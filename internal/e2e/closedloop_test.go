//go:build closedloop

// The closed-loop go-alp smoke across ALP and PyLearn (spec #66 Seam 3, Q28,
// Q38; ticket #75). It needs a PyLearn checkout and devenv, so it is behind
// the `closedloop` build tag and never part of `go test ./...`. Run it with
// scripts/smoke-closed-loop.sh (docs/integrations/CLOSED_LOOP_SMOKE.md).
//
// Everything learner-side is synthetic and throwaway: a fresh workspace with
// a synthetic profile and personas, an isolated HOME, and a throwaway PyLearn
// libSQL (SQLite file) database. The PyLearn checkout is only read, except
// for the files its own gates regenerate, which are restored afterwards; the
// smoke fails if it leaves the checkout's git status changed.
//
// Real Claude/Codex harness runs are out of scope: the authoring and
// assessing agents are played by this test with fixed synthetic inputs.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/gitexec"
	"go.yaml.in/yaml/v3"
)

const (
	smokeTarget = "go-alp"
	// The go-alp unit PyLearn #47 realized from an ALP target-skeleton plan.
	realizedLesson      = "go-alp-a3-sync-primitives"
	realizedCompetency  = "go.concurrency.sync"
	realizedQuizConcept = "go-alp-sync-primitives"
	// Files PyLearn's gates rewrite in place (compile:go's wasm manifest).
	pylearnGeneratedManifest = "apps/web/public/wasm/go/manifest.json"
	// PyLearn writes timestamps at second precision; a fixed answer time keeps
	// the exported events and imported evidence deterministic.
	quizAnsweredAtMs = 1791363600000 // 2026-10-07T09:00:00Z
	pylearnUser      = "smoke-pylearn-user"
)

// Synthetic learner: a senior engineer whose Go basics are already shown.
const (
	smokeProfile = `schemaVersion: 1
learner:
  id: smoke-learner
  displayName: Synthetic Smoke Learner
  professionalLevel: senior
experience:
  typescript:
    level: strong
    relativeRank: 1
    frameworks:
      nestjs: strong
  csharp:
    level: strong
    relativeRank: 2
    frameworks:
      dotnet: strong
goals:
  - id: production-go
    statement: Production-ready Go concurrency.
preferences:
  feedbackStyle: direct-technical
`
	smokeGlobalPersona = `schemaVersion: 1
scope: global
domain: null
teaching:
  pace: standard
  preferredActivityTypes: [debugging, code-reading]
  avoid: [toy-examples]
  emphasis: [explicit-tradeoffs]
  feedback: [direct]
analogyPolicy:
  defaultPriority: [dotnet, nestjs]
risks:
  - skimming past invariants when the syntax looks familiar
`
	smokeGoPersona = `schemaVersion: 1
scope: domain
domain: go
teaching:
  pace: senior-dense
  preferredActivityTypes: [concurrency-debugging]
  emphasis: [concurrency]
analogyPolicy:
  semanticOverrides:
    concurrency:
      prefer: [dotnet]
      reason: Start from lock, Interlocked and Task.WhenAll.
risks:
  - unguarded shared state across goroutines
`
	smokeConstraints = `schemaVersion: 1
platform: pylearn
target: go-alp
allowedModes: [skip, challenge, skim, full]
goal: [go.concurrency.sync, go.concurrency.races-deadlocks-leaks]
`
)

// startingCompetencies are demonstrated before the loop starts, so the
// skeleton plan proposes only what the go-alp goal still needs.
var startingCompetencies = []string{"go.language.syntax", "go.runtime.scheduler", "go.concurrency.goroutines"}

type smoke struct {
	t       *testing.T
	dir     string
	alpBin  string
	home    string
	ws      string
	pylearn string
	dbURL   string
	// Post-authoring PyLearn inputs.
	curriculum, mapping, constraints string
}

type alpRun struct {
	code           int
	stdout, stderr []byte
}

func TestClosedLoopGoALP(t *testing.T) {
	s := newSmoke(t)

	// Step 1: build alp, inspect go-alp through PyLearn's curriculum export.
	s.step(1, "inspect go-alp via export:curriculum")
	s.curriculum = s.path("curriculum-after.json")
	s.write(s.curriculum, s.pylearnRun("bun", "run", "--cwd", "apps/web", "export:curriculum", "--course", smokeTarget))
	s.mapping = filepath.Join(s.pylearn, "apps/web/content/alp/go-alp.mapping.yaml")
	s.constraints = s.path("go-alp.constraints.yaml")
	s.write(s.constraints, []byte(smokeConstraints))
	var inspect struct {
		Capabilities    []string
		AuthoringTarget *struct{ Skill string }
		Items           []struct {
			Ref   struct{ Item string }
			Phase string
		}
	}
	s.decode(s.alpOK("platform", "inspect", "--adapter", "pylearn", "--target", smokeTarget, "--curriculum", s.curriculum), &inspect)
	for _, capability := range []string{"activity-source", "curriculum-reader", "content-mapper", "authoring-target", "platform-validator"} {
		if !slices.Contains(inspect.Capabilities, capability) {
			t.Fatalf("inspect capabilities = %v, missing %s", inspect.Capabilities, capability)
		}
	}
	if inspect.AuthoringTarget == nil || inspect.AuthoringTarget.Skill != "pylearn-alp-authoring" {
		t.Fatalf("inspect authoringTarget = %+v", inspect.AuthoringTarget)
	}
	lessonPhase := ""
	for _, item := range inspect.Items {
		if item.Ref.Item == realizedLesson {
			lessonPhase = item.Phase
		}
	}
	if lessonPhase == "" {
		t.Fatalf("go-alp export does not list the realized lesson %s", realizedLesson)
	}
	var mappingCheck struct{ Valid bool }
	s.decode(s.alpOK("platform", "mapping", "validate", "--adapter", "pylearn", "--target", smokeTarget,
		"--curriculum", s.curriculum, "--mapping", s.mapping), &mappingCheck)
	t.Logf("go-alp: %d declared-stable items, %s in phase %s, mapping valid", len(inspect.Items), realizedLesson, lessonPhase)

	// The target as it was before authoring: what alp:bootstrap printed for a
	// course PyLearn had not declared yet, from the same library functions.
	pack := s.mappingPack()
	before := s.path("curriculum-before.json")
	beforeMapping := s.path("go-alp.mapping.before.yaml")
	s.write(before, s.pylearnRun("bash", "-c", `cd apps/web && exec ./node_modules/.bin/tsx "$0" "$@"`, s.helper(),
		"bootstrap", "--target", smokeTarget, "--title", "Go (ALP)", "--pack", pack, "--mapping-out", beforeMapping))

	// Step 2: starting evidence, then plan go-alp.
	s.step(2, "record starting evidence, plan go-alp")
	for _, competency := range startingCompetencies {
		s.recordAssessment(competency, "strong", "high", "2026-10-01T09:00:00Z", []string{s.recordEvidence(competency)})
	}
	first := s.plan(before, beforeMapping)
	if first.Authoring.Status != "explicit-intent-required" || first.Authoring.Intent != "target-skeleton" {
		t.Fatalf("default plan of the empty target authoring = %+v", first.Authoring)
	}
	skeleton := s.plan(before, beforeMapping, "--intent", "target-skeleton")
	if skeleton.Authoring.Status != "planned" || skeleton.AuthoringPlan == nil {
		t.Fatalf("skeleton plan authoring = %+v", skeleton.Authoring)
	}
	for _, document := range skeleton.Specifications.Persona.Documents {
		if !document.Present {
			t.Fatalf("synthetic persona document %s not used", document.Path)
		}
	}
	proposed := skeleton.unitFor(t, realizedCompetency, s.ws)
	// Proposed unit IDs derive from {target, competency}, so the synthetic
	// learner's plan names the unit PyLearn #47 realized and cited.
	if cited := s.citedUnitID(); proposed.ID != cited {
		t.Fatalf("skeleton unit for %s = %s, PyLearn mapping cites %s", realizedCompetency, proposed.ID, cited)
	}
	skeletonPlan := skeleton.AuthoringPlan.ID
	t.Logf("skeleton plan %s: %d units; %s proposed as %s v%d", skeletonPlan, len(skeleton.Specifications.Units),
		realizedCompetency, proposed.ID, proposed.Version)

	// PyLearn #47 saw a new spec ID when it planned again on the authored
	// export. Before gates record nothing is realized, so the authored lesson
	// is an unlinked existing item and gets an item-derived ID.
	interim := s.plan(s.curriculum, s.mapping)
	interimUnit := interim.unitAt(t, realizedLesson)
	t.Logf("re-plan before gates record: %s appears as %s (realized: %v)", realizedLesson, interimUnit.ID, interimUnit.Realized)

	// Step 3: the already-realized unit. Gates, then gates record.
	s.step(3, "validate:platform and gates record for the realized unit")
	gateResult := s.path("gate-result.json")
	s.write(gateResult, s.pylearnGates())
	var gates struct {
		Target      string
		Publishable bool
		Gates       []struct{ ID, Status string }
	}
	s.decode(s.read(gateResult), &gates)
	if !gates.Publishable || gates.Target != smokeTarget {
		t.Fatalf("validate:platform = %s", s.read(gateResult))
	}
	realization := s.path("realization.json")
	s.write(realization, s.realizationReport(skeletonPlan, proposed.ID, skeleton))
	record := s.gatesRecord(skeletonPlan, gateResult, realization)
	if record.Status != "recorded" {
		t.Fatalf("gates record status = %s", record.Status)
	}
	for _, unit := range record.Record.Units {
		want, reason := "unrealized", "realization-not-reported"
		if unit.Unit.ID == proposed.ID {
			want, reason = "realized", ""
		}
		if unit.Status != want || (reason != "" && (len(unit.Reasons) != 1 || unit.Reasons[0].Code != reason)) {
			t.Fatalf("gates record unit %s = %+v, want %s %s", unit.Unit.ID, unit, want, reason)
		}
	}
	if len(record.Realizations) != 1 || record.Realizations[0].Root.Item != realizedLesson || record.Realizations[0].Unit.ID != proposed.ID {
		t.Fatalf("realizations = %+v", record.Realizations)
	}
	if again := s.gatesRecord(skeletonPlan, gateResult, realization); again.Status != "already-recorded" {
		t.Fatalf("second gates record status = %s", again.Status)
	}
	t.Logf("gates: publishable %v (%s); %s v%d realized by %s", gates.Publishable, gateSummary(gates.Gates), proposed.ID, proposed.Version, record.Realizations[0].Path)

	// Spec-ID stability: the realized unit keeps its proposed ID.
	realized := s.plan(s.curriculum, s.mapping)
	kept := realized.unitAt(t, realizedLesson)
	if kept.ID != proposed.ID || kept.Version != proposed.Version || kept.Realized == nil || !*kept.Realized ||
		(kept.Status != "held" && kept.Status != "unchanged") {
		t.Fatalf("realized unit after gates record = %+v, want %s v%d realized", kept, proposed.ID, proposed.Version)
	}
	if realized.AuthoringPlan != nil && slices.Contains(realized.AuthoringPlan.unitIDs(), proposed.ID) {
		t.Fatalf("realized unit %s authored again by %s", proposed.ID, realized.AuthoringPlan.ID)
	}
	if again := s.plan(s.curriculum, s.mapping).unitAt(t, realizedLesson); again.ID != proposed.ID {
		t.Fatalf("second re-plan unit = %+v", again)
	}
	beforeImport := realized.Projection
	unitBefore := beforeImport.unit(t, realizedLesson)
	t.Logf("re-plan after realization: %s kept %s v%d (%s, realized); projection %s, %s mode %s",
		realizedLesson, kept.ID, kept.Version, kept.Status, beforeImport.Revision, realizedLesson, unitBefore.Mode)

	// Step 4: synthetic activity through PyLearn's persistence, export, import twice.
	s.step(4, "seed PyLearn activity, export:activity, import twice")
	s.pylearnRun("bun", "run", "--cwd", "apps/web", "db:migrate")
	s.pylearnRun("bun", "run", "--cwd", "apps/web", "db:sync-lessons")
	answers := `[{"questionId":"q1","picked":1,"correct":true},{"questionId":"q2","picked":2,"correct":true},` +
		`{"questionId":"q3","picked":0,"correct":true},{"questionId":"q4","picked":2,"correct":true}]`
	s.pylearnRun("bash", "-c", `cd apps/web && exec ./node_modules/.bin/tsx "$0" "$@"`, s.helper(),
		"seed-quiz", "--user", pylearnUser, "--lesson", realizedLesson, "--concept", realizedQuizConcept,
		"--answers", answers, "--now-ms", fmt.Sprint(quizAnsweredAtMs))
	export := s.path("activity-export.json")
	s.write(export, s.pylearnRun("bun", "run", "--cwd", "apps/web", "export:activity", "--user", pylearnUser))
	var activity activityExport
	s.decode(s.read(export), &activity)
	if records := activity.records(smokeTarget); len(records) != 4 {
		t.Fatalf("go-alp export records = %+v", records)
	}

	refused := s.alp(s.importArgs(export))
	if refused.code != 1 || s.errorCode(refused) != "platform-account-not-linked" {
		t.Fatalf("import before account link exit = %d, stdout = %s", refused.code, refused.stdout)
	}
	var link struct{ Status string }
	s.decode(s.alpOK("platform", "account", "link", "--adapter", "pylearn", "--instance", activity.Instance,
		"--user", activity.User.ID, "--confirm", "--workspace", s.ws), &link)
	if link.Status != "linked" {
		t.Fatalf("account link status = %s", link.Status)
	}
	evidenceBefore := s.evidenceFiles()
	firstImport := s.importActivity(export)
	if c := firstImport.Counts; c.Records != 4 || c.Imported != 4 || c.Skipped != 0 || c.Unmapped != 0 {
		t.Fatalf("first import counts = %+v", c)
	}
	imported := s.evidenceFiles()
	if len(imported) != len(evidenceBefore)+4 {
		t.Fatalf("evidence after first import = %v", imported)
	}
	secondImport := s.importActivity(export)
	if c := secondImport.Counts; c.Records != 4 || c.Imported != 0 || c.Skipped != 4 || c.Superseded != 0 {
		t.Fatalf("second import counts = %+v", c)
	}
	if after := s.evidenceFiles(); !reflect.DeepEqual(after, imported) {
		t.Fatalf("second import changed evidence: %v -> %v", imported, after)
	}
	// The caller's cursor round-trips: nothing changed since, so nothing comes back.
	incremental := s.path("activity-export.incremental.json")
	s.write(incremental, s.pylearnRun("bun", "run", "--cwd", "apps/web", "export:activity", "--user", pylearnUser,
		"--since", firstImport.Cursor.Next))
	cursorImport := s.importActivity(incremental, "--cursor", firstImport.Cursor.Next)
	if c := cursorImport.Counts; c.Records != 0 || c.Imported != 0 {
		t.Fatalf("incremental import counts = %+v", c)
	}
	t.Logf("import 1: %+v; import 2: %+v; since cursor: %+v", firstImport.Counts, secondImport.Counts, cursorImport.Counts)

	// Step 5: the projection changes only when the evidence warrants it.
	s.step(5, "projection changes where warranted, next plan from post-import state")
	unassessed := s.plan(s.curriculum, s.mapping)
	if unassessed.Projection.Revision != beforeImport.Revision {
		t.Fatalf("imported evidence alone changed the projection: %s -> %s", beforeImport.Revision, unassessed.Projection.Revision)
	}
	if unit := unassessed.unitAt(t, realizedLesson); unit.ID != proposed.ID || unit.Version != proposed.Version {
		t.Fatalf("unit re-specified without new learner state: %+v", unit)
	}
	t.Logf("re-plan after import, before assessment: projection %s unchanged (evidence is not learner state until assessed)", unassessed.Projection.Revision)

	var evidenceIDs []string
	for _, record := range firstImport.Records {
		evidenceIDs = append(evidenceIDs, record.Evidence...)
	}
	assessment := s.recordAssessment(realizedCompetency, "functional", "high", "2026-10-07T10:00:00Z", evidenceIDs)
	next := s.plan(s.curriculum, s.mapping)
	unitAfter := next.Projection.unit(t, realizedLesson)
	if next.Projection.Revision == beforeImport.Revision {
		t.Fatalf("projection unchanged after the assessed import: %s", next.Projection.Revision)
	}
	competency := unitAfter.competency(t, realizedCompetency)
	if competency.Status != "functional" || competency.AssessmentID != assessment ||
		unitBefore.ProposedMode != "full" || unitAfter.ProposedMode != "skim" {
		t.Fatalf("%s before = %+v, after = %+v", realizedLesson, unitBefore, unitAfter)
	}
	revised := next.unitAt(t, realizedLesson)
	if revised.ID != proposed.ID || revised.Version != proposed.Version+1 || revised.Status != "revised" {
		t.Fatalf("realized unit after evidence = %+v, want %s v%d revised", revised, proposed.ID, proposed.Version+1)
	}
	var spec struct {
		Change struct {
			MaterialFields []string
			EvidenceLinked bool
		}
		Provenance struct{ ProjectionRevision string }
	}
	s.decode(s.read(filepath.Join(s.ws, revised.Path)), &spec)
	if !spec.Change.EvidenceLinked || !slices.Contains(spec.Change.MaterialFields, "adaptationMode") ||
		spec.Provenance.ProjectionRevision != next.Projection.Revision {
		t.Fatalf("revised spec change = %+v, provenance = %+v", spec.Change, spec.Provenance)
	}
	if again := s.plan(s.curriculum, s.mapping); again.Projection.Revision != next.Projection.Revision ||
		again.unitAt(t, realizedLesson).Status != "unchanged" {
		t.Fatalf("re-plan of unchanged state = %s, unit %+v", again.Projection.Revision, again.unitAt(t, realizedLesson))
	}
	t.Logf("assessment %s (functional, citing %d imported evidence): projection %s -> %s; %s proposed %s -> %s; %s v%d (%s, evidence-linked %v)",
		assessment, len(evidenceIDs), beforeImport.Revision, next.Projection.Revision, realizedLesson,
		unitBefore.ProposedMode, unitAfter.ProposedMode, revised.ID, revised.Version, strings.Join(spec.Change.MaterialFields, ","), spec.Change.EvidenceLinked)
	if next.AuthoringPlan != nil {
		t.Logf("next adaptation: %s %s (%s)", next.Authoring.Status, next.Authoring.Intent, next.AuthoringPlan.ID)
	} else {
		t.Logf("next adaptation: %s", next.Authoring.Status)
	}
}

// ---- setup ----------------------------------------------------------------

func newSmoke(t *testing.T) *smoke {
	t.Helper()
	pylearn := os.Getenv("ALP_SMOKE_PYLEARN")
	if pylearn == "" {
		t.Fatal("ALP_SMOKE_PYLEARN must name a PyLearn checkout (run scripts/smoke-closed-loop.sh <pylearn-dir>)")
	}
	pylearn, err := filepath.Abs(pylearn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(pylearn, "apps/web/content/courses/go-alp.yaml")); err != nil {
		t.Fatalf("%s is not a PyLearn checkout with the go-alp course: %v", pylearn, err)
	}
	if _, err := exec.LookPath("devenv"); err != nil {
		t.Fatal("devenv is required to run PyLearn's toolchain")
	}
	dir := t.TempDir()
	s := &smoke{t: t, dir: dir, pylearn: pylearn,
		alpBin: filepath.Join(dir, "bin", "alp"), home: filepath.Join(dir, "home"), ws: filepath.Join(dir, "workspace"),
		dbURL: "file:" + filepath.Join(dir, "pylearn-smoke.db")}

	t.Log("step 0: build alp from this tree, prepare PyLearn")
	build := exec.Command("go", "build", "-o", s.alpBin, "./cmd/alp")
	build.Dir = moduleRoot(t)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
	for _, path := range []string{s.home, filepath.Join(s.ws, "profile"), filepath.Join(s.ws, "personas", "domains")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	s.write(filepath.Join(s.ws, "workspace.yaml"), []byte("schemaVersion: 2\nworkspaceId: ws_closed_loop_smoke\nlearnerId: smoke-learner\n"))
	s.write(filepath.Join(s.ws, "profile", "profile.yaml"), []byte(smokeProfile))
	s.write(filepath.Join(s.ws, "personas", "global.yaml"), []byte(smokeGlobalPersona))
	s.write(filepath.Join(s.ws, "personas", "domains", "go.yaml"), []byte(smokeGoPersona))

	status := s.pylearnStatus()
	t.Cleanup(func() {
		if after := s.pylearnStatus(); after != status {
			t.Errorf("smoke left the PyLearn checkout changed:\nbefore:\n%s\nafter:\n%s", status, after)
		}
	})
	s.pylearnRun("bun", "install", "--frozen-lockfile")
	t.Logf("alp %s; PyLearn %s @ %s; workspace %s; PyLearn DB %s", s.alpBin, s.pylearn, s.pylearnHead(), s.ws, s.dbURL)
	return s
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the module root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func (s *smoke) helper() string {
	return filepath.Join(moduleRoot(s.t), "internal", "e2e", "testdata", "pylearn-smoke.ts")
}

func (s *smoke) step(n int, title string) {
	s.t.Helper()
	s.t.Logf("step %d: %s", n, title)
}

func (s *smoke) path(name string) string { return filepath.Join(s.dir, name) }

func (s *smoke) write(path string, data []byte) {
	s.t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		s.t.Fatal(err)
	}
}

func (s *smoke) read(path string) []byte {
	s.t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		s.t.Fatal(err)
	}
	return data
}

func (s *smoke) decode(data []byte, target any) {
	s.t.Helper()
	if err := json.Unmarshal(data, target); err != nil {
		s.t.Fatalf("decode %T: %v\n%s", target, err, data)
	}
}

// ---- PyLearn --------------------------------------------------------------

// pylearnRun runs a command in PyLearn's devenv against the throwaway
// database and returns its stdout; any failure fails the smoke.
func (s *smoke) pylearnRun(args ...string) []byte {
	s.t.Helper()
	stdout, stderr, err := s.pylearnExec(args...)
	if err != nil {
		s.t.Fatalf("PyLearn %v: %v\nstdout:\n%s\nstderr:\n%s", args, err, stdout, stderr)
	}
	return stdout
}

func (s *smoke) pylearnExec(args ...string) ([]byte, []byte, error) {
	command := exec.Command("devenv", append([]string{"shell", "--"}, args...)...)
	command.Dir = s.pylearn
	command.Env = append(os.Environ(), "DATABASE_URL="+s.dbURL, "ALP_BIN="+s.alpBin)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// pylearnGates runs validate:platform and returns its result. compile:go
// rewrites its wasm manifest with toolchain-dependent sizes; the file is put
// back as it was so the smoke never edits the checkout.
func (s *smoke) pylearnGates() []byte {
	s.t.Helper()
	manifest := filepath.Join(s.pylearn, pylearnGeneratedManifest)
	saved := s.read(manifest)
	defer s.write(manifest, saved)
	return s.pylearnRun("bun", "run", "validate:platform", "--target", smokeTarget)
}

func (s *smoke) pylearnStatus() string {
	s.t.Helper()
	output, err := gitexec.Command(context.Background(), s.pylearn, "status", "--porcelain").Output()
	if err != nil {
		s.t.Fatalf("git status in %s: %v", s.pylearn, err)
	}
	return string(output)
}

func (s *smoke) pylearnHead() string {
	s.t.Helper()
	output, err := gitexec.Command(context.Background(), s.pylearn, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		s.t.Fatalf("git rev-parse in %s: %v", s.pylearn, err)
	}
	return strings.TrimSpace(string(output))
}

// mappingPack is the go-alp mapping's domain pack as alp:bootstrap takes it.
func (s *smoke) mappingPack() string {
	s.t.Helper()
	var mapping struct {
		Packs []struct {
			Domain      string `yaml:"domain"`
			PackVersion string `yaml:"packVersion"`
		} `yaml:"packs"`
	}
	if err := yaml.Unmarshal(s.read(s.mapping), &mapping); err != nil || len(mapping.Packs) != 1 {
		s.t.Fatalf("go-alp mapping packs = %+v, err = %v", mapping.Packs, err)
	}
	return mapping.Packs[0].Domain + "@" + mapping.Packs[0].PackVersion
}

// citedUnitID is the unit specification the go-alp mapping says the realized
// lesson realizes.
func (s *smoke) citedUnitID() string {
	s.t.Helper()
	ids := regexp.MustCompile(`uspec_[0-9a-f]{24}`).FindAllString(string(s.read(s.mapping)), -1)
	if len(ids) == 0 {
		s.t.Fatal("go-alp mapping cites no unit specification")
	}
	return ids[0]
}

// realizationReport is what the authoring skill hands back for the realized
// unit: its lesson and every declared-stable item under it, and one sourced
// claim per teachingIntent claim (the reel's lastVerified / stackVersions).
func (s *smoke) realizationReport(planID, unitID string, skeleton planOutput) []byte {
	s.t.Helper()
	var export struct {
		Targets []struct {
			Items []struct{ ID, Parent string }
		}
	}
	s.decode(s.read(s.curriculum), &export)
	items := []string{realizedLesson}
	for changed := true; changed; {
		changed = false
		for _, item := range export.Targets[0].Items {
			if item.Parent != "" && slices.Contains(items, item.Parent) && !slices.Contains(items, item.ID) {
				items, changed = append(items, item.ID), true
			}
		}
	}
	var claims []any
	for _, unit := range skeleton.AuthoringPlan.Public.Units {
		if unit.ID != unitID {
			continue
		}
		for _, claim := range unit.TeachingIntent.Claims {
			claims = append(claims, map[string]any{"domain": claim.Domain, "competency": claim.Competency,
				"sources": []any{map[string]any{"url": "https://pkg.go.dev/sync", "version": "go1.27", "verifiedAt": "2026-10-07"}}})
		}
	}
	report, err := json.Marshal(map[string]any{"schemaVersion": 1, "plan": planID,
		"units": []any{map[string]any{"unit": unitID, "items": items, "claims": claims}}})
	if err != nil {
		s.t.Fatal(err)
	}
	return report
}

// ---- ALP ------------------------------------------------------------------

func (s *smoke) alp(args []string) alpRun {
	s.t.Helper()
	command := exec.Command(s.alpBin, args...)
	command.Dir = s.dir
	command.Env = append(os.Environ(), "HOME="+s.home)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	code := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		s.t.Fatalf("alp %v: %v", args, err)
	}
	return alpRun{code: code, stdout: stdout.Bytes(), stderr: stderr.Bytes()}
}

func (s *smoke) alpOK(args ...string) []byte {
	s.t.Helper()
	run := s.alp(args)
	if run.code != 0 {
		s.t.Fatalf("alp %v exit = %d\nstdout:\n%s\nstderr:\n%s", args, run.code, run.stdout, run.stderr)
	}
	return run.stdout
}

func (s *smoke) errorCode(run alpRun) string {
	s.t.Helper()
	var body struct{ Error struct{ Code string } }
	s.decode(run.stdout, &body)
	return body.Error.Code
}

func (s *smoke) recordEvidence(competency string) string {
	s.t.Helper()
	id := "ev_smoke_start_" + strings.NewReplacer(".", "_", "-", "_").Replace(competency)
	file := s.path(id + ".yaml")
	s.write(file, []byte(`schemaVersion: 1
id: `+id+`
recordedAt: 2026-10-01T09:00:00Z
domain: go
competencies: [`+competency+`]
type: explanation
source: {kind: diagnostic, ref: closed-loop-smoke}
observation: Synthetic starting evidence for the closed-loop smoke.
result: pass
strength: moderate
`))
	s.alpOK("evidence", "add", "--workspace", s.ws, "--file", file)
	return id
}

// recordAssessment plays the assessing agent: a proposed assessment that ALP
// accepts against the cited evidence.
func (s *smoke) recordAssessment(competency, level, confidence, recordedAt string, evidence []string) string {
	s.t.Helper()
	id := "asmt_smoke_" + strings.NewReplacer(".", "_", "-", "_", ":", "", "T", "_", "Z", "").Replace(competency+"_"+recordedAt)
	cited, err := json.Marshal(evidence)
	if err != nil {
		s.t.Fatal(err)
	}
	file := s.path(id + ".yaml")
	s.write(file, []byte(`schemaVersion: 1
id: `+id+`
recordedAt: `+recordedAt+`
domain: go
competency: `+competency+`
evidence: `+string(cited)+`
rubric: {id: go-competency, version: "1"}
assessor: {type: agent, id: closed-loop-smoke}
judgment: {level: `+level+`}
confidence: `+confidence+`
rationale: Synthetic assessment for the closed-loop smoke.
status: proposed
`))
	s.alpOK("assessment", "add", "--workspace", s.ws, "--file", file)
	return id
}

func (s *smoke) evidenceFiles() []string {
	s.t.Helper()
	entries, err := os.ReadDir(filepath.Join(s.ws, "evidence"))
	if err != nil {
		s.t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

type planOutput struct {
	Projection     projection
	Specifications struct {
		Units   []specUnit
		Persona struct {
			Documents []struct {
				Path    string
				Present bool
			}
		}
	}
	Authoring     struct{ Status, Intent, Reason string }
	AuthoringPlan *authoringPlan
}

type specUnit struct {
	ID, Title, Status, Path string
	Version                 int
	Realized                *bool
	PlatformItem            *struct{ Item string }
}

type authoringPlan struct {
	ID     string
	Public struct {
		Units []struct {
			ID             string
			TeachingIntent struct {
				Claims []struct{ Domain, Competency string }
			}
		}
	}
}

func (p *authoringPlan) unitIDs() []string {
	var ids []string
	for _, unit := range p.Public.Units {
		ids = append(ids, unit.ID)
	}
	return ids
}

type projection struct {
	Revision string
	Units    []projectionUnit
}

type projectionUnit struct {
	Item         struct{ Item string }
	Mode         string
	ProposedMode string
	Competencies []struct{ ID, Status, AssessmentID string }
}

func (p projection) unit(t *testing.T, item string) projectionUnit {
	t.Helper()
	for _, unit := range p.Units {
		if unit.Item.Item == item {
			return unit
		}
	}
	t.Fatalf("projection has no unit %s: %+v", item, p.Units)
	return projectionUnit{}
}

func (u projectionUnit) competency(t *testing.T, id string) struct{ ID, Status, AssessmentID string } {
	t.Helper()
	for _, competency := range u.Competencies {
		if competency.ID == id {
			return competency
		}
	}
	t.Fatalf("unit %s has no competency %s: %+v", u.Item.Item, id, u.Competencies)
	return struct{ ID, Status, AssessmentID string }{}
}

// unitAt is the specification entry placed at a platform item.
func (p planOutput) unitAt(t *testing.T, item string) specUnit {
	t.Helper()
	for _, unit := range p.Specifications.Units {
		if unit.PlatformItem != nil && unit.PlatformItem.Item == item {
			return unit
		}
	}
	t.Fatalf("plan has no unit specification at %s: %+v", item, p.Specifications.Units)
	return specUnit{}
}

// unitFor is the proposed unit specification teaching a competency.
func (p planOutput) unitFor(t *testing.T, competency, workspace string) specUnit {
	t.Helper()
	for _, unit := range p.Specifications.Units {
		data, err := os.ReadFile(filepath.Join(workspace, unit.Path))
		if err != nil {
			t.Fatal(err)
		}
		var spec struct {
			Teaching struct{ Competencies []struct{ ID string } }
		}
		if err := json.Unmarshal(data, &spec); err != nil {
			t.Fatal(err)
		}
		if len(spec.Teaching.Competencies) == 1 && spec.Teaching.Competencies[0].ID == competency {
			return unit
		}
	}
	t.Fatalf("plan proposes no unit for %s: %+v", competency, p.Specifications.Units)
	return specUnit{}
}

func (s *smoke) plan(curriculum, mapping string, extra ...string) planOutput {
	s.t.Helper()
	args := append([]string{"platform", "plan", "--adapter", "pylearn", "--target", smokeTarget, "--curriculum", curriculum,
		"--mapping", mapping, "--constraints", s.constraints, "--workspace", s.ws}, extra...)
	var out planOutput
	s.decode(s.alpOK(args...), &out)
	return out
}

type gatesRecordOutput struct {
	Status string
	Record struct {
		Units []struct {
			Unit struct {
				ID      string
				Version int
			}
			Status  string
			Reasons []struct{ Code string }
		}
	}
	Realizations []struct {
		Path string
		Root struct{ Item string }
		Unit struct{ ID string }
	}
}

func (s *smoke) gatesRecord(planID, result, realization string) gatesRecordOutput {
	s.t.Helper()
	var out gatesRecordOutput
	s.decode(s.alpOK("platform", "gates", "record", "--adapter", "pylearn", "--target", smokeTarget, "--curriculum", s.curriculum,
		"--plan", planID, "--result", result, "--realization", realization, "--workspace", s.ws), &out)
	return out
}

func gateSummary(gates []struct{ ID, Status string }) string {
	var parts []string
	for _, gate := range gates {
		parts = append(parts, gate.ID+" "+gate.Status)
	}
	return strings.Join(parts, ", ")
}

type activityExport struct {
	Instance string
	User     struct{ ID string }
	Targets  []struct {
		ID      string
		Records []json.RawMessage
	}
}

func (e activityExport) records(target string) []json.RawMessage {
	for _, t := range e.Targets {
		if t.ID == target {
			return t.Records
		}
	}
	return nil
}

type importOutput struct {
	Counts  struct{ Records, Imported, Skipped, Superseded, Unmapped int }
	Cursor  struct{ Next string }
	Records []struct{ Evidence []string }
}

func (s *smoke) importArgs(export string, extra ...string) []string {
	return append([]string{"platform", "import", "--adapter", "pylearn", "--target", smokeTarget, "--curriculum", s.curriculum,
		"--mapping", s.mapping, "--export", export, "--workspace", s.ws}, extra...)
}

func (s *smoke) importActivity(export string, extra ...string) importOutput {
	s.t.Helper()
	var out importOutput
	s.decode(s.alpOK(s.importArgs(export, extra...)...), &out)
	return out
}
