package pylearn

import (
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
)

func validatedMapping(t *testing.T, adapter Adapter, mapping string) platform.MappingReport {
	t.Helper()
	curriculum := platform.Curriculum{
		Target: platform.ExternalID{Platform: AdapterID, Target: "go"},
		Items:  []platform.Item{{Ref: platform.ExternalID{Platform: AdapterID, Target: "go", Item: "go-context"}, Kind: "lesson", Phase: "B"}},
	}
	report, err := adapter.ValidateContentMapping([]byte(mapping), "mapping.yaml", curriculum)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

const contextMapping = `schemaVersion: 2
platform: pylearn
target: go
packs: [{domain: go, packVersion: ">=0.1.0 <0.2.0"}]
entries:
  - item: go-context
    competencies: [{id: go.runtime.context, role: assesses}]
`

func TestNormalizeIsIdempotentAndKeepsMasteryDerived(t *testing.T) {
	adapter := NewAdapter()
	mapping := validatedMapping(t, adapter, contextMapping)
	export := Export{
		SchemaVersion: 1,
		ExportedAt:    "2026-10-06T00:00:00Z",
		Attempts: []Attempt{
			{ID: "a1", ContentID: "go-context", Passed: true, CreatedAt: "2026-10-06T00:00:00Z"},
			{ID: "a1", ContentID: "go-context", Passed: true, CreatedAt: "2026-10-06T00:00:00Z"},
		},
		ConceptMastery: []ConceptMastery{
			{ID: "m1", ContentID: "go-context", Attempts: 3, Passes: 2, Failures: 1},
		},
	}
	result, err := adapter.Normalize(export, mapping, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence) != 1 {
		t.Fatalf("evidence count = %d", len(result.Evidence))
	}
	if record := result.Evidence[0].Record; record.Domain != "go" || len(record.Competencies) != 1 || record.Competencies[0] != "go.runtime.context" {
		t.Fatalf("evidence = %#v", record)
	}
	if len(result.DerivedSignals) != 1 || result.DerivedSignals[0].Kind != "concept-mastery-rollup" {
		t.Fatalf("signals = %#v", result.DerivedSignals)
	}
}

func TestNormalizeRefusesInvalidMapping(t *testing.T) {
	adapter := NewAdapter()
	mapping := validatedMapping(t, adapter, strings.Replace(contextMapping, "go.runtime.context", "go.not.real", 1))
	if _, err := adapter.Normalize(Export{SchemaVersion: 1}, mapping, nil); err == nil {
		t.Fatal("expected invalid mapping to be refused")
	}
}

func TestProfileConflictIsReportedNotApplied(t *testing.T) {
	conflicts := DetectProfileConflicts(
		map[string]any{"experience": map[string]any{"python": map[string]any{"level": "functional"}}},
		map[string]any{"experience": map[string]any{"python": map[string]any{"level": "strong"}}},
	)
	if len(conflicts) != 1 || conflicts[0].Field != "experience.python.level" {
		t.Fatalf("conflicts = %#v", conflicts)
	}
}
