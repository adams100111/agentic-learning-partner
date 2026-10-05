package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestBuilderProjectsRelevantContext(t *testing.T) {
	root := t.TempDir()
	writeContext(t, root, "profile/profile.yaml", `schemaVersion: 1
learner:
  id: test
  professionalLevel: senior
experience:
  php:
    level: expert
    relativeRank: 1
  typescript:
    level: strong
    relativeRank: 2
preferences:
  preferRealProjects: true
goals: []
`)
	writeContext(t, root, "personas/global.yaml", `schemaVersion: 1
scope: global
domain: null
teaching:
  pace: senior-dense
  avoid: [toy-examples]
`)
	writeContext(t, root, "personas/domains/go.yaml", `schemaVersion: 1
scope: domain
domain: go
teaching:
  pace: senior-dense
  emphasis: [concurrency]
`)
	writeContext(t, root, "state/competencies.yaml", `schemaVersion: 1
competencies:
  - id: go.runtime.context
    domain: go
    level: functional
    confidence: high
    assessmentId: asmt_1
    lastVerified: 2026-10-06T00:00:00Z
  - id: python.fastapi.routing
    domain: python
    level: strong
    confidence: high
    assessmentId: asmt_2
    lastVerified: 2026-10-06T00:00:00Z
`)

	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := (Builder{Root: root, Validator: validator}).Build(Request{
		Task:       "teach",
		Domain:     "go",
		Competency: "go.runtime.context",
	})
	if err != nil {
		t.Fatal(err)
	}
	items, ok := bundle.Competencies["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("competencies = %#v", bundle.Competencies)
	}
	if bundle.EstimatedTokens == nil || *bundle.EstimatedTokens <= 0 {
		t.Fatalf("estimated tokens = %#v", bundle.EstimatedTokens)
	}
	for _, source := range bundle.IncludedSources {
		if source.Ref == "evidence/" {
			t.Fatal("full evidence history must not be included")
		}
	}
}

func writeContext(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
