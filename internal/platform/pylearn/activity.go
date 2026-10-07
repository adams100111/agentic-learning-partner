package pylearn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

// ExportSchema is the embedded JSON Schema for PyLearn activity exports.
const ExportSchema = "pylearn-export.schema.json"

// ExportVersion is the only pylearn-export schemaVersion ALP reads (ADR-0059).
const ExportVersion = 2

var _ platform.ActivityReader = Adapter{}

type activityExport struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Platform      string               `json:"platform"`
	Instance      string               `json:"instance"`
	ExportedAt    string               `json:"exportedAt"`
	User          exportUser           `json:"user"`
	Cursor        platform.CursorRange `json:"cursor"`
	Targets       []activityTarget     `json:"targets"`
}

type exportUser struct {
	ID string `json:"id"`
}

type activityTarget struct {
	ID      string           `json:"id"`
	Records []activityRecord `json:"records"`
}

type activityRecord struct {
	Kind       string                 `json:"kind"`
	Item       string                 `json:"item"`
	Event      platform.EventIdentity `json:"event"`
	ObservedAt string                 `json:"observedAt"`
	Status     string                 `json:"status"`
	Picked     *int                   `json:"picked"`
	Correct    *bool                  `json:"correct"`
	Text       string                 `json:"text"`
	Variant    string                 `json:"variant"`
	Failure    string                 `json:"failure"`
}

// ReadActivity reads one target's activity from a `pylearn-export` v2
// document produced by PyLearn's `export:activity`, and applies PyLearn's
// signal-kind policy to each record.
func (a Adapter) ReadActivity(export []byte, target string, since *string) (platform.ActivityBatch, error) {
	var header struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(export, &header); err != nil {
		return platform.ActivityBatch{}, invalidActivity(target, []string{fmt.Sprintf("export is not JSON: %v", err)})
	}
	if header.SchemaVersion != ExportVersion {
		return platform.ActivityBatch{}, invalidActivity(target, []string{fmt.Sprintf(
			"pylearn-export schemaVersion %d is not supported: ALP reads schemaVersion %d (activity grouped by targets, a cursor, and an event identity per record); run PyLearn's export:activity",
			header.SchemaVersion, ExportVersion)})
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		return platform.ActivityBatch{}, fmt.Errorf("initialize validation: %w", err)
	}
	value, err := workspace.DecodeDocument("activity-export.json", export)
	if err != nil {
		return platform.ActivityBatch{}, invalidActivity(target, []string{err.Error()})
	}
	if issues := validator.ValidateValue(ExportSchema, "activity-export.json", value); len(issues) != 0 {
		problems := make([]string, 0, len(issues))
		for _, issue := range issues {
			problems = append(problems, issue.Error())
		}
		sort.Strings(problems)
		return platform.ActivityBatch{}, invalidActivity(target, problems)
	}
	var document activityExport
	decoder := json.NewDecoder(bytes.NewReader(export))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return platform.ActivityBatch{}, fmt.Errorf("decode schema-valid activity export: %w", err)
	}

	var selected *activityTarget
	available := make([]string, 0, len(document.Targets))
	for i := range document.Targets {
		available = append(available, document.Targets[i].ID)
		if document.Targets[i].ID == target {
			if selected != nil {
				return platform.ActivityBatch{}, invalidActivity(target, []string{fmt.Sprintf("target %q appears more than once", target)})
			}
			selected = &document.Targets[i]
		}
	}
	if selected == nil {
		sort.Strings(available)
		return platform.ActivityBatch{}, &platform.Error{
			Code:      platform.CodeUnknownTarget,
			Message:   fmt.Sprintf("activity export has no target %q (available: %s)", target, strings.Join(available, ", ")),
			Adapter:   AdapterID,
			Target:    target,
			Available: available,
		}
	}
	if err := platform.CheckCursor(AdapterID, target, since, document.Cursor.Since); err != nil {
		return platform.ActivityBatch{}, err
	}

	batch := platform.ActivityBatch{
		Target:         platform.ExternalID{Platform: AdapterID, Target: target},
		Instance:       document.Instance,
		PlatformUserID: document.User.ID,
		Cursor:         document.Cursor,
		Records:        make([]platform.ActivityRecord, 0, len(selected.Records)),
	}
	for _, record := range selected.Records {
		signal, ok := signalFor(record)
		if !ok {
			return platform.ActivityBatch{}, invalidActivity(target, []string{fmt.Sprintf("record kind %q has no signal policy", record.Kind)})
		}
		batch.Records = append(batch.Records, platform.ActivityRecord{
			Item:       record.Item,
			Kind:       record.Kind,
			Event:      record.Event,
			ObservedAt: record.ObservedAt,
			Signal:     signal,
		})
	}
	return batch, nil
}

// signalFor is PyLearn's signal-kind policy: what each kind of activity can
// show before Mapping Roles apply (ADR-0058). Navigation progress and
// reflections are never assessable; answered questions and checked exercise
// attempts are.
func signalFor(record activityRecord) (platform.Signal, bool) {
	switch record.Kind {
	case "progress":
		return platform.Signal{EvidenceType: "platform-event", Result: "neutral", Strength: "weak",
			Observation: "PyLearn lesson progress is " + strings.ReplaceAll(record.Status, "_", " ") + "."}, true
	case "quiz-answer":
		if *record.Correct {
			return platform.Signal{EvidenceType: "quiz", Result: "pass", Strength: "moderate", Assessable: true,
				Observation: fmt.Sprintf("PyLearn quiz answer was correct (picked option %d).", *record.Picked)}, true
		}
		return platform.Signal{EvidenceType: "quiz", Result: "fail", Strength: "weak", FailureClass: "conceptual-miss", Assessable: true,
			Observation: fmt.Sprintf("PyLearn quiz answer was incorrect (picked option %d).", *record.Picked)}, true
	case "reflection":
		return platform.Signal{EvidenceType: "reflection", Result: "neutral", Strength: "weak",
			Observation: "PyLearn learner reflection: " + strings.TrimSpace(record.Text)}, true
	case "attempt":
		switch record.Status {
		case "passed":
			return platform.Signal{EvidenceType: "exercise", Result: "pass", Strength: "moderate", Assessable: true,
				Observation: "PyLearn exercise variant " + record.Variant + " passed its configured checks."}, true
		case "failed":
			return platform.Signal{EvidenceType: "exercise", Result: "fail", Strength: "moderate", FailureClass: "implementation-miss", Assessable: true,
				Observation: failedAttempt("failed its checks", record)}, true
		default:
			return platform.Signal{EvidenceType: "exercise", Result: "fail", Strength: "weak", FailureClass: "implementation-miss", Assessable: true,
				Observation: failedAttempt("raised an error", record)}, true
		}
	}
	return platform.Signal{}, false
}

func failedAttempt(what string, record activityRecord) string {
	observation := "PyLearn exercise variant " + record.Variant + " " + what + "."
	if failure := failureSummary(record.Failure); failure != "" {
		observation += " First failure: " + failure
	}
	return observation
}

// maxFailureSummaryRunes bounds the failure text kept as evidence.
const maxFailureSummaryRunes = 200

var (
	terminalEscape = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	// localPath matches an absolute filesystem path (POSIX or Windows) that
	// starts a word; it keeps only the file name.
	localPath = regexp.MustCompile(`(^|[\s"'(=])(?:[A-Za-z]:[\\/]|/)(?:[^\s"'()/\\]+[\\/])+([^\s"'()/\\]*)`)
)

// failureSummary minimizes a failed attempt's failure output (PRIVACY.md):
// assessment needs what failed, not the raw runner output, which can carry
// local paths, environment values and unrelated logs. It keeps the first
// non-empty line, drops terminal escapes and control characters, reduces
// absolute paths to their file name, collapses whitespace and bounds the
// result to maxFailureSummaryRunes runes.
func failureSummary(failure string) string {
	var line string
	for _, candidate := range strings.Split(failure, "\n") {
		if strings.TrimSpace(candidate) != "" {
			line = candidate
			break
		}
	}
	line = terminalEscape.ReplaceAllString(line, "")
	line = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, line)
	line = localPath.ReplaceAllString(line, "${1}${2}")
	line = strings.Join(strings.Fields(line), " ")
	if runes := []rune(line); len(runes) > maxFailureSummaryRunes {
		line = strings.TrimSpace(string(runes[:maxFailureSummaryRunes-1])) + "…"
	}
	return line
}

func invalidActivity(target string, problems []string) error {
	return &platform.Error{
		Code:     platform.CodeInvalidActivityExport,
		Message:  fmt.Sprintf("invalid PyLearn activity export for target %q: %s", target, strings.Join(problems, "; ")),
		Adapter:  AdapterID,
		Target:   target,
		Problems: problems,
	}
}
