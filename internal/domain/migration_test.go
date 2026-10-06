package domain

import "testing"

func TestResolveCompetencyMigrations(t *testing.T) {
	pack := Pack{
		SchemaVersion: 1,
		Domain:        "go",
		Version:       "2.0.0",
		Competencies: []Competency{
			{ID: "go.new", Name: "New", Dimension: "runtime", Description: "new", FreshnessClass: "stable-concept"},
			{ID: "go.part.a", Name: "A", Dimension: "runtime", Description: "a", FreshnessClass: "stable-concept"},
			{ID: "go.part.b", Name: "B", Dimension: "runtime", Description: "b", FreshnessClass: "stable-concept"},
		},
		Migrations: []Migration{
			{From: "go.old", To: []string{"go.new"}, Strategy: "rename"},
			{From: "go.old-split", To: []string{"go.part.a", "go.part.b"}, Strategy: "split"},
			{From: "go.removed", Strategy: "remove"},
		},
	}
	if err := pack.Validate(); err != nil {
		t.Fatal(err)
	}
	rename, err := pack.ResolveCompetency("go.old")
	if err != nil || len(rename.Targets) != 1 || rename.RequiresReassessment {
		t.Fatalf("rename=%#v err=%v", rename, err)
	}
	split, err := pack.ResolveCompetency("go.old-split")
	if err != nil || !split.RequiresReassessment || len(split.Targets) != 2 {
		t.Fatalf("split=%#v err=%v", split, err)
	}
	removed, err := pack.ResolveCompetency("go.removed")
	if err != nil || !removed.Removed {
		t.Fatalf("removed=%#v err=%v", removed, err)
	}
}
