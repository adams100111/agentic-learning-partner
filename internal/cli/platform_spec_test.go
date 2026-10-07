package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/platform/pylearn"
)

// specRef is one specification entry of plan output.
type specRef struct {
	ID           string          `json:"id"`
	Version      int             `json:"version"`
	ContentHash  string          `json:"contentHash"`
	Path         string          `json:"path"`
	Status       string          `json:"status"`
	Title        string          `json:"title"`
	Group        string          `json:"group"`
	PlatformItem *map[string]any `json:"platformItem"`
	// Persona is the unit's private persona view, re-derived from the live
	// persona and profile documents on every plan and never stored.
	Persona map[string]any `json:"persona"`
}

type planSpecs struct {
	Curriculum specRef   `json:"curriculum"`
	Units      []specRef `json:"units"`
}

func specsOf(t *testing.T, body map[string]any) planSpecs {
	t.Helper()
	raw, ok := body["specifications"]
	if !ok {
		t.Fatalf("plan output has no specifications: %#v", body)
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var specs planSpecs
	if err := json.Unmarshal(encoded, &specs); err != nil {
		t.Fatal(err)
	}
	return specs
}

// unitSpecFor returns the plan's unit spec entry for a platform item, or for a
// proposed unit (no platform item) titled title.
func unitSpecFor(t *testing.T, specs planSpecs, itemOrTitle string) specRef {
	t.Helper()
	for _, unit := range specs.Units {
		if unit.PlatformItem != nil && (*unit.PlatformItem)["item"] == itemOrTitle {
			return unit
		}
		if unit.PlatformItem == nil && unit.Title == itemOrTitle {
			return unit
		}
	}
	t.Fatalf("no unit spec for %q in %#v", itemOrTitle, specs.Units)
	return specRef{}
}

func readSpecFile(t *testing.T, root, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read spec %s: %v", path, err)
	}
	return decodeJSON(t, data)
}

func TestPlatformPlanWritesVersionedSpecificationsInThePrivateWorkspace(t *testing.T) {
	root := adaLearnerState(t)
	body := requirePlan(t, planTarget(t, root))
	projection := projectionOf(t, body)
	specs := specsOf(t, body)

	curriculum := specs.Curriculum
	if !strings.HasPrefix(curriculum.ID, "cspec_") || curriculum.Version != 1 || curriculum.Status != "created" ||
		curriculum.Path != "specifications/curricula/"+curriculum.ID+"-v1.json" {
		t.Fatalf("curriculum spec = %#v", curriculum)
	}

	// Existing target units in curriculum order, with uncovered prerequisite
	// gaps of the target proposed as new units before the unit needing them.
	var order []string
	for _, unit := range specs.Units {
		if !strings.HasPrefix(unit.ID, "uspec_") || unit.Version != 1 || unit.Status != "created" ||
			unit.Path != "specifications/units/"+unit.ID+"-v1.json" {
			t.Fatalf("unit spec = %#v", unit)
		}
		if unit.PlatformItem != nil {
			order = append(order, (*unit.PlatformItem)["item"].(string))
		} else {
			order = append(order, "proposed:"+unit.Title)
		}
	}
	want := []string{
		"go-alp-a1-context",
		"proposed:Goroutines and scheduler mental model",
		"go-alp-a2-goroutines",
		"go-alp-a3-generics",
		"proposed:Channels and ownership",
		"proposed:Mutexes, WaitGroups, Once, atomics",
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("unit order = %v, want %v", order, want)
	}

	// Every spec records its full provenance.
	goroutines := readSpecFile(t, root, unitSpecFor(t, specs, "go-alp-a2-goroutines").Path)
	provenance := goroutines["provenance"].(map[string]any)
	if revision, _ := provenance["learnerStateRevision"].(string); !regexp.MustCompile(`^lsr_[0-9a-f]{24}$`).MatchString(revision) {
		t.Fatalf("learnerStateRevision = %#v", provenance["learnerStateRevision"])
	}
	if provenance["projectionRevision"] != projection["revision"] {
		t.Fatalf("projectionRevision = %v, projection %v", provenance["projectionRevision"], projection["revision"])
	}
	inputs := projection["inputs"].(map[string]any)
	if !reflect.DeepEqual(provenance["curriculum"], inputs["curriculum"]) || !reflect.DeepEqual(provenance["mapping"], inputs["mapping"]) ||
		!reflect.DeepEqual(provenance["packs"], []any{map[string]any{"domain": "go", "version": "0.1.0"}}) {
		t.Fatalf("target snapshot provenance = %#v", provenance)
	}
	sources := provenance["sources"].([]any)
	if !containsJSON(sources, map[string]any{"domain": "go", "key": "officialReleaseNotes", "value": "https://go.dev/doc/go1.27"}) {
		t.Fatalf("sources = %#v", sources)
	}
	if goroutines["learnerId"] != "ada" || goroutines["authoringIntent"] != "none" || goroutines["id"] != unitSpecFor(t, specs, "go-alp-a2-goroutines").ID {
		t.Fatalf("unit spec = %#v", goroutines)
	}
	if hash, _ := goroutines["contentHash"].(string); hash != unitSpecFor(t, specs, "go-alp-a2-goroutines").ContentHash ||
		!regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(hash) {
		t.Fatalf("contentHash = %#v", goroutines["contentHash"])
	}

	teaching := goroutines["teaching"].(map[string]any)
	if teaching["title"] != "Goroutine Lifetimes" {
		t.Fatalf("title = %#v", teaching["title"])
	}
	wantCompetencies := []any{
		map[string]any{"domain": "go", "id": "go.concurrency.goroutines", "name": "Goroutine lifecycle and ownership", "roles": []any{"teaches", "assesses"}},
		map[string]any{"domain": "go", "id": "go.concurrency.races-deadlocks-leaks", "name": "Race, deadlock, and leak diagnosis", "roles": []any{"reinforces"}},
	}
	if !reflect.DeepEqual(teaching["competencies"], wantCompetencies) {
		t.Fatalf("competencies = %#v", teaching["competencies"])
	}
	if !reflect.DeepEqual(teaching["objectives"], []any{"Start goroutines with explicit ownership, termination, error handling, and resource lifetime."}) {
		t.Fatalf("objectives = %#v", teaching["objectives"])
	}
	if !reflect.DeepEqual(teaching["prerequisites"], []any{map[string]any{"domain": "go", "id": "go.runtime.scheduler", "name": "Goroutines and scheduler mental model"}}) {
		t.Fatalf("prerequisites = %#v", teaching["prerequisites"])
	}
	scheduler := unitSpecFor(t, specs, "Goroutines and scheduler mental model")
	if !reflect.DeepEqual(teaching["dependsOn"], []any{scheduler.ID}) {
		t.Fatalf("dependsOn = %#v, want %s", teaching["dependsOn"], scheduler.ID)
	}
	if !reflect.DeepEqual(teaching["requiredEvidence"], []any{map[string]any{"domain": "go", "competency": "go.concurrency.goroutines", "role": "assesses", "minimumLevel": "functional"}}) {
		t.Fatalf("requiredEvidence = %#v", teaching["requiredEvidence"])
	}
	adaptation := goroutines["adaptation"].(map[string]any)
	if adaptation["mode"] != "skim" || adaptation["proposedMode"] != "skim" {
		t.Fatalf("adaptation = %#v", adaptation)
	}

	// Version-sensitive and source-required claims come from the pack's
	// freshness classes.
	schedulerSpec := readSpecFile(t, root, scheduler.Path)
	wantClaims := []any{map[string]any{"domain": "go", "competency": "go.runtime.scheduler", "freshnessClass": "version-sensitive-language-runtime", "versionSensitive": true, "sourceRequired": true}}
	if !reflect.DeepEqual(schedulerSpec["teaching"].(map[string]any)["claims"], wantClaims) {
		t.Fatalf("claims = %#v", schedulerSpec["teaching"].(map[string]any)["claims"])
	}
	if schedulerSpec["authoringIntent"] != "unit" || schedulerSpec["unit"].(map[string]any)["platformItem"] != nil {
		t.Fatalf("proposed unit spec = %#v", schedulerSpec)
	}

	curriculumSpec := readSpecFile(t, root, curriculum.Path)
	var unitIDs []any
	for _, unit := range specs.Units {
		unitIDs = append(unitIDs, unit.ID)
	}
	var listed []any
	for _, raw := range curriculumSpec["units"].([]any) {
		listed = append(listed, raw.(map[string]any)["id"])
	}
	if !reflect.DeepEqual(listed, unitIDs) || curriculumSpec["contentHash"] != curriculum.ContentHash {
		t.Fatalf("curriculum units = %v, want %v", listed, unitIDs)
	}
	var groups []string
	for _, raw := range curriculumSpec["groups"].([]any) {
		group := raw.(map[string]any)
		groups = append(groups, group["id"].(string)+":"+itemsCount(group["units"]))
	}
	if want := []string{"A:3", "stage-1:3"}; !reflect.DeepEqual(groups, want) {
		t.Fatalf("groups = %v, want %v", groups, want)
	}

	// Specs are platform-neutral: no platform-native rendering concepts.
	for _, path := range filesUnder(t, root, "specifications") {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if native := regexp.MustCompile(`(?i)reel|mdx|scene`).Find(data); native != nil {
			t.Fatalf("spec %s mentions platform-native concept %q", path, native)
		}
	}
	requireValidWorkspace(t, root)
}

func itemsCount(value any) string {
	list, _ := value.([]any)
	return strconv.Itoa(len(list))
}

func containsJSON(values []any, want any) bool {
	for _, value := range values {
		if reflect.DeepEqual(value, want) {
			return true
		}
	}
	return false
}

// filesUnder lists every regular file below root/dir.
func filesUnder(t *testing.T, root, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestPlatformPlanVersionsSpecificationsOnlyOnMaterialChange(t *testing.T) {
	root := adaLearnerState(t)
	first := specsOf(t, requirePlan(t, planTarget(t, root)))
	goroutinesV1 := unitSpecFor(t, first, "go-alp-a2-goroutines")
	v1Bytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(goroutinesV1.Path)))
	if err != nil {
		t.Fatal(err)
	}
	filesBefore := filesUnder(t, root, "specifications")

	// A recompute with new evidence that changes nothing material (the
	// learner is still functional on goroutines) creates no new version.
	seedAssessment(t, root, "go.concurrency.goroutines", "functional", "high", "2026-10-02T09:00:00Z")
	body := requirePlan(t, planTarget(t, root))
	second := specsOf(t, body)
	if projectionOf(t, body)["revision"] == readSpecFile(t, root, goroutinesV1.Path)["provenance"].(map[string]any)["projectionRevision"] {
		t.Fatal("the new assessment must change the projection, or this recompute proves nothing")
	}
	for _, unit := range append(second.Units, second.Curriculum) {
		if unit.Status != "unchanged" || unit.Version != 1 {
			t.Fatalf("non-material recompute changed %s: %#v", unit.ID, unit)
		}
	}
	if got := filesUnder(t, root, "specifications"); !reflect.DeepEqual(got, filesBefore) {
		t.Fatalf("non-material recompute wrote specifications: %v", got)
	}

	// Evidence that makes goroutines rusty changes the unit's adaptation mode
	// and adds a misconception: a new, superseding version with provenance.
	seedAssessment(t, root, "go.concurrency.goroutines", "rusty", "high", "2026-10-03T09:00:00Z", "unbounded goroutine fan-out without cancellation")
	body = requirePlan(t, planTarget(t, root))
	third := specsOf(t, body)
	revised := unitSpecFor(t, third, "go-alp-a2-goroutines")
	if revised.ID != goroutinesV1.ID || revised.Version != 2 || revised.Status != "revised" ||
		revised.Path != "specifications/units/"+revised.ID+"-v2.json" || revised.ContentHash == goroutinesV1.ContentHash {
		t.Fatalf("revised unit spec = %#v", revised)
	}
	v2 := readSpecFile(t, root, revised.Path)
	if !reflect.DeepEqual(v2["supersedes"], map[string]any{"version": float64(1), "contentHash": goroutinesV1.ContentHash}) {
		t.Fatalf("supersedes = %#v", v2["supersedes"])
	}
	if !reflect.DeepEqual(v2["change"], map[string]any{"materialFields": []any{"adaptationMode", "misconceptions"}, "evidenceLinked": true}) {
		t.Fatalf("change = %#v", v2["change"])
	}
	adaptation := v2["adaptation"].(map[string]any)
	misconceptions := adaptation["misconceptions"].([]any)
	if adaptation["mode"] != "full" || len(misconceptions) != 1 ||
		misconceptions[0].(map[string]any)["gap"] != "unbounded goroutine fan-out without cancellation" {
		t.Fatalf("adaptation = %#v", adaptation)
	}
	if v2["provenance"].(map[string]any)["projectionRevision"] != projectionOf(t, body)["revision"] {
		t.Fatalf("v2 provenance = %#v", v2["provenance"])
	}

	// Earlier versions are immutable; unaffected units keep their version;
	// the curriculum is re-versioned to reference the new unit version.
	after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(goroutinesV1.Path)))
	if err != nil || string(after) != string(v1Bytes) {
		t.Fatalf("version 1 changed or vanished: %v", err)
	}
	if context := unitSpecFor(t, third, "go-alp-a1-context"); context.Status != "unchanged" || context.Version != 1 {
		t.Fatalf("unaffected unit = %#v", context)
	}
	if third.Curriculum.Version != 2 || third.Curriculum.Status != "revised" {
		t.Fatalf("curriculum = %#v", third.Curriculum)
	}
	curriculum := readSpecFile(t, root, third.Curriculum.Path)
	if !reflect.DeepEqual(curriculum["change"], map[string]any{"materialFields": []any{"units"}, "evidenceLinked": true}) {
		t.Fatalf("curriculum change = %#v", curriculum["change"])
	}
	requireValidWorkspace(t, root)
}

func TestPlatformPlanRefusesModifiedSpecificationVersions(t *testing.T) {
	root := adaLearnerState(t)
	specs := specsOf(t, requirePlan(t, planTarget(t, root)))
	path := filepath.Join(root, filepath.FromSlash(unitSpecFor(t, specs, "go-alp-a2-goroutines").Path))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(data), `"mode": "skim"`, `"mode": "skip"`, 1)
	if tampered == string(data) {
		t.Fatal("fixture did not contain the adaptation mode")
	}
	if err := os.WriteFile(path, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}
	result := planTarget(t, root)
	if result.code != 1 || errorField(t, result)["code"] != "specification-integrity" {
		t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
	}
}

type authoringView struct {
	Status    string `json:"status"`
	Intent    string `json:"intent"`
	Selection string `json:"selection"`
	Reason    string `json:"reason"`
}

func authoringOf(t *testing.T, body map[string]any) (authoringView, map[string]any) {
	t.Helper()
	encoded, err := json.Marshal(body["authoring"])
	if err != nil {
		t.Fatal(err)
	}
	var view authoringView
	if err := json.Unmarshal(encoded, &view); err != nil {
		t.Fatal(err)
	}
	plan, _ := body["authoringPlan"].(map[string]any)
	return view, plan
}

func planUnitIDs(plan map[string]any) []string {
	var ids []string
	for _, raw := range plan["units"].([]any) {
		ids = append(ids, raw.(map[string]any)["id"].(string))
	}
	return ids
}

func TestPlatformPlanDefaultsToTheSmallestJustifiedAuthoringIntent(t *testing.T) {
	root := adaLearnerState(t)
	body := requirePlan(t, planTarget(t, root))
	specs := specsOf(t, body)
	authoring, plan := authoringOf(t, body)

	// No existing unit lacks the evidence it requires, so the smallest
	// justified intent is authoring the first proposed unit in sequence.
	scheduler := unitSpecFor(t, specs, "Goroutines and scheduler mental model")
	if authoring.Status != "planned" || authoring.Intent != "unit" || authoring.Selection != "default" || plan == nil {
		t.Fatalf("authoring = %#v, plan = %#v", authoring, plan)
	}
	if !regexp.MustCompile(`^apl_[0-9a-f]{24}$`).MatchString(plan["id"].(string)) || plan["status"] != "created" ||
		plan["path"] != "authoring-plans/"+plan["id"].(string)+".json" || plan["intent"] != "unit" {
		t.Fatalf("authoring plan = %#v", plan)
	}
	if !reflect.DeepEqual(planUnitIDs(plan), []string{scheduler.ID}) {
		t.Fatalf("plan units = %v, want %s", planUnitIDs(plan), scheduler.ID)
	}
	if !reflect.DeepEqual(plan["authoringTarget"], map[string]any{"platform": "pylearn", "skill": "pylearn-alp-authoring"}) {
		t.Fatalf("authoringTarget = %#v", plan["authoringTarget"])
	}
	if !reflect.DeepEqual(plan["curriculum"], map[string]any{"id": specs.Curriculum.ID, "version": float64(1), "contentHash": specs.Curriculum.ContentHash}) {
		t.Fatalf("plan curriculum = %#v", plan["curriculum"])
	}
	recorded := readSpecFile(t, root, plan["path"].(string))
	delete(plan, "path")
	delete(plan, "status")
	if !reflect.DeepEqual(recorded, plan) {
		t.Fatalf("recorded plan differs from output:\n%#v\n%#v", recorded, plan)
	}

	// The same intent over the same specification versions is the same plan.
	again := requirePlan(t, planTarget(t, root))
	_, replanned := authoringOf(t, again)
	if replanned["id"] != plan["id"] || replanned["status"] != "existing" {
		t.Fatalf("re-plan = %#v", replanned)
	}
	if files := filesIn(t, filepath.Join(root, "authoring-plans")); len(files) != 1 {
		t.Fatalf("authoring plans = %v", files)
	}

	// An existing unit that teaches a competency nothing in it assesses
	// justifies an activity, which is smaller than a unit.
	mapping := writeMapping(t, mustRead(t, goALPMappingFixture)+`  - item: go-alp-a3-generics
    competencies:
      - id: go.language.generics
        role: teaches
`)
	body = requirePlan(t, planTarget(t, root, "--mapping", mapping))
	authoring, plan = authoringOf(t, body)
	generics := unitSpecFor(t, specsOf(t, body), "go-alp-a3-generics")
	if authoring.Intent != "activity" || authoring.Selection != "default" || !reflect.DeepEqual(planUnitIDs(plan), []string{generics.ID}) {
		t.Fatalf("authoring = %#v, units = %v", authoring, planUnitIDs(plan))
	}
	requireValidWorkspace(t, root)
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPlatformPlanWholeCurriculumAndExplicitIntentsRequireExplicitRequests(t *testing.T) {
	root := adaLearnerState(t)
	specs := specsOf(t, requirePlan(t, planTarget(t, root)))

	whole := requirePlan(t, planTarget(t, root, "--intent", "curriculum"))
	authoring, plan := authoringOf(t, whole)
	want := []string{
		unitSpecFor(t, specs, "Goroutines and scheduler mental model").ID,
		unitSpecFor(t, specs, "Channels and ownership").ID,
		unitSpecFor(t, specs, "Mutexes, WaitGroups, Once, atomics").ID,
	}
	if authoring.Intent != "curriculum" || authoring.Selection != "explicit" || !reflect.DeepEqual(planUnitIDs(plan), want) {
		t.Fatalf("whole-curriculum authoring = %#v, units = %v", authoring, planUnitIDs(plan))
	}

	channels := unitSpecFor(t, specs, "Channels and ownership")
	explicit := requirePlan(t, planTarget(t, root, "--intent", "unit", "--unit", channels.ID))
	if authoring, plan := authoringOf(t, explicit); authoring.Selection != "explicit" || !reflect.DeepEqual(planUnitIDs(plan), []string{channels.ID}) {
		t.Fatalf("explicit unit = %#v", authoring)
	}
	patch := requirePlan(t, planTarget(t, root, "--intent", "patch", "--unit", "go-alp-a2-goroutines"))
	if authoring, plan := authoringOf(t, patch); authoring.Intent != "patch" ||
		!reflect.DeepEqual(planUnitIDs(plan), []string{unitSpecFor(t, specs, "go-alp-a2-goroutines").ID}) {
		t.Fatalf("explicit patch = %#v", authoring)
	}

	refusals := map[string][]string{
		"skeleton for a target with content": {"--intent", "target-skeleton"},
		"unit for an existing unit":          {"--intent", "unit", "--unit", "go-alp-a2-goroutines"},
		"activity for a proposed unit":       {"--intent", "activity", "--unit", channels.ID},
	}
	for name, args := range refusals {
		t.Run(name, func(t *testing.T) {
			result := planTarget(t, root, args...)
			if result.code != 1 || errorField(t, result)["code"] != "intent-not-applicable" {
				t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
			}
		})
	}
	if result := planTarget(t, root, "--unit", "go-alp-z9-missing"); result.code != 1 || errorField(t, result)["code"] != "unknown-unit" {
		t.Fatalf("unknown unit exit = %d, stdout = %s", result.code, result.stdout)
	}
	if result := planTarget(t, root, "--intent", "course"); result.code != 2 || errorField(t, result)["code"] != "usage" {
		t.Fatalf("unknown intent exit = %d, stdout = %s", result.code, result.stdout)
	}
}

// newTargetFixtures writes a curriculum export and mapping for a target with
// no content yet, and constraints naming its goal.
func newTargetFixtures(t *testing.T) (curriculum, mapping, constraints string) {
	t.Helper()
	dir := t.TempDir()
	curriculum = filepath.Join(dir, "curriculum.json")
	mapping = filepath.Join(dir, "go-alp.mapping.yaml")
	constraints = filepath.Join(dir, "go-alp.constraints.yaml")
	for path, content := range map[string]string{
		curriculum:  `{"schemaVersion": 1, "platform": "pylearn", "targets": [{"id": "go-alp", "title": "Go (ALP)", "contentHash": "` + opaqueHash + `", "phases": [], "items": []}]}`,
		mapping:     "schemaVersion: 2\nplatform: pylearn\ntarget: go-alp\npacks:\n  - domain: go\n    packVersion: \">=0.1.0 <0.2.0\"\nentries: []\n",
		constraints: "schemaVersion: 1\nplatform: pylearn\ntarget: go-alp\nallowedModes: [skip, challenge, skim, full]\ngoal: [go.concurrency.channels]\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return curriculum, mapping, constraints
}

func planNewTarget(t *testing.T, root string, extra ...string) platformRun {
	t.Helper()
	curriculum, mapping, constraints := newTargetFixtures(t)
	args := []string{"plan", "--adapter", "pylearn", "--target", "go-alp", "--curriculum", curriculum,
		"--mapping", mapping, "--constraints", constraints, "--workspace", root}
	return runPlatform(t, App{}, append(args, extra...)...)
}

func TestPlatformPlanAuthorsANewTargetSkeletonOnlyOnExplicitRequest(t *testing.T) {
	root := newLearnerWorkspace(t, "grace")
	seedAssessment(t, root, "go.runtime.scheduler", "strong", "high", "2026-10-01T09:00:00Z")

	// The goal's unshown prerequisites are proposed with it; demonstrated
	// ones are not.
	body := requirePlan(t, planNewTarget(t, root))
	specs := specsOf(t, body)
	var titles []string
	for _, unit := range specs.Units {
		titles = append(titles, unit.Group+":"+unit.Title)
	}
	if want := []string{"stage-1:Goroutine lifecycle and ownership", "stage-2:Channels and ownership"}; !reflect.DeepEqual(titles, want) {
		t.Fatalf("units = %v, want %v", titles, want)
	}
	authoring, plan := authoringOf(t, body)
	if authoring.Status != "explicit-intent-required" || authoring.Intent != "target-skeleton" || plan != nil {
		t.Fatalf("default authoring for a new target = %#v, plan = %#v", authoring, plan)
	}
	if curriculum := readSpecFile(t, root, specs.Curriculum.Path); curriculum["authoringIntent"] != "target-skeleton" ||
		!reflect.DeepEqual(curriculum["goal"], []any{map[string]any{"domain": "go", "id": "go.concurrency.channels"}}) {
		t.Fatalf("curriculum spec = %#v", curriculum)
	}

	skeleton := requirePlan(t, planNewTarget(t, root, "--intent", "target-skeleton"))
	authoring, plan = authoringOf(t, skeleton)
	if authoring.Status != "planned" || authoring.Intent != "target-skeleton" || authoring.Selection != "explicit" {
		t.Fatalf("skeleton authoring = %#v", authoring)
	}
	if !reflect.DeepEqual(planUnitIDs(plan), []string{specs.Units[0].ID, specs.Units[1].ID}) {
		t.Fatalf("skeleton units = %v", planUnitIDs(plan))
	}
	// The public face carries the grouping the authoring target turns into
	// the platform's own structure.
	groups := plan["public"].(map[string]any)["curriculum"].(map[string]any)["groups"].([]any)
	if len(groups) != 2 || groups[0].(map[string]any)["title"] != "Stage 1" ||
		!reflect.DeepEqual(groups[1].(map[string]any)["units"], []any{map[string]any{"id": specs.Units[1].ID, "title": "Channels and ownership"}}) {
		t.Fatalf("public groups = %#v", groups)
	}

	unknownGoal := filepath.Join(t.TempDir(), "constraints.yaml")
	if err := os.WriteFile(unknownGoal, []byte("schemaVersion: 1\nplatform: pylearn\ntarget: go-alp\nallowedModes: [full]\ngoal: [go.concurrency.telepathy]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if result := planNewTarget(t, root, "--constraints", unknownGoal); result.code != 1 || errorField(t, result)["code"] != "invalid-target-constraints" {
		t.Fatalf("unknown goal exit = %d, stdout = %s", result.code, result.stdout)
	}
	requireValidWorkspace(t, root)
}

func TestPlatformAuthoringPlanPublicFaceCarriesOnlySpecCitationsAndTeachingIntent(t *testing.T) {
	root := adaLearnerState(t)
	seedAssessment(t, root, "go.concurrency.goroutines", "rusty", "high", "2026-10-03T09:00:00Z", "unbounded goroutine fan-out without cancellation")
	cwd := t.TempDir()
	app := App{Getwd: func() (string, error) { return cwd, nil }}
	result := runPlatform(t, app, "plan", "--adapter", "pylearn", "--target", "go-alp", "--curriculum", goALPCurriculumFixture,
		"--mapping", goALPMappingFixture, "--workspace", root, "--intent", "curriculum")
	body := requirePlan(t, result)
	_, plan := authoringOf(t, body)
	public := plan["public"].(map[string]any)

	encoded, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{
		"ada", "learnerId", "asmt_", "ev_", "aad_", "tapr_", "lsr_", "assessmentId", "unbounded goroutine fan-out",
		"rusty", "demonstrated", "unassessed", "\"status\"", "\"level\"", "confidence", "\"mode\"", "proposedMode", "rationale", "misconception", "adaptation",
	} {
		if strings.Contains(string(encoded), private) {
			t.Fatalf("public face leaks %q: %s", private, encoded)
		}
	}
	if public["intent"] != "curriculum" {
		t.Fatalf("public intent = %#v", public["intent"])
	}
	// Every public unit is cited by ID, version and hash, with its teaching
	// intent exactly as specified.
	for _, raw := range public["units"].([]any) {
		unit := raw.(map[string]any)
		path := "specifications/units/" + unit["id"].(string) + "-v" + strconv.Itoa(int(unit["version"].(float64))) + ".json"
		spec := readSpecFile(t, root, path)
		if unit["contentHash"] != spec["contentHash"] || !reflect.DeepEqual(unit["teachingIntent"], spec["teaching"]) {
			t.Fatalf("public unit %v does not cite its spec: %#v vs %#v", unit["id"], unit, spec["teaching"])
		}
	}

	// Specifications and plans live only in the learner workspace.
	if entries, err := os.ReadDir(cwd); err != nil || len(entries) != 0 {
		t.Fatalf("plan wrote outside the workspace: %v %v", entries, err)
	}
	for _, path := range append(filesUnder(t, root, "specifications"), filesUnder(t, root, "authoring-plans")...) {
		if !strings.HasPrefix(path, root+string(filepath.Separator)) {
			t.Fatalf("artifact outside workspace: %s", path)
		}
	}
}

type noAuthoringAdapter struct{ pylearn.Adapter }

func (noAuthoringAdapter) ID() string { return "read-only" }
func (noAuthoringAdapter) Capabilities() []platform.Capability {
	return []platform.Capability{platform.ActivitySource, platform.CurriculumReader, platform.ContentMapper}
}

func TestPlatformPlanRequiresTheAuthoringTargetCapability(t *testing.T) {
	registry := platform.NewRegistry(noAuthoringAdapter{pylearn.NewAdapter()})
	root := adaLearnerState(t)
	result := runPlatform(t, App{Platforms: &registry}, "plan", "--adapter", "read-only", "--target", "go-alp",
		"--curriculum", goALPCurriculumFixture, "--mapping", goALPMappingFixture, "--workspace", root)
	if result.code != 1 {
		t.Fatalf("exit = %d, stdout = %s", result.code, result.stdout)
	}
	if got := errorField(t, result); got["code"] != "capability-not-declared" || got["capability"] != "authoring-target" {
		t.Fatalf("error = %#v", got)
	}
	for _, dir := range []string{"specifications", "authoring-plans", filepath.Join("state", "target-adaptations")} {
		if _, err := os.Stat(filepath.Join(root, dir)); !os.IsNotExist(err) {
			t.Fatalf("refused plan wrote %s: %v", dir, err)
		}
	}
}

func TestPlatformInspectNamesTheDeclaredAuthoringTargetSkill(t *testing.T) {
	declared := runPlatform(t, App{}, "inspect", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture)
	if declared.code != 0 {
		t.Fatalf("exit = %d, stdout = %s", declared.code, declared.stdout)
	}
	var output map[string]any
	if err := json.Unmarshal(declared.stdout, &output); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(output["authoringTarget"], map[string]any{"platform": "pylearn", "skill": "pylearn-alp-authoring"}) {
		t.Fatalf("authoringTarget = %#v", output["authoringTarget"])
	}

	// An adapter that does not declare Authoring Target names no skill, even
	// when its implementation could.
	registry := platform.NewRegistry(noAuthoringAdapter{pylearn.NewAdapter()})
	undeclared := runPlatform(t, App{Platforms: &registry}, "inspect", "--adapter", "read-only", "--target", "go", "--curriculum", pylearnCurriculumFixture)
	if undeclared.code != 0 {
		t.Fatalf("exit = %d, stdout = %s", undeclared.code, undeclared.stdout)
	}
	output = nil
	if err := json.Unmarshal(undeclared.stdout, &output); err != nil {
		t.Fatal(err)
	}
	if value, present := output["authoringTarget"]; !present || value != nil {
		t.Fatalf("authoringTarget = %#v (present %v), want null", value, present)
	}
}
