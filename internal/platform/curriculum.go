package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

// CurriculumExportSchema is the embedded JSON Schema for curriculum exports.
const CurriculumExportSchema = "platform-curriculum-export.schema.json"

// CurriculumExportVersion is the curriculum export schemaVersion ALP reads.
const CurriculumExportVersion = 1

// Curriculum is one Learning Target's structure as read from a versioned
// curriculum export (Q37). Item kinds and IDs are opaque platform strings.
type Curriculum struct {
	Target        ExternalID
	Title         string
	SchemaVersion int
	ContentHash   string
	Mapping       *MappingRef
	Phases        []Phase
	Items         []Item
}

// MappingRef points at the platform-owned content mapping for a target.
type MappingRef struct {
	Ref         string `json:"ref"`
	ContentHash string `json:"contentHash"`
}

// Phase is a platform-declared grouping of a target's items, in order.
type Phase struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// Item is a declared-stable platform item of a target.
type Item struct {
	Ref    ExternalID
	Kind   string
	Phase  string
	Title  string
	Parent *ExternalID
}

// ExportExpectation is what an adapter requires of a curriculum export.
type ExportExpectation struct {
	Platform    string
	Target      string
	StableKinds []string
}

type curriculumExport struct {
	SchemaVersion int            `json:"schemaVersion"`
	Platform      string         `json:"platform"`
	Targets       []exportTarget `json:"targets"`
}

type exportTarget struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	ContentHash string       `json:"contentHash"`
	Phases      []Phase      `json:"phases"`
	Items       []exportItem `json:"items"`
	Mapping     *MappingRef  `json:"mapping"`
}

type exportItem struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Phase  string `json:"phase"`
	Parent string `json:"parent"`
	Title  string `json:"title"`
}

// ReadCurriculumExport validates a curriculum export against the published
// schema and the adapter's expectation, and returns the requested target.
func ReadCurriculumExport(data []byte, expect ExportExpectation) (Curriculum, error) {
	validator, err := workspace.NewValidator()
	if err != nil {
		return Curriculum{}, fmt.Errorf("initialize validation: %w", err)
	}
	if issue := validator.ValidateDocument(CurriculumExportSchema, "curriculum-export.json", data); issue != nil {
		return Curriculum{}, invalidExport(expect, []string{issue.Error()})
	}

	var export curriculumExport
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&export); err != nil {
		return Curriculum{}, invalidExport(expect, []string{err.Error()})
	}
	if export.Platform != expect.Platform {
		return Curriculum{}, invalidExport(expect, []string{
			fmt.Sprintf("export is for platform %q, adapter reads platform %q", export.Platform, expect.Platform),
		})
	}

	var selected *exportTarget
	available := make([]string, 0, len(export.Targets))
	for i := range export.Targets {
		available = append(available, export.Targets[i].ID)
		if export.Targets[i].ID == expect.Target {
			if selected != nil {
				return Curriculum{}, invalidExport(expect, []string{fmt.Sprintf("target %q appears more than once", expect.Target)})
			}
			selected = &export.Targets[i]
		}
	}
	sort.Strings(available)
	if selected == nil {
		return Curriculum{}, &Error{
			Code:      CodeUnknownTarget,
			Message:   fmt.Sprintf("curriculum export has no target %q (available: %s)", expect.Target, strings.Join(available, ", ")),
			Adapter:   expect.Platform,
			Target:    expect.Target,
			Available: available,
		}
	}
	return buildCurriculum(export, *selected, expect)
}

func buildCurriculum(export curriculumExport, target exportTarget, expect ExportExpectation) (Curriculum, error) {
	namespace := ExternalID{Platform: export.Platform, Target: target.ID}
	stable := make(map[string]bool, len(expect.StableKinds))
	for _, kind := range expect.StableKinds {
		stable[kind] = true
	}

	var problems []string
	phases := make(map[string]bool, len(target.Phases))
	for _, phase := range target.Phases {
		if phases[phase.ID] {
			problems = append(problems, fmt.Sprintf("phase %q is declared more than once", phase.ID))
		}
		phases[phase.ID] = true
	}

	curriculum := Curriculum{
		Target:        namespace,
		Title:         target.Title,
		SchemaVersion: export.SchemaVersion,
		ContentHash:   target.ContentHash,
		Mapping:       target.Mapping,
		Phases:        append([]Phase{}, target.Phases...),
	}
	seen := make(map[string]bool, len(target.Items))
	for _, raw := range target.Items {
		if seen[raw.ID] {
			problems = append(problems, fmt.Sprintf("item %q is declared more than once", raw.ID))
		}
		if !stable[raw.Kind] {
			problems = append(problems, fmt.Sprintf("item %q has kind %q, which adapter %q does not declare stable", raw.ID, raw.Kind, expect.Platform))
		}
		if raw.Phase == "" && raw.Parent == "" {
			problems = append(problems, fmt.Sprintf("item %q has neither a phase nor a parent", raw.ID))
		}
		if raw.Phase != "" && !phases[raw.Phase] {
			problems = append(problems, fmt.Sprintf("item %q references undeclared phase %q", raw.ID, raw.Phase))
		}
		item := Item{Ref: namespace, Kind: raw.Kind, Phase: raw.Phase, Title: raw.Title}
		item.Ref.Item = raw.ID
		if raw.Parent != "" {
			if !seen[raw.Parent] {
				problems = append(problems, fmt.Sprintf("item %q references parent %q, which is not declared before it", raw.ID, raw.Parent))
			}
			parent := namespace
			parent.Item = raw.Parent
			item.Parent = &parent
		}
		seen[raw.ID] = true
		curriculum.Items = append(curriculum.Items, item)
	}
	if len(problems) != 0 {
		return Curriculum{}, invalidExport(expect, problems)
	}
	return curriculum, nil
}

func invalidExport(expect ExportExpectation, problems []string) error {
	return &Error{
		Code:     CodeInvalidExport,
		Message:  fmt.Sprintf("invalid curriculum export for target %q: %s", expect.Target, strings.Join(problems, "; ")),
		Adapter:  expect.Platform,
		Target:   expect.Target,
		Problems: problems,
	}
}
