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
