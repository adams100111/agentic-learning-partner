package platform

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/state"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

// Adaptation modes, ordered from least to most delivered content. A unit is
// skipped (already demonstrated), challenged (test out instead of re-learning
// what is claimed but unconfirmed), skimmed (condensed), or taken in full.
const (
	ModeSkip      = "skip"
	ModeChallenge = "challenge"
	ModeSkim      = "skim"
	ModeFull      = "full"
)

// Modes lists every adaptation mode from least to most delivered content.
var Modes = []string{ModeSkip, ModeChallenge, ModeSkim, ModeFull}

// Learner competency statuses, as read from the competency projection.
const (
	StatusDemonstrated = "demonstrated"
	StatusUnconfirmed  = "unconfirmed"
	StatusFunctional   = "functional"
	StatusRusty        = "rusty"
	StatusUnassessed   = "unassessed"
)

// TargetAdaptationVersion is the Target Adaptation Projection schemaVersion.
const TargetAdaptationVersion = 1

// Target adaptation error codes.
const (
	CodeInvalidConstraints = "invalid-target-constraints"
)

// TargetAdaptationProjection is one learner's derived, rebuildable adaptation
// of one shared Learning Target (ADR-0060, ADR-0061). It reads learner
// competency state but never judges competency itself (ADR-0055): every
// competency status cites the assessment it was read from. Its Revision is a
// content hash, so rebuilding from the same canonical inputs reproduces it.
type TargetAdaptationProjection struct {
	SchemaVersion int                 `json:"schemaVersion"`
	ID            string              `json:"id"`
	Revision      string              `json:"revision"`
	LearnerID     string              `json:"learnerId"`
	Target        ExternalID          `json:"target"`
	Inputs        AdaptationInputs    `json:"inputs"`
	Units         []AdaptedUnit       `json:"units"`
	Sequence      []string            `json:"sequence"`
	Reinforcement []ReinforcementNeed `json:"reinforcement"`
	Gaps          []CompetencyGap     `json:"gaps"`
}

// AdaptationInputs identifies the canonical inputs a projection was built from.
type AdaptationInputs struct {
	Curriculum  SourceRef     `json:"curriculum"`
	Mapping     MappingSource `json:"mapping"`
	Packs       []PackRef     `json:"packs"`
	Constraints *SourceRef    `json:"constraints"`
	// Decisions are the active Accepted Adaptation Decisions for the target.
	Decisions []string `json:"decisions"`
}

// SourceRef identifies a versioned input document by content hash.
type SourceRef struct {
	SchemaVersion int    `json:"schemaVersion"`
	ContentHash   string `json:"contentHash"`
}

// PackRef is a domain pack version a projection was resolved against.
type PackRef struct {
	Domain  string `json:"domain"`
	Version string `json:"version"`
}

// AdaptedUnit is one top-level unit of the target with its adaptation.
type AdaptedUnit struct {
	Item         ExternalID       `json:"item"`
	Phase        string           `json:"phase"`
	Title        string           `json:"title,omitempty"`
	Competencies []UnitCompetency `json:"competencies"`
	// ProposedMode is what learner state alone justifies; it is the proposal
	// an agent may put to the learner. Mode is the effective mode after
	// target constraints and Accepted Adaptation Decisions.
	ProposedMode string           `json:"proposedMode"`
	Mode         string           `json:"mode"`
	Rationale    string           `json:"rationale"`
	Decision     *AppliedDecision `json:"decision,omitempty"`
}

// UnitCompetency is a competency a unit maps to, with the learner's status
// read from the competency projection.
// Need is how urgently a competency calls for teaching, lowest first.
type Need int

const (
	// NeedUrgent: the learner is rusty on it or assessed as unknown.
	NeedUrgent Need = iota
	// NeedUnconfirmed: never assessed, or assessed but stale or low confidence.
	NeedUnconfirmed
	// NeedSettled: demonstrated or functional, and current.
	NeedSettled
)

type UnitCompetency struct {
	ID           string        `json:"id"`
	Domain       string        `json:"domain"`
	Roles        []MappingRole `json:"roles"`
	Status       string        `json:"status"`
	Level        string        `json:"level,omitempty"`
	Confidence   string        `json:"confidence,omitempty"`
	AssessmentID string        `json:"assessmentId,omitempty"`
	// Need is derived with Status from the same projection read, for
	// sequencing; it is not part of any stored or emitted document.
	Need Need `json:"-"`
}

// AppliedDecision is the active Accepted Adaptation Decision for a unit.
type AppliedDecision struct {
	ID      string `json:"id"`
	Mode    string `json:"mode"`
	Applied bool   `json:"applied"`
	Reason  string `json:"reason,omitempty"`
}

// ReinforcementNeed is a target competency the learner should reinforce.
type ReinforcementNeed struct {
	Competency string   `json:"competency"`
	Domain     string   `json:"domain"`
	Reason     string   `json:"reason"`
	Items      []string `json:"items"`
}

// CompetencyGap is a prerequisite of the target's competencies that the
// learner has not shown and that the target does not cover.
type CompetencyGap struct {
	Competency string   `json:"competency"`
	Domain     string   `json:"domain"`
	Status     string   `json:"status"`
	RequiredBy []string `json:"requiredBy"`
}

// TargetConstraints are target-level limits on adaptation, read from a
// versioned constraints document.
type TargetConstraints struct {
	Source        SourceRef
	AllowedModes  []string
	RequiredUnits []string
	// Goal lists competencies the target is for, beyond what its content
	// already covers; Curriculum Specifications propose units for them.
	Goal []string
}

// TargetConstraintsSchema is the embedded JSON Schema for target constraints.
const TargetConstraintsSchema = "target-constraints.schema.json"

type constraintsDocument struct {
	SchemaVersion int      `json:"schemaVersion"`
	Platform      string   `json:"platform"`
	Target        string   `json:"target"`
	AllowedModes  []string `json:"allowedModes"`
	RequiredUnits []string `json:"requiredUnits"`
	Goal          []string `json:"goal"`
}

// ReadTargetConstraints validates a target constraints document (YAML or
// JSON) for one Learning Target of curriculum.
func ReadTargetConstraints(data []byte, name string, curriculum Curriculum) (TargetConstraints, error) {
	fail := func(problems ...string) error {
		return &Error{
			Code:     CodeInvalidConstraints,
			Message:  fmt.Sprintf("invalid target constraints %s: %s", name, strings.Join(problems, "; ")),
			Adapter:  curriculum.Target.Platform,
			Target:   curriculum.Target.Target,
			Problems: problems,
		}
	}
	document, err := workspace.DecodeDocument(name, data)
	if err != nil {
		return TargetConstraints{}, fail(err.Error())
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		return TargetConstraints{}, fmt.Errorf("initialize validation: %w", err)
	}
	if issues := validator.ValidateValue(TargetConstraintsSchema, name, document); len(issues) != 0 {
		problems := make([]string, len(issues))
		for i, issue := range issues {
			problems[i] = issue.Error()
		}
		return TargetConstraints{}, fail(problems...)
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return TargetConstraints{}, fail(err.Error())
	}
	var parsed constraintsDocument
	if err := json.Unmarshal(encoded, &parsed); err != nil {
		return TargetConstraints{}, fail(err.Error())
	}
	var problems []string
	if parsed.Platform != curriculum.Target.Platform || parsed.Target != curriculum.Target.Target {
		problems = append(problems, fmt.Sprintf("constraints are for %s/%s, not %s/%s", parsed.Platform, parsed.Target, curriculum.Target.Platform, curriculum.Target.Target))
	}
	units := map[string]bool{}
	for _, unit := range targetUnits(curriculum) {
		units[unit.Ref.Item] = true
	}
	for _, unit := range parsed.RequiredUnits {
		if !units[unit] {
			problems = append(problems, fmt.Sprintf("required unit %q is not a unit of target %q", unit, curriculum.Target.Target))
		}
	}
	if len(problems) != 0 {
		return TargetConstraints{}, fail(problems...)
	}
	sum := sha256.Sum256(data)
	constraints := TargetConstraints{
		Source:        SourceRef{SchemaVersion: parsed.SchemaVersion, ContentHash: "sha256:" + hex.EncodeToString(sum[:])},
		AllowedModes:  parsed.AllowedModes,
		RequiredUnits: parsed.RequiredUnits,
		Goal:          parsed.Goal,
	}
	return constraints, nil
}

// AdaptationRequest holds the canonical inputs of a Target Adaptation
// Projection.
type AdaptationRequest struct {
	LearnerID    string
	Curriculum   Curriculum
	Mapping      MappingReport
	Packs        PackLoader
	LearnerState state.Projection
	Constraints  *TargetConstraints
	Decisions    []state.AdaptationDecision
}

// ProjectTargetAdaptation builds the learner's Target Adaptation Projection.
// It is a pure function of its inputs.
func ProjectTargetAdaptation(request AdaptationRequest) (TargetAdaptationProjection, error) {
	if !request.Mapping.Valid {
		return TargetAdaptationProjection{}, fmt.Errorf("target adaptation requires a valid mapping")
	}
	target := request.Curriculum.Target
	target.Item = ""
	projection := TargetAdaptationProjection{
		SchemaVersion: TargetAdaptationVersion,
		ID:            state.TargetAdaptationID(target.Platform, target.Target),
		LearnerID:     request.LearnerID,
		Target:        target,
		Inputs: AdaptationInputs{
			Curriculum: SourceRef{SchemaVersion: request.Curriculum.SchemaVersion, ContentHash: request.Curriculum.ContentHash},
			Mapping:    request.Mapping.Mapping,
			Packs:      []PackRef{},
			Decisions:  []string{},
		},
		Units:         []AdaptedUnit{},
		Sequence:      []string{},
		Reinforcement: []ReinforcementNeed{},
		Gaps:          []CompetencyGap{},
	}
	for _, pack := range request.Mapping.Packs {
		projection.Inputs.Packs = append(projection.Inputs.Packs, PackRef{Domain: pack.Domain, Version: pack.Version})
	}
	if request.Constraints != nil {
		source := request.Constraints.Source
		projection.Inputs.Constraints = &source
	}

	learner := newLearnerView(request.LearnerState)
	units := targetUnits(request.Curriculum)
	unitOf := rootUnits(request.Curriculum)
	position := make(map[string]int, len(request.Curriculum.Items))
	for i, item := range request.Curriculum.Items {
		position[item.Ref.Item] = i
	}

	// Collect each unit's competencies and every item that reinforces or
	// assesses a competency.
	type unitKey = string
	assignments := map[unitKey]map[string]*UnitCompetency{}
	practice := map[string]map[string]bool{}
	targetCompetencies := map[string]UnitCompetency{}
	for _, entry := range request.Mapping.Entries {
		unit, ok := unitOf[entry.Item.Item]
		if !ok {
			continue
		}
		if assignments[unit] == nil {
			assignments[unit] = map[string]*UnitCompetency{}
		}
		for _, assignment := range entry.Competencies {
			id := assignment.ResolvedID
			if id == "" {
				id = assignment.ID
			}
			key := assignment.Domain + "\x00" + id
			competency, ok := assignments[unit][key]
			if !ok {
				read := learner.read(assignment.Domain, id)
				competency = &read
				assignments[unit][key] = competency
				targetCompetencies[key] = read
			}
			if !containsRole(competency.Roles, assignment.Role) {
				competency.Roles = append(competency.Roles, assignment.Role)
			}
			if assignment.Role == RoleReinforces || assignment.Role == RoleAssesses {
				if practice[key] == nil {
					practice[key] = map[string]bool{}
				}
				practice[key][entry.Item.Item] = true
			}
		}
	}

	active := activeDecisions(request.Decisions, target)
	for _, decision := range active {
		projection.Inputs.Decisions = append(projection.Inputs.Decisions, decision.ID)
	}
	sort.Strings(projection.Inputs.Decisions)

	for _, item := range units {
		unit := AdaptedUnit{Item: item.Ref, Phase: item.Phase, Title: item.Title, Competencies: []UnitCompetency{}}
		for _, competency := range assignments[item.Ref.Item] {
			sort.Slice(competency.Roles, func(i, j int) bool { return roleOrder(competency.Roles[i]) < roleOrder(competency.Roles[j]) })
			unit.Competencies = append(unit.Competencies, *competency)
		}
		sort.Slice(unit.Competencies, func(i, j int) bool {
			if unit.Competencies[i].Domain != unit.Competencies[j].Domain {
				return unit.Competencies[i].Domain < unit.Competencies[j].Domain
			}
			return unit.Competencies[i].ID < unit.Competencies[j].ID
		})
		unit.ProposedMode, unit.Rationale = proposeMode(unit.Competencies)
		unit.Mode = request.Constraints.constrain(item.Ref.Item, unit.ProposedMode)
		if unit.Mode != unit.ProposedMode {
			unit.Rationale += fmt.Sprintf(" Target constraints do not allow %s for this unit.", unit.ProposedMode)
		}
		if decision, ok := active[item.Ref.Item]; ok {
			applied := &AppliedDecision{ID: decision.ID, Mode: decision.Mode, Applied: true}
			if request.Constraints.allows(item.Ref.Item, decision.Mode) {
				unit.Mode = decision.Mode
				unit.Rationale += fmt.Sprintf(" The learner accepted %s (decision %s).", decision.Mode, decision.ID)
			} else {
				applied.Applied = false
				applied.Reason = fmt.Sprintf("target constraints do not allow %s for this unit", decision.Mode)
			}
			unit.Decision = applied
		}
		if unit.Mode != ModeSkip {
			projection.Sequence = append(projection.Sequence, item.Ref.Item)
		}
		projection.Units = append(projection.Units, unit)
	}

	for key, competency := range targetCompetencies {
		reason := ""
		switch {
		case competency.Status == StatusRusty:
			reason = "rusty"
		case competency.Status == StatusUnconfirmed:
			reason = "unconfirmed"
		case len(learner.gaps(competency.Domain, competency.ID)) != 0:
			reason = "assessment-gaps"
		default:
			continue
		}
		items := []string{}
		for item := range practice[key] {
			items = append(items, item)
		}
		sort.Slice(items, func(i, j int) bool { return position[items[i]] < position[items[j]] })
		projection.Reinforcement = append(projection.Reinforcement, ReinforcementNeed{
			Competency: competency.ID, Domain: competency.Domain, Reason: reason, Items: items,
		})
	}
	sort.Slice(projection.Reinforcement, func(i, j int) bool {
		return competencyLess(projection.Reinforcement[i].Domain, projection.Reinforcement[i].Competency, projection.Reinforcement[j].Domain, projection.Reinforcement[j].Competency)
	})

	gaps := map[string]*CompetencyGap{}
	for _, competency := range targetCompetencies {
		pack, err := request.Packs.Load(competency.Domain)
		if err != nil {
			return TargetAdaptationProjection{}, fmt.Errorf("load domain pack %q: %w", competency.Domain, err)
		}
		for _, definition := range pack.Competencies {
			if definition.ID != competency.ID {
				continue
			}
			for _, prerequisite := range definition.Prerequisites {
				key := competency.Domain + "\x00" + prerequisite
				if _, covered := targetCompetencies[key]; covered {
					continue
				}
				status := learner.read(competency.Domain, prerequisite).Status
				if status != StatusUnassessed && status != StatusRusty {
					continue
				}
				gap, ok := gaps[key]
				if !ok {
					gap = &CompetencyGap{Competency: prerequisite, Domain: competency.Domain, Status: status}
					gaps[key] = gap
				}
				gap.RequiredBy = append(gap.RequiredBy, competency.ID)
			}
		}
	}
	for _, gap := range gaps {
		sort.Strings(gap.RequiredBy)
		projection.Gaps = append(projection.Gaps, *gap)
	}
	sort.Slice(projection.Gaps, func(i, j int) bool {
		return competencyLess(projection.Gaps[i].Domain, projection.Gaps[i].Competency, projection.Gaps[j].Domain, projection.Gaps[j].Competency)
	})

	revision, err := projectionRevision(projection)
	if err != nil {
		return TargetAdaptationProjection{}, err
	}
	projection.Revision = revision
	return projection, nil
}

// projectionRevision hashes the projection's content (without its revision).
func projectionRevision(projection TargetAdaptationProjection) (string, error) {
	projection.Revision = ""
	data, err := json.Marshal(projection)
	if err != nil {
		return "", fmt.Errorf("encode target adaptation projection: %w", err)
	}
	sum := sha256.Sum256(data)
	return "tapr_" + hex.EncodeToString(sum[:12]), nil
}

// EncodeTargetAdaptation renders a projection as deterministic JSON.
func EncodeTargetAdaptation(projection TargetAdaptationProjection) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(projection); err != nil {
		return nil, fmt.Errorf("encode target adaptation projection: %w", err)
	}
	return buffer.Bytes(), nil
}

// proposeMode derives a unit's mode from the learner's status on what the
// unit teaches or assesses (reinforced competencies only count when the unit
// does neither).
func proposeMode(competencies []UnitCompetency) (string, string) {
	if len(competencies) == 0 {
		return ModeFull, "The unit maps to no competencies, so learner state cannot justify adapting it."
	}
	var primary []UnitCompetency
	for _, competency := range competencies {
		if containsRole(competency.Roles, RoleTeaches) || containsRole(competency.Roles, RoleAssesses) {
			primary = append(primary, competency)
		}
	}
	if len(primary) == 0 {
		primary = competencies
	}
	counts := map[string][]string{}
	for _, competency := range primary {
		counts[competency.Status] = append(counts[competency.Status], competency.ID)
	}
	describe := func(status string) string { return strings.Join(counts[status], ", ") }
	switch {
	case len(counts[StatusUnassessed]) != 0:
		return ModeFull, fmt.Sprintf("No assessed competency for %s.", describe(StatusUnassessed))
	case len(counts[StatusRusty]) != 0:
		return ModeFull, fmt.Sprintf("Assessed as rusty: %s.", describe(StatusRusty))
	case len(counts[StatusUnconfirmed]) != 0:
		return ModeChallenge, fmt.Sprintf("Assessed strong but not confirmed (low confidence or needs reassessment): %s.", describe(StatusUnconfirmed))
	case len(counts[StatusFunctional]) != 0:
		return ModeSkim, fmt.Sprintf("Assessed functional: %s.", describe(StatusFunctional))
	default:
		return ModeSkip, fmt.Sprintf("Already demonstrated: %s.", describe(StatusDemonstrated))
	}
}

// allows reports whether the constraints permit mode for unit. Without
// constraints every mode is allowed.
func (c *TargetConstraints) allows(unit, mode string) bool {
	if c == nil {
		return true
	}
	if mode == ModeSkip {
		for _, required := range c.RequiredUnits {
			if required == unit {
				return false
			}
		}
	}
	for _, allowed := range c.AllowedModes {
		if allowed == mode {
			return true
		}
	}
	return false
}

// constrain returns the first allowed mode at or above mode in Modes order;
// full is always allowed.
func (c *TargetConstraints) constrain(unit, mode string) string {
	started := false
	for _, candidate := range Modes {
		if candidate == mode {
			started = true
		}
		if started && (candidate == ModeFull || c.allows(unit, candidate)) {
			return candidate
		}
	}
	return ModeFull
}

// AllowsMode reports whether a decision for mode on unit would take effect.
func AllowsMode(constraints *TargetConstraints, unit, mode string) bool {
	return constraints.allows(unit, mode)
}

// ActiveAdaptationDecisions returns the active accept decisions for target
// keyed by unit: records not superseded by any later record. When concurrent
// records leave several active for one unit, the latest wins.
func ActiveAdaptationDecisions(decisions []state.AdaptationDecision, target ExternalID) map[string]state.AdaptationDecision {
	return activeDecisions(decisions, target)
}

func activeDecisions(decisions []state.AdaptationDecision, target ExternalID) map[string]state.AdaptationDecision {
	superseded := map[string]bool{}
	for _, decision := range decisions {
		for _, id := range decision.Supersedes {
			superseded[id] = true
		}
	}
	active := map[string]state.AdaptationDecision{}
	for _, decision := range decisions {
		if decision.Action != state.DecisionAccept || superseded[decision.ID] ||
			decision.Target.Platform != target.Platform || decision.Target.Target != target.Target {
			continue
		}
		current, ok := active[decision.Unit]
		if !ok || later(decision, current) {
			active[decision.Unit] = decision
		}
	}
	return active
}

func later(left, right state.AdaptationDecision) bool {
	l, lerr := time.Parse(time.RFC3339, left.RecordedAt)
	r, rerr := time.Parse(time.RFC3339, right.RecordedAt)
	if lerr == nil && rerr == nil && !l.Equal(r) {
		return l.After(r)
	}
	return left.ID > right.ID
}

// targetUnits returns the target's top-level items (those placed in a phase),
// in phase order and then curriculum order.
func targetUnits(curriculum Curriculum) []Item {
	phaseOrder := make(map[string]int, len(curriculum.Phases))
	for i, phase := range curriculum.Phases {
		phaseOrder[phase.ID] = i
	}
	var units []Item
	for _, item := range curriculum.Items {
		if item.Parent == nil && item.Phase != "" {
			units = append(units, item)
		}
	}
	sort.SliceStable(units, func(i, j int) bool { return phaseOrder[units[i].Phase] < phaseOrder[units[j].Phase] })
	return units
}

// rootUnits maps every item to the unit that contains it.
func rootUnits(curriculum Curriculum) map[string]string {
	parent := make(map[string]string, len(curriculum.Items))
	for _, item := range curriculum.Items {
		if item.Parent != nil {
			parent[item.Ref.Item] = item.Parent.Item
		}
	}
	roots := make(map[string]string, len(curriculum.Items))
	for _, item := range curriculum.Items {
		root := item.Ref.Item
		for seen := 0; parent[root] != "" && seen <= len(curriculum.Items); seen++ {
			root = parent[root]
		}
		roots[item.Ref.Item] = root
	}
	return roots
}

type learnerView struct {
	byKey map[string]state.ProjectedCompetency
}

func newLearnerView(projection state.Projection) learnerView {
	view := learnerView{byKey: make(map[string]state.ProjectedCompetency, len(projection.Competencies))}
	for _, competency := range projection.Competencies {
		view.byKey[competency.Domain+"\x00"+competency.ID] = competency
	}
	return view
}

// read returns the learner's status on a competency, citing the assessment.
func (v learnerView) read(domainName, id string) UnitCompetency {
	result := UnitCompetency{ID: id, Domain: domainName, Roles: []MappingRole{}, Status: StatusUnassessed, Need: NeedUnconfirmed}
	projected, ok := v.byKey[domainName+"\x00"+id]
	if !ok {
		return result
	}
	result.Level, result.Confidence, result.AssessmentID = projected.Level, projected.Confidence, projected.AssessmentID
	result.Need = NeedSettled
	switch projected.Level {
	case "rusty":
		result.Status = StatusRusty
		result.Need = NeedUrgent
	case "functional":
		result.Status = StatusFunctional
	case "strong", "production-ready":
		result.Status = StatusDemonstrated
		if projected.Confidence == "low" || projected.NeedsReassessment {
			result.Status = StatusUnconfirmed
		}
	case "unknown":
		// Assessed and weak: it reads as unassessed for adaptation but is as
		// urgent as rusty for sequencing.
		result.Need = NeedUrgent
	default:
		result.Need = NeedUnconfirmed
	}
	// Whatever the level, a stale or low-confidence reading is never settled.
	if result.Need == NeedSettled && (projected.Confidence == "low" || projected.NeedsReassessment) {
		result.Need = NeedUnconfirmed
	}
	return result
}

func (v learnerView) gaps(domainName, id string) []string {
	return v.byKey[domainName+"\x00"+id].Gaps
}

func containsRole(roles []MappingRole, role MappingRole) bool {
	for _, value := range roles {
		if value == role {
			return true
		}
	}
	return false
}

func roleOrder(role MappingRole) int {
	switch role {
	case RoleTeaches:
		return 0
	case RoleReinforces:
		return 1
	default:
		return 2
	}
}

func competencyLess(leftDomain, leftID, rightDomain, rightID string) bool {
	if leftDomain != rightDomain {
		return leftDomain < rightDomain
	}
	return leftID < rightID
}
