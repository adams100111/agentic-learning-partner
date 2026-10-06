package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"context"
	contextbundle "github.com/adams100111/agentic-learning-partner/internal/context"
	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/gitexec"
	"github.com/adams100111/agentic-learning-partner/internal/learning"
	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/platform/pylearn"
	"github.com/adams100111/agentic-learning-partner/internal/state"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestV0ClosedLoopFromPlatformEvidenceToPlan(t *testing.T) {
	root, revision := makeWorkspace(t)
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	if issues := validator.ValidateWorkspace(root); len(issues) != 0 {
		t.Fatalf("initial workspace invalid: %#v", issues)
	}

	adapter := pylearn.NewAdapter()
	curriculum := platform.Curriculum{
		Target: platform.ExternalID{Platform: "pylearn", Target: "go"},
		Items: []platform.Item{
			{Ref: platform.ExternalID{Platform: "pylearn", Target: "go", Item: "go-context"}, Kind: "lesson", Phase: "B"},
			{Ref: platform.ExternalID{Platform: "pylearn", Target: "go", Item: "go-context#quiz:cancellation"}, Kind: "quiz",
				Parent: &platform.ExternalID{Platform: "pylearn", Target: "go", Item: "go-context"}},
		},
	}
	mapping, err := adapter.ValidateContentMapping([]byte(`schemaVersion: 2
platform: pylearn
target: go
packs: [{domain: go, packVersion: ">=0.1.0 <0.2.0"}]
entries:
  - item: go-context#quiz:cancellation
    competencies: [{id: go.runtime.context, role: assesses}]
`), "go.mapping.yaml", curriculum)
	if err != nil || !mapping.Valid {
		t.Fatalf("mapping = %+v, err = %v", mapping, err)
	}
	batch, err := adapter.ReadActivity([]byte(`{
  "schemaVersion": 2, "platform": "pylearn", "instance": "pylearn-local", "exportedAt": "2026-10-06T00:00:00Z",
  "user": {"id": "usr_test"}, "cursor": {"since": null, "next": "cursor-1"},
  "targets": [{"id": "go", "records": [{
    "kind": "quiz-answer", "item": "go-context#quiz:cancellation",
    "event": {"id": "quiz_answers:go-context#quiz:cancellation", "revision": "sha256:90932ecd56d1a359359d74376f272fe8ce0433581f894c2d64acdfa8e57e09ba", "synthetic": true},
    "observedAt": "2026-10-06T00:00:00Z", "picked": 2, "correct": true}]}]
}`), "go", nil)
	if err != nil {
		t.Fatal(err)
	}
	imported, err := platform.PlanImport(platform.ImportRequest{
		Batch: batch, Curriculum: curriculum, Mapping: mapping,
		Account: platform.Account{Platform: "pylearn", Instance: batch.Instance, PlatformUserID: batch.PlatformUserID, LearnerID: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.Evidence) != 1 || imported.Report.Counts.Imported != 1 {
		t.Fatalf("imported = %+v", imported.Report)
	}

	store := state.Store{
		Root:      root,
		Catalog:   domain.NewRegistry(),
		Validator: validator,
	}
	evidence := imported.Evidence[0]
	evidence, err = store.AppendEvidence(revision, evidence)
	if err != nil {
		t.Fatal(err)
	}

	assessment, err := store.AppendAssessment(revision, state.Assessment{
		SchemaVersion: 1,
		ID:            "asmt_context_e2e",
		RecordedAt:    "2026-10-06T00:01:00Z",
		Domain:        "go",
		Competency:    "go.runtime.context",
		Evidence:      []string{evidence.ID},
		Rubric:        state.RubricRef{ID: "go.runtime.context", Version: "1", DomainPackVersion: "0.1.0"},
		Assessor:      state.Assessor{Type: "agent", ID: "e2e"},
		Judgment:      state.Judgment{Level: "functional"},
		Confidence:    "high",
		Rationale:     "Answered an assessing question correctly; enough for functional, not production-ready.",
		Status:        "proposed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Status != "accepted" {
		t.Fatalf("assessment status = %q", assessment.Status)
	}

	projection, err := store.RebuildProjection()
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Competencies) != 1 || projection.Competencies[0].Level != "functional" {
		t.Fatalf("projection = %#v", projection)
	}

	projectionPath := filepath.Join(root, "state", "competencies.yaml")
	firstProjection, err := os.ReadFile(projectionPath)
	if err != nil {
		t.Fatal(err)
	}

	builder := contextbundle.Builder{Root: root, Validator: validator}
	request := contextbundle.Request{Task: "teach", Domain: "go", Competency: "go.runtime.context"}
	firstContext, err := builder.Build(request)
	if err != nil {
		t.Fatal(err)
	}
	secondContext, err := builder.Build(request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstContext, secondContext) {
		t.Fatal("same workspace/task must produce the same effective context across harness invocations")
	}
	if _, ok := firstContext.Competencies["items"]; !ok {
		t.Fatalf("context omitted current competency projection: %#v", firstContext.Competencies)
	}

	queue, plan, err := (learning.GoPlanner{
		Root: root, Validator: validator, Registry: domain.NewRegistry(),
	}).Build(time.Date(2026, 10, 6, 0, 5, 0, 0, time.UTC), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.Items) == 0 || len(plan.Items) == 0 {
		t.Fatalf("expected evidence-aware review/plan; queue=%#v plan=%#v", queue, plan)
	}
	foundContextPlan := false
	for _, item := range plan.Items {
		if item.Competency == "go.runtime.context" {
			foundContextPlan = true
			if item.Mode != "challenge-only" {
				t.Fatalf("functional context competency mode = %q, want challenge-only", item.Mode)
			}
		}
	}
	if !foundContextPlan {
		t.Fatal("plan should include go.runtime.context")
	}

	if err := os.Remove(projectionPath); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RebuildProjection(); err != nil {
		t.Fatal(err)
	}
	secondProjection, err := os.ReadFile(projectionPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstProjection, secondProjection) {
		t.Fatalf("projection cache did not reproduce from canonical records\nfirst:\n%s\nsecond:\n%s", firstProjection, secondProjection)
	}

	if err := domain.NewRegistry().CheckCompatibility("go", ">=2.0 <3.0"); err == nil {
		t.Fatal("incompatible platform/domain mapping range must fail explicitly")
	}

	if issues := validator.ValidateWorkspace(root); len(issues) != 0 {
		t.Fatalf("final workspace invalid: %#v", issues)
	}
}

func TestPortableAndClaudePluginManifestsShareOneSkillTree(t *testing.T) {
	root := repositoryRoot(t)
	var portable map[string]any
	readJSON(t, filepath.Join(root, "plugin.json"), &portable)
	if portable["name"] != "agentic-learning-partner" {
		t.Fatalf("portable plugin name = %#v", portable["name"])
	}

	var claude map[string]any
	readJSON(t, filepath.Join(root, ".claude-plugin", "plugin.json"), &claude)
	if claude["name"] != portable["name"] {
		t.Fatalf("Claude plugin name %v != portable name %v", claude["name"], portable["name"])
	}

	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	validSkills := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "skills", entry.Name(), "SKILL.md")); err == nil {
			validSkills++
		}
	}
	if validSkills < 7 {
		t.Fatalf("shared skill tree has %d valid skills", validSkills)
	}
}

func makeWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "alp@example.invalid")
	runGit(t, root, "config", "user.name", "ALP E2E")

	write(t, root, "workspace.yaml", "schemaVersion: 1\nlearnerId: test\n")
	write(t, root, "profile/profile.yaml", `schemaVersion: 1
learner:
  id: test
  professionalLevel: senior
experience:
  php:
    level: expert
    relativeRank: 1
  typescript:
    level: strong
    relativeRank: 2
goals:
  - id: production-go
    statement: Reach production-ready Go proficiency.
    priority: critical
preferences:
  teachingPace: senior-dense
  preferRealProjects: true
`)
	write(t, root, "personas/global.yaml", `schemaVersion: 1
scope: global
domain: null
teaching:
  pace: senior-dense
  preferredActivityTypes: [code-reading, debugging, implementation]
  avoid: [toy-examples]
`)
	write(t, root, "personas/domains/go.yaml", `schemaVersion: 1
scope: domain
domain: go
teaching:
  pace: senior-dense
  emphasis: [idiomatic-Go, concurrency, production-runtime]
`)

	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initialize learner workspace")
	info, err := workspace.Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, info.Revision
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := gitexec.Command(context.Background(), "", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func TestNoGitHubActionsWorkflowIsPresent(t *testing.T) {
	root := repositoryRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, ".github", "workflows"))
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	var workflows []string
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".yaml")) {
			workflows = append(workflows, entry.Name())
		}
	}
	if len(workflows) != 0 {
		t.Fatalf("GitHub Actions is intentionally disabled while quota is exhausted; found %v", workflows)
	}
}
