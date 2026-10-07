package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/platform/pylearn"
)

// seedAssessment records one accepted assessment (and the evidence it cites)
// through the ordinary CLI, so plan tests start from real learner state.
func seedAssessment(t *testing.T, root, competency, level, confidence, recordedAt string) {
	t.Helper()
	dir := t.TempDir()
	slug := fmt.Sprintf("%x", []byte(competency+recordedAt))
	evidence := filepath.Join(dir, "evidence.yaml")
	if err := os.WriteFile(evidence, []byte(`schemaVersion: 1
id: ev_seed_`+slug+`
recordedAt: `+recordedAt+`
domain: go
competencies:
  - `+competency+`
type: explanation
source:
  kind: diagnostic
  ref: plan-fixture
observation: Seeded for target adaptation planning.
result: pass
strength: moderate
`), 0o644); err != nil {
		t.Fatal(err)
	}
	assessment := filepath.Join(dir, "assessment.yaml")
	if err := os.WriteFile(assessment, []byte(`schemaVersion: 1
id: asmt_seed_`+slug+`
recordedAt: `+recordedAt+`
domain: go
competency: `+competency+`
evidence: [ev_seed_`+slug+`]
rubric: {id: go-competency, version: "1"}
assessor: {type: human, id: plan-fixture}
judgment: {level: `+level+`}
confidence: `+confidence+`
rationale: Seeded for target adaptation planning.
status: accepted
`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"evidence", "add", "--workspace", root, "--file", evidence},
		{"assessment", "add", "--workspace", root, "--file", assessment},
	} {
		var out, errOut bytes.Buffer
		app := App{Out: &out, ErrOut: &errOut, Getwd: func() (string, error) { return t.TempDir(), nil }}
		if code := app.Run(args); code != 0 {
			t.Fatalf("%v exit = %d, stderr = %s", args, code, errOut.String())
		}
	}
}

func planTarget(t *testing.T, root string, extra ...string) platformRun {
	t.Helper()
	args := []string{"plan", "--adapter", "pylearn", "--target", "go-alp",
		"--curriculum", goALPCurriculumFixture, "--mapping", goALPMappingFixture, "--workspace", root}
	return runPlatform(t, App{}, append(args, extra...)...)
}

func requirePlan(t *testing.T, result platformRun) map[string]any {
	t.Helper()
	if result.code != 0 {
		t.Fatalf("plan exit = %d, stderr = %s, stdout = %s", result.code, result.stderr, result.stdout)
	}
	return decodeJSON(t, result.stdout)
}

func projectionOf(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	projection, ok := body["projection"].(map[string]any)
	if !ok {
		t.Fatalf("plan output has no projection: %#v", body)
	}
	return projection
}

// unitModes returns item -> mode for every projected unit, in order.
func unitModes(t *testing.T, projection map[string]any) [][2]string {
	t.Helper()
	var modes [][2]string
	for _, raw := range projection["units"].([]any) {
		unit := raw.(map[string]any)
		item := unit["item"].(map[string]any)["item"].(string)
		modes = append(modes, [2]string{item, unit["mode"].(string)})
	}
	return modes
}

func stringList(values []any) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.(string)
	}
	return result
}

// adaLearnerState: context demonstrated, goroutines functional, nothing else.
func adaLearnerState(t *testing.T) string {
	t.Helper()
	root := newLearnerWorkspace(t, "ada")
	seedAssessment(t, root, "go.runtime.context", "strong", "high", "2026-10-01T09:00:00Z")
	seedAssessment(t, root, "go.concurrency.goroutines", "functional", "medium", "2026-10-01T09:05:00Z")
	return root
}

func TestPlatformPlanProjectsLearnerStateOntoSharedTarget(t *testing.T) {
	root := adaLearnerState(t)
	evidenceBefore := filesIn(t, filepath.Join(root, "evidence"))
	assessmentsBefore := filesIn(t, filepath.Join(root, "assessments"))

	body := requirePlan(t, planTarget(t, root))
	projection := projectionOf(t, body)

	if body["adapter"] != "pylearn" {
		t.Fatalf("adapter = %#v", body["adapter"])
	}
	if target := projection["target"].(map[string]any); target["platform"] != "pylearn" || target["target"] != "go-alp" {
		t.Fatalf("target = %#v", target)
	}
	if projection["learnerId"] != "ada" {
		t.Fatalf("learnerId = %#v", projection["learnerId"])
	}
	want := [][2]string{
		{"go-alp-a1-context", "skip"},
		{"go-alp-a2-goroutines", "skim"},
		{"go-alp-a3-generics", "full"},
	}
	if got := unitModes(t, projection); !reflect.DeepEqual(got, want) {
		t.Fatalf("unit modes = %v, want %v", got, want)
	}
	if got := stringList(projection["sequence"].([]any)); !reflect.DeepEqual(got, []string{"go-alp-a2-goroutines", "go-alp-a3-generics"}) {
		t.Fatalf("sequence = %v", got)
	}
	var gaps []string
	for _, raw := range projection["gaps"].([]any) {
		gaps = append(gaps, raw.(map[string]any)["competency"].(string))
	}
	if want := []string{"go.concurrency.channels", "go.concurrency.sync", "go.runtime.scheduler"}; !reflect.DeepEqual(gaps, want) {
		t.Fatalf("gaps = %v, want %v", gaps, want)
	}

	// The projection is a generated workspace file, not canonical state.
	path, _ := body["path"].(string)
	if path == "" {
		t.Fatalf("plan output has no projection path: %s", result(body))
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
		t.Fatalf("projection file not written: %v", err)
	}
	if !reflect.DeepEqual(filesIn(t, filepath.Join(root, "evidence")), evidenceBefore) ||
		!reflect.DeepEqual(filesIn(t, filepath.Join(root, "assessments")), assessmentsBefore) {
		t.Fatal("plan must not write evidence or assessments")
	}
	requireValidWorkspace(t, root)
}

func result(body map[string]any) string { return fmt.Sprintf("%#v", body) }

func TestPlatformPlanRebuildsIdenticallyFromSameCanonicalInputs(t *testing.T) {
	root := adaLearnerState(t)
	first := planTarget(t, root)
	body := requirePlan(t, first)
	path := filepath.Join(root, filepath.FromSlash(body["path"].(string)))
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// The projection is disposable: deleting it and re-planning reproduces it.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	second := planTarget(t, root)
	if second.code != 0 || !bytes.Equal(first.stdout, second.stdout) {
		t.Fatalf("rebuild differs:\nfirst  %s\nsecond %s\nstderr %s", first.stdout, second.stdout, second.stderr)
	}
	rebuilt, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, rebuilt) {
		t.Fatalf("rebuilt projection file differs:\n%s\n%s", written, rebuilt)
	}
	revision, _ := projectionOf(t, body)["revision"].(string)
	if len(revision) != len("tapr_")+24 || revision[:5] != "tapr_" {
		t.Fatalf("revision = %q", revision)
	}

	// New learner state changes the projection and its revision.
	seedAssessment(t, root, "go.concurrency.goroutines", "strong", "low", "2026-10-02T09:00:00Z")
	changed := projectionOf(t, requirePlan(t, planTarget(t, root)))
	if changed["revision"] == revision {
		t.Fatal("projection revision must change when learner state changes")
	}
	if got := unitModes(t, changed)[1]; got != [2]string{"go-alp-a2-goroutines", "challenge"} {
		t.Fatalf("unconfirmed strong goroutines unit = %v, want challenge", got)
	}
	var reinforcement []string
	for _, raw := range changed["reinforcement"].([]any) {
		need := raw.(map[string]any)
		reinforcement = append(reinforcement, need["competency"].(string)+":"+need["reason"].(string)+":"+fmt.Sprint(need["items"]))
	}
	if want := []string{"go.concurrency.goroutines:unconfirmed:[go-alp-a2-goroutines#quiz:goroutine-leaks]"}; !reflect.DeepEqual(reinforcement, want) {
		t.Fatalf("reinforcement = %v, want %v", reinforcement, want)
	}
}

func TestPlatformPlanIsPerLearnerOverSharedTargetWithoutSecondCompetencyState(t *testing.T) {
	ada := adaLearnerState(t)
	grace := newLearnerWorkspace(t, "grace")

	adaBody := requirePlan(t, planTarget(t, ada))
	graceBody := requirePlan(t, planTarget(t, grace))
	adaProjection, graceProjection := projectionOf(t, adaBody), projectionOf(t, graceBody)

	if adaProjection["id"] != graceProjection["id"] || adaBody["path"] != graceBody["path"] {
		t.Fatalf("one shared target has one projection identity per workspace: %v vs %v", adaProjection["id"], graceProjection["id"])
	}
	if graceProjection["learnerId"] != "grace" || adaProjection["revision"] == graceProjection["revision"] {
		t.Fatalf("each learner gets their own projection: ada %v, grace %v", adaProjection["revision"], graceProjection["revision"])
	}
	want := [][2]string{{"go-alp-a1-context", "full"}, {"go-alp-a2-goroutines", "full"}, {"go-alp-a3-generics", "full"}}
	if got := unitModes(t, graceProjection); !reflect.DeepEqual(got, want) {
		t.Fatalf("grace modes = %v, want %v", got, want)
	}

	// Competency status is read from, and cites, the one competency state.
	first := adaProjection["units"].([]any)[0].(map[string]any)["competencies"].([]any)[0].(map[string]any)
	if first["id"] != "go.runtime.context" || first["status"] != "demonstrated" || first["level"] != "strong" ||
		first["assessmentId"] != "asmt_seed_"+fmt.Sprintf("%x", []byte("go.runtime.context2026-10-01T09:00:00Z")) {
		t.Fatalf("unit competency = %#v", first)
	}
	for _, root := range []string{ada, grace} {
		if _, err := os.Stat(filepath.Join(root, "state", "competencies.yaml")); !os.IsNotExist(err) {
			t.Fatalf("plan must not write competency state: %v", err)
		}
		if files := filesIn(t, filepath.Join(root, "state", "target-adaptations")); len(files) != 1 {
			t.Fatalf("target adaptations = %v", files)
		}
	}
}

func decide(t *testing.T, root string, args ...string) platformRun {
	t.Helper()
	full := append([]string{"decision"}, args[0])
	full = append(full, "--adapter", "pylearn", "--target", "go-alp",
		"--curriculum", goALPCurriculumFixture, "--mapping", goALPMappingFixture, "--workspace", root)
	return runPlatform(t, App{}, append(full, args[1:]...)...)
}

func unitNamed(t *testing.T, projection map[string]any, item string) map[string]any {
	t.Helper()
	for _, raw := range projection["units"].([]any) {
		unit := raw.(map[string]any)
		if unit["item"].(map[string]any)["item"] == item {
			return unit
		}
	}
	t.Fatalf("no unit %q in projection", item)
	return nil
}

func TestPlatformDecisionRefusedWithoutLearnerConfirmation(t *testing.T) {
	root := adaLearnerState(t)
	revision := projectionOf(t, requirePlan(t, planTarget(t, root)))["revision"].(string)
	before := workspaceRevision(t, root)

	result := decide(t, root, "accept", "--unit", "go-alp-a3-generics", "--mode", "skim", "--basis", revision)
	if result.code != 1 {
		t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
	}
	if got := errorField(t, result); got["code"] != "learner-confirmation-required" {
		t.Fatalf("error = %#v", got)
	}
	if records := filesIn(t, filepath.Join(root, "adaptation-decisions")); len(records) != 0 {
		t.Fatalf("unconfirmed decision was recorded: %v", records)
	}
	if workspaceRevision(t, root) != before {
		t.Fatal("a refused decision must not change the workspace")
	}
}

func TestPlatformDecisionRequiresCurrentBasisRevision(t *testing.T) {
	root := adaLearnerState(t)
	stale := projectionOf(t, requirePlan(t, planTarget(t, root)))["revision"].(string)
	seedAssessment(t, root, "go.concurrency.goroutines", "strong", "high", "2026-10-02T09:00:00Z")

	result := decide(t, root, "accept", "--unit", "go-alp-a3-generics", "--mode", "skim", "--basis", stale, "--confirm")
	if result.code != 1 || errorField(t, result)["code"] != "stale-projection-revision" {
		t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
	}
	if records := filesIn(t, filepath.Join(root, "adaptation-decisions")); len(records) != 0 {
		t.Fatalf("decision on a stale basis was recorded: %v", records)
	}
}

func TestPlatformAcceptedDecisionIsRecordedAndSurvivesRebuild(t *testing.T) {
	root := adaLearnerState(t)
	basis := projectionOf(t, requirePlan(t, planTarget(t, root)))
	revision := basis["revision"].(string)
	before := workspaceRevision(t, root)

	result := decide(t, root, "accept", "--unit", "go-alp-a3-generics", "--mode", "skim", "--basis", revision,
		"--reason", "I use generics daily", "--confirm")
	if result.code != 0 {
		t.Fatalf("exit = %d, stderr = %s, stdout = %s", result.code, result.stderr, result.stdout)
	}
	body := decodeJSON(t, result.stdout)
	decision := body["decision"].(map[string]any)
	if body["status"] != "accepted" || decision["action"] != "accept" || decision["mode"] != "skim" || decision["unit"] != "go-alp-a3-generics" {
		t.Fatalf("decision = %#v", body)
	}
	confirmation := decision["confirmation"].(map[string]any)
	if confirmation["confirmedBy"] != "learner" || confirmation["confirmedAt"] == "" || decision["learnerId"] != "ada" {
		t.Fatalf("confirmation = %#v, learner = %v", confirmation, decision["learnerId"])
	}
	if decision["basis"].(map[string]any)["projectionRevision"] != revision || decision["recordedAt"] == "" {
		t.Fatalf("basis = %#v", decision["basis"])
	}
	id := decision["id"].(string)
	if records := filesIn(t, filepath.Join(root, "adaptation-decisions")); !reflect.DeepEqual(records, []string{id + ".yaml"}) {
		t.Fatalf("decision records = %v", records)
	}
	if workspaceRevision(t, root) == before {
		t.Fatal("an accepted decision is canonical state and must advance the workspace revision")
	}
	requireValidWorkspace(t, root)

	// The decision is an input, not part of the projection: deleting every
	// generated projection and re-planning still reflects it.
	if err := os.RemoveAll(filepath.Join(root, "state")); err != nil {
		t.Fatal(err)
	}
	rebuilt := projectionOf(t, requirePlan(t, planTarget(t, root)))
	unit := unitNamed(t, rebuilt, "go-alp-a3-generics")
	if unit["proposedMode"] != "full" || unit["mode"] != "skim" {
		t.Fatalf("unit = %#v", unit)
	}
	if applied := unit["decision"].(map[string]any); applied["id"] != id || applied["applied"] != true {
		t.Fatalf("applied decision = %#v", applied)
	}
	if got := stringList(rebuilt["inputs"].(map[string]any)["decisions"].([]any)); !reflect.DeepEqual(got, []string{id}) {
		t.Fatalf("inputs.decisions = %v", got)
	}
	if rebuilt["revision"] == revision || rebuilt["revision"] != projectionOf(t, body)["revision"] {
		t.Fatalf("revision after decision = %v, decision output %v, basis %v", rebuilt["revision"], projectionOf(t, body)["revision"], revision)
	}
}

func TestPlatformDecisionRevocationSupersedesTheDecision(t *testing.T) {
	root := adaLearnerState(t)
	revision := projectionOf(t, requirePlan(t, planTarget(t, root)))["revision"].(string)
	accepted := decide(t, root, "accept", "--unit", "go-alp-a2-goroutines", "--mode", "skip", "--basis", revision, "--confirm")
	if accepted.code != 0 {
		t.Fatalf("accept exit = %d, stdout = %s", accepted.code, accepted.stdout)
	}
	acceptedBody := decodeJSON(t, accepted.stdout)
	id := acceptedBody["decision"].(map[string]any)["id"].(string)
	afterAccept := projectionOf(t, acceptedBody)
	if unit := unitNamed(t, afterAccept, "go-alp-a2-goroutines"); unit["mode"] != "skip" {
		t.Fatalf("accepted skip not reflected: %#v", unit)
	}

	unconfirmed := decide(t, root, "revoke", "--decision", id, "--basis", afterAccept["revision"].(string))
	if unconfirmed.code != 1 || errorField(t, unconfirmed)["code"] != "learner-confirmation-required" {
		t.Fatalf("unconfirmed revoke exit = %d, stdout = %s", unconfirmed.code, unconfirmed.stdout)
	}

	revoked := decide(t, root, "revoke", "--decision", id, "--basis", afterAccept["revision"].(string), "--confirm")
	if revoked.code != 0 {
		t.Fatalf("revoke exit = %d, stderr = %s, stdout = %s", revoked.code, revoked.stderr, revoked.stdout)
	}
	revokedBody := decodeJSON(t, revoked.stdout)
	revocation := revokedBody["decision"].(map[string]any)
	if revokedBody["status"] != "revoked" || revocation["action"] != "revoke" ||
		!reflect.DeepEqual(stringList(revocation["supersedes"].([]any)), []string{id}) ||
		revocation["confirmation"].(map[string]any)["confirmedBy"] != "learner" {
		t.Fatalf("revocation = %#v", revokedBody)
	}
	// Append-only: the original record stays; the revocation supersedes it.
	if records := filesIn(t, filepath.Join(root, "adaptation-decisions")); len(records) != 2 {
		t.Fatalf("decision records = %v", records)
	}
	unit := unitNamed(t, projectionOf(t, requirePlan(t, planTarget(t, root))), "go-alp-a2-goroutines")
	if unit["mode"] != "skim" || unit["decision"] != nil {
		t.Fatalf("revoked decision still applied: %#v", unit)
	}

	again := decide(t, root, "revoke", "--decision", id, "--basis", projectionOf(t, revokedBody)["revision"].(string), "--confirm")
	if again.code != 1 || errorField(t, again)["code"] != "unknown-decision" {
		t.Fatalf("second revoke exit = %d, stdout = %s", again.code, again.stdout)
	}
	requireValidWorkspace(t, root)
}

func TestPlatformAcceptingANewModeSupersedesTheActiveDecision(t *testing.T) {
	root := adaLearnerState(t)
	revision := projectionOf(t, requirePlan(t, planTarget(t, root)))["revision"].(string)
	first := decodeJSON(t, decide(t, root, "accept", "--unit", "go-alp-a3-generics", "--mode", "skim", "--basis", revision, "--confirm").stdout)
	firstID := first["decision"].(map[string]any)["id"].(string)

	second := decide(t, root, "accept", "--unit", "go-alp-a3-generics", "--mode", "challenge", "--basis", projectionOf(t, first)["revision"].(string), "--confirm")
	if second.code != 0 {
		t.Fatalf("exit = %d, stdout = %s", second.code, second.stdout)
	}
	body := decodeJSON(t, second.stdout)
	if got := stringList(body["decision"].(map[string]any)["supersedes"].([]any)); !reflect.DeepEqual(got, []string{firstID}) {
		t.Fatalf("supersedes = %v", got)
	}
	if unit := unitNamed(t, projectionOf(t, body), "go-alp-a3-generics"); unit["mode"] != "challenge" {
		t.Fatalf("unit = %#v", unit)
	}
}

const goALPConstraintsFixture = "testdata/platform/pylearn/go-alp.constraints.yaml"

func TestPlatformPlanAppliesTargetConstraints(t *testing.T) {
	root := adaLearnerState(t)
	body := requirePlan(t, planTarget(t, root, "--constraints", goALPConstraintsFixture))
	projection := projectionOf(t, body)

	// context is demonstrated (skip) but required: never skipped, so the next
	// allowed mode is challenge. goroutines justifies skim, which the target
	// does not allow, so it is taken in full.
	want := [][2]string{{"go-alp-a1-context", "challenge"}, {"go-alp-a2-goroutines", "full"}, {"go-alp-a3-generics", "full"}}
	if got := unitModes(t, projection); !reflect.DeepEqual(got, want) {
		t.Fatalf("unit modes = %v, want %v", got, want)
	}
	if unit := unitNamed(t, projection, "go-alp-a1-context"); unit["proposedMode"] != "skip" {
		t.Fatalf("proposed mode = %v", unit["proposedMode"])
	}
	constraints, _ := projection["inputs"].(map[string]any)["constraints"].(map[string]any)
	if hash, _ := constraints["contentHash"].(string); len(hash) != len("sha256:")+64 {
		t.Fatalf("constraints input = %#v", constraints)
	}
	unconstrained := projectionOf(t, requirePlan(t, planTarget(t, root)))
	if unconstrained["revision"] == projection["revision"] {
		t.Fatal("constraints are a canonical input and must change the revision")
	}

	refused := decide(t, root, "accept", "--unit", "go-alp-a3-generics", "--mode", "skim", "--constraints", goALPConstraintsFixture,
		"--basis", projection["revision"].(string), "--confirm")
	if refused.code != 1 || errorField(t, refused)["code"] != "mode-not-allowed" {
		t.Fatalf("exit = %d, stdout = %s", refused.code, refused.stdout)
	}
	unknown := decide(t, root, "accept", "--unit", "go-alp-a1-context#quiz:context-cancellation", "--mode", "full", "--constraints", goALPConstraintsFixture,
		"--basis", projection["revision"].(string), "--confirm")
	if unknown.code != 1 || errorField(t, unknown)["code"] != "unknown-unit" {
		t.Fatalf("exit = %d, stdout = %s", unknown.code, unknown.stdout)
	}
	if records := filesIn(t, filepath.Join(root, "adaptation-decisions")); len(records) != 0 {
		t.Fatalf("refused decisions were recorded: %v", records)
	}
}

func TestPlatformPlanRejectsInvalidTargetConstraints(t *testing.T) {
	root := adaLearnerState(t)
	constraints := filepath.Join(t.TempDir(), "constraints.yaml")
	if err := os.WriteFile(constraints, []byte("schemaVersion: 1\nplatform: pylearn\ntarget: go-alp\nallowedModes: [skip]\nrequiredUnits: [go-alp-z9-missing]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := planTarget(t, root, "--constraints", constraints)
	if result.code != 1 || errorField(t, result)["code"] != "invalid-target-constraints" {
		t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
	}
	if files := filesIn(t, filepath.Join(root, "state", "target-adaptations")); len(files) != 0 {
		t.Fatalf("projection written from invalid constraints: %v", files)
	}
}

func TestPlatformPlanRefusesAdapterWithoutContentMapper(t *testing.T) {
	registry := platform.NewRegistry(curriculumOnlyAdapter{pylearn.NewAdapter()})
	root := newLearnerWorkspace(t, "ada")
	result := runPlatform(t, App{Platforms: &registry}, "plan", "--adapter", "curriculum-only", "--target", "go-alp",
		"--curriculum", goALPCurriculumFixture, "--mapping", goALPMappingFixture, "--workspace", root)
	if result.code != 1 {
		t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
	}
	if got := errorField(t, result); got["code"] != "capability-not-declared" || got["capability"] != "content-mapper" {
		t.Fatalf("error = %#v", got)
	}
}

func TestPlatformPlanAndDecisionUsageErrorsExitTwo(t *testing.T) {
	root := newLearnerWorkspace(t, "ada")
	base := []string{"--adapter", "pylearn", "--target", "go-alp", "--curriculum", goALPCurriculumFixture, "--mapping", goALPMappingFixture, "--workspace", root}
	cases := map[string][]string{
		"plan without mapping":     {"plan", "--adapter", "pylearn", "--target", "go-alp", "--curriculum", goALPCurriculumFixture},
		"decision without command": {"decision"},
		"accept without unit":      append([]string{"decision", "accept"}, append(base, "--mode", "skip", "--basis", "tapr_000000000000000000000000", "--confirm")...),
		"accept with unknown mode": append([]string{"decision", "accept"}, append(base, "--unit", "go-alp-a3-generics", "--mode", "drop", "--basis", "tapr_000000000000000000000000", "--confirm")...),
		"accept without basis":     append([]string{"decision", "accept"}, append(base, "--unit", "go-alp-a3-generics", "--mode", "skip", "--confirm")...),
		"revoke without decision":  append([]string{"decision", "revoke"}, append(base, "--basis", "tapr_000000000000000000000000", "--confirm")...),
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			result := runPlatform(t, App{}, args...)
			if result.code != 2 || errorField(t, result)["code"] != "usage" {
				t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
			}
		})
	}
}

func TestPlatformPlanOutputIsDeterministicJSON(t *testing.T) {
	first := planTarget(t, adaLearnerState(t))
	second := planTarget(t, adaLearnerState(t))
	if first.code != 0 || second.code != 0 {
		t.Fatalf("exit codes = %d, %d; stderr = %s", first.code, second.code, first.stderr)
	}
	golden := filepath.Join("testdata", "platform", "pylearn", "plan-go-alp.golden.json")
	if os.Getenv("ALP_WRITE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, first.stdout, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.stdout, want) {
		t.Fatalf("plan output mismatch\n--- got ---\n%s\n--- want ---\n%s", first.stdout, want)
	}
	if !bytes.Equal(first.stdout, second.stdout) {
		t.Fatal("plan output differs between identical learner states in fresh workspaces")
	}
}
