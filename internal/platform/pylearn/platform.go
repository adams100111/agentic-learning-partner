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
var _ platform.AuthoringTargetDeclaration = Adapter{}

// AuthoringSkillName is the PyLearn repo-local skill that realizes ALP
// Curriculum and Learning Unit Specifications as PyLearn-native content and
// runs PyLearn's own gates (ADR-0056, Q24). ALP only dispatches to it.
const AuthoringSkillName = "pylearn-alp-authoring"

func (Adapter) ID() string { return AdapterID }

// Capabilities declares what the PyLearn adapter supports today. Authoring
// Target is declared because PyLearn names its authoring target skill;
// validation is not declared until it is implemented.
func (Adapter) Capabilities() []platform.Capability {
	return []platform.Capability{platform.ActivitySource, platform.CurriculumReader, platform.ContentMapper, platform.AuthoringTarget}
}

// AuthoringSkill names the platform-declared authoring target skill.
func (Adapter) AuthoringSkill() string { return AuthoringSkillName }

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
