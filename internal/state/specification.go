package state

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Specification kinds and the workspace directories that hold their
// immutable versions (ADR-0060).
const (
	CurriculumSpecs = "curricula"
	UnitSpecs       = "units"
)

var specSchemas = map[string]string{
	CurriculumSpecs: "curriculum-specification.schema.json",
	UnitSpecs:       "learning-unit-specification.schema.json",
}

const authoringPlansDir = "authoring-plans"

// SpecificationPath is the workspace-relative path of one immutable version
// of a Curriculum or Learning Unit Specification.
func SpecificationPath(kind, id string, version int) string {
	return fmt.Sprintf("specifications/%s/%s-v%d.json", kind, id, version)
}

// AuthoringPlanPath is the workspace-relative path of an Authoring Plan.
func AuthoringPlanPath(id string) string {
	return authoringPlansDir + "/" + id + ".json"
}

// AppendSpecification writes a new specification version. Versions are
// append-only: an existing version is never replaced.
func (s Store) AppendSpecification(expectedRevision, kind, id string, version int, data []byte) (string, error) {
	schema, ok := specSchemas[kind]
	if !ok {
		return "", fmt.Errorf("unknown specification kind %q", kind)
	}
	if err := s.requireRevision(expectedRevision); err != nil {
		return "", err
	}
	path := SpecificationPath(kind, id, version)
	if err := s.validate(schema, path, data); err != nil {
		return "", err
	}
	if err := appendFile(filepath.Join(s.Root, filepath.FromSlash(path)), data); err != nil {
		return "", err
	}
	return path, nil
}

// LatestSpecification returns the highest stored version of a specification,
// validated against its schema; found is false when none exists.
func (s Store) LatestSpecification(kind, id string) (data []byte, path string, found bool, err error) {
	schema, ok := specSchemas[kind]
	if !ok {
		return nil, "", false, fmt.Errorf("unknown specification kind %q", kind)
	}
	matches, err := filepath.Glob(filepath.Join(s.Root, "specifications", kind, id+"-v*.json"))
	if err != nil {
		return nil, "", false, fmt.Errorf("list specification %s: %w", id, err)
	}
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(id) + `-v([1-9][0-9]*)\.json$`)
	latest := 0
	for _, match := range matches {
		groups := pattern.FindStringSubmatch(filepath.Base(match))
		if groups == nil {
			continue
		}
		version, err := strconv.Atoi(groups[1])
		if err != nil {
			continue
		}
		if version > latest {
			latest = version
		}
	}
	if latest == 0 {
		return nil, "", false, nil
	}
	path = SpecificationPath(kind, id, latest)
	data, err = os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(path)))
	if err != nil {
		return nil, "", false, fmt.Errorf("read %s: %w", path, err)
	}
	if err := s.validate(schema, path, data); err != nil {
		return nil, "", false, err
	}
	return data, path, true, nil
}

// RecordAuthoringPlan writes an Authoring Plan unless it is already recorded.
// Plan IDs are content hashes, so an existing plan must hold identical bytes.
func (s Store) RecordAuthoringPlan(expectedRevision, id string, data []byte) (path string, created bool, err error) {
	path = AuthoringPlanPath(id)
	full := filepath.Join(s.Root, filepath.FromSlash(path))
	if existing, err := os.ReadFile(full); err == nil {
		if !bytes.Equal(existing, data) {
			return "", false, fmt.Errorf("authoring plan %s exists with different content; authoring plans are immutable", id)
		}
		return path, false, nil
	} else if !os.IsNotExist(err) {
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	if err := s.requireRevision(expectedRevision); err != nil {
		return "", false, err
	}
	if err := s.validate("authoring-plan.schema.json", path, data); err != nil {
		return "", false, err
	}
	if err := appendFile(full, data); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// RealizationsDir holds the append-only Realization Links of unit
// specification versions.
const RealizationsDir = "specifications/realizations"

// RealizationPath is the workspace-relative path of a Realization Link.
func RealizationPath(id string) string {
	return RealizationsDir + "/" + id + ".json"
}

// GateRecordPath is the workspace-relative path of a Platform Gate Record,
// stored with the Authoring Plan it is attached to.
func GateRecordPath(planID, id string) string {
	return authoringPlansDir + "/" + planID + "/gate-results/" + id + ".json"
}

// AuthoringPlan returns a recorded Authoring Plan, validated against its
// schema; found is false when no plan has that ID.
func (s Store) AuthoringPlan(id string) (data []byte, path string, found bool, err error) {
	path = AuthoringPlanPath(id)
	data, err = os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(path)))
	if os.IsNotExist(err) {
		return nil, path, false, nil
	}
	if err != nil {
		return nil, path, false, fmt.Errorf("read %s: %w", path, err)
	}
	if err := s.validate("authoring-plan.schema.json", path, data); err != nil {
		return nil, path, false, err
	}
	return data, path, true, nil
}

// Specification returns one stored specification version, validated against
// its schema; found is false when that version does not exist.
func (s Store) Specification(kind, id string, version int) (data []byte, path string, found bool, err error) {
	schema, ok := specSchemas[kind]
	if !ok {
		return nil, "", false, fmt.Errorf("unknown specification kind %q", kind)
	}
	path = SpecificationPath(kind, id, version)
	data, err = os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(path)))
	if os.IsNotExist(err) {
		return nil, path, false, nil
	}
	if err != nil {
		return nil, path, false, fmt.Errorf("read %s: %w", path, err)
	}
	if err := s.validate(schema, path, data); err != nil {
		return nil, path, false, err
	}
	return data, path, true, nil
}

// RecordGateResult writes a Platform Gate Record unless it is already
// recorded. Record IDs hash the plan, result and realization report, so an
// existing record must hold the same ID; it is never replaced.
func (s Store) RecordGateResult(expectedRevision, planID, id string, data []byte) (path string, existing []byte, err error) {
	path = GateRecordPath(planID, id)
	full := filepath.Join(s.Root, filepath.FromSlash(path))
	if stored, err := os.ReadFile(full); err == nil {
		return path, stored, nil
	} else if !os.IsNotExist(err) {
		return "", nil, fmt.Errorf("read %s: %w", path, err)
	}
	if err := s.requireRevision(expectedRevision); err != nil {
		return "", nil, err
	}
	if err := s.validate("platform-gate-record.schema.json", path, data); err != nil {
		return "", nil, err
	}
	if err := appendFile(full, data); err != nil {
		return "", nil, err
	}
	return path, nil, nil
}

// AppendRealization writes a new Realization Link; links are append-only.
func (s Store) AppendRealization(expectedRevision, id string, data []byte) (string, error) {
	if err := s.requireRevision(expectedRevision); err != nil {
		return "", err
	}
	path := RealizationPath(id)
	if err := s.validate("realization-link.schema.json", path, data); err != nil {
		return "", err
	}
	if err := appendFile(filepath.Join(s.Root, filepath.FromSlash(path)), data); err != nil {
		return "", err
	}
	return path, nil
}

// StoredRecord is one stored record with its workspace-relative path.
type StoredRecord struct {
	Path string
	Data []byte
}

// Realizations returns every recorded Realization Link, validated against its
// schema, in path order.
func (s Store) Realizations() ([]StoredRecord, error) {
	matches, err := filepath.Glob(filepath.Join(s.Root, filepath.FromSlash(RealizationsDir), "*.json"))
	if err != nil {
		return nil, fmt.Errorf("list realization links: %w", err)
	}
	sort.Strings(matches)
	records := make([]StoredRecord, 0, len(matches))
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", match, err)
		}
		path := RealizationPath(strings.TrimSuffix(filepath.Base(match), ".json"))
		if err := s.validate("realization-link.schema.json", path, data); err != nil {
			return nil, err
		}
		records = append(records, StoredRecord{Path: path, Data: data})
	}
	return records, nil
}
