package platform

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

// MappingSchema is the embedded JSON Schema for platform content mappings.
const MappingSchema = "platform-mapping.schema.json"

// MappingVersion is the only platform mapping schemaVersion ALP accepts
// (ADR-0058). Version 1 lists competencies without Mapping Roles; roles cannot
// be inferred, so v1 is rejected with a rewrite instruction, never migrated.
const MappingVersion = 2

// MappingRole is how a mapped item relates to a competency (ADR-0058).
type MappingRole string

const (
	RoleTeaches    MappingRole = "teaches"
	RoleReinforces MappingRole = "reinforces"
	RoleAssesses   MappingRole = "assesses"
)

// Problem severities.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// Mapping problem codes. Each names one specific reason.
const (
	ProblemUnreadableMapping     = "unreadable-mapping"
	ProblemUnsupportedVersion    = "unsupported-mapping-version"
	ProblemSchema                = "schema"
	ProblemPlatformMismatch      = "platform-mismatch"
	ProblemTargetMismatch        = "target-mismatch"
	ProblemDuplicatePack         = "duplicate-pack"
	ProblemUnknownDomainPack     = "unknown-domain-pack"
	ProblemInvalidPackRange      = "invalid-pack-version-range"
	ProblemPackOutOfRange        = "pack-version-out-of-range"
	ProblemDuplicateItem         = "duplicate-item"
	ProblemUnstableIdentifier    = "unstable-identifier"
	ProblemUndeclaredDomain      = "undeclared-domain"
	ProblemUnknownCompetency     = "unknown-competency"
	ProblemInvalidPackMigration  = "invalid-pack-migration"
	ProblemCompetencyRenamed     = "competency-renamed"
	ProblemCompetencySplit       = "competency-split"
	ProblemCompetencyMerged      = "competency-merged"
	ProblemCompetencyReassess    = "competency-reassess"
	ProblemCompetencyRemoved     = "competency-removed"
	ProblemDuplicateCompetency   = "duplicate-competency"
	ProblemCompetencyNotResolved = "competency-not-resolved"
)

// PackLoader loads domain packs by name; domain.Registry implements it.
type PackLoader interface {
	Load(name string) (domain.Pack, error)
}

// MappingSubject is what a mapping is validated against: the adapter's
// declared-stable identifier kinds, the target's curriculum (the only source
// of which items exist and are stable), and the available domain packs.
type MappingSubject struct {
	Platform    string
	StableKinds []string
	Curriculum  Curriculum
	Packs       PackLoader
}

// MappingReport is the deterministic result of validating one mapping. Every
// problem found is listed with a specific code and reason; Valid is false
// whenever any problem has error severity.
type MappingReport struct {
	Mapping  MappingSource    `json:"mapping"`
	Packs    []PackStatus     `json:"packs"`
	Valid    bool             `json:"valid"`
	Summary  MappingSummary   `json:"summary"`
	Entries  []MappedItem     `json:"entries"`
	Problems []MappingProblem `json:"problems"`
}

// MappingSource identifies the validated mapping document.
type MappingSource struct {
	SchemaVersion int    `json:"schemaVersion,omitempty"`
	ContentHash   string `json:"contentHash"`
}

// PackStatus reports a declared pack compatibility range and its outcome.
type PackStatus struct {
	Domain      string `json:"domain"`
	PackVersion string `json:"packVersion"`
	Version     string `json:"version,omitempty"`
	Compatible  bool   `json:"compatible"`
}

// MappingSummary counts entries and problems by severity.
type MappingSummary struct {
	Entries  int `json:"entries"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

// MappedItem is one mapping entry as resolved against the curriculum and packs.
type MappedItem struct {
	Item            ExternalID         `json:"item"`
	Kind            string             `json:"kind,omitempty"`
	StrengthCeiling string             `json:"strengthCeiling,omitempty"`
	Competencies    []MappedCompetency `json:"competencies"`
}

// MappedCompetency is one competency assignment. ResolvedID is the current
// pack competency the mapped ID resolves to; it is absent when unresolved.
type MappedCompetency struct {
	ID         string      `json:"id"`
	Role       MappingRole `json:"role"`
	Domain     string      `json:"domain,omitempty"`
	ResolvedID string      `json:"resolvedId,omitempty"`
}

// MappingProblem is one specific mapping defect. Path is a JSON Pointer into
// the mapping document.
type MappingProblem struct {
	Severity     string   `json:"severity"`
	Code         string   `json:"code"`
	Path         string   `json:"path,omitempty"`
	Item         string   `json:"item,omitempty"`
	Competency   string   `json:"competency,omitempty"`
	Replacements []string `json:"replacements,omitempty"`
	Message      string   `json:"message"`
}

type mappingDocument struct {
	SchemaVersion int            `json:"schemaVersion"`
	Platform      string         `json:"platform"`
	Target        string         `json:"target"`
	Packs         []mappingPack  `json:"packs"`
	Entries       []mappingEntry `json:"entries"`
}

type mappingPack struct {
	Domain      string `json:"domain"`
	PackVersion string `json:"packVersion"`
}

type mappingEntry struct {
	Item            string              `json:"item"`
	StrengthCeiling string              `json:"strengthCeiling"`
	Competencies    []mappingCompetency `json:"competencies"`
}

type mappingCompetency struct {
	ID   string      `json:"id"`
	Role MappingRole `json:"role"`
}

// ValidateMapping validates a platform mapping document (JSON or YAML, chosen
// by name's extension) against subject. A returned error means validation
// could not run; mapping defects are reported in the MappingReport.
func ValidateMapping(data []byte, name string, subject MappingSubject) (MappingReport, error) {
	sum := sha256.Sum256(data)
	report := &mappingReport{MappingReport: MappingReport{
		Mapping:  MappingSource{ContentHash: "sha256:" + hex.EncodeToString(sum[:])},
		Packs:    []PackStatus{},
		Entries:  []MappedItem{},
		Problems: []MappingProblem{},
	}}

	value, err := workspace.DecodeDocument(name, data)
	if err != nil {
		report.fail(ProblemUnreadableMapping, "", fmt.Sprintf("mapping cannot be parsed: %v", err))
		return report.finish(), nil
	}
	version, ok := declaredVersion(value)
	if ok {
		report.Mapping.SchemaVersion = version
	}
	if ok && version != MappingVersion {
		report.fail(ProblemUnsupportedVersion, "/schemaVersion", unsupportedVersionMessage(version))
		return report.finish(), nil
	}

	validator, err := workspace.NewValidator()
	if err != nil {
		return MappingReport{}, fmt.Errorf("initialize validation: %w", err)
	}
	if issues := validator.ValidateValue(MappingSchema, name, value); len(issues) != 0 {
		// Schema evaluation order follows map iteration; report in a stable order.
		sort.SliceStable(issues, func(i, j int) bool {
			if issues[i].Path != issues[j].Path {
				return issues[i].Path < issues[j].Path
			}
			return issues[i].Reason < issues[j].Reason
		})
		for _, issue := range issues {
			report.fail(ProblemSchema, issue.Path, issue.Reason)
		}
		return report.finish(), nil
	}

	normalized, err := json.Marshal(value)
	if err != nil {
		return MappingReport{}, fmt.Errorf("normalize mapping: %w", err)
	}
	var document mappingDocument
	decoder := json.NewDecoder(bytes.NewReader(normalized))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return MappingReport{}, fmt.Errorf("decode schema-valid mapping: %w", err)
	}
	report.check(document, subject)
	return report.finish(), nil
}

func declaredVersion(value any) (int, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return 0, false
	}
	number, ok := object["schemaVersion"].(float64)
	if !ok || number != float64(int(number)) {
		return 0, false
	}
	return int(number), true
}

func unsupportedVersionMessage(version int) string {
	if version == 1 {
		return "platform mapping schemaVersion 1 is not accepted: v1 lists competencies without Mapping Roles, and ALP will not infer roles. " +
			"Rewrite it as schemaVersion 2: packs[] with semver packVersion ranges, and entries[] mapping each declared-stable item " +
			"to competencies[] with a role (teaches, reinforces or assesses). See docs/PLATFORM_MAPPING.md."
	}
	return fmt.Sprintf("platform mapping schemaVersion %d is not supported (supported: %d)", version, MappingVersion)
}

type mappingReport struct {
	MappingReport
}

func (r *mappingReport) add(problem MappingProblem) {
	r.Problems = append(r.Problems, problem)
}

func (r *mappingReport) fail(code, path, message string) {
	r.add(MappingProblem{Severity: SeverityError, Code: code, Path: path, Message: message})
}

func (r *mappingReport) finish() MappingReport {
	for _, problem := range r.Problems {
		if problem.Severity == SeverityError {
			r.Summary.Errors++
		} else {
			r.Summary.Warnings++
		}
	}
	r.Summary.Entries = len(r.Entries)
	r.Valid = r.Summary.Errors == 0
	return r.MappingReport
}

func (r *mappingReport) check(document mappingDocument, subject MappingSubject) {
	target := subject.Curriculum.Target
	if document.Platform != subject.Platform {
		r.fail(ProblemPlatformMismatch, "/platform",
			fmt.Sprintf("mapping is for platform %q, adapter validates platform %q", document.Platform, subject.Platform))
	}
	if document.Target != target.Target {
		r.fail(ProblemTargetMismatch, "/target",
			fmt.Sprintf("mapping is for target %q, but target %q is being validated", document.Target, target.Target))
	}

	declaredDomains := make([]string, 0, len(document.Packs))
	usable := map[string]domain.Pack{} // declared packs that loaded and satisfy their range
	declared := map[string]bool{}
	for index, declaration := range document.Packs {
		path := fmt.Sprintf("/packs/%d", index)
		if declared[declaration.Domain] {
			r.fail(ProblemDuplicatePack, path+"/domain", fmt.Sprintf("domain pack %q is declared more than once", declaration.Domain))
			continue
		}
		declared[declaration.Domain] = true
		declaredDomains = append(declaredDomains, declaration.Domain)
		status := PackStatus{Domain: declaration.Domain, PackVersion: declaration.PackVersion}
		pack, err := subject.Packs.Load(declaration.Domain)
		if err != nil {
			r.fail(ProblemUnknownDomainPack, path+"/domain",
				fmt.Sprintf("domain pack %q cannot be loaded: %v; its competencies were not resolved", declaration.Domain, err))
			r.Packs = append(r.Packs, status)
			continue
		}
		status.Version = pack.Version
		compatible, err := pack.Supports(declaration.PackVersion)
		switch {
		case err != nil:
			r.fail(ProblemInvalidPackRange, path+"/packVersion",
				fmt.Sprintf("packVersion %q for domain %q is not a valid semver range: %v; its competencies were not resolved", declaration.PackVersion, declaration.Domain, err))
		case !compatible:
			r.fail(ProblemPackOutOfRange, path+"/packVersion",
				fmt.Sprintf("mapping requires domain pack %q %s, but ALP has version %s; its competencies were not resolved", declaration.Domain, declaration.PackVersion, pack.Version))
		default:
			status.Compatible = true
			usable[declaration.Domain] = pack
		}
		r.Packs = append(r.Packs, status)
	}

	kinds := make(map[string]string, len(subject.Curriculum.Items))
	for _, item := range subject.Curriculum.Items {
		kinds[item.Ref.Item] = item.Kind
	}
	stableKinds := append([]string(nil), subject.StableKinds...)
	sort.Strings(stableKinds)

	firstEntry := map[string]int{}
	for index, entry := range document.Entries {
		path := fmt.Sprintf("/entries/%d", index)
		ref := target
		ref.Item = entry.Item
		mapped := MappedItem{Item: ref, Kind: kinds[entry.Item], StrengthCeiling: entry.StrengthCeiling}

		if previous, duplicate := firstEntry[entry.Item]; duplicate {
			r.add(MappingProblem{Severity: SeverityError, Code: ProblemDuplicateItem, Path: path + "/item", Item: entry.Item,
				Message: fmt.Sprintf("item %q is already mapped by /entries/%d; combine its competencies into one entry", entry.Item, previous)})
		} else {
			firstEntry[entry.Item] = index
		}
		if _, listed := kinds[entry.Item]; !listed {
			r.add(MappingProblem{Severity: SeverityError, Code: ProblemUnstableIdentifier, Path: path + "/item", Item: entry.Item,
				Message: fmt.Sprintf("item %q is not a declared-stable identifier of target %q: the curriculum export does not list it. "+
					"Mappings may only reference declared-stable items (%s declares %s); heading-derived or positional IDs are not stable",
					entry.Item, target.Target, subject.Platform, strings.Join(stableKinds, ", "))})
		}

		resolvedIn := map[string]string{}
		for competencyIndex, assignment := range entry.Competencies {
			competencyPath := fmt.Sprintf("%s/competencies/%d", path, competencyIndex)
			owner, resolved := r.resolve(assignment, entry.Item, competencyPath, document.Packs, declaredDomains, usable)
			mapped.Competencies = append(mapped.Competencies, MappedCompetency{ID: assignment.ID, Role: assignment.Role, Domain: owner, ResolvedID: resolved})
			if resolved == "" {
				continue
			}
			if earlier, duplicate := resolvedIn[resolved]; duplicate {
				r.add(MappingProblem{Severity: SeverityError, Code: ProblemDuplicateCompetency, Path: competencyPath + "/id", Item: entry.Item, Competency: assignment.ID,
					Message: fmt.Sprintf("item %q maps competency %q more than once (also at %s); give it a single role", entry.Item, resolved, earlier)})
				continue
			}
			resolvedIn[resolved] = competencyPath
		}
		r.Entries = append(r.Entries, mapped)
	}
}

// resolve resolves one mapped competency through its pack's migrations. It
// returns the owning declared domain ("" when none) and the current competency
// ID, which is "" when the competency cannot be used as mapped.
func (r *mappingReport) resolve(assignment mappingCompetency, item, path string, packs []mappingPack, declaredDomains []string, usable map[string]domain.Pack) (string, string) {
	problem := func(severity, code, message string, replacements []string) {
		r.add(MappingProblem{Severity: severity, Code: code, Path: path + "/id", Item: item, Competency: assignment.ID,
			Replacements: replacements, Message: message})
	}

	owner := ""
	for _, declaration := range packs {
		if strings.HasPrefix(assignment.ID, declaration.Domain+".") {
			owner = declaration.Domain
			break
		}
	}
	if owner == "" {
		problem(SeverityError, ProblemUndeclaredDomain,
			fmt.Sprintf("competency %q does not belong to any domain pack declared in packs (declared: %s)", assignment.ID, strings.Join(declaredDomains, ", ")), nil)
		return owner, ""
	}
	pack, ok := usable[owner]
	if !ok {
		problem(SeverityError, ProblemCompetencyNotResolved,
			fmt.Sprintf("competency %q was not resolved because domain pack %q is unavailable or incompatible (see /packs)", assignment.ID, owner), nil)
		return owner, ""
	}
	packName := fmt.Sprintf("domain pack %q %s", pack.Domain, pack.Version)

	resolution, err := pack.ResolveCompetency(assignment.ID)
	if err != nil {
		if hasMigration(pack, assignment.ID) {
			problem(SeverityError, ProblemInvalidPackMigration,
				fmt.Sprintf("competency %q cannot be resolved because %s declares an invalid migration for it: %v", assignment.ID, packName, err), nil)
			return owner, ""
		}
		problem(SeverityError, ProblemUnknownCompetency,
			fmt.Sprintf("competency %q is not in %s and no pack migration covers it", assignment.ID, packName), nil)
		return owner, ""
	}

	targets := resolution.Targets
	switch resolution.Strategy {
	case "unchanged":
		return owner, assignment.ID
	case "rename":
		problem(SeverityWarning, ProblemCompetencyRenamed,
			fmt.Sprintf("competency %q was renamed to %q in %s; the mapping still resolves, but update it to the new ID", assignment.ID, targets[0], packName), targets)
		return owner, targets[0]
	case "split":
		problem(SeverityError, ProblemCompetencySplit,
			fmt.Sprintf("competency %q was split into %s in %s; ALP will not guess which part item %q %s. Re-map the item to the specific competencies with explicit roles",
				assignment.ID, strings.Join(targets, ", "), packName, item, assignment.Role), targets)
	case "merge":
		problem(SeverityError, ProblemCompetencyMerged,
			fmt.Sprintf("competency %q was merged into %q in %s; merged competencies require reassessment, so the %s role is not carried over automatically. Re-map the item to %q explicitly",
				assignment.ID, targets[0], packName, assignment.Role, targets[0]), targets)
	case "reassess":
		problem(SeverityError, ProblemCompetencyReassess,
			fmt.Sprintf("competency %q was replaced by %s in %s and requires reassessment; re-map the item explicitly",
				assignment.ID, strings.Join(targets, ", "), packName), targets)
	case "remove":
		problem(SeverityError, ProblemCompetencyRemoved,
			fmt.Sprintf("competency %q was removed from %s; remove it from the mapping or map the item to a current competency", assignment.ID, packName), nil)
	default:
		problem(SeverityError, ProblemInvalidPackMigration,
			fmt.Sprintf("competency %q resolves through unsupported migration strategy %q in %s", assignment.ID, resolution.Strategy, packName), targets)
	}
	return owner, ""
}

func hasMigration(pack domain.Pack, id string) bool {
	for _, migration := range pack.Migrations {
		if migration.From == id {
			return true
		}
	}
	return false
}
