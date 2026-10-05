package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

type Catalog interface {
	HasCompetency(domain, id string) bool
}

type ProductionGate interface {
	ApproveProductionReady(assessment Assessment, evidence []Evidence) error
}

type Store struct {
	Root           string
	Catalog        Catalog
	ProductionGate ProductionGate
	Validator      *workspace.Validator
}

func (s Store) AppendEvidence(expectedRevision string, evidence Evidence) (Evidence, error) {
	if err := s.requireRevision(expectedRevision); err != nil {
		return Evidence{}, err
	}
	if evidence.ID == "" {
		id, err := newID("ev")
		if err != nil {
			return Evidence{}, err
		}
		evidence.ID = id
	}
	if evidence.SchemaVersion == 0 {
		evidence.SchemaVersion = 1
	}
	for _, competency := range evidence.Competencies {
		if s.Catalog == nil || !s.Catalog.HasCompetency(evidence.Domain, competency) {
			return Evidence{}, fmt.Errorf("unknown competency %q for domain %q", competency, evidence.Domain)
		}
	}
	data, err := yaml.Marshal(evidence)
	if err != nil {
		return Evidence{}, fmt.Errorf("marshal evidence %s: %w", evidence.ID, err)
	}
	if err := s.validate("evidence.schema.json", evidence.ID+".yaml", data); err != nil {
		return Evidence{}, err
	}
	if err := appendFile(filepath.Join(s.Root, "evidence", evidence.ID+".yaml"), data); err != nil {
		return Evidence{}, err
	}
	return evidence, nil
}

func (s Store) AppendAssessment(expectedRevision string, assessment Assessment) (Assessment, error) {
	if err := s.requireRevision(expectedRevision); err != nil {
		return Assessment{}, err
	}
	if assessment.ID == "" {
		id, err := newID("asmt")
		if err != nil {
			return Assessment{}, err
		}
		assessment.ID = id
	}
	if assessment.SchemaVersion == 0 {
		assessment.SchemaVersion = 1
	}
	if s.Catalog == nil || !s.Catalog.HasCompetency(assessment.Domain, assessment.Competency) {
		return Assessment{}, fmt.Errorf("unknown competency %q for domain %q", assessment.Competency, assessment.Domain)
	}
	evidence, err := s.loadEvidence(assessment.Evidence)
	if err != nil {
		return Assessment{}, err
	}
	for _, record := range evidence {
		if record.Domain != assessment.Domain {
			return Assessment{}, fmt.Errorf("assessment %s references evidence %s from domain %q", assessment.ID, record.ID, record.Domain)
		}
	}

	if assessment.Status == "" || assessment.Status == "proposed" {
		if assessment.Judgment.Level == "production-ready" {
			if s.ProductionGate == nil {
				return Assessment{}, fmt.Errorf("production-ready assessment %s requires a production gate", assessment.ID)
			}
			if err := s.ProductionGate.ApproveProductionReady(assessment, evidence); err != nil {
				return Assessment{}, fmt.Errorf("production-ready assessment %s rejected: %w", assessment.ID, err)
			}
		}
		assessment.Status = "accepted"
	}
	if assessment.Status == "accepted" && assessment.Judgment.Level == "production-ready" && s.ProductionGate == nil {
		return Assessment{}, fmt.Errorf("production-ready assessment %s requires a production gate", assessment.ID)
	}

	if err := s.ensureAssessmentReferences(assessment.Supersedes); err != nil {
		return Assessment{}, err
	}
	assessment.WorkspaceRevision = expectedRevision

	data, err := yaml.Marshal(assessment)
	if err != nil {
		return Assessment{}, fmt.Errorf("marshal assessment %s: %w", assessment.ID, err)
	}
	if err := s.validate("assessment.schema.json", assessment.ID+".yaml", data); err != nil {
		return Assessment{}, err
	}
	if err := appendFile(filepath.Join(s.Root, "assessments", assessment.ID+".yaml"), data); err != nil {
		return Assessment{}, err
	}
	return assessment, nil
}

func (s Store) RebuildProjection() (Projection, error) {
	assessments, err := s.loadAssessments()
	if err != nil {
		return Projection{}, err
	}

	superseded := map[string]struct{}{}
	for _, assessment := range assessments {
		if assessment.Status != "accepted" {
			continue
		}
		for _, id := range assessment.Supersedes {
			superseded[id] = struct{}{}
		}
	}

	active := map[string][]Assessment{}
	for _, assessment := range assessments {
		if assessment.Status != "accepted" {
			continue
		}
		if _, removed := superseded[assessment.ID]; removed {
			continue
		}
		key := assessment.Domain + "\x00" + assessment.Competency
		active[key] = append(active[key], assessment)
	}

	projection := Projection{SchemaVersion: 1}
	for _, group := range active {
		sort.Slice(group, func(i, j int) bool {
			if group[i].RecordedAt == group[j].RecordedAt {
				return group[i].ID < group[j].ID
			}
			return group[i].RecordedAt < group[j].RecordedAt
		})
		latest := group[len(group)-1]
		confidence := latest.Confidence
		needsReassessment := false
		if len(group) > 1 {
			previous := group[len(group)-2]
			if previous.Judgment.Level != latest.Judgment.Level {
				confidence = lowerConfidence(confidence)
				needsReassessment = true
			}
		}
		projection.Competencies = append(projection.Competencies, ProjectedCompetency{
			ID:                latest.Competency,
			Domain:            latest.Domain,
			Level:             latest.Judgment.Level,
			Confidence:        confidence,
			AssessmentID:      latest.ID,
			LastVerified:      latest.RecordedAt,
			NeedsReassessment: needsReassessment,
			Gaps:              append([]string(nil), latest.Judgment.Gaps...),
		})
	}

	sort.Slice(projection.Competencies, func(i, j int) bool {
		return projection.Competencies[i].ID < projection.Competencies[j].ID
	})
	data, err := yaml.Marshal(projection)
	if err != nil {
		return Projection{}, fmt.Errorf("marshal projection: %w", err)
	}
	if err := s.validate("projection.schema.json", "state/competencies.yaml", data); err != nil {
		return Projection{}, err
	}
	if err := writeGenerated(filepath.Join(s.Root, "state", "competencies.yaml"), data); err != nil {
		return Projection{}, err
	}
	return projection, nil
}

func (s Store) requireRevision(expected string) error {
	if strings.TrimSpace(expected) == "" {
		return errors.New("expected workspace revision is required")
	}
	info, err := workspace.Inspect(s.Root)
	if err != nil {
		return err
	}
	if info.Revision != expected {
		return fmt.Errorf("workspace revision changed: expected %s, found %s", expected, info.Revision)
	}
	return nil
}

func (s Store) validate(schemaName, file string, data []byte) error {
	if s.Validator == nil {
		return errors.New("workspace validator is required")
	}
	if issue := s.Validator.ValidateDocument(schemaName, file, data); issue != nil {
		return issue
	}
	return nil
}

func (s Store) loadEvidence(ids []string) ([]Evidence, error) {
	result := make([]Evidence, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf("duplicate evidence reference %q", id)
		}
		seen[id] = struct{}{}
		data, err := os.ReadFile(filepath.Join(s.Root, "evidence", id+".yaml"))
		if err != nil {
			return nil, fmt.Errorf("read evidence %s: %w", id, err)
		}
		var record Evidence
		if err := yaml.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("parse evidence %s: %w", id, err)
		}
		result = append(result, record)
	}
	return result, nil
}

func (s Store) ensureAssessmentReferences(ids []string) error {
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, duplicate := seen[id]; duplicate {
			return fmt.Errorf("duplicate superseded assessment %q", id)
		}
		seen[id] = struct{}{}
		if _, err := os.Stat(filepath.Join(s.Root, "assessments", id+".yaml")); err != nil {
			return fmt.Errorf("superseded assessment %s: %w", id, err)
		}
	}
	return nil
}

func (s Store) loadAssessments() ([]Assessment, error) {
	paths, err := filepath.Glob(filepath.Join(s.Root, "assessments", "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("list assessments: %w", err)
	}
	sort.Strings(paths)
	result := make([]Assessment, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var assessment Assessment
		if err := yaml.Unmarshal(data, &assessment); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		result = append(result, assessment)
	}
	return result, nil
}

func appendFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("append-only record already exists: %s", filepath.Base(path))
		}
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeGenerated(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o644); err != nil {
		return fmt.Errorf("write generated projection: %w", err)
	}
	if err := os.Rename(temp, path); err != nil {
		return fmt.Errorf("replace generated projection: %w", err)
	}
	return nil
}

func lowerConfidence(value string) string {
	switch value {
	case "high":
		return "medium"
	case "medium":
		return "low"
	default:
		return "low"
	}
}
