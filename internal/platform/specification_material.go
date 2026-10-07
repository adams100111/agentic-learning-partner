package platform

import (
	"bytes"
	"encoding/json"
)

// Material-change rules: the fields whose change makes a new specification
// version (Q25), each compared in the form that matters for teaching.

// materialField is one material field of a draft and its latest version.
type materialField struct {
	name          string
	draft, latest any
}

func changedFields(fields []materialField) []string {
	changed := []string{}
	for _, field := range fields {
		left, _ := json.Marshal(field.draft)
		right, _ := json.Marshal(field.latest)
		if !bytes.Equal(left, right) {
			changed = append(changed, field.name)
		}
	}
	return changed
}

func materialCompetencies(values []SpecCompetency) []SpecCompetency {
	result := make([]SpecCompetency, len(values))
	for i, value := range values {
		result[i] = SpecCompetency{Domain: value.Domain, ID: value.ID, Roles: value.Roles}
	}
	return result
}

func materialPrerequisites(values []SpecPrerequisite) []CompetencyRef {
	result := make([]CompetencyRef, len(values))
	for i, value := range values {
		result[i] = CompetencyRef{Domain: value.Domain, ID: value.ID}
	}
	return result
}

func materialMisconceptions(values []Misconception) []Misconception {
	result := make([]Misconception, len(values))
	for i, value := range values {
		result[i] = Misconception{Domain: value.Domain, Competency: value.Competency, Gap: value.Gap}
	}
	return result
}

// materialShape is the derived teaching shape a version stores. Persona
// edits that leave it unchanged (risks, override reasons, experience levels,
// document hashes) are not material: they change only the private persona
// view, which is re-derived on every plan and never stored.
func materialShape(spec LearningUnitSpec) any {
	return spec.Teaching.Shape
}

func materialUnits(values []CurriculumUnit) []CurriculumUnit {
	result := make([]CurriculumUnit, len(values))
	for i, value := range values {
		result[i] = CurriculumUnit{ID: value.ID, Version: value.Version, Group: value.Group}
	}
	return result
}
