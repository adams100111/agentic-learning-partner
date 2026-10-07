package state

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
