package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

const (
	goALPCurriculumFixture = "testdata/platform/pylearn/curriculum-export.go-alp.json"
	goALPMappingFixture    = "testdata/platform/pylearn/go-alp.mapping.yaml"
	activityExportFixture  = "testdata/platform/pylearn/activity-export.v2.json"
	revisedExportFixture   = "testdata/platform/pylearn/activity-export.v2.revised.json"
)

// newLearnerWorkspace creates an isolated Local Store workspace for learner.
func newLearnerWorkspace(t *testing.T, learner string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	manifest := "schemaVersion: 2\nworkspaceId: ws_platform_import\nlearnerId: " + learner + "\n"
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func linkAccount(t *testing.T, root, instance, user string) platformRun {
	t.Helper()
	return runPlatform(t, App{}, "account", "link", "--adapter", "pylearn", "--instance", instance, "--user", user, "--confirm", "--workspace", root)
}

func filesIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

func workspaceRevision(t *testing.T, root string) storepkg.Revision {
	t.Helper()
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	local, err := storepkg.OpenLocal(root, validator)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := local.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return revision
}

func requireValidWorkspace(t *testing.T, root string) {
	t.Helper()
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	if issues := validator.ValidateWorkspace(root); len(issues) != 0 {
		t.Fatalf("workspace invalid: %#v", issues)
	}
}

func TestPlatformAccountLinkRecordsLearnerConfirmedLink(t *testing.T) {
	root := newLearnerWorkspace(t, "ada")

	first := linkAccount(t, root, "pylearn-local", "usr_7f3a")
	if first.code != 0 {
		t.Fatalf("exit = %d, stderr = %s, stdout = %s", first.code, first.stderr, first.stdout)
	}
	body := decodeJSON(t, first.stdout)
	if body["status"] != "linked" {
		t.Fatalf("status = %#v", body["status"])
	}
	link := body["link"].(map[string]any)
	if link["platform"] != "pylearn" || link["instance"] != "pylearn-local" || link["platformUserId"] != "usr_7f3a" || link["learnerId"] != "ada" {
		t.Fatalf("link = %#v", link)
	}
	if confirmation := link["confirmation"].(map[string]any); confirmation["confirmedBy"] != "learner" || confirmation["confirmedAt"] == "" {
		t.Fatalf("confirmation = %#v", confirmation)
	}
	records := filesIn(t, filepath.Join(root, "platform-accounts"))
	if len(records) != 1 || records[0] != link["id"].(string)+".yaml" {
		t.Fatalf("platform account records = %v, link id = %v", records, link["id"])
	}
	requireValidWorkspace(t, root)

	again := linkAccount(t, root, "pylearn-local", "usr_7f3a")
	if again.code != 0 || decodeJSON(t, again.stdout)["status"] != "already-linked" {
		t.Fatalf("relink exit = %d, stdout = %s, stderr = %s", again.code, again.stdout, again.stderr)
	}
	if records := filesIn(t, filepath.Join(root, "platform-accounts")); len(records) != 1 {
		t.Fatalf("relinking must not append a record: %v", records)
	}
}

func TestPlatformAccountLinkRequiresLearnerConfirmation(t *testing.T) {
	root := newLearnerWorkspace(t, "ada")
	result := runPlatform(t, App{}, "account", "link", "--adapter", "pylearn", "--instance", "pylearn-local", "--user", "usr_7f3a", "--workspace", root)
	if result.code != 1 {
		t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
	}
	if got := errorField(t, result); got["code"] != "learner-confirmation-required" {
		t.Fatalf("error = %#v", got)
	}
	if records := filesIn(t, filepath.Join(root, "platform-accounts")); len(records) != 0 {
		t.Fatalf("unconfirmed link was recorded: %v", records)
	}
}

func importActivity(t *testing.T, root, export string, extra ...string) platformRun {
	t.Helper()
	args := []string{"import", "--adapter", "pylearn", "--target", "go-alp",
		"--curriculum", goALPCurriculumFixture, "--mapping", goALPMappingFixture,
		"--export", export, "--workspace", root}
	return runPlatform(t, App{}, append(args, extra...)...)
}

// linkedWorkspace is a workspace whose learner has linked the fixture
// export's PyLearn account.
func linkedWorkspace(t *testing.T) string {
	t.Helper()
	root := newLearnerWorkspace(t, "ada")
	if result := linkAccount(t, root, "pylearn-local", "usr_7f3a"); result.code != 0 {
		t.Fatalf("link exit = %d, stderr = %s", result.code, result.stderr)
	}
	return root
}

func counts(t *testing.T, result platformRun) map[string]any {
	t.Helper()
	if result.code != 0 {
		t.Fatalf("import exit = %d, stderr = %s, stdout = %s", result.code, result.stderr, result.stdout)
	}
	return decodeJSON(t, result.stdout)["counts"].(map[string]any)
}

func requireCounts(t *testing.T, result platformRun, want map[string]float64) {
	t.Helper()
	got := counts(t, result)
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("counts[%s] = %v, want %v (counts = %v)", key, got[key], value, got)
		}
	}
}

type evidenceFile struct {
	ID           string         `yaml:"id"`
	Competencies []string       `yaml:"competencies"`
	Type         string         `yaml:"type"`
	Result       string         `yaml:"result"`
	Strength     string         `yaml:"strength"`
	FailureClass string         `yaml:"failureClassification"`
	Supersedes   []string       `yaml:"supersedes"`
	Metadata     map[string]any `yaml:"metadata"`
}

func (e evidenceFile) item() string {
	activity, _ := e.Metadata["activity"].(map[string]any)
	item, _ := activity["item"].(string)
	return item
}

func readEvidence(t *testing.T, root string) []evidenceFile {
	t.Helper()
	var records []evidenceFile
	for _, name := range filesIn(t, filepath.Join(root, "evidence")) {
		data, err := os.ReadFile(filepath.Join(root, "evidence", name))
		if err != nil {
			t.Fatal(err)
		}
		var record evidenceFile
		if err := yaml.Unmarshal(data, &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

// evidenceFor returns the evidence for item covering competency.
func evidenceFor(t *testing.T, records []evidenceFile, item, competency string) []evidenceFile {
	t.Helper()
	var found []evidenceFile
	for _, record := range records {
		if record.item() != item {
			continue
		}
		for _, id := range record.Competencies {
			if id == competency {
				found = append(found, record)
			}
		}
	}
	return found
}

func onlyEvidence(t *testing.T, records []evidenceFile, item, competency string) evidenceFile {
	t.Helper()
	found := evidenceFor(t, records, item, competency)
	if len(found) != 1 {
		t.Fatalf("evidence for %s/%s = %d records, want 1", item, competency, len(found))
	}
	return found[0]
}

func TestPlatformImportRefusesUnlinkedPlatformUser(t *testing.T) {
	cases := map[string]func(t *testing.T) string{
		"no link": func(t *testing.T) string { return newLearnerWorkspace(t, "ada") },
		"other account linked": func(t *testing.T) string {
			root := newLearnerWorkspace(t, "ada")
			if result := linkAccount(t, root, "pylearn-local", "usr_other"); result.code != 0 {
				t.Fatalf("link exit = %d", result.code)
			}
			return root
		},
		"same user on another instance": func(t *testing.T) string {
			root := newLearnerWorkspace(t, "ada")
			if result := linkAccount(t, root, "pylearn-staging", "usr_7f3a"); result.code != 0 {
				t.Fatalf("link exit = %d", result.code)
			}
			return root
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			root := setup(t)
			result := importActivity(t, root, activityExportFixture)
			if result.code != 1 {
				t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
			}
			got := errorField(t, result)
			if got["code"] != "platform-account-not-linked" {
				t.Fatalf("error = %#v", got)
			}
			if records := filesIn(t, filepath.Join(root, "evidence")); len(records) != 0 {
				t.Fatalf("unlinked import wrote evidence: %v", records)
			}
		})
	}
}

func TestPlatformImportGradesActivityByMappingRole(t *testing.T) {
	root := linkedWorkspace(t)
	result := importActivity(t, root, activityExportFixture)
	requireCounts(t, result, map[string]float64{"records": 7, "imported": 5, "skipped": 0, "superseded": 0, "unmapped": 2})
	requireValidWorkspace(t, root)

	records := readEvidence(t, root)
	if len(records) != 6 {
		t.Fatalf("evidence records = %d, want 6", len(records))
	}

	// teaches: lesson progress is exposure, never assessment-grade.
	progress := onlyEvidence(t, records, "go-alp-a1-context", "go.runtime.context")
	if progress.Metadata["evidenceGrade"] != "exposure" || progress.Type != "platform-event" || progress.Result != "neutral" || progress.Strength != "weak" {
		t.Fatalf("teaches progress evidence = %+v", progress)
	}
	// assesses (through the question's quiz): a correct answer is assessment-grade.
	correct := onlyEvidence(t, records, "go-alp-a1-context#quiz:context-cancellation/q1", "go.runtime.context")
	if correct.Metadata["evidenceGrade"] != "assessment" || correct.Type != "quiz" || correct.Result != "pass" || correct.Strength != "moderate" {
		t.Fatalf("assesses correct answer evidence = %+v", correct)
	}
	if correct.Metadata["activity"].(map[string]any)["mappedItem"] != "go-alp-a1-context#quiz:context-cancellation" {
		t.Fatalf("mapped item = %#v", correct.Metadata["activity"])
	}
	wrong := onlyEvidence(t, records, "go-alp-a1-context#quiz:context-cancellation/q2", "go.runtime.context")
	if wrong.Metadata["evidenceGrade"] != "assessment" || wrong.Result != "fail" || wrong.FailureClass != "conceptual-miss" {
		t.Fatalf("assesses wrong answer evidence = %+v", wrong)
	}
	// One answer on an item that assesses one competency and reinforces another
	// yields assessment evidence (capped by the entry's weak ceiling) and
	// practice evidence.
	leaks := "go-alp-a2-goroutines#quiz:goroutine-leaks/q1"
	assessed := onlyEvidence(t, records, leaks, "go.concurrency.goroutines")
	if assessed.Metadata["evidenceGrade"] != "assessment" || assessed.Result != "pass" || assessed.Strength != "weak" {
		t.Fatalf("ceiling-capped assessment evidence = %+v", assessed)
	}
	practiced := onlyEvidence(t, records, leaks, "go.concurrency.races-deadlocks-leaks")
	if practiced.Metadata["evidenceGrade"] != "practice" || practiced.Type != "platform-event" || practiced.Result != "neutral" || practiced.Strength != "weak" {
		t.Fatalf("reinforces evidence = %+v", practiced)
	}
	if len(assessed.Competencies) != 1 || len(practiced.Competencies) != 1 {
		t.Fatalf("assessment and practice competencies must not mix: %v / %v", assessed.Competencies, practiced.Competencies)
	}
	reflection := onlyEvidence(t, records, "go-alp-a2-goroutines", "go.concurrency.goroutines")
	if reflection.Metadata["evidenceGrade"] != "exposure" || reflection.Result != "neutral" {
		t.Fatalf("reflection evidence = %+v", reflection)
	}

	body := decodeJSON(t, result.stdout)
	unmapped := map[string]string{}
	for _, raw := range body["records"].([]any) {
		record := raw.(map[string]any)
		if record["outcome"] == "unmapped" {
			unmapped[record["item"].(string)] = record["reason"].(string)
		}
	}
	want := map[string]string{"go-alp-a3-generics": "not-mapped", "go-alp-a0-retired#quiz:legacy/q1": "not-in-curriculum"}
	if len(unmapped) != len(want) || unmapped["go-alp-a3-generics"] != want["go-alp-a3-generics"] || unmapped["go-alp-a0-retired#quiz:legacy/q1"] != want["go-alp-a0-retired#quiz:legacy/q1"] {
		t.Fatalf("unmapped records = %v, want %v", unmapped, want)
	}
	if cursor := body["cursor"].(map[string]any); cursor["since"] != nil || cursor["next"] != "cursor-0001" {
		t.Fatalf("cursor = %#v", cursor)
	}
}

func TestPlatformImportOutputIsDeterministicJSON(t *testing.T) {
	first := importActivity(t, linkedWorkspace(t), activityExportFixture)
	second := importActivity(t, linkedWorkspace(t), activityExportFixture)
	if first.code != 0 || second.code != 0 {
		t.Fatalf("exit codes = %d, %d; stderr = %s", first.code, second.code, first.stderr)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "platform", "pylearn", "import-go-alp.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.stdout, want) {
		t.Fatalf("import output mismatch\n--- got ---\n%s\n--- want ---\n%s", first.stdout, want)
	}
	if !bytes.Equal(first.stdout, second.stdout) {
		t.Fatal("import output differs between identical imports into fresh workspaces")
	}
}

func TestPlatformDoubleImportYieldsNoNewEvidence(t *testing.T) {
	root := linkedWorkspace(t)
	requireCounts(t, importActivity(t, root, activityExportFixture), map[string]float64{"imported": 5})
	before := filesIn(t, filepath.Join(root, "evidence"))
	revision := workspaceRevision(t, root)

	again := importActivity(t, root, activityExportFixture)
	requireCounts(t, again, map[string]float64{"records": 7, "imported": 0, "skipped": 5, "superseded": 0, "unmapped": 2})
	if after := filesIn(t, filepath.Join(root, "evidence")); !reflect.DeepEqual(before, after) {
		t.Fatalf("double import changed evidence:\nbefore %v\nafter  %v", before, after)
	}
	if workspaceRevision(t, root) != revision {
		t.Fatal("an import that adds no evidence must not advance the workspace revision")
	}
	for _, raw := range decodeJSON(t, again.stdout)["records"].([]any) {
		record := raw.(map[string]any)
		if record["outcome"] == "skipped" && record["reason"] != "already-imported" {
			t.Fatalf("skip reason = %#v", record)
		}
	}
}

func TestPlatformImportSupersedesHigherRevisionOfSameEvent(t *testing.T) {
	root := linkedWorkspace(t)
	requireCounts(t, importActivity(t, root, activityExportFixture), map[string]float64{"imported": 5})
	question := "go-alp-a1-context#quiz:context-cancellation/q2"
	original := onlyEvidence(t, readEvidence(t, root), question, "go.runtime.context")

	revised := importActivity(t, root, revisedExportFixture, "--cursor", "cursor-0001")
	requireCounts(t, revised, map[string]float64{"records": 1, "imported": 0, "skipped": 0, "superseded": 1, "unmapped": 0})
	records := readEvidence(t, root)
	if len(records) != 7 {
		t.Fatalf("evidence records = %d, want 7", len(records))
	}
	var replacement *evidenceFile
	for _, record := range evidenceFor(t, records, question, "go.runtime.context") {
		if record.ID != original.ID {
			replacement = &record
		}
	}
	if replacement == nil || replacement.Result != "pass" || !reflect.DeepEqual(replacement.Supersedes, []string{original.ID}) {
		t.Fatalf("replacement = %+v, original = %s", replacement, original.ID)
	}
	if cursor := decodeJSON(t, revised.stdout)["cursor"].(map[string]any); cursor["since"] != "cursor-0001" || cursor["next"] != "cursor-0002" {
		t.Fatalf("cursor = %#v", cursor)
	}

	// The revision is now the imported one: importing it again is a no-op, and
	// the older full snapshot cannot reinstate the superseded answer.
	requireCounts(t, importActivity(t, root, revisedExportFixture, "--cursor", "cursor-0001"), map[string]float64{"skipped": 1, "superseded": 0})
	stale := importActivity(t, root, activityExportFixture)
	requireCounts(t, stale, map[string]float64{"imported": 0, "skipped": 5, "superseded": 0})
	for _, raw := range decodeJSON(t, stale.stdout)["records"].([]any) {
		record := raw.(map[string]any)
		if record["item"] == question && record["reason"] != "stale-revision" {
			t.Fatalf("stale snapshot record = %#v", record)
		}
	}
	if len(readEvidence(t, root)) != 7 {
		t.Fatal("stale snapshot wrote evidence")
	}
}

func TestPlatformImportRequiresMatchingCursor(t *testing.T) {
	root := linkedWorkspace(t)
	cases := map[string][]string{
		"incremental export without cursor":    nil,
		"incremental export with other cursor": {"--cursor", "cursor-0007"},
	}
	for name, extra := range cases {
		t.Run(name, func(t *testing.T) {
			result := importActivity(t, root, revisedExportFixture, extra...)
			if result.code != 1 {
				t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
			}
			if got := errorField(t, result); got["code"] != "cursor-mismatch" {
				t.Fatalf("error = %#v", got)
			}
		})
	}
	// A full snapshot covers every cursor.
	requireCounts(t, importActivity(t, root, activityExportFixture, "--cursor", "cursor-0001"), map[string]float64{"imported": 5})
	if len(readEvidence(t, root)) != 6 {
		t.Fatal("cursor-mismatched imports wrote evidence")
	}
}

func TestPlatformImportRejectsUnsupportedExports(t *testing.T) {
	root := linkedWorkspace(t)
	cases := map[string]string{
		"v1 export": `{"schemaVersion": 1, "exportedAt": "2026-10-06T00:00:00Z", "courses": [], "progress": []}`,
		"missing event identity": `{"schemaVersion": 2, "platform": "pylearn", "instance": "pylearn-local", "exportedAt": "2026-10-07T09:00:00Z",
  "user": {"id": "usr_7f3a"}, "cursor": {"since": null, "next": "c"},
  "targets": [{"id": "go-alp", "records": [{"kind": "progress", "item": "go-alp-a1-context", "observedAt": "2026-10-07T08:10:00Z", "status": "completed"}]}]}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "export.json")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			result := importActivity(t, root, path)
			if result.code != 1 {
				t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
			}
			if got := errorField(t, result); got["code"] != "invalid-activity-export" {
				t.Fatalf("error = %#v", got)
			}
		})
	}
}

func TestPlatformImportRefusesInvalidMapping(t *testing.T) {
	root := linkedWorkspace(t)
	mapping := filepath.Join(t.TempDir(), "go-alp.mapping.yaml")
	data, err := os.ReadFile(goALPMappingFixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mapping, bytes.Replace(data, []byte("go.runtime.context"), []byte("go.not.real"), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	result := runPlatform(t, App{}, "import", "--adapter", "pylearn", "--target", "go-alp",
		"--curriculum", goALPCurriculumFixture, "--mapping", mapping, "--export", activityExportFixture, "--workspace", root)
	if result.code != 1 {
		t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
	}
	if got := errorField(t, result); got["code"] != "invalid-mapping" {
		t.Fatalf("error = %#v", got)
	}
	if len(readEvidence(t, root)) != 0 {
		t.Fatal("import with an invalid mapping wrote evidence")
	}
}

func TestPlatformImportRecordsAnEventReturningToAnEarlierRevision(t *testing.T) {
	root := linkedWorkspace(t)
	requireCounts(t, importActivity(t, root, activityExportFixture), map[string]float64{"imported": 5})
	requireCounts(t, importActivity(t, root, revisedExportFixture, "--cursor", "cursor-0001"), map[string]float64{"superseded": 1})
	question := "go-alp-a1-context#quiz:context-cancellation/q2"
	var correct evidenceFile
	for _, record := range evidenceFor(t, readEvidence(t, root), question, "go.runtime.context") {
		if record.Result == "pass" {
			correct = record
		}
	}

	// The learner picks the original wrong option again: same content, so the
	// same synthetic revision as the first import, but later than the answer
	// it replaces.
	reverted := filepath.Join(t.TempDir(), "export.json")
	if err := os.WriteFile(reverted, []byte(`{
  "schemaVersion": 2, "platform": "pylearn", "instance": "pylearn-local", "exportedAt": "2026-10-07T11:00:00Z",
  "user": {"id": "usr_7f3a"}, "cursor": {"since": "cursor-0002", "next": "cursor-0003"},
  "targets": [{"id": "go-alp", "records": [{
    "kind": "quiz-answer", "item": "`+question+`",
    "event": {"id": "quiz_answers:`+question+`", "revision": "sha256:417bb098606dbf78272226208d4749082d2344f0d27fe6790c41a018668df183", "synthetic": true},
    "observedAt": "2026-10-07T10:30:00Z", "picked": 0, "correct": false}]}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result := importActivity(t, root, reverted, "--cursor", "cursor-0002")
	requireCounts(t, result, map[string]float64{"records": 1, "superseded": 1, "skipped": 0})
	records := evidenceFor(t, readEvidence(t, root), question, "go.runtime.context")
	if len(records) != 3 {
		t.Fatalf("evidence for %s = %d records, want 3", question, len(records))
	}
	latest := decodeJSON(t, result.stdout)["records"].([]any)[0].(map[string]any)
	if supersedes := latest["supersedes"].([]any); len(supersedes) != 1 || supersedes[0] != correct.ID {
		t.Fatalf("reverted answer supersedes %v, want [%s]", supersedes, correct.ID)
	}
	requireCounts(t, importActivity(t, root, reverted, "--cursor", "cursor-0002"), map[string]float64{"skipped": 1, "superseded": 0})
}
