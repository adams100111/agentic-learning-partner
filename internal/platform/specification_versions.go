package platform

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
)

// Specification versioning: reconciling a draft with the latest stored
// version under the regeneration policy, and the content-hashed, immutable
// encoding of stored versions (ADR-0060).

// UnitRevision is a unit draft reconciled with its latest stored version.
type UnitRevision struct {
	Spec   LearningUnitSpec
	Status string
	// HeldFields are the material fields that changed without new evidence
	// for a realized unit, so its stored version was kept (SpecHeld).
	HeldFields []string
}

// ReviseUnitSpec reconciles a unit draft with the latest stored version under
// the regeneration policy: a new version only when a material field changed
// (competencies, prerequisites, adaptation mode, misconceptions, required
// evidence, teaching shape). Otherwise the stored version stands, with its
// provenance. Once a unit is realized its content exists on the platform, so a
// material change produces a new version (and so a content change) only when
// it is evidence-linked: the learner state changed with it. Other material
// changes to a realized unit are held.
func ReviseUnitSpec(draft LearningUnitSpec, latest *LearningUnitSpec, realized bool) (UnitRevision, error) {
	if latest == nil {
		draft.Version = 1
		hash, err := specHash(draft)
		draft.ContentHash = hash
		return UnitRevision{Spec: draft, Status: SpecCreated}, err
	}
	changed := changedFields([]materialField{
		{"competencies", materialCompetencies(draft.Teaching.Competencies), materialCompetencies(latest.Teaching.Competencies)},
		{"prerequisites", materialPrerequisites(draft.Teaching.Prerequisites), materialPrerequisites(latest.Teaching.Prerequisites)},
		{"adaptationMode", draft.Adaptation.Mode, latest.Adaptation.Mode},
		{"misconceptions", materialMisconceptions(draft.Adaptation.Misconceptions), materialMisconceptions(latest.Adaptation.Misconceptions)},
		{"requiredEvidence", draft.Teaching.RequiredEvidence, latest.Teaching.RequiredEvidence},
		{"teachingShape", materialShape(draft), materialShape(*latest)},
	})
	if len(changed) == 0 {
		return UnitRevision{Spec: *latest, Status: SpecUnchanged}, nil
	}
	evidenceLinked := draft.Provenance.LearnerStateRevision != latest.Provenance.LearnerStateRevision
	if realized && !evidenceLinked {
		return UnitRevision{Spec: *latest, Status: SpecHeld, HeldFields: changed}, nil
	}
	draft.Version = latest.Version + 1
	draft.Supersedes = &SpecVersionRef{Version: latest.Version, ContentHash: latest.ContentHash}
	draft.Change = &SpecChange{MaterialFields: changed, EvidenceLinked: evidenceLinked}
	hash, err := specHash(draft)
	draft.ContentHash = hash
	return UnitRevision{Spec: draft, Status: SpecRevised}, err
}

// ReviseCurriculumSpec reconciles a composed curriculum with the latest stored
// version: a new version only when its goal, groups, unit versions or sequence changed.
func ReviseCurriculumSpec(draft CurriculumSpec, latest *CurriculumSpec) (CurriculumSpec, string, error) {
	if latest == nil {
		draft.Version = 1
		hash, err := specHash(draft)
		draft.ContentHash = hash
		return draft, SpecCreated, err
	}
	changed := changedFields([]materialField{
		{"goal", draft.Goal, latest.Goal},
		{"groups", draft.Groups, latest.Groups},
		{"units", materialUnits(draft.Units), materialUnits(latest.Units)},
		{"sequence", draft.Sequence, latest.Sequence},
	})
	if len(changed) == 0 {
		return *latest, SpecUnchanged, nil
	}
	draft.Version = latest.Version + 1
	draft.Supersedes = &SpecVersionRef{Version: latest.Version, ContentHash: latest.ContentHash}
	draft.Change = &SpecChange{MaterialFields: changed, EvidenceLinked: draft.Provenance.LearnerStateRevision != latest.Provenance.LearnerStateRevision}
	hash, err := specHash(draft)
	draft.ContentHash = hash
	return draft, SpecRevised, err
}

// specHash hashes a specification's canonical JSON without its content hash.
func specHash(spec any) (string, error) {
	value := reflect.ValueOf(&spec).Elem().Elem()
	copied := reflect.New(value.Type()).Elem()
	copied.Set(value)
	if field := copied.FieldByName("ContentHash"); field.IsValid() {
		field.SetString("")
	}
	data, err := json.Marshal(copied.Interface())
	if err != nil {
		return "", fmt.Errorf("encode specification: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// EncodeSpec renders a specification or Authoring Plan as deterministic JSON.
func EncodeSpec(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("encode specification: %w", err)
	}
	return buffer.Bytes(), nil
}

// DecodeUnitSpec parses a stored Learning Unit Specification and verifies its
// content hash: stored versions are immutable.
func DecodeUnitSpec(path string, data []byte) (LearningUnitSpec, error) {
	var spec LearningUnitSpec
	if err := decodeStoredSpec(path, data, &spec); err != nil {
		return LearningUnitSpec{}, err
	}
	return spec, verifySpecHash(path, spec, spec.ContentHash)
}

// DecodeCurriculumSpec parses a stored Curriculum Specification and verifies
// its content hash.
func DecodeCurriculumSpec(path string, data []byte) (CurriculumSpec, error) {
	var spec CurriculumSpec
	if err := decodeStoredSpec(path, data, &spec); err != nil {
		return CurriculumSpec{}, err
	}
	return spec, verifySpecHash(path, spec, spec.ContentHash)
}

func decodeStoredSpec(path string, data []byte, into any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		return &Error{Code: CodeSpecIntegrity, Message: fmt.Sprintf("stored specification %s cannot be read: %v", path, err)}
	}
	return nil
}

func verifySpecHash(path string, spec any, recorded string) error {
	hash, err := specHash(spec)
	if err != nil {
		return err
	}
	if hash != recorded {
		return &Error{Code: CodeSpecIntegrity, Message: fmt.Sprintf(
			"stored specification %s was modified: its content hashes to %s, not its recorded %s; specification versions are immutable", path, hash, recorded)}
	}
	return nil
}
