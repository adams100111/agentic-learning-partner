package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	godomain "github.com/adams100111/agentic-learning-partner/domains/go"
	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"go.yaml.in/yaml/v3"
)

const syncPlatformUnit = "go-alp-a3-sync"

// sequencingFixtures is a go-alp target with one existing platform unit (in
// platform phase A) teaching go.concurrency.sync, and a goal of every Go
// pack competency.
func sequencingFixtures(t *testing.T) (curriculum, mapping, constraints string, pack domain.Pack) {
	t.Helper()
	data, err := godomain.Files.ReadFile("competencies.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &pack); err != nil {
		t.Fatal(err)
	}
	var goal []string
	for _, competency := range pack.Competencies {
		goal = append(goal, competency.ID)
	}
	dir := t.TempDir()
	curriculum = filepath.Join(dir, "curriculum.json")
	mapping = filepath.Join(dir, "go-alp.mapping.yaml")
	constraints = filepath.Join(dir, "go-alp.constraints.yaml")
	for path, content := range map[string]string{
		curriculum: `{"schemaVersion": 1, "platform": "pylearn", "targets": [{"id": "go-alp", "title": "Go (ALP)", "contentHash": "` + opaqueHash + `",
  "phases": [{"id": "A", "title": "Runtime Foundations"}],
  "items": [{"id": "` + syncPlatformUnit + `", "kind": "lesson", "phase": "A", "title": "Mutexes and WaitGroups"}]}]}`,
		mapping: "schemaVersion: 2\nplatform: pylearn\ntarget: go-alp\npacks:\n  - domain: go\n    packVersion: \">=0.1.0 <0.2.0\"\n" +
			"entries:\n  - item: " + syncPlatformUnit + "\n    competencies:\n      - id: go.concurrency.sync\n        role: teaches\n",
		constraints: "schemaVersion: 1\nplatform: pylearn\ntarget: go-alp\nallowedModes: [skip, challenge, skim, full]\ngoal: [" + strings.Join(goal, ", ") + "]\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return curriculum, mapping, constraints, pack
}

func planSequencing(t *testing.T, root, curriculum, mapping, constraints string) map[string]any {
	t.Helper()
	return requirePlan(t, runPlatform(t, App{}, "plan", "--adapter", "pylearn", "--target", "go-alp", "--curriculum", curriculum,
		"--mapping", mapping, "--constraints", constraints, "--workspace", root))
}

// sequenceCompetencies returns the primary competency of each unit in
// curriculum order, read from the stored unit specifications.
func sequenceCompetencies(t *testing.T, root string, specs planSpecs) []string {
	t.Helper()
	var result []string
	for _, unit := range specs.Units {
		teaching := readSpecFile(t, root, unit.Path)["teaching"].(map[string]any)
		result = append(result, teaching["competencies"].([]any)[0].(map[string]any)["id"].(string))
	}
	return result
}

func position(order []string, id string) int {
	for i, value := range order {
		if value == id {
			return i
		}
	}
	return -1
}

func ancestors(pack domain.Pack, id string, into map[string]bool) {
	for _, competency := range pack.Competencies {
		if competency.ID != id {
			continue
		}
		for _, prerequisite := range competency.Prerequisites {
			if !into[prerequisite] {
				into[prerequisite] = true
				ancestors(pack, prerequisite, into)
			}
		}
	}
}

// The observed bug: a goal over the whole Go pack with one existing platform
// unit teaching go.concurrency.sync put concurrency at #2-4 and stage-2
// units ahead of stage-1 units.
func TestPlatformPlanSequencesUnitsByPrerequisiteLayersThenLearnerNeed(t *testing.T) {
	root := newLearnerWorkspace(t, "ada")
	seedAssessment(t, root, "go.language.slices", "rusty", "high", "2026-10-01T09:00:00Z")
	seedAssessment(t, root, "go.concurrency.goroutines", "rusty", "high", "2026-10-01T09:05:00Z")
	seedAssessment(t, root, "go.runtime.context", "unknown", "high", "2026-10-01T09:10:00Z")
	curriculum, mapping, constraints, pack := sequencingFixtures(t)

	body := planSequencing(t, root, curriculum, mapping, constraints)
	specs := specsOf(t, body)
	order := sequenceCompetencies(t, root, specs)
	if len(order) != len(pack.Competencies) {
		t.Fatalf("sequence has %d units, want %d: %v", len(order), len(pack.Competencies), order)
	}

	// 1. A topological order of the pack's prerequisite graph.
	for i, id := range order {
		found := map[string]bool{}
		ancestors(pack, id, found)
		for ancestor := range found {
			if at := position(order, ancestor); at > i {
				t.Errorf("%s (#%d) precedes its prerequisite %s (#%d)", id, i+1, ancestor, at+1)
			}
		}
	}

	// 2. Language foundations come before concurrency and runtime, whatever
	// the platform unit's phase says.
	lastLanguage, firstConcurrency := -1, len(order)
	for i, id := range order {
		switch {
		case strings.HasPrefix(id, "go.language."):
			lastLanguage = i
		case strings.HasPrefix(id, "go.concurrency."), strings.HasPrefix(id, "go.runtime."):
			if i < firstConcurrency {
				firstConcurrency = i
			}
		}
	}
	if lastLanguage > firstConcurrency {
		t.Errorf("a language unit (#%d) follows the first runtime/concurrency unit (#%d): %v", lastLanguage+1, firstConcurrency+1, order)
	}
	if order[0] != "go.language.syntax" {
		t.Errorf("first unit = %s, want go.language.syntax", order[0])
	}

	// 3. The existing platform unit is placed like any other: after its
	// prerequisites and after the foundations, never pulled forward.
	syncAt := position(order, "go.concurrency.sync")
	if syncAt < position(order, "go.concurrency.goroutines") || syncAt < lastLanguage {
		t.Errorf("platform unit at #%d, want after goroutines and language: %v", syncAt+1, order)
	}
	if specs.Units[syncAt].PlatformItem == nil || (*specs.Units[syncAt].PlatformItem)["item"] != syncPlatformUnit {
		t.Errorf("unit #%d is not the platform unit: %#v", syncAt+1, specs.Units[syncAt])
	}

	// 4. Among ready units of one area, rusty before unassessed.
	if position(order, "go.language.slices") > position(order, "go.language.pointers") {
		t.Errorf("rusty slices must precede unassessed pointers: %v", order)
	}

	// 5. Stage = prerequisite depth, and the emitted group order follows the
	// sequence (no stage-2 group before stage-1).
	curriculumSpec := readSpecFile(t, root, specs.Curriculum.Path)
	var groups []string
	for _, raw := range curriculumSpec["groups"].([]any) {
		groups = append(groups, raw.(map[string]any)["id"].(string))
	}
	var firstSeen []string
	for _, unit := range specs.Units {
		if !contains(firstSeen, unit.Group) {
			firstSeen = append(firstSeen, unit.Group)
		}
	}
	if !reflect.DeepEqual(groups, firstSeen) {
		t.Errorf("groups = %v, want sequence order %v", groups, firstSeen)
	}
	var stages []string
	for _, group := range groups {
		if strings.HasPrefix(group, "stage-") {
			stages = append(stages, group)
		}
	}
	for i := 1; i < len(stages); i++ {
		if stages[i-1] > stages[i] {
			t.Errorf("stage groups out of order: %v", stages)
		}
	}
	var sequence []any
	for _, unit := range specs.Units {
		sequence = append(sequence, unit.ID)
	}
	if !reflect.DeepEqual(curriculumSpec["sequence"], sequence) {
		t.Errorf("sequence = %v, want %v", curriculumSpec["sequence"], sequence)
	}
	t.Logf("sequence: %v", order)
}

// A change in learner need can reorder the sequence without changing what
// any unit teaches: units keep their versions, the curriculum is re-versioned.
func TestPlatformPlanSequenceOnlyChangeVersionsTheCurriculumNotTheUnits(t *testing.T) {
	root := newLearnerWorkspace(t, "ada")
	seedAssessment(t, root, "go.language.slices", "rusty", "high", "2026-10-01T09:00:00Z")
	curriculum, mapping, constraints, _ := sequencingFixtures(t)
	first := specsOf(t, planSequencing(t, root, curriculum, mapping, constraints))
	before := sequenceCompetencies(t, root, first)

	// "unknown" leaves every unit's mode and misconceptions as they were
	// (still unassessed) but makes maps more urgent than pointers.
	seedAssessment(t, root, "go.language.maps", "unknown", "high", "2026-10-02T09:00:00Z")
	second := specsOf(t, planSequencing(t, root, curriculum, mapping, constraints))
	after := sequenceCompetencies(t, root, second)
	if reflect.DeepEqual(before, after) {
		t.Fatal("the new assessment must reorder the sequence, or this test proves nothing")
	}
	if position(after, "go.language.maps") > position(after, "go.language.pointers") {
		t.Errorf("unknown maps must precede unassessed pointers: %v", after)
	}
	for _, unit := range second.Units {
		if unit.Status != "unchanged" || unit.Version != 1 {
			t.Errorf("sequence-only change versioned unit %s: %#v", unit.Title, unit)
		}
	}
	if second.Curriculum.Version != 2 || second.Curriculum.Status != "revised" {
		t.Fatalf("curriculum = %#v", second.Curriculum)
	}
	change := readSpecFile(t, root, second.Curriculum.Path)["change"].(map[string]any)
	if !contains(stringList(change["materialFields"].([]any)), "sequence") {
		t.Errorf("curriculum change = %#v, want sequence among material fields", change)
	}
}
