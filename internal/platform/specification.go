package platform

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/state"
)

// SpecificationVersion is the schemaVersion of Curriculum and Learning Unit
// Specifications.
const SpecificationVersion = 1

// Specification error codes.
const (
	CodeSpecIntegrity = "specification-integrity"
)

// Specification statuses reported when a plan reconciles a draft with the
// latest stored version.
const (
	SpecCreated   = "created"
	SpecRevised   = "revised"
	SpecUnchanged = "unchanged"
	// SpecHeld: a realized unit changed materially without new evidence, so
	// its stored version (and its platform content) stands.
	SpecHeld = "held"
)

// minimumEvidenceLevel is the level required evidence must show for a unit to
// be done.
const minimumEvidenceLevel = "functional"

// SpecProvenance records what a specification version was derived from
// (ADR-0060). It describes the inputs when the version was created; a later
// recompute that changes nothing material keeps the version and its
// provenance.
type SpecProvenance struct {
	LearnerStateRevision string        `json:"learnerStateRevision"`
	ProjectionRevision   string        `json:"projectionRevision"`
	Curriculum           SourceRef     `json:"curriculum"`
	Mapping              MappingSource `json:"mapping"`
	Constraints          *SourceRef    `json:"constraints"`
	Packs                []PackRef     `json:"packs"`
	Sources              []PackSource  `json:"sources"`
}

// PackSource is one source a domain pack was verified against.
type PackSource struct {
	Domain string `json:"domain"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

// SpecVersionRef identifies one immutable version of a specification.
type SpecVersionRef struct {
	Version     int    `json:"version"`
	ContentHash string `json:"contentHash"`
}

// SpecChange explains why a version superseded its predecessor: which
// material fields changed, and whether the change followed new learner state
// (evidence-linked) rather than a target, mapping or pack change alone.
type SpecChange struct {
	MaterialFields []string `json:"materialFields"`
	EvidenceLinked bool     `json:"evidenceLinked"`
}

// LearningUnitSpec is an immutable, versioned Learning Unit Specification:
// the platform-neutral intent for one learner-facing unit (ADR-0056,
// ADR-0060). Teaching is learner-free teaching intent; Adaptation is the
// private learner basis that must never leave the learner workspace.
type LearningUnitSpec struct {
	SchemaVersion   int             `json:"schemaVersion"`
	ID              string          `json:"id"`
	Version         int             `json:"version"`
	ContentHash     string          `json:"contentHash"`
	Supersedes      *SpecVersionRef `json:"supersedes,omitempty"`
	Change          *SpecChange     `json:"change,omitempty"`
	LearnerID       string          `json:"learnerId"`
	Target          ExternalID      `json:"target"`
	Provenance      SpecProvenance  `json:"provenance"`
	AuthoringIntent string          `json:"authoringIntent"`
	Unit            SpecUnit        `json:"unit"`
	Teaching        UnitTeaching    `json:"teaching"`
	Adaptation      UnitAdaptation  `json:"adaptation"`
}

// SpecUnit places a unit in the target: the declared-stable platform item it
// adapts (absent for a proposed unit the platform has not realized) and its
// group in the Curriculum Specification.
type SpecUnit struct {
	PlatformItem *ExternalID `json:"platformItem,omitempty"`
	Group        string      `json:"group"`
}

// UnitTeaching is the learner-free teaching intent of a unit.
type UnitTeaching struct {
	Title            string             `json:"title"`
	Objectives       []string           `json:"objectives"`
	Competencies     []SpecCompetency   `json:"competencies"`
	Prerequisites    []SpecPrerequisite `json:"prerequisites"`
	DependsOn        []string           `json:"dependsOn"`
	RequiredEvidence []RequiredEvidence `json:"requiredEvidence"`
	Claims           []SpecClaim        `json:"claims"`
}

// SpecCompetency is a competency a unit teaches, reinforces or assesses.
type SpecCompetency struct {
	Domain string        `json:"domain"`
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Roles  []MappingRole `json:"roles"`
}

// SpecPrerequisite is prior knowledge the unit assumes.
type SpecPrerequisite struct {
	Domain string `json:"domain"`
	ID     string `json:"id"`
	Name   string `json:"name"`
}

// RequiredEvidence is evidence the unit must make it possible to produce.
type RequiredEvidence struct {
	Domain       string      `json:"domain"`
	Competency   string      `json:"competency"`
	Role         MappingRole `json:"role"`
	MinimumLevel string      `json:"minimumLevel"`
}

// SpecClaim marks a competency's technical claims as version-sensitive or
// source-required: the authoring target must re-verify them at authoring
// time and record source provenance (Q27).
type SpecClaim struct {
	Domain           string `json:"domain"`
	Competency       string `json:"competency"`
	FreshnessClass   string `json:"freshnessClass"`
	VersionSensitive bool   `json:"versionSensitive"`
	SourceRequired   bool   `json:"sourceRequired"`
}

// UnitAdaptation is the private, learner-derived basis of a unit: its
// adaptation mode, the learner statuses it was read from, and misconceptions
// recorded by assessments.
type UnitAdaptation struct {
	Mode           string           `json:"mode"`
	ProposedMode   string           `json:"proposedMode"`
	Decision       string           `json:"decision,omitempty"`
	Rationale      string           `json:"rationale"`
	Competencies   []UnitCompetency `json:"competencies"`
	Misconceptions []Misconception  `json:"misconceptions"`
}

// Misconception is an assessment-recorded gap in one of the unit's
// competencies.
type Misconception struct {
	Domain       string `json:"domain"`
	Competency   string `json:"competency"`
	Gap          string `json:"gap"`
	AssessmentID string `json:"assessmentId"`
}

// CurriculumSpec is an immutable, versioned Curriculum Specification: the
// target's goal, groups and unit graph for one learner (ADR-0060).
type CurriculumSpec struct {
	SchemaVersion   int              `json:"schemaVersion"`
	ID              string           `json:"id"`
	Version         int              `json:"version"`
	ContentHash     string           `json:"contentHash"`
	Supersedes      *SpecVersionRef  `json:"supersedes,omitempty"`
	Change          *SpecChange      `json:"change,omitempty"`
	LearnerID       string           `json:"learnerId"`
	Target          ExternalID       `json:"target"`
	Provenance      SpecProvenance   `json:"provenance"`
	AuthoringIntent string           `json:"authoringIntent"`
	Title           string           `json:"title,omitempty"`
	Goal            []CompetencyRef  `json:"goal"`
	Groups          []SpecGroup      `json:"groups"`
	Units           []CurriculumUnit `json:"units"`
	Sequence        []string         `json:"sequence"`
}

// CompetencyRef names one domain competency.
type CompetencyRef struct {
	Domain string `json:"domain"`
	ID     string `json:"id"`
}

// SpecGroup is an ordered grouping of units. PlatformPhase names the
// platform's existing phase for groups of existing units; proposed units are
// grouped into ALP stages the authoring target maps to its own structure.
type SpecGroup struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	PlatformPhase string   `json:"platformPhase,omitempty"`
	Units         []string `json:"units"`
}

// CurriculumUnit references one unit specification version, in sequence order.
type CurriculumUnit struct {
	ID           string      `json:"id"`
	Version      int         `json:"version"`
	ContentHash  string      `json:"contentHash"`
	Title        string      `json:"title"`
	Group        string      `json:"group"`
	Mode         string      `json:"mode"`
	PlatformItem *ExternalID `json:"platformItem,omitempty"`
}

// SpecRequest holds everything specifications are derived from.
type SpecRequest struct {
	Projection   TargetAdaptationProjection
	Curriculum   Curriculum
	Mapping      MappingReport
	Packs        PackLoader
	LearnerState state.Projection
	Constraints  *TargetConstraints
	// Realizations are the target's recorded Realization Links: a unit-level
	// item that realized a specification keeps that specification's ID.
	Realizations Realizations
}

// SpecDraft is the current derivation of a target's specifications, before
// it is reconciled with stored versions.
type SpecDraft struct {
	Curriculum CurriculumSpec
	Units      []LearningUnitSpec
}

// LearnerStateRevision hashes the learner's competency projection: the
// learner revision specifications are derived from.
func LearnerStateRevision(projection state.Projection) (string, error) {
	data, err := json.Marshal(projection)
	if err != nil {
		return "", fmt.Errorf("encode learner state: %w", err)
	}
	sum := sha256.Sum256(data)
	return "lsr_" + hex.EncodeToString(sum[:12]), nil
}

// CurriculumSpecID is the ALP-owned Curriculum Specification ID of a target.
func CurriculumSpecID(target ExternalID) string {
	return "cspec_" + shortHash(target.Platform, target.Target)
}

// unitSpecID is the ALP-owned Learning Unit Specification ID of a unit: an
// existing unit is keyed by its declared-stable item, a proposed unit by the
// competency it is proposed for.
func unitSpecID(target ExternalID, key ...string) string {
	return "uspec_" + shortHash(append([]string{target.Platform, target.Target}, key...)...)
}

func shortHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:12])
}

// DraftSpecifications derives a target's Curriculum and Learning Unit
// Specifications from its Target Adaptation Projection. It is a pure function
// of its inputs. Units are the target's existing units plus proposed units
// for uncovered prerequisite gaps and uncovered goal competencies; each
// proposed unit is placed before the first existing unit that needs it.
func DraftSpecifications(request SpecRequest) (SpecDraft, error) {
	projection := request.Projection
	target := projection.Target
	learnerRevision, err := LearnerStateRevision(request.LearnerState)
	if err != nil {
		return SpecDraft{}, err
	}
	packs := map[string]domain.Pack{}
	for _, ref := range projection.Inputs.Packs {
		pack, err := request.Packs.Load(ref.Domain)
		if err != nil {
			return SpecDraft{}, fmt.Errorf("load domain pack %q: %w", ref.Domain, err)
		}
		packs[ref.Domain] = pack
	}
	provenance := SpecProvenance{
		LearnerStateRevision: learnerRevision,
		ProjectionRevision:   projection.Revision,
		Curriculum:           projection.Inputs.Curriculum,
		Mapping:              projection.Inputs.Mapping,
		Constraints:          projection.Inputs.Constraints,
		Packs:                append([]PackRef{}, projection.Inputs.Packs...),
		Sources:              packSources(projection.Inputs.Packs, packs),
	}
	builder := specBuilder{
		target: target, learnerID: projection.LearnerID, provenance: provenance, packs: packs,
		learner: newLearnerView(request.LearnerState), constraints: request.Constraints,
		covered: map[string]bool{}, assessed: assessedByUnit(request.Curriculum, request.Mapping),
		realizations: request.Realizations,
	}

	goal, err := resolveGoal(request)
	if err != nil {
		return SpecDraft{}, err
	}

	// Existing units, keyed by the competencies they teach or assess.
	existing := make([]LearningUnitSpec, 0, len(projection.Units))
	teachers := map[string]string{}
	for _, unit := range projection.Units {
		spec := builder.existingUnit(unit)
		existing = append(existing, spec)
		for _, competency := range unit.Competencies {
			builder.covered[competencyKey(competency.Domain, competency.ID)] = true
			if containsRole(competency.Roles, RoleTeaches) || containsRole(competency.Roles, RoleAssesses) {
				key := competencyKey(competency.Domain, competency.ID)
				if _, ok := teachers[key]; !ok {
					teachers[key] = spec.ID
				}
			}
		}
	}

	// Proposed units: uncovered prerequisite gaps of the target, uncovered
	// goal competencies, and the goal's uncovered prerequisites the learner
	// has not shown.
	proposed := map[string]CompetencyRef{}
	for _, gap := range projection.Gaps {
		proposed[competencyKey(gap.Domain, gap.Competency)] = CompetencyRef{Domain: gap.Domain, ID: gap.Competency}
	}
	pending := append([]CompetencyRef{}, goal...)
	for len(pending) != 0 {
		next := pending[0]
		pending = pending[1:]
		key := competencyKey(next.Domain, next.ID)
		if builder.covered[key] || proposed[key] != (CompetencyRef{}) {
			continue
		}
		proposed[key] = next
		definition, _ := builder.definition(next.Domain, next.ID)
		for _, prerequisite := range definition.Prerequisites {
			status := builder.learner.read(next.Domain, prerequisite).Status
			if status == StatusUnassessed || status == StatusRusty {
				pending = append(pending, CompetencyRef{Domain: next.Domain, ID: prerequisite})
			}
		}
	}
	proposedSpecs := map[string]LearningUnitSpec{}
	for key, ref := range proposed {
		spec := builder.proposedUnit(ref)
		proposedSpecs[key] = spec
		teachers[key] = spec.ID
	}

	// Dependencies: units that teach a unit's prerequisites.
	resolveDependencies := func(spec *LearningUnitSpec) {
		spec.Teaching.DependsOn = []string{}
		for _, prerequisite := range spec.Teaching.Prerequisites {
			if teacher, ok := teachers[competencyKey(prerequisite.Domain, prerequisite.ID)]; ok && teacher != spec.ID && !contains(spec.Teaching.DependsOn, teacher) {
				spec.Teaching.DependsOn = append(spec.Teaching.DependsOn, teacher)
			}
		}
	}
	for i := range existing {
		resolveDependencies(&existing[i])
	}
	for key, spec := range proposedSpecs {
		resolveDependencies(&spec)
		proposedSpecs[key] = spec
	}

	// Stage proposed units by prerequisite depth among proposed units.
	byID := map[string]LearningUnitSpec{}
	for _, spec := range proposedSpecs {
		byID[spec.ID] = spec
	}
	depth := map[string]int{}
	var depthOf func(id string, seen map[string]bool) int
	depthOf = func(id string, seen map[string]bool) int {
		if value, ok := depth[id]; ok {
			return value
		}
		if seen[id] {
			return 1
		}
		seen[id] = true
		level := 1
		for _, dependency := range byID[id].Teaching.DependsOn {
			if _, isProposed := byID[dependency]; isProposed {
				if candidate := depthOf(dependency, seen) + 1; candidate > level {
					level = candidate
				}
			}
		}
		depth[id] = level
		return level
	}
	proposedIDs := make([]string, 0, len(byID))
	for id := range byID {
		proposedIDs = append(proposedIDs, id)
	}
	sort.Slice(proposedIDs, func(i, j int) bool {
		left, right := byID[proposedIDs[i]], byID[proposedIDs[j]]
		leftDepth, rightDepth := depthOf(left.ID, map[string]bool{}), depthOf(right.ID, map[string]bool{})
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return builder.packOrder(left.Teaching.Competencies[0]) < builder.packOrder(right.Teaching.Competencies[0])
	})
	for _, id := range proposedIDs {
		spec := byID[id]
		spec.Unit.Group = fmt.Sprintf("stage-%d", depthOf(id, map[string]bool{}))
		byID[id] = spec
	}

	// Order: existing units in target order, each preceded by the proposed
	// units it (transitively) depends on; remaining proposed units last.
	var ordered []LearningUnitSpec
	emitted := map[string]bool{}
	var emit func(id string)
	emit = func(id string) {
		spec, ok := byID[id]
		if !ok || emitted[id] {
			return
		}
		emitted[id] = true
		for _, dependency := range spec.Teaching.DependsOn {
			emit(dependency)
		}
		ordered = append(ordered, spec)
	}
	for _, spec := range existing {
		for _, dependency := range spec.Teaching.DependsOn {
			emit(dependency)
		}
		ordered = append(ordered, spec)
	}
	for _, id := range proposedIDs {
		emit(id)
	}

	curriculum := CurriculumSpec{
		SchemaVersion: SpecificationVersion,
		ID:            CurriculumSpecID(target),
		LearnerID:     projection.LearnerID,
		Target:        target,
		Provenance:    provenance,
		Title:         request.Curriculum.Title,
		Goal:          goal,
		Groups:        []SpecGroup{},
		Units:         []CurriculumUnit{},
		Sequence:      []string{},
	}
	return SpecDraft{Curriculum: curriculum, Units: ordered}, nil
}

// ComposeCurriculum completes a curriculum draft with the reconciled unit
// specification versions, in draft order.
func ComposeCurriculum(draft CurriculumSpec, units []LearningUnitSpec, phases []Phase) CurriculumSpec {
	phaseTitles := map[string]string{}
	isPhase := map[string]bool{}
	for _, phase := range phases {
		phaseTitles[phase.ID] = phase.Title
		isPhase[phase.ID] = true
	}
	groups := map[string]int{}
	newTarget := true
	for _, unit := range units {
		if unit.Unit.PlatformItem != nil {
			newTarget = false
		}
		entry := CurriculumUnit{
			ID: unit.ID, Version: unit.Version, ContentHash: unit.ContentHash, Title: unit.Teaching.Title,
			Group: unit.Unit.Group, Mode: unit.Adaptation.Mode, PlatformItem: unit.Unit.PlatformItem,
		}
		draft.Units = append(draft.Units, entry)
		if entry.Mode != ModeSkip {
			draft.Sequence = append(draft.Sequence, entry.ID)
		}
		index, ok := groups[entry.Group]
		if !ok {
			group := SpecGroup{ID: entry.Group, Title: stageTitle(entry.Group), Units: []string{}}
			// A realized proposed unit keeps its ALP stage group until a new
			// version places it in its platform phase.
			if unit.Unit.PlatformItem != nil && isPhase[entry.Group] {
				group.Title, group.PlatformPhase = phaseTitles[entry.Group], entry.Group
				if group.Title == "" {
					group.Title = entry.Group
				}
			}
			draft.Groups = append(draft.Groups, group)
			index = len(draft.Groups) - 1
			groups[entry.Group] = index
		}
		draft.Groups[index].Units = append(draft.Groups[index].Units, entry.ID)
	}
	draft.AuthoringIntent = curriculumIntent(units, newTarget)
	return draft
}

func stageTitle(group string) string {
	if number, ok := strings.CutPrefix(group, "stage-"); ok {
		return "Stage " + number
	}
	return group
}

// curriculumIntent is the smallest Authoring Intent the target justifies: a
// new target needs its skeleton; otherwise the smallest unit-level need.
func curriculumIntent(units []LearningUnitSpec, newTarget bool) string {
	if newTarget && len(units) != 0 {
		return IntentTargetSkeleton
	}
	smallest := IntentNone
	for _, unit := range units {
		if intentRank(unit.AuthoringIntent) > intentRank(smallest) {
			smallest = unit.AuthoringIntent
		}
	}
	return smallest
}

type specBuilder struct {
	target      ExternalID
	learnerID   string
	provenance  SpecProvenance
	packs       map[string]domain.Pack
	learner     learnerView
	constraints *TargetConstraints
	covered     map[string]bool
	// assessed maps each unit item to the competencies its item tree assesses.
	assessed     map[string]map[string]bool
	realizations Realizations
}

func (b specBuilder) definition(domainName, id string) (domain.Competency, bool) {
	for _, competency := range b.packs[domainName].Competencies {
		if competency.ID == id {
			return competency, true
		}
	}
	return domain.Competency{}, false
}

func (b specBuilder) packOrder(competency SpecCompetency) int {
	for index, definition := range b.packs[competency.Domain].Competencies {
		if definition.ID == competency.ID {
			return index
		}
	}
	return len(b.packs[competency.Domain].Competencies)
}

func (b specBuilder) newUnit(id string) LearningUnitSpec {
	return LearningUnitSpec{
		SchemaVersion: SpecificationVersion,
		ID:            id,
		LearnerID:     b.learnerID,
		Target:        b.target,
		Provenance:    b.provenance,
		Teaching: UnitTeaching{
			Objectives: []string{}, Competencies: []SpecCompetency{}, Prerequisites: []SpecPrerequisite{},
			DependsOn: []string{}, RequiredEvidence: []RequiredEvidence{}, Claims: []SpecClaim{},
		},
		Adaptation: UnitAdaptation{Competencies: []UnitCompetency{}, Misconceptions: []Misconception{}},
	}
}

// existingUnit specifies an existing unit of the target from its projection.
func (b specBuilder) existingUnit(unit AdaptedUnit) LearningUnitSpec {
	item := unit.Item
	id := unitSpecID(b.target, "item", item.Item)
	// A unit-level item that realized a specification (for example a
	// proposed unit the authoring target created) keeps that specification's
	// ID, so the Realization Link stays the path back to it.
	if realized, ok := b.realizations.RootItemOf(item.Item); ok {
		id = realized
	}
	spec := b.newUnit(id)
	spec.Unit = SpecUnit{PlatformItem: &item, Group: unit.Phase}
	spec.Teaching.Title = unit.Title
	spec.Adaptation.Mode, spec.Adaptation.ProposedMode, spec.Adaptation.Rationale = unit.Mode, unit.ProposedMode, unit.Rationale
	if unit.Decision != nil && unit.Decision.Applied {
		spec.Adaptation.Decision = unit.Decision.ID
	}
	b.fill(&spec, unit.Competencies)
	spec.AuthoringIntent = IntentNone
	if spec.Adaptation.Mode != ModeSkip {
		for _, required := range spec.Teaching.RequiredEvidence {
			if !b.assessed[item.Item][competencyKey(required.Domain, required.Competency)] {
				spec.AuthoringIntent = IntentActivity
			}
		}
	}
	return spec
}

// proposedUnit specifies a new unit teaching one competency.
func (b specBuilder) proposedUnit(ref CompetencyRef) LearningUnitSpec {
	spec := b.newUnit(unitSpecID(b.target, "competency", ref.Domain, ref.ID))
	competency := b.learner.read(ref.Domain, ref.ID)
	competency.Roles = []MappingRole{RoleTeaches}
	definition, _ := b.definition(ref.Domain, ref.ID)
	spec.Teaching.Title = definition.Name
	spec.Adaptation.ProposedMode, spec.Adaptation.Rationale = proposeMode([]UnitCompetency{competency})
	spec.Adaptation.Mode = b.constraints.constrain("", spec.Adaptation.ProposedMode)
	if spec.Adaptation.Mode != spec.Adaptation.ProposedMode {
		spec.Adaptation.Rationale += fmt.Sprintf(" Target constraints do not allow %s for this unit.", spec.Adaptation.ProposedMode)
	}
	b.fill(&spec, []UnitCompetency{competency})
	spec.AuthoringIntent = IntentNone
	if spec.Adaptation.Mode != ModeSkip {
		spec.AuthoringIntent = IntentUnit
	}
	return spec
}

// fill derives teaching intent and learner basis from a unit's competencies.
func (b specBuilder) fill(spec *LearningUnitSpec, competencies []UnitCompetency) {
	inUnit := map[string]bool{}
	for _, competency := range competencies {
		inUnit[competencyKey(competency.Domain, competency.ID)] = true
	}
	prerequisites := map[string]bool{}
	for _, competency := range competencies {
		definition, _ := b.definition(competency.Domain, competency.ID)
		primary := containsRole(competency.Roles, RoleTeaches) || containsRole(competency.Roles, RoleAssesses)
		spec.Teaching.Competencies = append(spec.Teaching.Competencies, SpecCompetency{
			Domain: competency.Domain, ID: competency.ID, Name: definition.Name, Roles: append([]MappingRole{}, competency.Roles...),
		})
		if containsRole(competency.Roles, RoleTeaches) {
			spec.Teaching.Objectives = append(spec.Teaching.Objectives, definition.Description)
		}
		if primary {
			for _, prerequisite := range definition.Prerequisites {
				key := competencyKey(competency.Domain, prerequisite)
				if inUnit[key] || prerequisites[key] {
					continue
				}
				prerequisites[key] = true
				required, _ := b.definition(competency.Domain, prerequisite)
				spec.Teaching.Prerequisites = append(spec.Teaching.Prerequisites, SpecPrerequisite{Domain: competency.Domain, ID: prerequisite, Name: required.Name})
			}
			if spec.Adaptation.Mode != ModeSkip {
				spec.Teaching.RequiredEvidence = append(spec.Teaching.RequiredEvidence, RequiredEvidence{
					Domain: competency.Domain, Competency: competency.ID, Role: RoleAssesses, MinimumLevel: minimumEvidenceLevel,
				})
			}
		}
		if claim, ok := claimFor(competency.Domain, competency.ID, definition.FreshnessClass); ok {
			spec.Teaching.Claims = append(spec.Teaching.Claims, claim)
		}
		basis := competency
		basis.Roles = append([]MappingRole{}, competency.Roles...)
		spec.Adaptation.Competencies = append(spec.Adaptation.Competencies, basis)
		for _, gap := range b.learner.gaps(competency.Domain, competency.ID) {
			spec.Adaptation.Misconceptions = append(spec.Adaptation.Misconceptions, Misconception{
				Domain: competency.Domain, Competency: competency.ID, Gap: gap, AssessmentID: competency.AssessmentID,
			})
		}
	}
	sort.Slice(spec.Teaching.Prerequisites, func(i, j int) bool {
		return competencyLess(spec.Teaching.Prerequisites[i].Domain, spec.Teaching.Prerequisites[i].ID, spec.Teaching.Prerequisites[j].Domain, spec.Teaching.Prerequisites[j].ID)
	})
}

// claimFor declares a competency's claims version-sensitive or
// source-required from its pack freshness class.
func claimFor(domainName, id, freshness string) (SpecClaim, bool) {
	claim := SpecClaim{Domain: domainName, Competency: id, FreshnessClass: freshness}
	switch freshness {
	case "version-sensitive-language-runtime", "operational-platform", "security-sensitive":
		claim.VersionSensitive, claim.SourceRequired = true, true
	case "ecosystem-choice":
		claim.SourceRequired = true
	default:
		return SpecClaim{}, false
	}
	return claim, true
}

// assessedByUnit maps each unit item to the competencies an item in its tree
// assesses.
func assessedByUnit(curriculum Curriculum, mapping MappingReport) map[string]map[string]bool {
	roots := rootUnits(curriculum)
	assessed := map[string]map[string]bool{}
	for _, entry := range mapping.Entries {
		unit, ok := roots[entry.Item.Item]
		if !ok {
			continue
		}
		for _, competency := range entry.Competencies {
			if competency.Role != RoleAssesses {
				continue
			}
			id := competency.ResolvedID
			if id == "" {
				id = competency.ID
			}
			if assessed[unit] == nil {
				assessed[unit] = map[string]bool{}
			}
			assessed[unit][competencyKey(competency.Domain, id)] = true
		}
	}
	return assessed
}

func packSources(refs []PackRef, packs map[string]domain.Pack) []PackSource {
	sources := []PackSource{}
	for _, ref := range refs {
		pack := packs[ref.Domain]
		keys := make([]string, 0, len(pack.VerifiedAgainst))
		for key := range pack.VerifiedAgainst {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			sources = append(sources, PackSource{Domain: ref.Domain, Key: key, Value: pack.VerifiedAgainst[key]})
		}
	}
	return sources
}

// resolveGoal resolves the target constraints' goal competencies against the
// packs the mapping declares.
func resolveGoal(request SpecRequest) ([]CompetencyRef, error) {
	goal := []CompetencyRef{}
	if request.Constraints == nil {
		return goal, nil
	}
	var problems []string
	for _, id := range request.Constraints.Goal {
		owner := ""
		for _, pack := range request.Mapping.Packs {
			if pack.Compatible && strings.HasPrefix(id, pack.Domain+".") {
				owner = pack.Domain
			}
		}
		if owner == "" {
			problems = append(problems, fmt.Sprintf("goal competency %q does not belong to a domain pack the mapping declares", id))
			continue
		}
		pack, err := request.Packs.Load(owner)
		if err != nil {
			return nil, fmt.Errorf("load domain pack %q: %w", owner, err)
		}
		found := false
		for _, competency := range pack.Competencies {
			found = found || competency.ID == id
		}
		if !found {
			problems = append(problems, fmt.Sprintf("goal competency %q is not in domain pack %q %s", id, owner, pack.Version))
			continue
		}
		goal = append(goal, CompetencyRef{Domain: owner, ID: id})
	}
	if len(problems) != 0 {
		return nil, &Error{
			Code:     CodeInvalidConstraints,
			Message:  "invalid target constraints: " + strings.Join(problems, "; "),
			Adapter:  request.Curriculum.Target.Platform,
			Target:   request.Curriculum.Target.Target,
			Problems: problems,
		}
	}
	return goal, nil
}

func competencyKey(domainName, id string) string { return domainName + "\x00" + id }

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// UnitRevision is a unit draft reconciled with its latest stored version.
type UnitRevision struct {
	Spec   LearningUnitSpec
	Status string
	// HeldFields are the material fields that changed without new evidence
	// for a realized unit, so its stored version was kept (SpecHeld).
	HeldFields []string
}

// ReviseUnitSpec reconciles a unit draft with the latest stored version under
// the regeneration policy: a new version only when a material field changed
// (competencies, prerequisites, adaptation mode, misconceptions, required
// evidence). Otherwise the stored version stands, with its provenance. Once a
// unit is realized its content exists on the platform, so a material change
// produces a new version (and so a content change) only when it is
// evidence-linked: the learner state changed with it. Other material changes
// to a realized unit are held.
func ReviseUnitSpec(draft LearningUnitSpec, latest *LearningUnitSpec, realized bool) (UnitRevision, error) {
	if latest == nil {
		draft.Version = 1
		hash, err := specHash(draft)
		draft.ContentHash = hash
		return UnitRevision{Spec: draft, Status: SpecCreated}, err
	}
	changed := changedFields([]materialField{
		{"competencies", materialCompetencies(draft.Teaching.Competencies), materialCompetencies(latest.Teaching.Competencies)},
		{"prerequisites", materialPrerequisites(draft.Teaching.Prerequisites), materialPrerequisites(latest.Teaching.Prerequisites)},
		{"adaptationMode", draft.Adaptation.Mode, latest.Adaptation.Mode},
		{"misconceptions", materialMisconceptions(draft.Adaptation.Misconceptions), materialMisconceptions(latest.Adaptation.Misconceptions)},
		{"requiredEvidence", draft.Teaching.RequiredEvidence, latest.Teaching.RequiredEvidence},
	})
	if len(changed) == 0 {
		return UnitRevision{Spec: *latest, Status: SpecUnchanged}, nil
	}
	evidenceLinked := draft.Provenance.LearnerStateRevision != latest.Provenance.LearnerStateRevision
	if realized && !evidenceLinked {
		return UnitRevision{Spec: *latest, Status: SpecHeld, HeldFields: changed}, nil
	}
	draft.Version = latest.Version + 1
	draft.Supersedes = &SpecVersionRef{Version: latest.Version, ContentHash: latest.ContentHash}
	draft.Change = &SpecChange{MaterialFields: changed, EvidenceLinked: evidenceLinked}
	hash, err := specHash(draft)
	draft.ContentHash = hash
	return UnitRevision{Spec: draft, Status: SpecRevised}, err
}

// ReviseCurriculumSpec reconciles a composed curriculum with the latest stored
// version: a new version only when its goal, groups or unit versions changed.
func ReviseCurriculumSpec(draft CurriculumSpec, latest *CurriculumSpec) (CurriculumSpec, string, error) {
	if latest == nil {
		draft.Version = 1
		hash, err := specHash(draft)
		draft.ContentHash = hash
		return draft, SpecCreated, err
	}
	changed := changedFields([]materialField{
		{"goal", draft.Goal, latest.Goal},
		{"groups", draft.Groups, latest.Groups},
		{"units", materialUnits(draft.Units), materialUnits(latest.Units)},
	})
	if len(changed) == 0 {
		return *latest, SpecUnchanged, nil
	}
	draft.Version = latest.Version + 1
	draft.Supersedes = &SpecVersionRef{Version: latest.Version, ContentHash: latest.ContentHash}
	draft.Change = &SpecChange{MaterialFields: changed, EvidenceLinked: draft.Provenance.LearnerStateRevision != latest.Provenance.LearnerStateRevision}
	hash, err := specHash(draft)
	draft.ContentHash = hash
	return draft, SpecRevised, err
}

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

func materialUnits(values []CurriculumUnit) []CurriculumUnit {
	result := make([]CurriculumUnit, len(values))
	for i, value := range values {
		result[i] = CurriculumUnit{ID: value.ID, Version: value.Version, Group: value.Group}
	}
	return result
}

// specHash hashes a specification's canonical JSON without its content hash.
func specHash(spec any) (string, error) {
	value := reflect.ValueOf(&spec).Elem().Elem()
	copied := reflect.New(value.Type()).Elem()
	copied.Set(value)
	if field := copied.FieldByName("ContentHash"); field.IsValid() {
		field.SetString("")
	}
	data, err := json.Marshal(copied.Interface())
	if err != nil {
		return "", fmt.Errorf("encode specification: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// EncodeSpec renders a specification or Authoring Plan as deterministic JSON.
func EncodeSpec(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("encode specification: %w", err)
	}
	return buffer.Bytes(), nil
}

// DecodeUnitSpec parses a stored Learning Unit Specification and verifies its
// content hash: stored versions are immutable.
func DecodeUnitSpec(path string, data []byte) (LearningUnitSpec, error) {
	var spec LearningUnitSpec
	if err := decodeStoredSpec(path, data, &spec); err != nil {
		return LearningUnitSpec{}, err
	}
	return spec, verifySpecHash(path, spec, spec.ContentHash)
}

// DecodeCurriculumSpec parses a stored Curriculum Specification and verifies
// its content hash.
func DecodeCurriculumSpec(path string, data []byte) (CurriculumSpec, error) {
	var spec CurriculumSpec
	if err := decodeStoredSpec(path, data, &spec); err != nil {
		return CurriculumSpec{}, err
	}
	return spec, verifySpecHash(path, spec, spec.ContentHash)
}

func decodeStoredSpec(path string, data []byte, into any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		return &Error{Code: CodeSpecIntegrity, Message: fmt.Sprintf("stored specification %s cannot be read: %v", path, err)}
	}
	return nil
}

func verifySpecHash(path string, spec any, recorded string) error {
	hash, err := specHash(spec)
	if err != nil {
		return err
	}
	if hash != recorded {
		return &Error{Code: CodeSpecIntegrity, Message: fmt.Sprintf(
			"stored specification %s was modified: its content hashes to %s, not its recorded %s; specification versions are immutable", path, hash, recorded)}
	}
	return nil
}
