package view

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompetencyViewExplainsAssessmentAndEvidence(t *testing.T) {
	root := t.TempDir()
	writeView(t, root, "state/competencies.yaml", `schemaVersion: 1
competencies:
  - id: go.runtime.context
    domain: go
    level: functional
    confidence: high
    assessmentId: asmt_context
    lastVerified: 2026-10-06T00:00:00Z
`)
	writeView(t, root, "assessments/asmt_context.yaml", `schemaVersion: 1
id: asmt_context
recordedAt: 2026-10-06T00:00:00Z
domain: go
competency: go.runtime.context
evidence: [ev_context]
rubric:
  id: go.runtime.context
  version: "1"
assessor:
  type: agent
  id: test
judgment:
  level: functional
confidence: high
rationale: Correct cancellation reasoning.
status: accepted
`)
	output, err := (Renderer{Root: root}).Competency("go.runtime.context", Markdown)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"functional", "Correct cancellation reasoning.", "ev_context"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output missing %q:\n%s", expected, output)
		}
	}
}

func TestPersonaViewShowsGlobalAndDomainPersona(t *testing.T) {
	root := t.TempDir()
	writeView(t, root, "profile/profile.yaml", `schemaVersion: 1
learner:
  id: test
  professionalLevel: senior
experience:
  php:
    level: expert
    relativeRank: 1
goals: []
preferences:
  preferRealProjects: true
`)
	writeView(t, root, "personas/global.yaml", `schemaVersion: 1
scope: global
domain: null
teaching:
  pace: senior-dense
`)
	writeView(t, root, "personas/domains/go.yaml", `schemaVersion: 1
scope: domain
domain: go
teaching:
  emphasis: [concurrency]
`)
	output, err := (Renderer{Root: root}).Persona("go", Text)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"php: expert", "Global persona", "Domain persona: go"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output missing %q:\n%s", expected, output)
		}
	}
}

func writeView(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
