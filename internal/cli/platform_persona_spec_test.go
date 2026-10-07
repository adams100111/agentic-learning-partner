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

// Persona fixtures: a learner whose profile lists three stacks, a Global
// Persona and a Go Domain Persona. The Go persona prefers a stack the profile
// does not list (kotlin) for concurrency, which must never be used.
const (
	personaProfileFixture = `schemaVersion: 1
learner:
  id: ada
  displayName: Ada Example
  professionalLevel: senior
experience:
  php:
    level: expert
    relativeRank: 1
    frameworks:
      laravel: expert
  typescript:
    level: strong
    relativeRank: 2
    frameworks:
      nestjs: strong
  csharp:
    level: strong
    relativeRank: 3
    frameworks:
      dotnet: strong
goals:
  - id: production-go
    statement: Production-ready Go.
preferences:
  feedbackStyle: direct-technical
`
	personaGlobalFixture = `schemaVersion: 1
scope: global
domain: null
teaching:
  pace: standard
  preferredActivityTypes: [debugging, code-reading]
  avoid: [toy-examples]
  emphasis: [explicit-tradeoffs]
  feedback: [direct]
analogyPolicy:
  defaultPriority: [laravel, nestjs, dotnet, fastapi]
  semanticOverrides:
    cancellation:
      prefer: [dotnet]
      reason: CancellationToken is the closest familiar scaffold.
risks:
  - disengaging when material turns into beginner instruction
  - creating interfaces before a consumer needs abstraction
`
	personaGoFixture = `schemaVersion: 1
scope: domain
domain: go
teaching:
  pace: senior-dense
  preferredActivityTypes: [code-reading, concurrency-debugging]
  avoid: [framework-first-Go]
  emphasis: [concurrency]
analogyPolicy:
  semanticOverrides:
    context:
      prefer: [dotnet]
      reason: Start from CancellationToken, then widen to deadlines.
    concurrency:
      prefer: [kotlin, typescript]
      reason: Compare coroutines and the event loop.
risks:
  - unbounded goroutines or hidden goroutine ownership
`
)

func writeWorkspaceFile(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// adaWithPersonas is adaLearnerState with a learner profile and personas.
func adaWithPersonas(t *testing.T) string {
	t.Helper()
	root := adaLearnerState(t)
	writeWorkspaceFile(t, root, "profile/profile.yaml", personaProfileFixture)
	writeWorkspaceFile(t, root, "personas/global.yaml", personaGlobalFixture)
	writeWorkspaceFile(t, root, "personas/domains/go.yaml", personaGoFixture)
	return root
}

func analogy(competency, basis string, concepts, sources []any) map[string]any {
	return map[string]any{"domain": "go", "competency": competency, "basis": basis, "concepts": concepts, "sources": sources}
}

func TestPlatformPlanShapesUnitTeachingFromTheLearnersPersonasAndProfile(t *testing.T) {
	root := adaWithPersonas(t)
	specs := specsOf(t, requirePlan(t, planTarget(t, root)))

	goroutines := readSpecFile(t, root, unitSpecFor(t, specs, "go-alp-a2-goroutines").Path)
	shape, ok := goroutines["teaching"].(map[string]any)["shape"].(map[string]any)
	if !ok {
		t.Fatalf("unit teaching has no shape: %#v", goroutines["teaching"])
	}
	// The Domain Persona refines the Global Persona: its pace wins, its lists
	// extend the global ones, and the profile's feedback style is kept.
	want := map[string]any{
		"pace":          "senior-dense",
		"activityTypes": []any{"debugging", "code-reading", "concurrency-debugging"},
		"avoid":         []any{"toy-examples", "framework-first-Go"},
		"emphasis":      []any{"explicit-tradeoffs", "concurrency"},
		"feedback":      []any{"direct", "direct-technical"},
		// Semantic overrides match competencies by concept; a preferred stack
		// the profile does not list (kotlin) is dropped.
		"analogies": []any{
			analogy("go.concurrency.goroutines", "semantic-override", []any{"concurrency"}, []any{"typescript"}),
			analogy("go.concurrency.races-deadlocks-leaks", "semantic-override", []any{"concurrency"}, []any{"typescript"}),
		},
	}
	if !reflect.DeepEqual(shape, want) {
		t.Fatalf("teaching shape =\n%#v\nwant\n%#v", shape, want)
	}

	// A Domain Persona override and a Global Persona override can both match
	// one competency (context by its ID, cancellation by its shared scaffold).
	context := readSpecFile(t, root, unitSpecFor(t, specs, "go-alp-a1-context").Path)
	analogies := context["teaching"].(map[string]any)["shape"].(map[string]any)["analogies"]
	if want := []any{analogy("go.runtime.context", "semantic-override", []any{"context", "cancellation"}, []any{"dotnet"})}; !reflect.DeepEqual(analogies, want) {
		t.Fatalf("context analogies = %#v", analogies)
	}

	// Without a matching override, the default priority applies, restricted
	// to stacks the profile lists (fastapi is not one).
	scheduler := readSpecFile(t, root, unitSpecFor(t, specs, "Goroutines and scheduler mental model").Path)
	analogies = scheduler["teaching"].(map[string]any)["shape"].(map[string]any)["analogies"]
	if want := []any{analogy("go.runtime.scheduler", "default-priority", []any{}, []any{"laravel", "nestjs", "dotnet"})}; !reflect.DeepEqual(analogies, want) {
		t.Fatalf("scheduler analogies = %#v", analogies)
	}

	// Risks are private. Every risk of the unit's Global and Domain Personas
	// is relevant (dropping one is worse than keeping one); each names the
	// unit concepts it mentions, so general risks name none.
	persona, ok := goroutines["adaptation"].(map[string]any)["persona"].(map[string]any)
	if !ok {
		t.Fatalf("unit adaptation has no persona basis: %#v", goroutines["adaptation"])
	}
	wantRisks := []any{
		map[string]any{"risk": "disengaging when material turns into beginner instruction", "concepts": []any{}},
		map[string]any{"risk": "creating interfaces before a consumer needs abstraction", "concepts": []any{}},
		map[string]any{"risk": "unbounded goroutines or hidden goroutine ownership", "concepts": []any{"goroutines"}},
	}
	if !reflect.DeepEqual(persona["risks"], wantRisks) {
		t.Fatalf("risks = %#v", persona["risks"])
	}
	requireValidWorkspace(t, root)
}

func TestPlatformPlanUsesDocumentedDefaultsWhenPersonasAreMissing(t *testing.T) {
	// No profile and no personas: plan still succeeds, every unit gets the
	// default shape, and the plan output reports the missing documents and
	// each default.
	root := adaLearnerState(t)
	body := requirePlan(t, planTarget(t, root))
	specs := specsOf(t, body)
	goroutines := readSpecFile(t, root, unitSpecFor(t, specs, "go-alp-a2-goroutines").Path)
	want := map[string]any{
		"pace": "standard", "activityTypes": []any{}, "avoid": []any{}, "emphasis": []any{}, "feedback": []any{},
		"analogies": []any{
			analogy("go.concurrency.goroutines", "none", []any{}, []any{}),
			analogy("go.concurrency.races-deadlocks-leaks", "none", []any{}, []any{}),
		},
	}
	if shape := goroutines["teaching"].(map[string]any)["shape"]; !reflect.DeepEqual(shape, want) {
		t.Fatalf("default shape = %#v", shape)
	}
	if personas := goroutines["provenance"].(map[string]any)["personas"]; !reflect.DeepEqual(personas, map[string]any{"profile": nil, "global": nil, "domains": []any{}}) {
		t.Fatalf("persona provenance = %#v", personas)
	}
	report := body["specifications"].(map[string]any)["persona"].(map[string]any)
	wantDocuments := []any{
		map[string]any{"path": "profile/profile.yaml", "present": false},
		map[string]any{"path": "personas/global.yaml", "present": false},
		map[string]any{"path": "personas/domains/go.yaml", "domain": "go", "present": false},
	}
	if !reflect.DeepEqual(report["documents"], wantDocuments) {
		t.Fatalf("persona documents = %#v", report["documents"])
	}
	var defaulted []string
	for _, raw := range report["defaults"].([]any) {
		entry := raw.(map[string]any)
		defaulted = append(defaulted, entry["domain"].(string)+":"+entry["field"].(string)+"="+entry["value"].(string))
		if entry["reason"] == "" {
			t.Fatalf("default without a reason: %#v", entry)
		}
	}
	if want := []string{"go:activityTypes=[]", "go:analogies=none", "go:avoid=[]", "go:emphasis=[]", "go:feedback=[]", "go:pace=standard"}; !reflect.DeepEqual(defaulted, want) {
		t.Fatalf("defaults = %v, want %v", defaulted, want)
	}

	// A profile without personas: analogies follow the profile's stacks by
	// relative rank, and the report says so.
	writeWorkspaceFile(t, root, "profile/profile.yaml", personaProfileFixture)
	body = requirePlan(t, planTarget(t, root))
	scheduler := readSpecFile(t, root, unitSpecFor(t, specsOf(t, body), "Goroutines and scheduler mental model").Path)
	analogies := scheduler["teaching"].(map[string]any)["shape"].(map[string]any)["analogies"]
	if want := []any{analogy("go.runtime.scheduler", "default-priority", []any{}, []any{"php", "typescript", "csharp"})}; !reflect.DeepEqual(analogies, want) {
		t.Fatalf("profile-only analogies = %#v", analogies)
	}
	if !containsJSON(body["specifications"].(map[string]any)["persona"].(map[string]any)["defaults"].([]any), map[string]any{
		"domain": "go", "field": "analogyPolicy.defaultPriority", "value": "php,typescript,csharp",
		"reason": "no persona layer sets analogyPolicy.defaultPriority; the profile's stacks by relative rank apply",
	}) {
		t.Fatalf("defaults = %#v", body["specifications"].(map[string]any)["persona"])
	}
	requireValidWorkspace(t, root)
}

func TestPlatformPlanVersionsUnitsOnPersonaChangesOnlyWhenTheTeachingShapeChanges(t *testing.T) {
	root := adaWithPersonas(t)
	first := specsOf(t, requirePlan(t, planTarget(t, root)))
	goroutinesV1 := unitSpecFor(t, first, "go-alp-a2-goroutines")
	provenance := readSpecFile(t, root, goroutinesV1.Path)["provenance"].(map[string]any)["personas"].(map[string]any)
	for _, source := range []any{provenance["profile"], provenance["global"], provenance["domains"].([]any)[0]} {
		if hash, _ := source.(map[string]any)["contentHash"].(string); len(hash) != len("sha256:")+64 {
			t.Fatalf("persona provenance = %#v", provenance)
		}
	}
	if domain := provenance["domains"].([]any)[0].(map[string]any); domain["domain"] != "go" || domain["path"] != "personas/domains/go.yaml" {
		t.Fatalf("domain persona provenance = %#v", domain)
	}

	// Persona and profile edits that derive the same teaching shape (an
	// override's reason, an experience level) are not material.
	writeWorkspaceFile(t, root, "personas/domains/go.yaml", strings.Replace(personaGoFixture, "Compare coroutines and the event loop.", "Contrast with async/await.", 1))
	writeWorkspaceFile(t, root, "profile/profile.yaml", strings.Replace(personaProfileFixture, "    level: strong\n    relativeRank: 3", "    level: expert\n    relativeRank: 3", 1))
	second := specsOf(t, requirePlan(t, planTarget(t, root)))
	for _, unit := range append(second.Units, second.Curriculum) {
		if unit.Status != "unchanged" {
			t.Fatalf("a persona edit that keeps the teaching shape changed %s: %#v", unit.ID, unit)
		}
	}

	// A persona edit that changes the derived teaching shape is a material,
	// non-evidence-linked change of every unit it shapes.
	writeWorkspaceFile(t, root, "personas/domains/go.yaml", strings.Replace(personaGoFixture, "emphasis: [concurrency]", "emphasis: [concurrency, testing]", 1))
	third := specsOf(t, requirePlan(t, planTarget(t, root)))
	revised := unitSpecFor(t, third, "go-alp-a2-goroutines")
	if revised.Version != 2 || revised.Status != "revised" {
		t.Fatalf("revised unit = %#v", revised)
	}
	v2 := readSpecFile(t, root, revised.Path)
	if !reflect.DeepEqual(v2["change"], map[string]any{"materialFields": []any{"teachingShape"}, "evidenceLinked": false}) {
		t.Fatalf("change = %#v", v2["change"])
	}
	if emphasis := v2["teaching"].(map[string]any)["shape"].(map[string]any)["emphasis"]; !reflect.DeepEqual(emphasis, []any{"explicit-tradeoffs", "concurrency", "testing"}) {
		t.Fatalf("emphasis = %#v", emphasis)
	}
	newDomain := v2["provenance"].(map[string]any)["personas"].(map[string]any)["domains"].([]any)[0].(map[string]any)
	if newDomain["contentHash"] == provenance["domains"].([]any)[0].(map[string]any)["contentHash"] {
		t.Fatalf("v2 does not record the new domain persona hash: %#v", newDomain)
	}
	requireValidWorkspace(t, root)
}

func TestPlatformAuthoringPlanPublishesTeachingShapeButKeepsPersonaBasisPrivate(t *testing.T) {
	root := adaWithPersonas(t)
	body := requirePlan(t, planTarget(t, root, "--intent", "curriculum"))
	_, plan := authoringOf(t, body)
	public := plan["public"].(map[string]any)
	units := public["units"].([]any)
	if len(units) == 0 {
		t.Fatalf("public units = %#v", units)
	}
	// Analogy stacks and teaching constraints are teaching intent.
	for _, raw := range units {
		shape, ok := raw.(map[string]any)["teachingIntent"].(map[string]any)["shape"].(map[string]any)
		if !ok || shape["pace"] != "senior-dense" || !reflect.DeepEqual(shape["avoid"], []any{"toy-examples", "framework-first-Go"}) {
			t.Fatalf("public teaching shape = %#v", raw)
		}
	}
	// Learner identity, experience levels and ranks, risks and raw persona
	// text never appear in the public face.
	encoded, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{
		"Ada Example", "senior\"", "expert", "relativeRank", "\"level\"", "\"stack\"", "sourceExperience", "analogyReasons", "risks",
		"unbounded goroutines", "beginner instruction", "CancellationToken", "Compare coroutines", "personas/", "profile",
	} {
		if strings.Contains(string(encoded), private) {
			t.Fatalf("public face leaks %q: %s", private, encoded)
		}
	}
}

func TestPlatformPlanReshapesSpecificationVersionsCreatedBeforeTeachingShapes(t *testing.T) {
	root := adaWithPersonas(t)
	specs := specsOf(t, requirePlan(t, planTarget(t, root)))
	entry := unitSpecFor(t, specs, "go-alp-a2-goroutines")

	// Rewrite version 1 as a pre-teaching-shape version: no shape, persona
	// basis or persona provenance, with a consistent content hash.
	path := filepath.Join(root, filepath.FromSlash(entry.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := platform.DecodeUnitSpec(entry.Path, data)
	if err != nil {
		t.Fatal(err)
	}
	stored.Teaching.Shape, stored.Adaptation.Persona, stored.Provenance.Personas = nil, nil, nil
	legacy, _, err := platform.ReviseUnitSpec(stored, nil)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := platform.EncodeSpec(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "shape") {
		t.Fatalf("legacy version still has a shape: %s", encoded)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}

	revised := unitSpecFor(t, specsOf(t, requirePlan(t, planTarget(t, root))), "go-alp-a2-goroutines")
	if revised.Version != 2 || revised.Status != "revised" {
		t.Fatalf("legacy version was not reshaped: %#v", revised)
	}
	if change := readSpecFile(t, root, revised.Path)["change"]; !reflect.DeepEqual(change, map[string]any{"materialFields": []any{"teachingShape"}, "evidenceLinked": false}) {
		t.Fatalf("change = %#v", change)
	}
}
