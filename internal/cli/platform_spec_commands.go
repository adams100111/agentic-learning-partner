package cli

import (
	"fmt"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/state"
)

// planSpecEntry reports one specification version a plan reconciled.
type planSpecEntry struct {
	ID           string               `json:"id"`
	Version      int                  `json:"version"`
	ContentHash  string               `json:"contentHash"`
	Path         string               `json:"path"`
	Status       string               `json:"status"`
	Title        string               `json:"title,omitempty"`
	PlatformItem *platform.ExternalID `json:"platformItem,omitempty"`
	Group        string               `json:"group,omitempty"`
	Mode         string               `json:"mode,omitempty"`
	// HeldFields are material changes held back for a realized unit because
	// no new evidence motivated them (status held).
	HeldFields []string `json:"heldFields,omitempty"`
	// Realized reports whether this version is realized on the platform.
	Realized bool `json:"realized,omitempty"`
	// Persona is the unit's private persona view (risks, analogy override
	// reasons, analogy source experience), re-derived from the live persona
	// and profile documents for this plan. It is never stored in a
	// specification version or Authoring Plan, and must never be copied into
	// platform content.
	Persona *platform.PersonaBasis `json:"persona,omitempty"`
}

type planSpecifications struct {
	Curriculum planSpecEntry   `json:"curriculum"`
	Units      []planSpecEntry `json:"units"`
	// Persona reports the persona documents that shaped unit teaching and
	// the documented defaults that stood in for missing ones.
	Persona platform.PersonaReport `json:"persona"`
}

// planAuthoringRecord is the Authoring Plan with where it is recorded.
type planAuthoringRecord struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	platform.AuthoringPlan
}

// specifyTarget reconciles the target's Curriculum and Learning Unit
// Specifications with their stored versions, writes the versions that
// changed materially, and records the selected Authoring Plan, all in the
// workspace transaction.
func specifyTarget(ws platformWorkspace, adapter platform.Adapter, inputs adaptationInputs, projection platform.TargetAdaptationProjection, flags platformFlags) (planSpecifications, platform.AuthoringOutcome, *planAuthoringRecord, error) {
	declaration, ok := adapter.(platform.AuthoringTargetDeclaration)
	if !ok {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, fmt.Errorf("platform adapter %q declares %s but does not implement it", adapter.ID(), platform.AuthoringTarget)
	}
	learnerState, err := ws.engine.CompetencyProjection()
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	domains := make([]string, 0, len(projection.Inputs.Packs))
	for _, pack := range projection.Inputs.Packs {
		domains = append(domains, pack.Domain)
	}
	documents, err := ws.engine.PersonaDocuments(domains)
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	persona, err := platform.ReadTeachingPersona(documents)
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	links, err := loadRealizations(ws)
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	realizations := platform.IndexRealizations(links, projection.Target)
	draft, err := platform.DraftSpecifications(platform.SpecRequest{
		Projection:   projection,
		Curriculum:   inputs.curriculum,
		Mapping:      inputs.mapping,
		Packs:        domain.NewRegistry(),
		LearnerState: learnerState,
		Constraints:  inputs.constraints,
		Persona:      persona,
		Realizations: realizations,
	})
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}

	output := planSpecifications{Units: []planSpecEntry{}, Persona: persona.Report}
	units := make([]platform.LearningUnitSpec, 0, len(draft.Units))
	for _, unitDraft := range draft.Units {
		var latest *platform.LearningUnitSpec
		data, path, found, err := ws.engine.LatestSpecification(state.UnitSpecs, unitDraft.ID)
		if err != nil {
			return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
		}
		if found {
			stored, err := platform.DecodeUnitSpec(path, data)
			if err != nil {
				return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
			}
			latest = &stored
		}
		revision, err := platform.ReviseUnitSpec(unitDraft, latest, realizations.Realized(unitDraft.ID))
		if err != nil {
			return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
		}
		spec, status := revision.Spec, revision.Status
		path, err = writeSpec(ws, state.UnitSpecs, spec.ID, spec.Version, status, spec)
		if err != nil {
			return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
		}
		spec = realizedView(spec, unitDraft, status, realizations)
		units = append(units, spec)
		output.Units = append(output.Units, planSpecEntry{
			ID: spec.ID, Version: spec.Version, ContentHash: spec.ContentHash, Path: path, Status: status,
			Title: spec.Teaching.Title, PlatformItem: spec.Unit.PlatformItem, Group: spec.Unit.Group, Mode: spec.Adaptation.Mode,
			HeldFields: revision.HeldFields, Realized: realizations.VersionRealized(spec.ID, spec.Version),
			Persona: personaView(draft.PersonaViews, unitDraft.ID),
		})
	}

	composed := platform.ComposeCurriculum(draft.Curriculum, units, inputs.curriculum.Phases)
	var latestCurriculum *platform.CurriculumSpec
	data, path, found, err := ws.engine.LatestSpecification(state.CurriculumSpecs, composed.ID)
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	if found {
		stored, err := platform.DecodeCurriculumSpec(path, data)
		if err != nil {
			return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
		}
		latestCurriculum = &stored
	}
	curriculum, status, err := platform.ReviseCurriculumSpec(composed, latestCurriculum)
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	path, err = writeSpec(ws, state.CurriculumSpecs, curriculum.ID, curriculum.Version, status, curriculum)
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	output.Curriculum = planSpecEntry{ID: curriculum.ID, Version: curriculum.Version, ContentHash: curriculum.ContentHash, Path: path, Status: status, Title: curriculum.Title}

	// A kept curriculum version justifies what its units justify now: once
	// a new target's skeleton is realized it is no longer a new target.
	if status == platform.SpecUnchanged {
		curriculum.AuthoringIntent = composed.AuthoringIntent
	}
	outcome, err := platform.PlanAuthoring(platform.AuthoringRequest{
		Curriculum: curriculum, Units: units, Intent: flags.intent, Unit: flags.unit, Skill: declaration.AuthoringSkill(),
	})
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	if outcome.Plan == nil {
		return output, outcome, nil, nil
	}
	data, err = platform.EncodeSpec(outcome.Plan)
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	planPath, created, err := ws.engine.RecordAuthoringPlan(ws.expected, outcome.Plan.ID, data)
	if err != nil {
		return planSpecifications{}, platform.AuthoringOutcome{}, nil, err
	}
	record := &planAuthoringRecord{Path: planPath, Status: "existing", AuthoringPlan: *outcome.Plan}
	if created {
		record.Status = "created"
	}
	return output, outcome, record, nil
}

// personaView returns the unit's private persona view, if it has one.
func personaView(views map[string]platform.PersonaBasis, unitID string) *platform.PersonaBasis {
	view, ok := views[unitID]
	if !ok {
		return nil
	}
	return &view
}

// realizedView is how planning sees a stored unit version once the unit is
// realized: placed at the unit-level item that realized it (its Realization
// Link), and, when the stored version was kept, justifying only what the
// current draft of the realized unit justifies, so realized content is never
// authored again. The stored version itself is immutable and unchanged.
func realizedView(spec, draft platform.LearningUnitSpec, status string, realizations platform.Realizations) platform.LearningUnitSpec {
	root, realized := realizations.ByUnit[spec.ID]
	if !realized {
		return spec
	}
	if spec.Unit.PlatformItem == nil {
		item := root
		spec.Unit.PlatformItem = &item
	}
	if status == platform.SpecUnchanged || status == platform.SpecHeld {
		spec.AuthoringIntent = platform.IntentNone
		if draft.Unit.PlatformItem != nil {
			spec.AuthoringIntent = draft.AuthoringIntent
		}
	}
	return spec
}

// writeSpec appends a new specification version, or returns the path of the
// stored version a plan kept unchanged.
func writeSpec(ws platformWorkspace, kind, id string, version int, status string, spec any) (string, error) {
	if status == platform.SpecUnchanged || status == platform.SpecHeld {
		return state.SpecificationPath(kind, id, version), nil
	}
	data, err := platform.EncodeSpec(spec)
	if err != nil {
		return "", err
	}
	return ws.engine.AppendSpecification(ws.expected, kind, id, version, data)
}
