package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/state"
)

// Evidence grades. Only activity on an item that assesses a competency, with a
// signal that can be assessed, yields assessment-grade evidence (ADR-0058).
const (
	GradeExposure   = "exposure"
	GradePractice   = "practice"
	GradeAssessment = "assessment"
)

// Import record outcomes and reasons.
const (
	OutcomeImported   = "imported"
	OutcomeSkipped    = "skipped"
	OutcomeSuperseded = "superseded"
	OutcomeUnmapped   = "unmapped"

	ReasonAlreadyImported  = "already-imported"
	ReasonStaleRevision    = "stale-revision"
	ReasonRevisionConflict = "revision-conflict"
	ReasonNotMapped        = "not-mapped"
	ReasonNotInCurriculum  = "not-in-curriculum"
)

// Account is the linked platform account activity is imported for.
type Account struct {
	Platform       string
	Instance       string
	PlatformUserID string
	LearnerID      string
}

// ImportRequest is everything an import decision depends on. Existing is the
// workspace's current evidence; import never reads anything else.
type ImportRequest struct {
	Batch      ActivityBatch
	Curriculum Curriculum
	Mapping    MappingReport
	Account    Account
	Existing   []state.Evidence
}

// ImportPlan is the evidence to append, in order, and the import report.
type ImportPlan struct {
	Evidence []state.Evidence
	Report   ImportReport
}

// ImportReport is the deterministic result of an import.
type ImportReport struct {
	Counts  ImportCounts     `json:"counts"`
	Records []ImportedRecord `json:"records"`
}

// ImportCounts counts activity records by outcome.
type ImportCounts struct {
	Records    int `json:"records"`
	Imported   int `json:"imported"`
	Skipped    int `json:"skipped"`
	Superseded int `json:"superseded"`
	Unmapped   int `json:"unmapped"`
}

// ImportedRecord reports what happened to one activity record.
type ImportedRecord struct {
	Item       string        `json:"item"`
	Kind       string        `json:"kind"`
	Event      EventIdentity `json:"event"`
	Outcome    string        `json:"outcome"`
	Reason     string        `json:"reason,omitempty"`
	MappedItem string        `json:"mappedItem,omitempty"`
	Evidence   []string      `json:"evidence"`
	Supersedes []string      `json:"supersedes,omitempty"`
}

// PlanImport decides, deterministically, which evidence an activity batch
// adds to a workspace (ADR-0059):
//
//   - evidence identity is a hash of platform instance, target, learner
//     (workspace learner and platform user), item, event identity and event
//     revision, per competency domain and evidence grade; timestamps are never
//     part of it;
//   - a revision already imported is skipped;
//   - a later revision of an event supersedes the active evidence for it; an
//     earlier one is skipped as stale;
//   - activity on items the mapping does not cover is reported as unmapped.
func PlanImport(request ImportRequest) (ImportPlan, error) {
	if !request.Mapping.Valid {
		return ImportPlan{}, &Error{Code: CodeInvalidMapping, Target: request.Batch.Target.Target,
			Message: fmt.Sprintf("platform mapping is not valid (%d errors); run alp platform mapping validate", request.Mapping.Summary.Errors)}
	}
	planner := newImportPlanner(request)
	records := append([]ActivityRecord(nil), request.Batch.Records...)
	sort.SliceStable(records, func(i, j int) bool {
		left, right := records[i], records[j]
		if left.Item != right.Item {
			return left.Item < right.Item
		}
		if left.Event.ID != right.Event.ID {
			return left.Event.ID < right.Event.ID
		}
		if !sameInstant(left.ObservedAt, right.ObservedAt) {
			return before(left.ObservedAt, right.ObservedAt)
		}
		return left.Event.Revision < right.Event.Revision
	})

	plan := ImportPlan{Report: ImportReport{Records: make([]ImportedRecord, 0, len(records))}}
	for _, record := range records {
		reported, evidence, err := planner.plan(record)
		if err != nil {
			return ImportPlan{}, err
		}
		plan.Evidence = append(plan.Evidence, evidence...)
		plan.Report.Records = append(plan.Report.Records, reported)
		plan.Report.Counts.Records++
		switch reported.Outcome {
		case OutcomeImported:
			plan.Report.Counts.Imported++
		case OutcomeSkipped:
			plan.Report.Counts.Skipped++
		case OutcomeSuperseded:
			plan.Report.Counts.Superseded++
		case OutcomeUnmapped:
			plan.Report.Counts.Unmapped++
		}
	}
	return plan, nil
}

// importedEvent is the import provenance ALP keeps in evidence metadata.
type importedEvent struct {
	id         string
	key        string
	revision   string
	observedAt string
}

type importPlanner struct {
	request  ImportRequest
	mapped   map[string]MappedItem
	parents  map[string]string
	listed   map[string]bool
	known    map[string]bool            // evidence IDs present
	byKey    map[string][]importedEvent // event key -> evidence
	replaced map[string]map[string]bool // event key -> superseded evidence IDs
}

func newImportPlanner(request ImportRequest) *importPlanner {
	planner := &importPlanner{
		request:  request,
		mapped:   make(map[string]MappedItem, len(request.Mapping.Entries)),
		parents:  make(map[string]string, len(request.Curriculum.Items)),
		listed:   make(map[string]bool, len(request.Curriculum.Items)),
		known:    make(map[string]bool, len(request.Existing)),
		byKey:    map[string][]importedEvent{},
		replaced: map[string]map[string]bool{},
	}
	for _, entry := range request.Mapping.Entries {
		planner.mapped[entry.Item.Item] = entry
	}
	for _, item := range request.Curriculum.Items {
		planner.listed[item.Ref.Item] = true
		if item.Parent != nil {
			planner.parents[item.Ref.Item] = item.Parent.Item
		}
	}
	for _, evidence := range request.Existing {
		planner.remember(evidence)
	}
	return planner
}

// Metadata keys of imported evidence.
const (
	metaActivity      = "activity"
	metaEventKey      = "eventKey"
	metaEventRevision = "eventRevision"
	metaObservedAt    = "observedAt"
)

func (p *importPlanner) remember(evidence state.Evidence) {
	p.known[evidence.ID] = true
	activity, _ := evidence.Metadata[metaActivity].(map[string]any)
	key, _ := activity[metaEventKey].(string)
	if key == "" {
		return
	}
	revision, _ := activity[metaEventRevision].(string)
	observedAt, _ := activity[metaObservedAt].(string)
	p.byKey[key] = append(p.byKey[key], importedEvent{id: evidence.ID, key: key, revision: revision, observedAt: observedAt})
	if p.replaced[key] == nil {
		p.replaced[key] = map[string]bool{}
	}
	for _, id := range evidence.Supersedes {
		p.replaced[key][id] = true
	}
}

// active returns the evidence for key that no other evidence supersedes,
// latest first.
func (p *importPlanner) active(key string) []importedEvent {
	var active []importedEvent
	for _, event := range p.byKey[key] {
		if !p.replaced[key][event.id] {
			active = append(active, event)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		if !sameInstant(active[i].observedAt, active[j].observedAt) {
			return before(active[j].observedAt, active[i].observedAt)
		}
		return active[i].id < active[j].id
	})
	return active
}

// mappedAncestor returns the nearest mapped item at or above item.
func (p *importPlanner) mappedAncestor(item string) (MappedItem, bool) {
	seen := map[string]bool{}
	for current := item; current != "" && !seen[current]; current = p.parents[current] {
		seen[current] = true
		if entry, ok := p.mapped[current]; ok {
			return entry, true
		}
	}
	return MappedItem{}, false
}

// facet is one evidence record's worth of a mapped activity record: the
// competencies of one domain that share an evidence grade.
type facet struct {
	domain       string
	grade        string
	competencies []string
	roles        map[string]any
}

func facets(entry MappedItem, signal Signal) []facet {
	var result []facet
	for _, competency := range entry.Competencies {
		if competency.ResolvedID == "" {
			continue
		}
		grade := gradeFor(competency.Role, signal)
		index := -1
		for i := range result {
			if result[i].domain == competency.Domain && result[i].grade == grade {
				index = i
			}
		}
		if index < 0 {
			result = append(result, facet{domain: competency.Domain, grade: grade, roles: map[string]any{}})
			index = len(result) - 1
		}
		result[index].competencies = append(result[index].competencies, competency.ResolvedID)
		result[index].roles[competency.ResolvedID] = string(competency.Role)
	}
	return result
}

// gradeFor applies a Mapping Role to a signal: teaches is exposure, reinforces
// is practice, and assesses is assessment only when the signal can be
// assessed at all.
func gradeFor(role MappingRole, signal Signal) string {
	switch {
	case role == RoleAssesses && signal.Assessable:
		return GradeAssessment
	case role == RoleTeaches:
		return GradeExposure
	default:
		return GradePractice
	}
}

func (p *importPlanner) plan(record ActivityRecord) (ImportedRecord, []state.Evidence, error) {
	reported := ImportedRecord{Item: record.Item, Kind: record.Kind, Event: record.Event, Evidence: []string{}}
	if !p.listed[record.Item] {
		reported.Outcome, reported.Reason = OutcomeUnmapped, ReasonNotInCurriculum
		return reported, nil, nil
	}
	entry, ok := p.mappedAncestor(record.Item)
	if !ok {
		reported.Outcome, reported.Reason = OutcomeUnmapped, ReasonNotMapped
		return reported, nil, nil
	}
	reported.MappedItem = entry.Item.Item
	groups := facets(entry, record.Signal)
	if len(groups) == 0 {
		reported.Outcome, reported.Reason = OutcomeUnmapped, ReasonNotMapped
		return reported, nil, nil
	}

	var appended []state.Evidence
	for _, group := range groups {
		evidence, reason, err := p.planFacet(record, entry, group)
		if err != nil {
			return ImportedRecord{}, nil, err
		}
		if evidence == nil {
			if reported.Reason == "" {
				reported.Reason = reason
			}
			continue
		}
		reported.Evidence = append(reported.Evidence, evidence.ID)
		reported.Supersedes = append(reported.Supersedes, evidence.Supersedes...)
		appended = append(appended, *evidence)
		p.remember(*evidence)
	}
	switch {
	case len(reported.Supersedes) != 0:
		reported.Outcome, reported.Reason = OutcomeSuperseded, ""
	case len(appended) != 0:
		reported.Outcome, reported.Reason = OutcomeImported, ""
	default:
		reported.Outcome = OutcomeSkipped
	}
	return reported, appended, nil
}

// planFacet returns the evidence to append for one facet, or nil and the
// reason it is skipped.
func (p *importPlanner) planFacet(record ActivityRecord, entry MappedItem, group facet) (*state.Evidence, string, error) {
	account, target := p.request.Account, p.request.Batch.Target.Target
	key := digest(account.Platform, account.Instance, target, account.LearnerID, account.PlatformUserID,
		record.Item, record.Event.ID, group.domain, group.grade)
	id := "ev_" + digest(key, record.Event.Revision)

	var supersedes []string
	active := p.active(key)
	if len(active) != 0 {
		latest := active[0]
		switch {
		case latest.revision == record.Event.Revision:
			return nil, ReasonAlreadyImported, nil
		case sameInstant(record.ObservedAt, latest.observedAt):
			return nil, ReasonRevisionConflict, nil
		case before(record.ObservedAt, latest.observedAt):
			return nil, ReasonStaleRevision, nil
		}
		for _, event := range active {
			supersedes = append(supersedes, event.id)
		}
		sort.Strings(supersedes)
		if p.known[id] {
			// The event returned to an earlier revision: record it again, after
			// the evidence it now supersedes.
			id = "ev_" + digest(key, record.Event.Revision, "after", strings.Join(supersedes, ","))
		}
	}
	if p.known[id] {
		return nil, ReasonAlreadyImported, nil
	}

	evidence := gradedEvidence(record, entry, group)
	evidence.ID = id
	evidence.Supersedes = supersedes
	evidence.Source = state.EvidenceSource{Kind: "platform", Ref: account.Platform + ":" + account.Instance + ":" + target + ":" + record.Event.ID, ContentID: record.Item}
	evidence.Metadata = map[string]any{
		"platform":      account.Platform,
		"evidenceGrade": group.grade,
		"mappingRoles":  group.roles,
		metaActivity: map[string]any{
			"instance":        account.Instance,
			"target":          target,
			"platformUserId":  account.PlatformUserID,
			"item":            record.Item,
			"mappedItem":      entry.Item.Item,
			"kind":            record.Kind,
			"eventId":         record.Event.ID,
			metaEventRevision: record.Event.Revision,
			"syntheticEvent":  record.Event.Synthetic,
			metaObservedAt:    record.ObservedAt,
			"observedResult":  record.Signal.Result,
			metaEventKey:      key,
		},
	}
	return &evidence, "", nil
}

// gradedEvidence builds the evidence body for a facet. Assessment-grade
// evidence keeps the signal's type, result and strength (capped by the
// mapping's ceiling); exposure and practice evidence never claims an outcome.
func gradedEvidence(record ActivityRecord, entry MappedItem, group facet) state.Evidence {
	evidence := state.Evidence{
		SchemaVersion: 1,
		RecordedAt:    record.ObservedAt,
		Domain:        group.domain,
		Competencies:  append([]string(nil), group.competencies...),
	}
	if group.grade == GradeAssessment {
		evidence.Type = record.Signal.EvidenceType
		evidence.Result = record.Signal.Result
		evidence.Strength = capStrength(record.Signal.Strength, entry.StrengthCeiling)
		evidence.FailureClass = record.Signal.FailureClass
		evidence.Observation = record.Signal.Observation
		return evidence
	}
	evidence.Type = "platform-event"
	evidence.Result = "neutral"
	evidence.Strength = "weak"
	evidence.Observation = fmt.Sprintf("%s Recorded as %s evidence, not as an assessment of these competencies.", record.Signal.Observation, group.grade)
	return evidence
}

var strengthOrder = map[string]int{"weak": 0, "moderate": 1, "strong": 2, "production": 3}

func capStrength(strength, ceiling string) string {
	if ceiling == "" {
		return strength
	}
	if strengthOrder[strength] > strengthOrder[ceiling] {
		return ceiling
	}
	return strength
}

func digest(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:16])
}

func instant(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed, err == nil
}

func sameInstant(left, right string) bool {
	l, lok := instant(left)
	r, rok := instant(right)
	if lok && rok {
		return l.Equal(r)
	}
	return left == right
}

func before(left, right string) bool {
	l, lok := instant(left)
	r, rok := instant(right)
	if lok && rok {
		return l.Before(r)
	}
	return left < right
}
