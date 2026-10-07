package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
)

// Real PyLearn output, produced at PyLearn integration/66-alp-reference-integration
// @ 9c7ba35 (no PyLearn changes):
//   - curriculum-export.go.json: `bun run --cwd apps/web export:curriculum --course go`;
//   - go.pylearn.mapping.yaml: apps/web/content/alp/go.mapping.yaml;
//   - gate-result.go.json: `bun run validate:platform --target go` (publishable);
//   - gate-result.go-alp.unpublishable.json: `bun run validate:platform --target go-alp`
//     before the go-alp course existed (not publishable).
const (
	pylearnGoCurriculumFixture = "testdata/platform/pylearn/curriculum-export.go.json"
	realGoMappingFixture       = "testdata/platform/pylearn/go.pylearn.mapping.yaml"
	pylearnGoGateFixture       = "testdata/platform/pylearn/gate-result.go.json"
	pylearnGoALPFailedGate     = "testdata/platform/pylearn/gate-result.go-alp.unpublishable.json"
)

// planPyLearnGo plans the real PyLearn `go` target; with no learner state the
// smallest justified intent is an activity for go-a1-first-delivery.
func planPyLearnGo(t *testing.T, root string) (planID, unitID string, plan map[string]any) {
	t.Helper()
	body := requirePlan(t, runPlatform(t, App{}, "plan", "--adapter", "pylearn", "--target", "go",
		"--curriculum", pylearnGoCurriculumFixture, "--mapping", realGoMappingFixture, "--workspace", root))
	authoring, plan := authoringOf(t, body)
	if authoring.Status != "planned" || authoring.Intent != "activity" {
		t.Fatalf("authoring = %#v", authoring)
	}
	units := planUnitIDs(plan)
	if len(units) != 1 {
		t.Fatalf("plan units = %v", units)
	}
	return plan["id"].(string), units[0], plan
}

// writeRealization writes the realization report the authoring target skill
// hands to gates record.
func writeRealization(t *testing.T, report map[string]any) string {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "realization.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// goFirstDeliveryClaims is source provenance for both version-sensitive
// claims of the go-a1-first-delivery unit.
func goFirstDeliveryClaims() []any {
	source := func(url string) []any {
		return []any{map[string]any{"url": url, "version": "go1.27", "verifiedAt": "2026-10-07"}}
	}
	return []any{
		map[string]any{"domain": "go", "competency": "go.backend.http-stdlib", "sources": source("https://pkg.go.dev/net/http@go1.27")},
		map[string]any{"domain": "go", "competency": "go.language.packages-modules", "sources": source("https://go.dev/ref/mod")},
	}
}

func recordGates(t *testing.T, root, target, curriculum, planID, result string, extra ...string) platformRun {
	t.Helper()
	args := []string{"gates", "record", "--adapter", "pylearn", "--target", target, "--curriculum", curriculum,
		"--plan", planID, "--result", result, "--workspace", root}
	return runPlatform(t, App{}, append(args, extra...)...)
}

func requireRecorded(t *testing.T, run platformRun) map[string]any {
	t.Helper()
	if run.code != 0 {
		t.Fatalf("gates record exit = %d, stderr = %s, stdout = %s", run.code, run.stderr, run.stdout)
	}
	return decodeJSON(t, run.stdout)
}

// unitOutcome returns the recorded outcome for a unit spec.
func unitOutcome(t *testing.T, body map[string]any, unitID string) map[string]any {
	t.Helper()
	for _, raw := range body["record"].(map[string]any)["units"].([]any) {
		unit := raw.(map[string]any)
		if unit["unit"].(map[string]any)["id"] == unitID {
			return unit
		}
	}
	t.Fatalf("no outcome for %s in %s", unitID, body)
	return nil
}

func reasonCodes(unit map[string]any) []string {
	codes := []string{}
	if reasons, ok := unit["reasons"].([]any); ok {
		for _, raw := range reasons {
			codes = append(codes, raw.(map[string]any)["code"].(string))
		}
	}
	return codes
}

func TestPlatformGatesRecordRealizesUnitFromPublishablePyLearnResult(t *testing.T) {
	root := newLearnerWorkspace(t, "ada")
	planID, unitID, _ := planPyLearnGo(t, root)
	realization := writeRealization(t, map[string]any{
		"schemaVersion": 1,
		"plan":          planID,
		"units": []any{map[string]any{
			"unit":   unitID,
			"items":  []any{"go-a1-first-delivery", "go-a1-first-delivery#quiz:go-module-basics", "go-a1-first-delivery#quiz:go-module-basics/q1"},
			"claims": goFirstDeliveryClaims(),
		}},
	})

	body := requireRecorded(t, recordGates(t, root, "go", pylearnGoCurriculumFixture, planID, pylearnGoGateFixture, "--realization", realization))
	if body["status"] != "recorded" {
		t.Fatalf("status = %v", body["status"])
	}
	record := body["record"].(map[string]any)
	if !strings.HasPrefix(record["id"].(string), "pgr_") || record["plan"].(map[string]any)["id"] != planID || record["learnerId"] != "ada" {
		t.Fatalf("record = %#v", record)
	}
	if want := "authoring-plans/" + planID + "/gate-results/" + record["id"].(string) + ".json"; body["path"] != want {
		t.Fatalf("path = %v, want %s", body["path"], want)
	}
	// The PyLearn result is recorded verbatim.
	var original map[string]any
	if err := json.Unmarshal([]byte(mustRead(t, pylearnGoGateFixture)), &original); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(record["result"], original) {
		t.Fatalf("recorded result differs from the PyLearn gate JSON:\n%#v", record["result"])
	}
	stored := readSpecFile(t, root, body["path"].(string))
	if !reflect.DeepEqual(stored, record) {
		t.Fatalf("stored record differs from output")
	}

	unit := unitOutcome(t, body, unitID)
	if unit["status"] != "realized" || len(reasonCodes(unit)) != 0 {
		t.Fatalf("unit outcome = %#v", unit)
	}
	realizations := body["realizations"].([]any)
	if len(realizations) != 1 {
		t.Fatalf("realizations = %#v", realizations)
	}
	link := realizations[0].(map[string]any)
	if link["id"] != unit["realization"] || link["path"] != "specifications/realizations/"+link["id"].(string)+".json" {
		t.Fatalf("realization link = %#v, unit = %#v", link, unit)
	}
	item := func(id string) any { return map[string]any{"platform": "pylearn", "target": "go", "item": id} }
	if !reflect.DeepEqual(link["root"], item("go-a1-first-delivery")) ||
		!reflect.DeepEqual(link["items"], []any{item("go-a1-first-delivery"), item("go-a1-first-delivery#quiz:go-module-basics"), item("go-a1-first-delivery#quiz:go-module-basics/q1")}) ||
		link["unit"].(map[string]any)["id"] != unitID || link["gateRecord"] != record["id"] || link["plan"] != planID {
		t.Fatalf("realization link = %#v", link)
	}
	claims := link["claims"].([]any)
	if len(claims) != 2 || claims[0].(map[string]any)["competency"] != "go.backend.http-stdlib" ||
		claims[0].(map[string]any)["versionSensitive"] != true ||
		!reflect.DeepEqual(claims[0].(map[string]any)["sources"], goFirstDeliveryClaims()[0].(map[string]any)["sources"]) {
		t.Fatalf("link claims = %#v", claims)
	}
	stripped := map[string]any{}
	for key, value := range link {
		if key != "path" {
			stripped[key] = value
		}
	}
	if storedLink := readSpecFile(t, root, link["path"].(string)); !reflect.DeepEqual(storedLink, stripped) {
		t.Fatalf("stored link = %#v, output = %#v", storedLink, stripped)
	}
	requireValidWorkspace(t, root)

	// Recording the same result again records nothing new.
	again := requireRecorded(t, recordGates(t, root, "go", pylearnGoCurriculumFixture, planID, pylearnGoGateFixture, "--realization", realization))
	if again["status"] != "already-recorded" || again["record"].(map[string]any)["id"] != record["id"] {
		t.Fatalf("re-record = %#v", again)
	}
	if files := filesIn(t, filepath.Join(root, "specifications", "realizations")); len(files) != 1 {
		t.Fatalf("realization files = %v", files)
	}
}

func TestPlatformGatesRecordLeavesUnitUnrealizedForNonPublishableResult(t *testing.T) {
	root := adaLearnerState(t)
	_, plan := authoringOf(t, requirePlan(t, planTarget(t, root)))
	planID, units := plan["id"].(string), planUnitIDs(plan)
	if len(units) != 1 {
		t.Fatalf("plan units = %v", units)
	}

	// The real PyLearn validator output for go-alp before the course existed.
	body := requireRecorded(t, recordGates(t, root, "go-alp", goALPCurriculumFixture, planID, pylearnGoALPFailedGate))
	unit := unitOutcome(t, body, units[0])
	if unit["status"] != "unrealized" || unit["realization"] != nil {
		t.Fatalf("unit outcome = %#v", unit)
	}
	if want := []string{"not-publishable", "realization-not-reported"}; !reflect.DeepEqual(reasonCodes(unit), want) {
		t.Fatalf("reasons = %v, want %v", reasonCodes(unit), want)
	}
	message := unit["reasons"].([]any)[0].(map[string]any)["message"].(string)
	if message != `platform pylearn reported the result for target "go-alp" not publishable (failed gates: gate:reels, lint:lessons, gate:alp-mapping)` {
		t.Fatalf("not-publishable reason = %q", message)
	}
	if len(body["realizations"].([]any)) != 0 || len(filesIn(t, filepath.Join(root, "specifications", "realizations"))) != 0 {
		t.Fatalf("a non-publishable result realized something: %s", body["realizations"])
	}
	// The result is still recorded with the plan: it is the audit trail.
	if record := readSpecFile(t, root, body["path"].(string)); record["result"].(map[string]any)["publishable"] != false {
		t.Fatalf("stored record = %#v", record)
	}
	requireValidWorkspace(t, root)

	// A report of realized items does not make an unpublishable result realize.
	realization := writeRealization(t, map[string]any{
		"schemaVersion": 1, "plan": planID,
		"units": []any{map[string]any{"unit": units[0], "items": []any{"go-alp-a2-goroutines"}, "claims": []any{
			map[string]any{"domain": "go", "competency": "go.runtime.scheduler", "sources": []any{map[string]any{"url": "https://go.dev/doc/go1.27", "version": "go1.27", "verifiedAt": "2026-10-07"}}},
		}}},
	})
	again := requireRecorded(t, recordGates(t, root, "go-alp", goALPCurriculumFixture, planID, pylearnGoALPFailedGate, "--realization", realization))
	if unit := unitOutcome(t, again, units[0]); unit["status"] != "unrealized" || !reflect.DeepEqual(reasonCodes(unit), []string{"not-publishable"}) {
		t.Fatalf("unit outcome with a report = %#v", unit)
	}
}

func TestPlatformGatesRecordRefusesRealizationWithoutRequiredProvenance(t *testing.T) {
	root := newLearnerWorkspace(t, "ada")
	planID, unitID, _ := planPyLearnGo(t, root)
	items := []any{"go-a1-first-delivery"}
	record := func(claims []any) map[string]any {
		t.Helper()
		realization := writeRealization(t, map[string]any{
			"schemaVersion": 1, "plan": planID,
			"units": []any{map[string]any{"unit": unitID, "items": items, "claims": claims}},
		})
		return requireRecorded(t, recordGates(t, root, "go", pylearnGoCurriculumFixture, planID, pylearnGoGateFixture, "--realization", realization))
	}

	// No provenance at all: both version-sensitive claims are missing it.
	unit := unitOutcome(t, record([]any{}), unitID)
	if unit["status"] != "unrealized" || !reflect.DeepEqual(reasonCodes(unit), []string{"provenance-missing", "provenance-missing"}) {
		t.Fatalf("unit outcome = %#v", unit)
	}
	first := unit["reasons"].([]any)[0].(map[string]any)
	if first["competency"] != "go.backend.http-stdlib" || first["domain"] != "go" ||
		first["message"] != "claim go.backend.http-stdlib (version-sensitive-language-runtime) requires source provenance: re-verify it at authoring time and record the source URL, version and date" {
		t.Fatalf("reason = %#v", first)
	}

	// A version-sensitive claim's source must name the version verified.
	claims := goFirstDeliveryClaims()
	unversioned := claims[1].(map[string]any)
	unversioned["sources"] = []any{map[string]any{"url": "https://go.dev/ref/mod", "verifiedAt": "2026-10-07"}}
	unit = unitOutcome(t, record(claims), unitID)
	if unit["status"] != "unrealized" || !reflect.DeepEqual(reasonCodes(unit), []string{"provenance-missing"}) ||
		unit["reasons"].([]any)[0].(map[string]any)["competency"] != "go.language.packages-modules" {
		t.Fatalf("unit outcome = %#v", unit)
	}
	if files := filesIn(t, filepath.Join(root, "specifications", "realizations")); len(files) != 0 {
		t.Fatalf("realization links written without provenance: %v", files)
	}

	// Provenance for a claim the unit does not declare is refused outright.
	stray := append(goFirstDeliveryClaims(), map[string]any{"domain": "go", "competency": "go.runtime.scheduler",
		"sources": []any{map[string]any{"url": "https://go.dev/doc/go1.27", "version": "go1.27", "verifiedAt": "2026-10-07"}}})
	realization := writeRealization(t, map[string]any{
		"schemaVersion": 1, "plan": planID,
		"units": []any{map[string]any{"unit": unitID, "items": items, "claims": stray}},
	})
	refused := recordGates(t, root, "go", pylearnGoCurriculumFixture, planID, pylearnGoGateFixture, "--realization", realization)
	if refused.code != 1 || errorField(t, refused)["code"] != "invalid-realization-report" {
		t.Fatalf("stray claim exit = %d, stdout = %s", refused.code, refused.stdout)
	}
	requireValidWorkspace(t, root)
}

func TestPlatformGatesRecordLinksOnlyDeclaredStableItems(t *testing.T) {
	root := newLearnerWorkspace(t, "ada")
	planID, unitID, _ := planPyLearnGo(t, root)
	cases := map[string]struct {
		items []any
		code  string
		want  string
	}{
		"positional scene ID": {[]any{"go-a1-first-delivery", "go-a1-first-delivery#scene-3"}, "invalid-realization-link",
			`"go-a1-first-delivery#scene-3" is not a declared-stable item of target "go" in the curriculum export`},
		"heading-derived section slug": {[]any{"go-a1-first-delivery", "go-a1-first-delivery#why-modules"}, "invalid-realization-link",
			`"go-a1-first-delivery#why-modules" is not a declared-stable item`},
		"item of another target": {[]any{"go-alp-a2-goroutines"}, "invalid-realization-link",
			`"go-alp-a2-goroutines" is not a declared-stable item of target "go"`},
		"no unit-level item": {[]any{"go-a1-first-delivery#init"}, "invalid-realization-link",
			"no reported item is a unit-level item"},
		"two unit-level items": {[]any{"go-a1-first-delivery", "go-a2-model-the-domain"}, "invalid-realization-link",
			`items "go-a1-first-delivery" and "go-a2-model-the-domain" are both unit-level items`},
		"another unit's descendant": {[]any{"go-a1-first-delivery", "go-a2-model-the-domain#methods"}, "invalid-realization-link",
			`item "go-a2-model-the-domain#methods" does not descend from unit-level item "go-a1-first-delivery"`},
		"not the existing unit's item": {[]any{"go-a2-model-the-domain"}, "invalid-realization-link",
			`unit-level item "go-a2-model-the-domain" is not the unit's platform item "go-a1-first-delivery"`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			realization := writeRealization(t, map[string]any{
				"schemaVersion": 1, "plan": planID,
				"units": []any{map[string]any{"unit": unitID, "items": tc.items, "claims": goFirstDeliveryClaims()}},
			})
			run := recordGates(t, root, "go", pylearnGoCurriculumFixture, planID, pylearnGoGateFixture, "--realization", realization)
			failure := errorField(t, run)
			if run.code != 1 || failure["code"] != tc.code || !strings.Contains(failure["message"].(string), tc.want) {
				t.Fatalf("exit = %d, error = %#v, want %s containing %q", run.code, failure, tc.code, tc.want)
			}
		})
	}
	// A refused report records nothing: no gate record, no link.
	if files := filesIn(t, filepath.Join(root, "authoring-plans", planID, "gate-results")); len(files) != 0 {
		t.Fatalf("gate records written for refused reports: %v", files)
	}
	if files := filesIn(t, filepath.Join(root, "specifications", "realizations")); len(files) != 0 {
		t.Fatalf("links written for refused reports: %v", files)
	}
}

func TestPlatformGatesRecordRefusesResultsAndPlansThatDoNotMatch(t *testing.T) {
	root := newLearnerWorkspace(t, "ada")
	planID, _, _ := planPyLearnGo(t, root)
	cases := map[string]struct {
		args []string
		code string
	}{
		// The go-alp result cannot be attached to a go plan.
		"result for another target": {[]string{"--plan", planID, "--result", pylearnGoALPFailedGate}, "invalid-gate-result"},
		"unknown plan":              {[]string{"--plan", "apl_000000000000000000000000", "--result", pylearnGoGateFixture}, "unknown-authoring-plan"},
		"not a gate result":         {[]string{"--plan", planID, "--result", pylearnGoCurriculumFixture}, "invalid-gate-result"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"gates", "record", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnGoCurriculumFixture, "--workspace", root}, tc.args...)
			run := runPlatform(t, App{}, args...)
			if run.code != 1 || errorField(t, run)["code"] != tc.code {
				t.Fatalf("exit = %d, stdout = %s", run.code, run.stdout)
			}
		})
	}
	usage := runPlatform(t, App{}, "gates", "record", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnGoCurriculumFixture, "--workspace", root)
	if usage.code != 2 || errorField(t, usage)["code"] != "usage" {
		t.Fatalf("missing --plan/--result exit = %d", usage.code)
	}
}

func TestPlatformGatesRecordRequiresThePlatformValidatorCapability(t *testing.T) {
	registry := platform.NewRegistry(noAuthoringAdapter{})
	run := runPlatform(t, App{Platforms: &registry}, "gates", "record", "--adapter", "read-only", "--target", "go",
		"--curriculum", pylearnGoCurriculumFixture, "--plan", "apl_000000000000000000000000", "--result", pylearnGoGateFixture)
	if failure := errorField(t, run); run.code != 1 || failure["code"] != "capability-not-declared" || failure["capability"] != "platform-validator" {
		t.Fatalf("exit = %d, error = %#v", run.code, failure)
	}
}

// publishableGoALPResult is the real publishable PyLearn result re-targeted at
// go-alp, standing in for `validate:platform --target go-alp` once the
// authoring skill has realized go-alp units: same gates, provenance and shape.
func publishableGoALPResult(t *testing.T) string {
	t.Helper()
	text := strings.ReplaceAll(mustRead(t, pylearnGoGateFixture), "--course go ", "--course go-alp ")
	var result map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatal(err)
	}
	result["target"] = "go-alp"
	for _, raw := range result["gates"].([]any) {
		for _, diagnostic := range raw.(map[string]any)["diagnostics"].([]any) {
			if item, ok := diagnostic.(map[string]any)["item"].(map[string]any); ok {
				item["target"] = "go-alp"
			}
		}
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "gate-result.go-alp.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// claimsWithProvenance records a source for every claim of a planned unit.
func claimsWithProvenance(unit map[string]any) []any {
	claims := []any{}
	for _, raw := range unit["teachingIntent"].(map[string]any)["claims"].([]any) {
		claim := raw.(map[string]any)
		claims = append(claims, map[string]any{"domain": claim["domain"], "competency": claim["competency"],
			"sources": []any{map[string]any{"url": "https://go.dev/doc/go1.27", "version": "go1.27", "verifiedAt": "2026-10-07"}}})
	}
	return claims
}

// realizedGoALPFixtures is the go-alp target after the authoring skill
// realized the skeleton: one lesson (with an assessing quiz) per proposed unit.
func realizedGoALPFixtures(t *testing.T) (curriculum, mapping, constraints string) {
	t.Helper()
	dir := t.TempDir()
	curriculum = filepath.Join(dir, "curriculum.json")
	mapping = filepath.Join(dir, "go-alp.mapping.yaml")
	constraints = filepath.Join(dir, "go-alp.constraints.yaml")
	for path, content := range map[string]string{
		curriculum: `{"schemaVersion": 1, "platform": "pylearn", "targets": [{"id": "go-alp", "title": "Go (ALP)", "contentHash": "` + opaqueHash + `",
  "phases": [{"id": "S1", "title": "Stage 1"}, {"id": "S2", "title": "Stage 2"}],
  "items": [
    {"id": "go-alp-s1-goroutines", "kind": "lesson", "phase": "S1", "title": "Goroutine lifecycle and ownership"},
    {"id": "go-alp-s1-goroutines#quiz:lifetimes", "kind": "quiz", "parent": "go-alp-s1-goroutines"},
    {"id": "go-alp-s2-channels", "kind": "lesson", "phase": "S2", "title": "Channels and ownership"},
    {"id": "go-alp-s2-channels#quiz:ownership", "kind": "quiz", "parent": "go-alp-s2-channels"}
  ]}]}`,
		mapping: `schemaVersion: 2
platform: pylearn
target: go-alp
packs:
  - domain: go
    packVersion: ">=0.1.0 <0.2.0"
entries:
  - item: go-alp-s1-goroutines
    competencies: [{id: go.concurrency.goroutines, role: teaches}]
  - item: go-alp-s1-goroutines#quiz:lifetimes
    competencies: [{id: go.concurrency.goroutines, role: assesses}]
  - item: go-alp-s2-channels
    competencies: [{id: go.concurrency.channels, role: teaches}]
  - item: go-alp-s2-channels#quiz:ownership
    competencies: [{id: go.concurrency.channels, role: assesses}]
`,
		constraints: "schemaVersion: 1\nplatform: pylearn\ntarget: go-alp\nallowedModes: [skip, challenge, skim, full]\ngoal: [go.concurrency.channels]\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return curriculum, mapping, constraints
}

func TestPlatformRealizedUnitsKeepTheirSpecificationIDsAndChangeOnlyOnEvidence(t *testing.T) {
	root := newLearnerWorkspace(t, "grace")
	seedAssessment(t, root, "go.runtime.scheduler", "strong", "high", "2026-10-01T09:00:00Z")
	skeleton := requirePlan(t, planNewTarget(t, root, "--intent", "target-skeleton"))
	_, plan := authoringOf(t, skeleton)
	planID := plan["id"].(string)
	proposed := specsOf(t, skeleton).Units
	goroutines, channels := proposed[0], proposed[1]
	if goroutines.Title != "Goroutine lifecycle and ownership" || channels.Title != "Channels and ownership" {
		t.Fatalf("proposed units = %#v", proposed)
	}

	// The authoring skill realized the skeleton; the post-authoring
	// curriculum export lists the new lessons.
	curriculum, mapping, constraints := realizedGoALPFixtures(t)
	public := plan["public"].(map[string]any)["units"].([]any)
	report := map[string]any{"schemaVersion": 1, "plan": planID, "units": []any{
		map[string]any{"unit": goroutines.ID, "items": []any{"go-alp-s1-goroutines", "go-alp-s1-goroutines#quiz:lifetimes"}, "claims": claimsWithProvenance(public[0].(map[string]any))},
		map[string]any{"unit": channels.ID, "items": []any{"go-alp-s2-channels", "go-alp-s2-channels#quiz:ownership"}, "claims": claimsWithProvenance(public[1].(map[string]any))},
	}}
	recorded := requireRecorded(t, recordGates(t, root, "go-alp", curriculum, planID, publishableGoALPResult(t), "--realization", writeRealization(t, report)))
	for _, id := range []string{goroutines.ID, channels.ID} {
		if unit := unitOutcome(t, recorded, id); unit["status"] != "realized" {
			t.Fatalf("unit %s = %#v", id, unit)
		}
	}

	replan := func() map[string]any {
		t.Helper()
		return requirePlan(t, runPlatform(t, App{}, "plan", "--adapter", "pylearn", "--target", "go-alp", "--curriculum", curriculum,
			"--mapping", mapping, "--constraints", constraints, "--workspace", root))
	}
	// The realized lessons are the proposed units: same specification IDs,
	// not new item-derived ones. The quizzes add an assessing role (a
	// material change), but no new evidence motivates it, so the realized
	// versions are held.
	body := replan()
	specs := specsOf(t, body)
	if len(specs.Units) != 2 {
		t.Fatalf("units after realization = %#v", specs.Units)
	}
	for i, want := range []struct {
		spec specRef
		item string
	}{{goroutines, "go-alp-s1-goroutines"}, {channels, "go-alp-s2-channels"}} {
		got := specs.Units[i]
		if got.ID != want.spec.ID || got.Version != 1 || got.Status != "held" || got.PlatformItem == nil || (*got.PlatformItem)["item"] != want.item {
			t.Fatalf("unit %d after realization = %#v, want %s v1 held as %s", i, got, want.spec.ID, want.item)
		}
	}
	held := body["specifications"].(map[string]any)["units"].([]any)[0].(map[string]any)
	if !reflect.DeepEqual(held["heldFields"], []any{"competencies"}) || held["realized"] != true {
		t.Fatalf("held unit entry = %#v", held)
	}
	// Realized units are not authored again.
	if authoring, plan := authoringOf(t, body); authoring.Status != "nothing-to-author" || plan != nil {
		t.Fatalf("authoring after realization = %#v, plan = %#v", authoring, plan)
	}
	if files := filesIn(t, filepath.Join(root, "specifications", "units")); len(files) != 2 {
		t.Fatalf("unit spec versions = %v", files)
	}

	// New evidence that changes the unit materially does produce a new
	// version, under the same ID, now placed at its realized item.
	seedAssessment(t, root, "go.concurrency.goroutines", "functional", "medium", "2026-10-02T09:00:00Z")
	revised := unitSpecFor(t, specsOf(t, replan()), "go-alp-s1-goroutines")
	if revised.ID != goroutines.ID || revised.Version != 2 || revised.Status != "revised" {
		t.Fatalf("revised unit = %#v", revised)
	}
	spec := readSpecFile(t, root, revised.Path)
	change := spec["change"].(map[string]any)
	if change["evidenceLinked"] != true || spec["unit"].(map[string]any)["platformItem"].(map[string]any)["item"] != "go-alp-s1-goroutines" {
		t.Fatalf("revised spec change = %#v, unit = %#v", change, spec["unit"])
	}
	requireValidWorkspace(t, root)
}

func TestPlatformGatesRecordRefusesAnItemRealizingTwoUnits(t *testing.T) {
	root := newLearnerWorkspace(t, "grace")
	seedAssessment(t, root, "go.runtime.scheduler", "strong", "high", "2026-10-01T09:00:00Z")
	skeleton := requirePlan(t, planNewTarget(t, root, "--intent", "target-skeleton"))
	_, plan := authoringOf(t, skeleton)
	planID := plan["id"].(string)
	units := specsOf(t, skeleton).Units
	public := plan["public"].(map[string]any)["units"].([]any)
	curriculum, _, _ := realizedGoALPFixtures(t)
	result := publishableGoALPResult(t)
	report := func(entries ...any) string {
		return writeRealization(t, map[string]any{"schemaVersion": 1, "plan": planID, "units": entries})
	}
	entry := func(i int, items ...any) any {
		return map[string]any{"unit": units[i].ID, "items": items, "claims": claimsWithProvenance(public[i].(map[string]any))}
	}

	both := recordGates(t, root, "go-alp", curriculum, planID, result, "--realization", report(entry(0, "go-alp-s1-goroutines"), entry(1, "go-alp-s1-goroutines")))
	if both.code != 1 || errorField(t, both)["code"] != "realization-conflict" {
		t.Fatalf("same item for two units exit = %d, stdout = %s", both.code, both.stdout)
	}

	requireRecorded(t, recordGates(t, root, "go-alp", curriculum, planID, result, "--realization", report(entry(0, "go-alp-s1-goroutines"))))
	later := recordGates(t, root, "go-alp", curriculum, planID, result, "--realization", report(entry(1, "go-alp-s1-goroutines")))
	if failure := errorField(t, later); later.code != 1 || failure["code"] != "realization-conflict" ||
		!strings.Contains(failure["message"].(string), `item "go-alp-s1-goroutines" already realizes `+units[0].ID) {
		t.Fatalf("item already realizing another unit exit = %d, error = %#v", later.code, failure)
	}
	moved := recordGates(t, root, "go-alp", curriculum, planID, result, "--realization", report(entry(0, "go-alp-s2-channels")))
	if failure := errorField(t, moved); moved.code != 1 || failure["code"] != "realization-conflict" {
		t.Fatalf("realized unit moved to another item exit = %d, error = %#v", moved.code, failure)
	}
}
