package platform

import (
	"fmt"
)

// Authoring Intents, from the largest scope to the smallest (Q21). IntentNone
// means a unit or target justifies no authoring.
const (
	IntentTargetSkeleton = "target-skeleton"
	IntentCurriculum     = "curriculum"
	IntentUnit           = "unit"
	IntentActivity       = "activity"
	IntentPatch          = "patch"
	IntentNone           = "none"
)

// AuthoringIntents lists every Authoring Intent from largest to smallest.
var AuthoringIntents = []string{IntentTargetSkeleton, IntentCurriculum, IntentUnit, IntentActivity, IntentPatch}

// intentRank orders intents by scope: a higher rank is a smaller intent.
func intentRank(intent string) int {
	for rank, known := range AuthoringIntents {
		if known == intent {
			return rank
		}
	}
	return -1
}

// ValidIntent reports whether intent is a known Authoring Intent.
func ValidIntent(intent string) bool { return intentRank(intent) >= 0 }

// AuthoringPlanVersion is the Authoring Plan schemaVersion.
const AuthoringPlanVersion = 1

// Authoring outcome statuses.
const (
	AuthoringPlanned                = "planned"
	AuthoringNothingToAuthor        = "nothing-to-author"
	AuthoringExplicitIntentRequired = "explicit-intent-required"
)

// Authoring selections: whether the intent was the smallest justified
// default or explicitly requested.
const (
	SelectionDefault  = "default"
	SelectionExplicit = "explicit"
)

// Authoring error codes.
const (
	CodeIntentNotApplicable = "intent-not-applicable"
	CodeUnknownUnitSpec     = "unknown-unit"
)

// AuthoringTargetDeclaration is implemented by adapters that declare the
// Authoring Target capability: the platform names the authoring target skill
// that realizes specifications in its native representation (ADR-0056, Q24).
// ALP holds no platform authoring knowledge.
type AuthoringTargetDeclaration interface {
	AuthoringSkill() string
}

// SpecRef identifies one immutable specification version.
type SpecRef struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	ContentHash string `json:"contentHash"`
}

// AuthoringPlan is a versioned proposal for realizing specifications in a
// target platform (ADR-0056). Its ID is a content hash, so the same intent
// over the same specification versions is the same plan. Public is the only
// part that may appear in platform content or PRs (ADR-0060).
type AuthoringPlan struct {
	SchemaVersion   int                `json:"schemaVersion"`
	ID              string             `json:"id"`
	LearnerID       string             `json:"learnerId"`
	Target          ExternalID         `json:"target"`
	Intent          string             `json:"intent"`
	Justification   string             `json:"justification"`
	AuthoringTarget AuthoringTargetRef `json:"authoringTarget"`
	Curriculum      SpecRef            `json:"curriculum"`
	Units           []SpecRef          `json:"units"`
	Public          PublicAuthoring    `json:"public"`
}

// AuthoringTargetRef names the platform-declared authoring target skill.
type AuthoringTargetRef struct {
	Platform string `json:"platform"`
	Skill    string `json:"skill"`
}

// PublicAuthoring is the learner-free face of an Authoring Plan: spec IDs,
// versions and hashes plus teaching intent. It carries no learner ID,
// evidence, assessment, status, misconception or adaptation mode.
type PublicAuthoring struct {
	Intent     string             `json:"intent"`
	Curriculum PublicCurriculum   `json:"curriculum"`
	Units      []PublicUnitIntent `json:"units"`
}

// PublicCurriculum is a Curriculum Specification's citation and teaching
// intent: its goal and grouping.
type PublicCurriculum struct {
	SpecRef
	Title  string          `json:"title,omitempty"`
	Goal   []CompetencyRef `json:"goal"`
	Groups []PublicGroup   `json:"groups"`
}

// PublicGroup is a group of the curriculum with its unit citations.
type PublicGroup struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	PlatformPhase string             `json:"platformPhase,omitempty"`
	Units         []PublicGroupEntry `json:"units"`
}

// PublicGroupEntry cites one unit specification in a group.
type PublicGroupEntry struct {
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	PlatformItem *ExternalID `json:"platformItem,omitempty"`
}

// PublicUnitIntent is a Learning Unit Specification's citation, placement
// and teaching intent.
type PublicUnitIntent struct {
	SpecRef
	PlatformItem   *ExternalID  `json:"platformItem,omitempty"`
	Group          string       `json:"group"`
	TeachingIntent UnitTeaching `json:"teachingIntent"`
}

// AuthoringRequest selects what to author from reconciled specifications.
// Intent and Unit are empty unless explicitly requested.
type AuthoringRequest struct {
	Curriculum CurriculumSpec
	Units      []LearningUnitSpec
	Intent     string
	Unit       string
	Skill      string
}

// AuthoringOutcome is the result of selecting an Authoring Intent.
type AuthoringOutcome struct {
	Status    string         `json:"status"`
	Intent    string         `json:"intent,omitempty"`
	Selection string         `json:"selection"`
	Reason    string         `json:"reason"`
	Plan      *AuthoringPlan `json:"-"`
}

// PlanAuthoring selects the Authoring Intent and the specifications in its
// scope. By default it picks the smallest justified intent (patch → activity
// → unit) for the first unit in curriculum order that justifies it. A target
// skeleton (only for a target with no content yet) and whole-curriculum
// authoring happen only when explicitly requested.
func PlanAuthoring(request AuthoringRequest) (AuthoringOutcome, error) {
	selection := SelectionDefault
	if request.Intent != "" || request.Unit != "" {
		selection = SelectionExplicit
	}
	outcome := AuthoringOutcome{Selection: selection}
	target := request.Curriculum.Target
	newTarget := request.Curriculum.AuthoringIntent == IntentTargetSkeleton

	var unit *LearningUnitSpec
	if request.Unit != "" {
		for i := range request.Units {
			candidate := &request.Units[i]
			if candidate.ID == request.Unit || (candidate.Unit.PlatformItem != nil && candidate.Unit.PlatformItem.Item == request.Unit) {
				unit = candidate
			}
		}
		if unit == nil {
			return AuthoringOutcome{}, &Error{Code: CodeUnknownUnitSpec, Adapter: target.Platform, Target: target.Target,
				Message: fmt.Sprintf("%q is neither a unit specification ID nor a unit item of target %q", request.Unit, target.Target)}
		}
	}
	notApplicable := func(message string) error {
		return &Error{Code: CodeIntentNotApplicable, Adapter: target.Platform, Target: target.Target, Message: message}
	}

	var scope []LearningUnitSpec
	switch request.Intent {
	case IntentTargetSkeleton:
		if unit != nil {
			return AuthoringOutcome{}, notApplicable("a target skeleton covers the whole target; it takes no unit")
		}
		if !newTarget {
			return AuthoringOutcome{}, notApplicable(fmt.Sprintf("target %q already has content; a target skeleton is only authored for a new target", target.Target))
		}
		scope = request.Units
		outcome.Reason = fmt.Sprintf("Explicitly requested skeleton for new target %q: %d groups, %d units.", target.Target, len(request.Curriculum.Groups), len(scope))
	case IntentCurriculum:
		if unit != nil {
			return AuthoringOutcome{}, notApplicable("whole-curriculum authoring covers the whole target; it takes no unit")
		}
		for _, candidate := range request.Units {
			if candidate.AuthoringIntent != IntentNone {
				scope = append(scope, candidate)
			}
		}
		outcome.Reason = fmt.Sprintf("Explicitly requested whole-curriculum authoring: %d units justify authoring.", len(scope))
	case IntentUnit, IntentActivity, IntentPatch:
		if unit == nil {
			unit = firstWithIntent(request.Units, request.Intent)
			if unit == nil {
				outcome.Status, outcome.Intent = AuthoringNothingToAuthor, request.Intent
				outcome.Reason = fmt.Sprintf("No unit of target %q justifies %s authoring.", target.Target, request.Intent)
				return outcome, nil
			}
		}
		proposed := unit.Unit.PlatformItem == nil
		if request.Intent == IntentUnit && !proposed {
			return AuthoringOutcome{}, notApplicable(fmt.Sprintf("unit %s already exists as platform item %q; author an activity or a patch instead", unit.ID, unit.Unit.PlatformItem.Item))
		}
		if request.Intent != IntentUnit && proposed {
			return AuthoringOutcome{}, notApplicable(fmt.Sprintf("unit %s is not realized on the platform yet; author the unit before an %s", unit.ID, request.Intent))
		}
		scope = []LearningUnitSpec{*unit}
		outcome.Reason = fmt.Sprintf("Explicitly requested %s for unit %s (%s).", request.Intent, unit.ID, unit.Teaching.Title)
	case "":
		if newTarget && unit == nil {
			outcome.Status, outcome.Intent = AuthoringExplicitIntentRequired, IntentTargetSkeleton
			outcome.Reason = fmt.Sprintf("Target %q has no content yet: its structure must be authored first. Request --intent %s explicitly.", target.Target, IntentTargetSkeleton)
			return outcome, nil
		}
		if unit == nil {
			unit = firstWithIntent(request.Units, request.Curriculum.AuthoringIntent)
		}
		if unit == nil || unit.AuthoringIntent == IntentNone {
			outcome.Status = AuthoringNothingToAuthor
			outcome.Reason = "No unit justifies authoring: every unit is skipped or already provides the evidence it requires."
			if unit != nil {
				outcome.Reason = fmt.Sprintf("Unit %s (%s) justifies no authoring.", unit.ID, unit.Teaching.Title)
			}
			return outcome, nil
		}
		request.Intent = unit.AuthoringIntent
		scope = []LearningUnitSpec{*unit}
		outcome.Reason = justification(*unit)
	default:
		return AuthoringOutcome{}, fmt.Errorf("unknown authoring intent %q", request.Intent)
	}
	if len(scope) == 0 {
		outcome.Status, outcome.Intent = AuthoringNothingToAuthor, request.Intent
		outcome.Reason = fmt.Sprintf("No unit of target %q justifies authoring.", target.Target)
		return outcome, nil
	}

	plan := AuthoringPlan{
		SchemaVersion:   AuthoringPlanVersion,
		LearnerID:       request.Curriculum.LearnerID,
		Target:          target,
		Intent:          request.Intent,
		Justification:   outcome.Reason,
		AuthoringTarget: AuthoringTargetRef{Platform: target.Platform, Skill: request.Skill},
		Curriculum:      SpecRef{ID: request.Curriculum.ID, Version: request.Curriculum.Version, ContentHash: request.Curriculum.ContentHash},
		Units:           []SpecRef{},
		Public:          PublicAuthoring{Intent: request.Intent, Curriculum: publicCurriculum(request.Curriculum), Units: []PublicUnitIntent{}},
	}
	for _, spec := range scope {
		ref := SpecRef{ID: spec.ID, Version: spec.Version, ContentHash: spec.ContentHash}
		plan.Units = append(plan.Units, ref)
		plan.Public.Units = append(plan.Public.Units, PublicUnitIntent{
			SpecRef: ref, PlatformItem: spec.Unit.PlatformItem, Group: spec.Unit.Group, TeachingIntent: spec.Teaching,
		})
	}
	id, err := specHash(plan)
	if err != nil {
		return AuthoringOutcome{}, err
	}
	plan.ID = "apl_" + id[len("sha256:"):len("sha256:")+24]
	outcome.Status, outcome.Intent, outcome.Plan = AuthoringPlanned, request.Intent, &plan
	return outcome, nil
}

func firstWithIntent(units []LearningUnitSpec, intent string) *LearningUnitSpec {
	for i := range units {
		if units[i].AuthoringIntent == intent {
			return &units[i]
		}
	}
	return nil
}

// justification explains, privately, why a unit justifies its intent.
func justification(unit LearningUnitSpec) string {
	switch unit.AuthoringIntent {
	case IntentUnit:
		return fmt.Sprintf("Smallest justified intent is unit: %s (%s) is proposed for this target, not yet on the platform, and the learner's adaptation mode for it is %s.",
			unit.ID, unit.Teaching.Title, unit.Adaptation.Mode)
	case IntentActivity:
		return fmt.Sprintf("Smallest justified intent is activity: %s (%s) exists on the platform but nothing in it assesses every competency it requires evidence for.",
			unit.ID, unit.Teaching.Title)
	default:
		return fmt.Sprintf("Smallest justified intent is %s for %s (%s).", unit.AuthoringIntent, unit.ID, unit.Teaching.Title)
	}
}

func publicCurriculum(spec CurriculumSpec) PublicCurriculum {
	public := PublicCurriculum{
		SpecRef: SpecRef{ID: spec.ID, Version: spec.Version, ContentHash: spec.ContentHash},
		Title:   spec.Title, Goal: spec.Goal, Groups: []PublicGroup{},
	}
	titles := map[string]CurriculumUnit{}
	for _, unit := range spec.Units {
		titles[unit.ID] = unit
	}
	for _, group := range spec.Groups {
		entry := PublicGroup{ID: group.ID, Title: group.Title, PlatformPhase: group.PlatformPhase, Units: []PublicGroupEntry{}}
		for _, id := range group.Units {
			entry.Units = append(entry.Units, PublicGroupEntry{ID: id, Title: titles[id].Title, PlatformItem: titles[id].PlatformItem})
		}
		public.Groups = append(public.Groups, entry)
	}
	return public
}
