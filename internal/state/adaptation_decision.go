package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

// AdaptationDecision is an Accepted Adaptation Decision (ADR-0060, ADR-0061):
// a canonical, append-only record of an adaptation choice the learner
// explicitly confirmed for one unit of a shared Learning Target. It is an
// input to the Target Adaptation Projection, never part of it, so it survives
// projection rebuilds. A decision is revoked or replaced only by a later
// record that supersedes it.
type AdaptationDecision struct {
	SchemaVersion int                  `yaml:"schemaVersion" json:"schemaVersion"`
	ID            string               `yaml:"id" json:"id"`
	RecordedAt    string               `yaml:"recordedAt" json:"recordedAt"`
	LearnerID     string               `yaml:"learnerId" json:"learnerId"`
	Target        DecisionTarget       `yaml:"target" json:"target"`
	Unit          string               `yaml:"unit,omitempty" json:"unit,omitempty"`
	Action        string               `yaml:"action" json:"action"`
	Mode          string               `yaml:"mode,omitempty" json:"mode,omitempty"`
	Reason        string               `yaml:"reason,omitempty" json:"reason,omitempty"`
	Basis         DecisionBasis        `yaml:"basis" json:"basis"`
	Supersedes    []string             `yaml:"supersedes,omitempty" json:"supersedes,omitempty"`
	Confirmation  DecisionConfirmation `yaml:"confirmation" json:"confirmation"`
}

// DecisionTarget is the shared Learning Target a decision applies to.
type DecisionTarget struct {
	Platform string `yaml:"platform" json:"platform"`
	Target   string `yaml:"target" json:"target"`
}

// DecisionBasis is the Target Adaptation Projection revision the learner saw
// when confirming the decision.
type DecisionBasis struct {
	ProjectionRevision string `yaml:"projectionRevision" json:"projectionRevision"`
}

// DecisionConfirmation records who confirmed a decision and when.
type DecisionConfirmation struct {
	ConfirmedBy string `yaml:"confirmedBy" json:"confirmedBy"`
	ConfirmedAt string `yaml:"confirmedAt" json:"confirmedAt"`
}

// Adaptation decision actions.
const (
	DecisionAccept = "accept"
	DecisionRevoke = "revoke"
)

const adaptationDecisionsDir = "adaptation-decisions"

// AppendAdaptationDecision appends a learner-confirmed decision. Every
// superseded decision must exist, be active, and belong to the same target.
func (s Store) AppendAdaptationDecision(expectedRevision string, decision AdaptationDecision) (AdaptationDecision, error) {
	if err := s.requireRevision(expectedRevision); err != nil {
		return AdaptationDecision{}, err
	}
	if decision.Confirmation.ConfirmedBy != LearnerConfirmation {
		return AdaptationDecision{}, fmt.Errorf("accepted adaptation decision requires learner confirmation")
	}
	existing, err := s.AdaptationDecisions()
	if err != nil {
		return AdaptationDecision{}, err
	}
	byID := make(map[string]AdaptationDecision, len(existing))
	superseded := map[string]bool{}
	for _, record := range existing {
		byID[record.ID] = record
		for _, id := range record.Supersedes {
			superseded[id] = true
		}
	}
	for _, id := range decision.Supersedes {
		previous, ok := byID[id]
		switch {
		case !ok:
			return AdaptationDecision{}, fmt.Errorf("superseded adaptation decision %s does not exist", id)
		case superseded[id]:
			return AdaptationDecision{}, fmt.Errorf("adaptation decision %s is already superseded", id)
		case previous.Target != decision.Target:
			return AdaptationDecision{}, fmt.Errorf("adaptation decision %s belongs to another Learning Target", id)
		}
	}
	if decision.SchemaVersion == 0 {
		decision.SchemaVersion = 1
	}
	if decision.ID == "" {
		id, err := newID("aad")
		if err != nil {
			return AdaptationDecision{}, err
		}
		decision.ID = id
	}
	data, err := yaml.Marshal(decision)
	if err != nil {
		return AdaptationDecision{}, fmt.Errorf("marshal adaptation decision %s: %w", decision.ID, err)
	}
	if err := s.validate("accepted-adaptation-decision.schema.json", decision.ID+".yaml", data); err != nil {
		return AdaptationDecision{}, err
	}
	if err := appendFile(filepath.Join(s.Root, adaptationDecisionsDir, decision.ID+".yaml"), data); err != nil {
		return AdaptationDecision{}, err
	}
	return decision, nil
}

// AdaptationDecisions returns every Accepted Adaptation Decision record,
// ordered by ID.
func (s Store) AdaptationDecisions() ([]AdaptationDecision, error) {
	paths, err := filepath.Glob(filepath.Join(s.Root, adaptationDecisionsDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("list adaptation decisions: %w", err)
	}
	sort.Strings(paths)
	decisions := make([]AdaptationDecision, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if err := s.validate("accepted-adaptation-decision.schema.json", filepath.Base(path), data); err != nil {
			return nil, err
		}
		var decision AdaptationDecision
		if err := yaml.Unmarshal(data, &decision); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		decisions = append(decisions, decision)
	}
	return decisions, nil
}

// TargetAdaptationID is the deterministic ID of a learner's Target Adaptation
// Projection for one shared Learning Target; target identities stay opaque.
func TargetAdaptationID(platform, target string) string {
	sum := sha256.Sum256([]byte(platform + "\x00" + target))
	return "tap_" + hex.EncodeToString(sum[:12])
}

// TargetAdaptationPath is the workspace-relative path of a generated Target
// Adaptation Projection.
func TargetAdaptationPath(platform, target string) string {
	return "state/target-adaptations/" + TargetAdaptationID(platform, target) + ".json"
}

// WriteTargetAdaptation validates and writes a generated Target Adaptation
// Projection. It is derived state: it is replaced wholesale on every rebuild.
func (s Store) WriteTargetAdaptation(platform, target string, data []byte) error {
	path := TargetAdaptationPath(platform, target)
	if err := s.validate("target-adaptation-projection.schema.json", path, data); err != nil {
		return err
	}
	return writeGenerated(filepath.Join(s.Root, filepath.FromSlash(path)), data)
}
