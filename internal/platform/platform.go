// Package platform is ALP's platform-neutral adaptation contract (ADR-0054,
// ADR-0057). Adapters register in a Registry and declare the Platform
// Capabilities they support honestly; callers require a capability before
// using it. Nothing here knows any platform's native content model.
package platform

import (
	"fmt"
	"sort"
	"strings"
)

// Capability is a Platform Capability family an adapter may declare.
type Capability string

const (
	ActivitySource    Capability = "activity-source"
	CurriculumReader  Capability = "curriculum-reader"
	ContentMapper     Capability = "content-mapper"
	AuthoringTarget   Capability = "authoring-target"
	PlatformValidator Capability = "platform-validator"
)

// Capabilities lists every capability family in canonical order.
var Capabilities = []Capability{ActivitySource, CurriculumReader, ContentMapper, AuthoringTarget, PlatformValidator}

// ExternalID is an opaque, platform-owned identity namespaced by platform and
// Learning Target (ADR-0057). ALP compares these strings but never derives
// meaning from them.
type ExternalID struct {
	Platform string `json:"platform"`
	Target   string `json:"target"`
	Item     string `json:"item,omitempty"`
}

// Adapter is the umbrella Platform Adapter. Capability-specific behaviour is
// exposed through the capability interfaces (for example CurriculumSource).
type Adapter interface {
	// ID is the adapter identifier used with --adapter.
	ID() string
	// Capabilities returns the capability families the adapter declares.
	Capabilities() []Capability
	// StableIdentifierKinds names the platform item kinds whose identifiers the
	// adapter declares stable. Only these may be referenced by ALP.
	StableIdentifierKinds() []string
}

// CurriculumSource is implemented by adapters that declare CurriculumReader.
type CurriculumSource interface {
	// ReadCurriculum reads one Learning Target from a versioned curriculum export.
	ReadCurriculum(export []byte, target string) (Curriculum, error)
}

// Error is a platform contract failure with a stable machine-readable code.
type Error struct {
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	Adapter    string     `json:"adapter,omitempty"`
	Target     string     `json:"target,omitempty"`
	Capability Capability `json:"capability,omitempty"`
	Available  []string   `json:"available,omitempty"`
	Problems   []string   `json:"problems,omitempty"`
}

func (e *Error) Error() string { return e.Message }

const (
	CodeUnknownAdapter        = "unknown-adapter"
	CodeUnknownTarget         = "unknown-target"
	CodeCapabilityNotDeclared = "capability-not-declared"
	CodeInvalidExport         = "invalid-curriculum-export"
)

// Registry holds the adapters ALP can reach.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry builds a registry from adapters; duplicate IDs panic because
// they are a programming error in the composition root.
func NewRegistry(adapters ...Adapter) Registry {
	registry := Registry{adapters: make(map[string]Adapter, len(adapters))}
	for _, adapter := range adapters {
		if _, duplicate := registry.adapters[adapter.ID()]; duplicate {
			panic(fmt.Sprintf("platform adapter %q registered twice", adapter.ID()))
		}
		registry.adapters[adapter.ID()] = adapter
	}
	return registry
}

// IDs returns the registered adapter IDs in sorted order.
func (r Registry) IDs() []string {
	ids := make([]string, 0, len(r.adapters))
	for id := range r.adapters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Lookup returns the adapter registered under id.
func (r Registry) Lookup(id string) (Adapter, error) {
	adapter, ok := r.adapters[id]
	if !ok {
		return nil, &Error{
			Code:      CodeUnknownAdapter,
			Message:   fmt.Sprintf("unknown platform adapter %q (registered: %s)", id, strings.Join(r.IDs(), ", ")),
			Adapter:   id,
			Available: r.IDs(),
		}
	}
	return adapter, nil
}

// Require returns the adapter only if it declares capability.
func (r Registry) Require(id string, capability Capability) (Adapter, error) {
	adapter, err := r.Lookup(id)
	if err != nil {
		return nil, err
	}
	if !Declares(adapter, capability) {
		declared := DeclaredCapabilities(adapter)
		names := make([]string, len(declared))
		for i, value := range declared {
			names[i] = string(value)
		}
		return nil, &Error{
			Code:       CodeCapabilityNotDeclared,
			Message:    fmt.Sprintf("platform adapter %q does not declare the %s capability (declared: %s)", id, capability, strings.Join(names, ", ")),
			Adapter:    id,
			Capability: capability,
			Available:  names,
		}
	}
	return adapter, nil
}

// Declares reports whether adapter declares capability.
func Declares(adapter Adapter, capability Capability) bool {
	for _, declared := range adapter.Capabilities() {
		if declared == capability {
			return true
		}
	}
	return false
}

// DeclaredCapabilities returns the adapter's known declared capabilities in
// canonical order, without duplicates.
func DeclaredCapabilities(adapter Adapter) []Capability {
	var declared []Capability
	for _, capability := range Capabilities {
		if Declares(adapter, capability) {
			declared = append(declared, capability)
		}
	}
	return declared
}
