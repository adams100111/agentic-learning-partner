package state

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

type fakeCatalog map[string]bool

func (c fakeCatalog) HasCompetency(domain, id string) bool {
	return c[domain+"|"+id]
}

type allowProduction struct{}

func (allowProduction) ApproveProductionReady(Assessment, []Evidence) error { return nil }

type denyProduction struct{}

func (denyProduction) ApproveProductionReady(Assessment, []Evidence) error {
	return errors.New("missing hard evidence")
}

func TestStoreAppendAndDeterministicProjection(t *testing.T) {
	root, revision := makeStateWorkspace(t)
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	store := Store{
		Root:      root,
		Catalog:   fakeCatalog{"go|go.runtime.context": true},
		Validator: validator,
	}

	evidence, err := store.AppendEvidence(revision, Evidence{
		SchemaVersion: 1,
		ID:            "ev_context",
		RecordedAt:    "2026-10-06T00:00:00Z",
		Domain:        "go",
		Competencies:  []string{"go.runtime.context"},
		Type:          "exercise",
		Source:        EvidenceSource{Kind: "diagnostic"},
		Observation:   "Explained cancellation propagation correctly.",
		Result:        "pass",
		Strength:      "strong",
	})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.ID != "ev_context" {
		t.Fatalf("id = %q", evidence.ID)
	}

	first, err := store.AppendAssessment(revision, Assessment{
		SchemaVersion: 1,
		ID:            "asmt_context_1",
		RecordedAt:    "2026-10-06T00:01:00Z",
		Domain:        "go",
		Competency:    "go.runtime.context",
		Evidence:      []string{"ev_context"},
		Rubric:        RubricRef{ID: "go.runtime.context", Version: "1"},
		Assessor:      Assessor{Type: "agent", ID: "test"},
		Judgment:      Judgment{Level: "functional"},
		Confidence:    "high",
		Rationale:     "Independent explanation.",
		Status:        "proposed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "accepted" {
		t.Fatalf("status = %q", first.Status)
	}

	if _, err := store.AppendAssessment(revision, Assessment{
		SchemaVersion: 1,
		ID:            "asmt_context_2",
		RecordedAt:    "2026-10-06T00:02:00Z",
		Domain:        "go",
		Competency:    "go.runtime.context",
		Evidence:      []string{"ev_context"},
		Rubric:        RubricRef{ID: "go.runtime.context", Version: "1"},
		Assessor:      Assessor{Type: "agent", ID: "test"},
		Judgment:      Judgment{Level: "rusty", Gaps: []string{"deadline ownership"}},
		Confidence:    "high",
		Rationale:     "Later implementation exposed a conceptual gap.",
		Status:        "proposed",
	}); err != nil {
		t.Fatal(err)
	}

	projection, err := store.RebuildProjection()
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Competencies) != 1 {
		t.Fatalf("projection = %#v", projection)
	}
	got := projection.Competencies[0]
	if got.Level != "rusty" || got.Confidence != "medium" || !got.NeedsReassessment {
		t.Fatalf("projected competency = %#v", got)
	}

	path := filepath.Join(root, "state", "competencies.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RebuildProjection(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("projection rebuild is not deterministic\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestStoreRejectsDuplicateAndStaleWrites(t *testing.T) {
	root, revision := makeStateWorkspace(t)
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	store := Store{
		Root:      root,
		Catalog:   fakeCatalog{"go|go.runtime.context": true},
		Validator: validator,
	}
	record := Evidence{
		SchemaVersion: 1,
		ID:            "ev_duplicate",
		RecordedAt:    "2026-10-06T00:00:00Z",
		Domain:        "go",
		Competencies:  []string{"go.runtime.context"},
		Type:          "exercise",
		Source:        EvidenceSource{Kind: "diagnostic"},
		Observation:   "Observation",
		Strength:      "moderate",
	}
	if _, err := store.AppendEvidence(revision, record); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendEvidence(revision, record); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate error = %v", err)
	}
	if _, err := store.AppendEvidence("stale-revision", Evidence{
		SchemaVersion: 1,
		RecordedAt:    "2026-10-06T00:03:00Z",
		Domain:        "go",
		Competencies:  []string{"go.runtime.context"},
		Type:          "exercise",
		Source:        EvidenceSource{Kind: "diagnostic"},
		Observation:   "Another observation",
		Strength:      "moderate",
	}); err == nil || !strings.Contains(err.Error(), "revision changed") {
		t.Fatalf("stale revision error = %v", err)
	}
}

func TestProductionReadyRequiresGate(t *testing.T) {
	root, revision := makeStateWorkspace(t)
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	base := Store{
		Root:      root,
		Catalog:   fakeCatalog{"go|go.runtime.context": true},
		Validator: validator,
	}
	if _, err := base.AppendEvidence(revision, Evidence{
		SchemaVersion: 1,
		ID:            "ev_prod",
		RecordedAt:    "2026-10-06T00:00:00Z",
		Domain:        "go",
		Competencies:  []string{"go.runtime.context"},
		Type:          "project-implementation",
		Source:        EvidenceSource{Kind: "repository"},
		Observation:   "Production implementation evidence.",
		Strength:      "production",
	}); err != nil {
		t.Fatal(err)
	}

	assessment := Assessment{
		SchemaVersion: 1,
		ID:            "asmt_prod",
		RecordedAt:    "2026-10-06T00:01:00Z",
		Domain:        "go",
		Competency:    "go.runtime.context",
		Evidence:      []string{"ev_prod"},
		Rubric:        RubricRef{ID: "go.runtime.context", Version: "1"},
		Assessor:      Assessor{Type: "agent", ID: "test"},
		Judgment:      Judgment{Level: "production-ready"},
		Confidence:    "high",
		Rationale:     "Candidate production-ready assessment.",
		Status:        "proposed",
	}

	if _, err := base.AppendAssessment(revision, assessment); err == nil || !strings.Contains(err.Error(), "requires a production gate") {
		t.Fatalf("missing gate error = %v", err)
	}

	base.ProductionGate = denyProduction{}
	if _, err := base.AppendAssessment(revision, assessment); err == nil || !strings.Contains(err.Error(), "missing hard evidence") {
		t.Fatalf("denied gate error = %v", err)
	}

	base.ProductionGate = allowProduction{}
	accepted, err := base.AppendAssessment(revision, assessment)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != "accepted" {
		t.Fatalf("status = %q", accepted.Status)
	}
}

func makeStateWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	runStateGit(t, root, "init")
	runStateGit(t, root, "config", "user.email", "alp@example.invalid")
	runStateGit(t, root, "config", "user.name", "ALP Test")
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 1\nlearnerId: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runStateGit(t, root, "add", "workspace.yaml")
	runStateGit(t, root, "commit", "-m", "init")

	info, err := workspace.Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, info.Revision
}

func runStateGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
