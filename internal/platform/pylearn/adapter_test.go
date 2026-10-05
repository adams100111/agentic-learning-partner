package pylearn

import "testing"

func TestNormalizeIsIdempotentAndKeepsMasteryDerived(t *testing.T) {
	adapter := NewAdapter()
	mapping := Mapping{
		SchemaVersion: 1,
		Platform:      "pylearn",
		Mappings: []ContentMapping{
			{ContentID: "go-context", Domain: "go", PackVersion: ">=0.1 <0.2", Competencies: []string{"go.runtime.context"}},
		},
	}
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
	if len(result.DerivedSignals) != 1 || result.DerivedSignals[0].Kind != "concept-mastery-rollup" {
		t.Fatalf("signals = %#v", result.DerivedSignals)
	}
}

func TestMappingRejectsUnknownCompetency(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.ValidateMapping(Mapping{
		SchemaVersion: 1,
		Platform:      "pylearn",
		Mappings: []ContentMapping{
			{ContentID: "bad", Domain: "go", PackVersion: ">=0.1 <0.2", Competencies: []string{"go.not.real"}},
		},
	})
	if err == nil {
		t.Fatal("expected unknown competency error")
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
