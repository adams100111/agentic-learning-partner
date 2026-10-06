package pylearn

import "github.com/adams100111/agentic-learning-partner/internal/platform"

// AdapterID is the PyLearn adapter's registry identifier and platform ID.
const AdapterID = "pylearn"

// stableIdentifierKinds are the PyLearn identifiers declared stable across
// ordinary authoring edits (ADR-0057): lesson front-matter id, explicit Scene
// id, and quiz/question IDs. Heading-derived section slugs and positional
// scene IDs are deliberately absent.
var stableIdentifierKinds = []string{"lesson", "question", "quiz", "scene"}

var _ platform.Adapter = Adapter{}
var _ platform.CurriculumSource = Adapter{}
var _ platform.ContentMappingValidator = Adapter{}

func (Adapter) ID() string { return AdapterID }

// Capabilities declares what the PyLearn adapter supports today. Authoring
// and validation are not declared until they are implemented.
func (Adapter) Capabilities() []platform.Capability {
	return []platform.Capability{platform.ActivitySource, platform.CurriculumReader, platform.ContentMapper}
}

func (Adapter) StableIdentifierKinds() []string {
	return append([]string(nil), stableIdentifierKinds...)
}

// ReadCurriculum reads a target from PyLearn's `export:curriculum` output.
func (a Adapter) ReadCurriculum(export []byte, target string) (platform.Curriculum, error) {
	return platform.ReadCurriculumExport(export, platform.ExportExpectation{
		Platform:    AdapterID,
		Target:      target,
		StableKinds: a.StableIdentifierKinds(),
	})
}

// ValidateContentMapping validates a PyLearn-owned mapping (schema v2) for a
// target against its curriculum export and ALP's domain packs.
func (a Adapter) ValidateContentMapping(mapping []byte, name string, curriculum platform.Curriculum) (platform.MappingReport, error) {
	return platform.ValidateMapping(mapping, name, platform.MappingSubject{
		Platform:    AdapterID,
		StableKinds: a.StableIdentifierKinds(),
		Curriculum:  curriculum,
		Packs:       a.Domains,
	})
}
