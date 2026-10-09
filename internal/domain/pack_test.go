package domain

import "testing"

func TestPackRejectsPrerequisiteCycle(t *testing.T) {
	pack := Pack{
		SchemaVersion: 1,
		Domain:        "go",
		Version:       "0.1.0",
		Competencies: []Competency{
			{
				ID:             "go.a",
				Name:           "A",
				Dimension:      "runtime",
				Description:    "A",
				Prerequisites:  []string{"go.b"},
				FreshnessClass: "stable-concept",
			},
			{
				ID:             "go.b",
				Name:           "B",
				Dimension:      "runtime",
				Description:    "B",
				Prerequisites:  []string{"go.a"},
				FreshnessClass: "stable-concept",
			},
		},
	}
	if err := pack.Validate(); err == nil {
		t.Fatal("expected prerequisite-cycle validation error")
	}
}

func TestPackRejectsInvalidProductionRequirement(t *testing.T) {
	pack := Pack{
		SchemaVersion: 1,
		Domain:        "go",
		Version:       "0.1.0",
		Competencies: []Competency{
			{
				ID:                      "go.a",
				Name:                    "A",
				Dimension:               "runtime",
				Description:             "A",
				FreshnessClass:          "stable-concept",
				ProductionReadyRequires: []string{""},
			},
		},
	}
	if err := pack.Validate(); err == nil {
		t.Fatal("expected empty production requirement error")
	}
}

func TestRegistryCompatibility(t *testing.T) {
	registry := NewRegistry()
	if err := registry.CheckCompatibility("go", ">=0.1 <0.2"); err != nil {
		t.Fatal(err)
	}
	if err := registry.CheckCompatibility("go", ">=1.0 <2.0"); err == nil {
		t.Fatal("expected incompatible-range error")
	}
}

func areaPack(areas ...string) Pack {
	competency := func(id string) Competency {
		return Competency{ID: id, Name: id, Dimension: "runtime", Description: id, FreshnessClass: "stable-concept"}
	}
	return Pack{
		SchemaVersion: 1, Domain: "go", Version: "1.0.0", Areas: areas,
		Competencies: []Competency{competency("go.language.a"), competency("go.runtime.b"), competency("go.testing.c")},
	}
}

func TestPackAreasRankCompetencyAreasAndMustMatchThePack(t *testing.T) {
	ordered := areaPack("testing", "language", "runtime")
	if err := ordered.Validate(); err != nil {
		t.Fatal(err)
	}
	if ordered.AreaRank("go.testing.c") != 0 || ordered.AreaRank("go.language.a") != 1 || ordered.AreaRank("go.runtime.b") != 2 {
		t.Errorf("ranks = %d %d %d", ordered.AreaRank("go.testing.c"), ordered.AreaRank("go.language.a"), ordered.AreaRank("go.runtime.b"))
	}
	// Without a list, areas rank by first appearance.
	implicit := areaPack()
	if err := implicit.Validate(); err != nil {
		t.Fatal(err)
	}
	if implicit.AreaRank("go.language.a") != 0 || implicit.AreaRank("go.testing.c") != 2 {
		t.Errorf("implicit ranks = %d %d", implicit.AreaRank("go.language.a"), implicit.AreaRank("go.testing.c"))
	}
	for name, pack := range map[string]Pack{
		"missing area":   areaPack("language", "runtime"),
		"unknown area":   areaPack("language", "runtime", "testing", "backend"),
		"repeated area":  areaPack("language", "runtime", "testing", "runtime"),
		"qualified area": areaPack("go.language", "runtime", "testing"),
	} {
		if err := pack.Validate(); err == nil {
			t.Errorf("%s: Validate accepted %v", name, pack.Areas)
		}
	}
}
