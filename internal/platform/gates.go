package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

// GateResultSchema is the embedded JSON Schema of a Platform Gate Result.
const GateResultSchema = "platform-gate-result.schema.json"

// RealizationReportSchema is the embedded JSON Schema of the realization
// report an authoring target skill hands to `alp platform gates record`.
const RealizationReportSchema = "platform-realization-report.schema.json"

// Platform gate and realization error codes.
const (
	CodeInvalidGateResult        = "invalid-gate-result"
	CodeInvalidRealizationReport = "invalid-realization-report"
	CodeInvalidRealizationLink   = "invalid-realization-link"
	CodeRealizationConflict      = "realization-conflict"
	CodeUnknownAuthoringPlan     = "unknown-authoring-plan"
	CodeAuthoringPlanIntegrity   = "authoring-plan-integrity"
)

// Unit realization statuses recorded with a gate result.
const (
	UnitRealized   = "realized"
	UnitUnrealized = "unrealized"
)

// Reasons a unit stays unrealized.
const (
	ReasonNotPublishable         = "not-publishable"
	ReasonRealizationNotReported = "realization-not-reported"
	ReasonProvenanceMissing      = "provenance-missing"
)

// GateResultReader is implemented by adapters that declare the Platform
// Validator capability: they read the platform's own structured gate output
// as a Platform Gate Result for one Learning Target. ALP records the gates'
// commands, artifacts and diagnostics but never interprets them.
type GateResultReader interface {
	ReadGateResult(data []byte, target string) (GateResult, error)
}

// GateResult is a Platform Gate Result (schemas/platform-gate-result.schema.json).
type GateResult struct {
	SchemaVersion int    `json:"schemaVersion"`
	Platform      string `json:"platform"`
	Target        string `json:"target"`
	Publishable   bool   `json:"publishable"`
	Gates         []Gate `json:"gates"`
}

// Gate is one platform quality gate's result.
type Gate struct {
	ID          string           `json:"id"`
	Status      string           `json:"status"`
	Reason      string           `json:"reason,omitempty"`
	Provenance  GateProvenance   `json:"provenance"`
	Artifacts   []GateArtifact   `json:"artifacts"`
	Diagnostics []GateDiagnostic `json:"diagnostics"`
}

// GateProvenance is the platform-native command and version that ran a gate.
type GateProvenance struct {
	Command    string `json:"command"`
	Version    string `json:"version"`
	ExitCode   *int   `json:"exitCode,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

// GateArtifact is a platform artifact a gate checked.
type GateArtifact struct {
	Ref         string `json:"ref"`
	ContentHash string `json:"contentHash,omitempty"`
}

// GateDiagnostic is one platform-native finding of a gate.
type GateDiagnostic struct {
	Severity string      `json:"severity"`
	Code     string      `json:"code,omitempty"`
	Message  string      `json:"message"`
	Item     *ExternalID `json:"item,omitempty"`
	Artifact string      `json:"artifact,omitempty"`
	Line     int         `json:"line,omitempty"`
}

// GateExpectation is what an adapter requires of a gate result.
type GateExpectation struct {
	Platform string
	Target   string
}

// ReadGateResult validates a Platform Gate Result against the published
// schema and the adapter's platform and target.
func ReadGateResult(data []byte, expect GateExpectation) (GateResult, error) {
	invalid := func(problems []string) error {
		return &Error{
			Code:     CodeInvalidGateResult,
			Message:  fmt.Sprintf("invalid platform gate result for target %q: %s", expect.Target, strings.Join(problems, "; ")),
			Adapter:  expect.Platform,
			Target:   expect.Target,
			Problems: problems,
		}
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		return GateResult{}, fmt.Errorf("initialize validation: %w", err)
	}
	if issue := validator.ValidateDocument(GateResultSchema, "gate-result.json", data); issue != nil {
		return GateResult{}, invalid([]string{issue.Error()})
	}
	var result GateResult
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return GateResult{}, invalid([]string{err.Error()})
	}
	var problems []string
	if result.Platform != expect.Platform {
		problems = append(problems, fmt.Sprintf("result is for platform %q, adapter validates platform %q", result.Platform, expect.Platform))
	}
	if result.Target != expect.Target {
		problems = append(problems, fmt.Sprintf("result is for target %q, not %q", result.Target, expect.Target))
	}
	seen := map[string]bool{}
	for _, gate := range result.Gates {
		if seen[gate.ID] {
			problems = append(problems, fmt.Sprintf("gate %q is reported more than once", gate.ID))
		}
		seen[gate.ID] = true
	}
	if len(problems) != 0 {
		return GateResult{}, invalid(problems)
	}
	return result, nil
}

// RealizationReport is what the authoring target skill reports it realized
// for an Authoring Plan: per unit specification, the declared-stable items
// that realize it and the source provenance of its claims (Q27).
type RealizationReport struct {
	SchemaVersion int            `json:"schemaVersion"`
	Plan          string         `json:"plan"`
	Units         []ReportedUnit `json:"units"`
}

// ReportedUnit is one unit specification's realization as reported.
type ReportedUnit struct {
	Unit   string          `json:"unit"`
	Items  []string        `json:"items"`
	Claims []ReportedClaim `json:"claims"`
}

// ReportedClaim is the source provenance recorded for one claim.
type ReportedClaim struct {
	Domain     string        `json:"domain"`
	Competency string        `json:"competency"`
	Sources    []ClaimSource `json:"sources"`
}

// ClaimSource is one source a claim was re-verified against at authoring time.
type ClaimSource struct {
	URL        string `json:"url"`
	Version    string `json:"version,omitempty"`
	VerifiedAt string `json:"verifiedAt"`
}

// ReadRealizationReport validates a realization report against its schema.
func ReadRealizationReport(data []byte, target ExternalID) (RealizationReport, error) {
	invalid := func(problems []string) error {
		return &Error{
			Code:     CodeInvalidRealizationReport,
			Message:  fmt.Sprintf("invalid realization report for target %q: %s", target.Target, strings.Join(problems, "; ")),
			Adapter:  target.Platform,
			Target:   target.Target,
			Problems: problems,
		}
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		return RealizationReport{}, fmt.Errorf("initialize validation: %w", err)
	}
	if issue := validator.ValidateDocument(RealizationReportSchema, "realization.json", data); issue != nil {
		return RealizationReport{}, invalid([]string{issue.Error()})
	}
	var report RealizationReport
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return RealizationReport{}, invalid([]string{err.Error()})
	}
	return report, nil
}

// GateRecordVersion is the schemaVersion of Platform Gate Records and
// Realization Links.
const GateRecordVersion = 1

// GateRecord is the canonical, append-only record of one Platform Gate Result
// attached to an Authoring Plan, with the realization outcome of every unit
// in the plan's scope (schemas/platform-gate-record.schema.json).
type GateRecord struct {
	SchemaVersion int           `json:"schemaVersion"`
	ID            string        `json:"id"`
	LearnerID     string        `json:"learnerId"`
	Target        ExternalID    `json:"target"`
	Plan          GatePlanRef   `json:"plan"`
	RecordedAt    string        `json:"recordedAt"`
	Result        GateResult    `json:"result"`
	Units         []UnitOutcome `json:"units"`
}

// GatePlanRef cites the Authoring Plan a gate result is attached to.
type GatePlanRef struct {
	ID     string `json:"id"`
	Intent string `json:"intent"`
}

// UnitOutcome is whether one unit specification version is realized by the
// recorded gate result, and why not.
type UnitOutcome struct {
	Unit        SpecRef      `json:"unit"`
	Status      string       `json:"status"`
	Reasons     []UnitReason `json:"reasons"`
	Realization string       `json:"realization,omitempty"`
}

// UnitReason explains why a unit is not realized.
type UnitReason struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Domain     string `json:"domain,omitempty"`
	Competency string `json:"competency,omitempty"`
}

// RealizationLink records which declared-stable platform items realized one
// Learning Unit Specification version (ADR-0057): the path platform activity
// is traced back to the specification through. Root is the unit-level item
// (placed in a phase); every other item descends from it.
type RealizationLink struct {
	SchemaVersion int             `json:"schemaVersion"`
	ID            string          `json:"id"`
	LearnerID     string          `json:"learnerId"`
	Target        ExternalID      `json:"target"`
	Unit          SpecRef         `json:"unit"`
	Plan          string          `json:"plan"`
	GateRecord    string          `json:"gateRecord"`
	RecordedAt    string          `json:"recordedAt"`
	Root          ExternalID      `json:"root"`
	Items         []ExternalID    `json:"items"`
	Claims        []RealizedClaim `json:"claims"`
}

// RealizedClaim is a unit claim with the source provenance recorded for it.
type RealizedClaim struct {
	SpecClaim
	Sources []ClaimSource `json:"sources"`
}

// Realizations indexes a target's recorded Realization Links.
type Realizations struct {
	// ByUnit maps a unit specification ID to the root item that realized it.
	ByUnit map[string]ExternalID
	// ByItem maps every realizing item to its unit specification ID.
	ByItem map[string]string
	// Versions holds "<unit spec ID>@<version>" for realized versions.
	Versions map[string]bool
}

// IndexRealizations indexes the links recorded for target.
func IndexRealizations(links []RealizationLink, target ExternalID) Realizations {
	index := Realizations{ByUnit: map[string]ExternalID{}, ByItem: map[string]string{}, Versions: map[string]bool{}}
	for _, link := range links {
		if link.Target.Platform != target.Platform || link.Target.Target != target.Target {
			continue
		}
		index.ByUnit[link.Unit.ID] = link.Root
		for _, item := range link.Items {
			index.ByItem[item.Item] = link.Unit.ID
		}
		index.Versions[versionKey(link.Unit.ID, link.Unit.Version)] = true
	}
	return index
}

// RootItemOf returns the unit spec ID realized as a unit-level item.
func (r Realizations) RootItemOf(item string) (string, bool) {
	for unit, root := range r.ByUnit {
		if root.Item == item {
			return unit, true
		}
	}
	return "", false
}

// Realized reports whether any version of a unit specification is realized.
func (r Realizations) Realized(unit string) bool {
	_, ok := r.ByUnit[unit]
	return ok
}

// VersionRealized reports whether one unit specification version is realized.
func (r Realizations) VersionRealized(unit string, version int) bool {
	return r.Versions[versionKey(unit, version)]
}

func versionKey(unit string, version int) string { return fmt.Sprintf("%s@%d", unit, version) }

// GateRequest is everything a gate result's realization outcome is derived
// from.
type GateRequest struct {
	Plan         AuthoringPlan
	Units        []LearningUnitSpec
	Result       GateResult
	Report       *RealizationReport
	Curriculum   Curriculum
	Realizations Realizations
	RecordedAt   string
}

// GateOutcome is a gate record with the Realization Links it creates.
type GateOutcome struct {
	Record GateRecord
	Links  []RealizationLink
}

// RecordGates attaches a Platform Gate Result to an Authoring Plan and decides
// each unit's realization (ADR-0057, Q22, Q27): a unit is realized only when
// the platform reports the result publishable, the authoring target reported
// declared-stable items realizing it, and every one of its version-sensitive
// or source-required claims has recorded source provenance. A report that
// references unknown units, claims, or items that are not declared-stable
// items of the target's curriculum is refused outright.
func RecordGates(request GateRequest) (GateOutcome, error) {
	plan := request.Plan
	target := plan.Target
	reported := map[string]ReportedUnit{}
	if report := request.Report; report != nil {
		if report.Plan != plan.ID {
			return GateOutcome{}, reportError(target, fmt.Sprintf("realization report is for authoring plan %q, not %q", report.Plan, plan.ID))
		}
		inPlan := map[string]bool{}
		for _, unit := range plan.Units {
			inPlan[unit.ID] = true
		}
		for _, unit := range report.Units {
			if !inPlan[unit.Unit] {
				return GateOutcome{}, reportError(target, fmt.Sprintf("unit %q is not in authoring plan %s", unit.Unit, plan.ID))
			}
			if _, duplicate := reported[unit.Unit]; duplicate {
				return GateOutcome{}, reportError(target, fmt.Sprintf("unit %q is reported more than once", unit.Unit))
			}
			reported[unit.Unit] = unit
		}
	}

	id, err := specHash(struct {
		Plan   string             `json:"plan"`
		Result GateResult         `json:"result"`
		Report *RealizationReport `json:"report"`
	}{plan.ID, request.Result, request.Report})
	if err != nil {
		return GateOutcome{}, err
	}
	record := GateRecord{
		SchemaVersion: GateRecordVersion,
		ID:            "pgr_" + id[len("sha256:"):len("sha256:")+24],
		LearnerID:     plan.LearnerID,
		Target:        target,
		Plan:          GatePlanRef{ID: plan.ID, Intent: plan.Intent},
		RecordedAt:    request.RecordedAt,
		Result:        request.Result,
		Units:         []UnitOutcome{},
	}
	outcome := GateOutcome{Links: []RealizationLink{}}
	claimed := map[string]string{}
	for _, spec := range request.Units {
		unit := UnitOutcome{Unit: SpecRef{ID: spec.ID, Version: spec.Version, ContentHash: spec.ContentHash}, Reasons: []UnitReason{}}
		if !request.Result.Publishable {
			unit.Reasons = append(unit.Reasons, UnitReason{Code: ReasonNotPublishable, Message: notPublishable(request.Result)})
		}
		var root ExternalID
		var items []ExternalID
		var claims []RealizedClaim
		if report, ok := reported[spec.ID]; ok {
			root, items, err = realizationItems(spec, report.Items, request)
			if err != nil {
				return GateOutcome{}, err
			}
			for _, item := range items {
				if other, ok := claimed[item.Item]; ok && other != spec.ID {
					return GateOutcome{}, linkError(target, CodeRealizationConflict, fmt.Sprintf("item %q is reported for both %s and %s", item.Item, other, spec.ID))
				}
				claimed[item.Item] = spec.ID
			}
			var reasons []UnitReason
			claims, reasons, err = claimProvenance(spec, report.Claims)
			if err != nil {
				return GateOutcome{}, err
			}
			unit.Reasons = append(unit.Reasons, reasons...)
		} else {
			unit.Reasons = append(unit.Reasons, UnitReason{Code: ReasonRealizationNotReported,
				Message: fmt.Sprintf("the authoring target reported no platform items realizing %s v%d", spec.ID, spec.Version)})
		}
		unit.Status = UnitUnrealized
		if len(unit.Reasons) == 0 {
			unit.Status = UnitRealized
			linkID, err := specHash(struct {
				Unit       SpecRef `json:"unit"`
				GateRecord string  `json:"gateRecord"`
			}{unit.Unit, record.ID})
			if err != nil {
				return GateOutcome{}, err
			}
			link := RealizationLink{
				SchemaVersion: GateRecordVersion,
				ID:            "rlz_" + linkID[len("sha256:"):len("sha256:")+24],
				LearnerID:     plan.LearnerID,
				Target:        target,
				Unit:          unit.Unit,
				Plan:          plan.ID,
				GateRecord:    record.ID,
				RecordedAt:    request.RecordedAt,
				Root:          root,
				Items:         items,
				Claims:        claims,
			}
			unit.Realization = link.ID
			outcome.Links = append(outcome.Links, link)
		}
		record.Units = append(record.Units, unit)
	}
	outcome.Record = record
	return outcome, nil
}

func reportError(target ExternalID, message string) error {
	return &Error{Code: CodeInvalidRealizationReport, Message: message, Adapter: target.Platform, Target: target.Target}
}

func linkError(target ExternalID, code, message string) error {
	return &Error{Code: code, Message: message, Adapter: target.Platform, Target: target.Target}
}

// notPublishable explains a result the platform judged not publishable,
// naming the gates that failed or were skipped without interpreting them.
func notPublishable(result GateResult) string {
	var failed, skipped []string
	for _, gate := range result.Gates {
		switch gate.Status {
		case "fail":
			failed = append(failed, gate.ID)
		case "skipped":
			skipped = append(skipped, gate.ID)
		}
	}
	message := fmt.Sprintf("platform %s reported the result for target %q not publishable", result.Platform, result.Target)
	var details []string
	if len(failed) != 0 {
		details = append(details, "failed gates: "+strings.Join(failed, ", "))
	}
	if len(skipped) != 0 {
		details = append(details, "skipped gates: "+strings.Join(skipped, ", "))
	}
	if len(details) != 0 {
		message += " (" + strings.Join(details, "; ") + ")"
	}
	return message
}

// realizationItems resolves reported item IDs against the target's
// curriculum: every item must be a declared-stable item of the target,
// exactly one must be a unit-level item (placed in a phase), and the rest
// must descend from it. An existing unit is realized by its own platform
// item, and a unit already realized keeps its root, so its specification ID
// stays stable.
func realizationItems(spec LearningUnitSpec, reported []string, request GateRequest) (ExternalID, []ExternalID, error) {
	target := request.Plan.Target
	invalid := func(message string) error {
		return linkError(target, CodeInvalidRealizationLink, fmt.Sprintf("realization of %s: %s", spec.ID, message))
	}
	byID := map[string]Item{}
	for _, item := range request.Curriculum.Items {
		byID[item.Ref.Item] = item
	}
	var root *Item
	seen := map[string]bool{}
	items := make([]ExternalID, 0, len(reported))
	for _, id := range reported {
		item, ok := byID[id]
		if !ok {
			return ExternalID{}, nil, invalid(fmt.Sprintf("%q is not a declared-stable item of target %q in the curriculum export", id, target.Target))
		}
		if seen[id] {
			return ExternalID{}, nil, invalid(fmt.Sprintf("item %q is reported more than once", id))
		}
		seen[id] = true
		if item.Parent == nil {
			if root != nil {
				return ExternalID{}, nil, invalid(fmt.Sprintf("items %q and %q are both unit-level items; a unit is realized as one unit-level item and its descendants", root.Ref.Item, id))
			}
			found := item
			root = &found
		}
		items = append(items, item.Ref)
	}
	if root == nil {
		return ExternalID{}, nil, invalid("no reported item is a unit-level item (one placed in a phase)")
	}
	for _, ref := range items {
		if ref.Item != root.Ref.Item && rootOf(byID, ref.Item) != root.Ref.Item {
			return ExternalID{}, nil, invalid(fmt.Sprintf("item %q does not descend from unit-level item %q", ref.Item, root.Ref.Item))
		}
	}
	if existing := spec.Unit.PlatformItem; existing != nil && existing.Item != root.Ref.Item {
		return ExternalID{}, nil, invalid(fmt.Sprintf("unit-level item %q is not the unit's platform item %q", root.Ref.Item, existing.Item))
	}
	if previous, ok := request.Realizations.ByUnit[spec.ID]; ok && previous.Item != root.Ref.Item {
		return ExternalID{}, nil, linkError(target, CodeRealizationConflict,
			fmt.Sprintf("%s is already realized as unit-level item %q, not %q", spec.ID, previous.Item, root.Ref.Item))
	}
	for _, ref := range items {
		if other, ok := request.Realizations.ByItem[ref.Item]; ok && other != spec.ID {
			return ExternalID{}, nil, linkError(target, CodeRealizationConflict,
				fmt.Sprintf("item %q already realizes %s; it cannot also realize %s", ref.Item, other, spec.ID))
		}
	}
	if other, ok := request.Realizations.RootItemOf(root.Ref.Item); ok && other != spec.ID {
		return ExternalID{}, nil, linkError(target, CodeRealizationConflict,
			fmt.Sprintf("unit-level item %q already realizes %s; it cannot also realize %s", root.Ref.Item, other, spec.ID))
	}
	return root.Ref, items, nil
}

// rootOf returns the unit-level ancestor of an item.
func rootOf(byID map[string]Item, id string) string {
	for depth := 0; depth <= len(byID); depth++ {
		item := byID[id]
		if item.Parent == nil {
			return id
		}
		id = item.Parent.Item
	}
	return ""
}

// claimProvenance matches reported source provenance to the unit's claims.
// Every claim requires at least one source with a URL and verification date;
// a version-sensitive claim's sources must also name the version verified.
func claimProvenance(spec LearningUnitSpec, reported []ReportedClaim) ([]RealizedClaim, []UnitReason, error) {
	target := spec.Target
	byKey := map[string]ReportedClaim{}
	for _, claim := range reported {
		key := competencyKey(claim.Domain, claim.Competency)
		if _, duplicate := byKey[key]; duplicate {
			return nil, nil, reportError(target, fmt.Sprintf("claim %s %s of %s is reported more than once", claim.Domain, claim.Competency, spec.ID))
		}
		found := false
		for _, declared := range spec.Teaching.Claims {
			found = found || (declared.Domain == claim.Domain && declared.Competency == claim.Competency)
		}
		if !found {
			return nil, nil, reportError(target, fmt.Sprintf("%s declares no claim for %s %s", spec.ID, claim.Domain, claim.Competency))
		}
		byKey[key] = claim
	}
	claims := []RealizedClaim{}
	reasons := []UnitReason{}
	for _, declared := range spec.Teaching.Claims {
		if !declared.VersionSensitive && !declared.SourceRequired {
			continue
		}
		reportedClaim := byKey[competencyKey(declared.Domain, declared.Competency)]
		sources := append([]ClaimSource{}, reportedClaim.Sources...)
		missing := func(message string) {
			reasons = append(reasons, UnitReason{Code: ReasonProvenanceMissing, Domain: declared.Domain, Competency: declared.Competency, Message: message})
		}
		switch {
		case len(sources) == 0:
			missing(fmt.Sprintf("claim %s (%s) requires source provenance: re-verify it at authoring time and record the source URL, version and date", declared.Competency, declared.FreshnessClass))
		case declared.VersionSensitive && !anyVersioned(sources):
			missing(fmt.Sprintf("claim %s is version-sensitive (%s): its source provenance must name the version verified", declared.Competency, declared.FreshnessClass))
		}
		claims = append(claims, RealizedClaim{SpecClaim: declared, Sources: sources})
	}
	sort.SliceStable(reasons, func(i, j int) bool {
		return competencyLess(reasons[i].Domain, reasons[i].Competency, reasons[j].Domain, reasons[j].Competency)
	})
	return claims, reasons, nil
}

func anyVersioned(sources []ClaimSource) bool {
	for _, source := range sources {
		if source.Version != "" {
			return true
		}
	}
	return false
}

// DecodeAuthoringPlan parses a recorded Authoring Plan and verifies that its
// content still hashes to its ID: authoring plans are immutable.
func DecodeAuthoringPlan(path string, data []byte) (AuthoringPlan, error) {
	var plan AuthoringPlan
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return AuthoringPlan{}, &Error{Code: CodeAuthoringPlanIntegrity, Message: fmt.Sprintf("recorded authoring plan %s cannot be read: %v", path, err)}
	}
	unsigned := plan
	unsigned.ID = ""
	hash, err := specHash(unsigned)
	if err != nil {
		return AuthoringPlan{}, err
	}
	if want := "apl_" + hash[len("sha256:"):len("sha256:")+24]; want != plan.ID {
		return AuthoringPlan{}, &Error{Code: CodeAuthoringPlanIntegrity, Message: fmt.Sprintf(
			"recorded authoring plan %s was modified: its content hashes to %s; authoring plans are immutable", path, want)}
	}
	return plan, nil
}

// DecodeRealizationLink parses a recorded Realization Link.
func DecodeRealizationLink(path string, data []byte) (RealizationLink, error) {
	var link RealizationLink
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&link); err != nil {
		return RealizationLink{}, fmt.Errorf("read realization link %s: %w", path, err)
	}
	return link, nil
}

// DecodeGateRecord parses a recorded Platform Gate Record.
func DecodeGateRecord(path string, data []byte) (GateRecord, error) {
	var record GateRecord
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return GateRecord{}, fmt.Errorf("read gate record %s: %w", path, err)
	}
	return record, nil
}
