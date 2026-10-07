package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
)

const pylearnCurriculumFixture = "testdata/platform/pylearn/curriculum-export.v1.json"

type platformRun struct {
	code   int
	stdout []byte
	stderr string
}

func runPlatform(t *testing.T, app App, args ...string) platformRun {
	t.Helper()
	var out, errOut bytes.Buffer
	app.Out = &out
	app.ErrOut = &errOut
	app.Getwd = func() (string, error) { return t.TempDir(), nil }
	code := app.Run(append([]string{"platform"}, args...))
	return platformRun{code: code, stdout: out.Bytes(), stderr: errOut.String()}
}

func TestPlatformInspectDescribesPyLearnTargetFromCurriculumExport(t *testing.T) {
	result := runPlatform(t, App{}, "inspect", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture)
	if result.code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", result.code, result.stderr, result.stdout)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "platform", "pylearn", "inspect-go.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.stdout, want) {
		t.Fatalf("inspect output mismatch\n--- got ---\n%s\n--- want ---\n%s", result.stdout, want)
	}

	again := runPlatform(t, App{}, "inspect", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture)
	if !bytes.Equal(again.stdout, result.stdout) {
		t.Fatalf("inspect output is not deterministic across runs")
	}
}

func decodeJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("stdout is not a JSON object: %v\n%s", err, data)
	}
	return value
}

func errorField(t *testing.T, result platformRun) map[string]any {
	t.Helper()
	body := decodeJSON(t, result.stdout)
	value, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("stdout has no error object: %s", result.stdout)
	}
	return value
}

func TestPlatformInspectRejectsUnknownAdapter(t *testing.T) {
	result := runPlatform(t, App{}, "inspect", "--adapter", "moodle", "--target", "go", "--curriculum", pylearnCurriculumFixture)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	got := errorField(t, result)
	if got["code"] != "unknown-adapter" || got["adapter"] != "moodle" {
		t.Fatalf("error = %#v", got)
	}
	if available, _ := got["available"].([]any); len(available) != 1 || available[0] != "pylearn" {
		t.Fatalf("available adapters = %#v", got["available"])
	}
	if !strings.Contains(result.stderr, `unknown platform adapter "moodle"`) {
		t.Fatalf("stderr = %q", result.stderr)
	}
}

func TestPlatformInspectRejectsUnknownTarget(t *testing.T) {
	result := runPlatform(t, App{}, "inspect", "--adapter", "pylearn", "--target", "go-alp", "--curriculum", pylearnCurriculumFixture)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	got := errorField(t, result)
	if got["code"] != "unknown-target" || got["target"] != "go-alp" {
		t.Fatalf("error = %#v", got)
	}
	available, _ := got["available"].([]any)
	if len(available) != 2 || available[0] != "go" || available[1] != "pylearn" {
		t.Fatalf("available targets = %#v", got["available"])
	}
}

// activityOnlyAdapter is a read-side platform that declares only Activity Source.
type activityOnlyAdapter struct{}

func (activityOnlyAdapter) ID() string { return "activity-only" }
func (activityOnlyAdapter) Capabilities() []platform.Capability {
	return []platform.Capability{platform.ActivitySource}
}
func (activityOnlyAdapter) StableIdentifierKinds() []string { return []string{"activity"} }

func TestPlatformCommandRefusesUndeclaredCapability(t *testing.T) {
	registry := platform.NewRegistry(activityOnlyAdapter{})
	result := runPlatform(t, App{Platforms: &registry}, "inspect", "--adapter", "activity-only", "--target", "go", "--curriculum", pylearnCurriculumFixture)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	got := errorField(t, result)
	if got["code"] != "capability-not-declared" || got["capability"] != "curriculum-reader" || got["adapter"] != "activity-only" {
		t.Fatalf("error = %#v", got)
	}
	if declared, _ := got["available"].([]any); len(declared) != 1 || declared[0] != "activity-source" {
		t.Fatalf("declared capabilities = %#v", got["available"])
	}
	if !strings.Contains(result.stderr, "does not declare the curriculum-reader capability") {
		t.Fatalf("stderr = %q", result.stderr)
	}
}

func TestPlatformCommandUsageErrorsExitTwo(t *testing.T) {
	cases := map[string][]string{
		"no subcommand":      {},
		"unknown subcommand": {"teleport", "--adapter", "pylearn", "--target", "go"},
		"missing adapter":    {"inspect", "--target", "go", "--curriculum", pylearnCurriculumFixture},
		"missing target":     {"inspect", "--adapter", "pylearn", "--curriculum", pylearnCurriculumFixture},
		"missing curriculum": {"inspect", "--adapter", "pylearn", "--target", "go"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			result := runPlatform(t, App{}, args...)
			if result.code != 2 {
				t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
			}
			if got := errorField(t, result); got["code"] != "usage" {
				t.Fatalf("error = %#v", got)
			}
		})
	}
}

func TestPlatformCommandsRejectFlagsTheyDoNotTake(t *testing.T) {
	cases := map[string]struct {
		args []string
		flag string
	}{
		"inspect --confirm":          {[]string{"inspect", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture, "--confirm"}, "confirm"},
		"inspect --workspace":        {[]string{"inspect", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture, "--workspace", "ws"}, "workspace"},
		"mapping validate --export":  {[]string{"mapping", "validate", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture, "--mapping", "m.yaml", "--export", "e.json"}, "export"},
		"account link --curriculum":  {[]string{"account", "link", "--adapter", "pylearn", "--instance", "i", "--user", "u", "--confirm", "--curriculum", pylearnCurriculumFixture}, "curriculum"},
		"import --confirm":           {[]string{"import", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture, "--mapping", "m.yaml", "--export", "e.json", "--confirm"}, "confirm"},
		"plan --mode":                {[]string{"plan", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture, "--mapping", "m.yaml", "--mode", "skip"}, "mode"},
		"decision accept --decision": {[]string{"decision", "accept", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture, "--mapping", "m.yaml", "--decision", "d", "--unit", "u", "--mode", "skip", "--basis", "b", "--confirm"}, "decision"},
		"decision revoke --mode":     {[]string{"decision", "revoke", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture, "--mapping", "m.yaml", "--decision", "d", "--mode", "skip", "--basis", "b", "--confirm"}, "mode"},
		"gates record --confirm":     {[]string{"gates", "record", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture, "--plan", "p", "--result", "r.json", "--confirm"}, "confirm"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := runPlatform(t, App{}, tc.args...)
			if result.code != 2 {
				t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
			}
			got := errorField(t, result)
			if got["code"] != "usage" || !strings.Contains(got["message"].(string), "--"+tc.flag) {
				t.Fatalf("error = %#v", got)
			}
		})
	}
}

func TestTopLevelUsageListsPlatformCommands(t *testing.T) {
	var out, errOut bytes.Buffer
	App{Out: &out, ErrOut: &errOut}.Run(nil)
	for _, command := range []string{"platform inspect", "platform mapping validate", "platform account link", "platform import", "platform plan", "platform decision accept", "platform decision revoke", "platform gates record"} {
		if !strings.Contains(errOut.String(), command) {
			t.Errorf("top-level usage does not list %q:\n%s", command, errOut.String())
		}
	}
}

func writeExport(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "curriculum-export.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const opaqueHash = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestPlatformInspectTreatsExternalIdentitiesAsOpaque(t *testing.T) {
	// IDs that look like competency IDs, phase codes, or paths must pass
	// through verbatim; placement comes only from the export's fields.
	export := writeExport(t, `{
  "schemaVersion": 1,
  "platform": "pylearn",
  "targets": [{
    "id": "go.runtime.context",
    "contentHash": "`+opaqueHash+`",
    "phases": [{"id": "A"}],
    "items": [
      {"id": "go-b5-context", "kind": "lesson", "phase": "A"},
      {"id": "B/7::scene-3", "kind": "scene", "parent": "go-b5-context"}
    ]
  }]
}`)
	result := runPlatform(t, App{}, "inspect", "--adapter", "pylearn", "--target", "go.runtime.context", "--curriculum", export)
	if result.code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", result.code, result.stderr)
	}
	body := decodeJSON(t, result.stdout)
	if target := body["target"].(map[string]any); target["platform"] != "pylearn" || target["target"] != "go.runtime.context" || len(target) != 2 {
		t.Fatalf("target = %#v", target)
	}
	items := body["items"].([]any)
	lesson := items[0].(map[string]any)
	if ref := lesson["ref"].(map[string]any); ref["item"] != "go-b5-context" || ref["target"] != "go.runtime.context" || lesson["phase"] != "A" {
		t.Fatalf("lesson = %#v", lesson)
	}
	scene := items[1].(map[string]any)
	if ref := scene["ref"].(map[string]any); ref["item"] != "B/7::scene-3" {
		t.Fatalf("scene ref = %#v", ref)
	}
	if _, hasPhase := scene["phase"]; hasPhase {
		t.Fatalf("phase must not be derived from an item ID: %#v", scene)
	}
	if parent := scene["parent"].(map[string]any); parent["platform"] != "pylearn" || parent["target"] != "go.runtime.context" || parent["item"] != "go-b5-context" {
		t.Fatalf("parent = %#v", parent)
	}
	if body["mapping"] != nil {
		t.Fatalf("unmapped target must report mapping null, got %#v", body["mapping"])
	}
}

func TestPlatformInspectRejectsInvalidCurriculumExports(t *testing.T) {
	cases := map[string]struct {
		export  string
		problem string
	}{
		"unsupported schema version": {
			export:  `{"schemaVersion": 2, "platform": "pylearn", "targets": []}`,
			problem: "schemaVersion",
		},
		"other platform's export": {
			export:  `{"schemaVersion": 1, "platform": "moodle", "targets": [{"id": "go", "contentHash": "` + opaqueHash + `", "phases": [], "items": []}]}`,
			problem: `export is for platform "moodle"`,
		},
		"undeclared-stable item kind": {
			export:  `{"schemaVersion": 1, "platform": "pylearn", "targets": [{"id": "go", "contentHash": "` + opaqueHash + `", "phases": [{"id": "A"}], "items": [{"id": "go-a1", "kind": "lesson", "phase": "A"}, {"id": "no-project-file", "kind": "section", "parent": "go-a1"}]}]}`,
			problem: `item "no-project-file" has kind "section", which adapter "pylearn" does not declare stable`,
		},
		"duplicate item": {
			export:  `{"schemaVersion": 1, "platform": "pylearn", "targets": [{"id": "go", "contentHash": "` + opaqueHash + `", "phases": [{"id": "A"}], "items": [{"id": "go-a1", "kind": "lesson", "phase": "A"}, {"id": "go-a1", "kind": "lesson", "phase": "A"}]}]}`,
			problem: `item "go-a1" is declared more than once`,
		},
		"undeclared phase": {
			export:  `{"schemaVersion": 1, "platform": "pylearn", "targets": [{"id": "go", "contentHash": "` + opaqueHash + `", "phases": [{"id": "A"}], "items": [{"id": "go-a1", "kind": "lesson", "phase": "Z"}]}]}`,
			problem: `item "go-a1" references undeclared phase "Z"`,
		},
		"unknown parent": {
			export:  `{"schemaVersion": 1, "platform": "pylearn", "targets": [{"id": "go", "contentHash": "` + opaqueHash + `", "phases": [{"id": "A"}], "items": [{"id": "quiz-1", "kind": "quiz", "parent": "go-a1"}]}]}`,
			problem: `item "quiz-1" references parent "go-a1", which is not declared before it`,
		},
		"unplaced item": {
			export:  `{"schemaVersion": 1, "platform": "pylearn", "targets": [{"id": "go", "contentHash": "` + opaqueHash + `", "phases": [], "items": [{"id": "go-a1", "kind": "lesson"}]}]}`,
			problem: `item "go-a1" has neither a phase nor a parent`,
		},
		"malformed content hash": {
			export:  `{"schemaVersion": 1, "platform": "pylearn", "targets": [{"id": "go", "contentHash": "md5:abc", "phases": [], "items": []}]}`,
			problem: "contentHash",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := runPlatform(t, App{}, "inspect", "--adapter", "pylearn", "--target", "go", "--curriculum", writeExport(t, tc.export))
			if result.code != 1 {
				t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
			}
			got := errorField(t, result)
			if got["code"] != "invalid-curriculum-export" {
				t.Fatalf("error = %#v", got)
			}
			problems, _ := got["problems"].([]any)
			found := false
			for _, problem := range problems {
				if text, _ := problem.(string); strings.Contains(text, tc.problem) {
					found = true
				}
			}
			if !found {
				t.Fatalf("problems %#v do not mention %q", problems, tc.problem)
			}
		})
	}
}
