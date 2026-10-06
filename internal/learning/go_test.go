package learning

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestPlanDoesNotDowngradeStrongButStaleCompetency(t *testing.T) {
	root := t.TempDir()
	writeLearning(t, root, "state/competencies.yaml", `schemaVersion: 1
competencies:
  - id: go.runtime.context
    domain: go
    level: strong
    confidence: high
    assessmentId: asmt_context
    lastVerified: 2025-01-01T00:00:00Z
`)
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	queue, plan, err := (GoPlanner{Root: root, Validator: validator, Registry: domain.NewRegistry()}).Build(
		time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC), 100,
	)
	if err != nil {
		t.Fatal(err)
	}
	foundReview := false
	for _, item := range queue.Items {
		if item.Competency == "go.runtime.context" {
			foundReview = true
		}
	}
	if !foundReview {
		t.Fatal("stale strong context competency should be due for review")
	}
	for _, item := range plan.Items {
		if item.Competency == "go.runtime.context" && item.Mode == "full" {
			t.Fatal("stale strong competency must not be downgraded to full reteaching")
		}
	}
}

func TestDiagnosticSkipsActivitiesWhoseCompetenciesAreStrong(t *testing.T) {
	root := t.TempDir()
	writeLearning(t, root, "state/competencies.yaml", `schemaVersion: 1
competencies:
  - id: go.language.slices
    domain: go
    level: strong
    confidence: high
    assessmentId: asmt_slices
    lastVerified: 2026-10-01T00:00:00Z
`)
	diagnostic, err := (GoPlanner{Root: root, Registry: domain.NewRegistry()}).Diagnostic(100)
	if err != nil {
		t.Fatal(err)
	}
	for _, activity := range diagnostic.Activities {
		if activity.ID == "go-diag-slice-aliasing" {
			t.Fatal("slice activity should be skipped when its only competency is strong")
		}
	}
}

func writeLearning(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
