package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	godomain "github.com/adams100111/agentic-learning-partner/domains/go"
	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/platform/pylearn"
	"go.yaml.in/yaml/v3"
)

const pylearnGoMappingFixture = "testdata/platform/pylearn/go.mapping.yaml"

func validateMapping(t *testing.T, app App, mappingPath string, extra ...string) platformRun {
	t.Helper()
	args := []string{"mapping", "validate", "--adapter", "pylearn", "--target", "go",
		"--curriculum", pylearnCurriculumFixture, "--mapping", mappingPath}
	return runPlatform(t, app, append(args, extra...)...)
}

func TestPlatformMappingValidateAcceptsV2MappingAgainstGoPack(t *testing.T) {
	result := validateMapping(t, App{}, pylearnGoMappingFixture)
	if result.code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", result.code, result.stderr, result.stdout)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "platform", "pylearn", "mapping-validate-go.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.stdout, want) {
		t.Fatalf("mapping validate output mismatch\n--- got ---\n%s\n--- want ---\n%s", result.stdout, want)
	}
	again := validateMapping(t, App{}, pylearnGoMappingFixture)
	if !bytes.Equal(again.stdout, result.stdout) {
		t.Fatalf("mapping validate output is not deterministic across runs")
	}
}

func writeMapping(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mapping.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const goPackHeader = `schemaVersion: 2
platform: pylearn
target: go
packs:
  - domain: go
    packVersion: ">=0.1.0 <0.2.0"
`

type problem struct {
	Severity     string   `json:"severity"`
	Code         string   `json:"code"`
	Path         string   `json:"path"`
	Item         string   `json:"item"`
	Competency   string   `json:"competency"`
	Replacements []string `json:"replacements"`
	Message      string   `json:"message"`
}

type mappingReport struct {
	Valid   bool `json:"valid"`
	Summary struct {
		Entries  int `json:"entries"`
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
	} `json:"summary"`
	Entries []struct {
		Item struct {
			Item string `json:"item"`
		} `json:"item"`
		Competencies []struct {
			ID         string `json:"id"`
			Role       string `json:"role"`
			ResolvedID string `json:"resolvedId"`
		} `json:"competencies"`
	} `json:"entries"`
	Problems []problem `json:"problems"`
}

func decodeReport(t *testing.T, result platformRun) mappingReport {
	t.Helper()
	var report mappingReport
	if err := json.Unmarshal(result.stdout, &report); err != nil {
		t.Fatalf("stdout is not a mapping report: %v\n%s", err, result.stdout)
	}
	return report
}

// assertProblems checks the report lists exactly the wanted problems, in
// document order, each with a message containing the wanted fragment.
func assertProblems(t *testing.T, report mappingReport, want []problem) {
	t.Helper()
	if len(report.Problems) != len(want) {
		t.Fatalf("problems = %+v, want %d problems %+v", report.Problems, len(want), want)
	}
	for i, wanted := range want {
		got := report.Problems[i]
		if got.Severity != wanted.Severity || got.Code != wanted.Code || got.Path != wanted.Path ||
			got.Item != wanted.Item || got.Competency != wanted.Competency ||
			strings.Join(got.Replacements, ",") != strings.Join(wanted.Replacements, ",") ||
			!strings.Contains(got.Message, wanted.Message) {
			t.Fatalf("problem %d = %+v, want %+v", i, got, wanted)
		}
	}
}

func TestPlatformMappingValidateRejectsNonDeclaredStableIdentifiers(t *testing.T) {
	mapping := writeMapping(t, goPackHeader+`entries:
  - item: go-a1-first-delivery#project-file
    competencies:
      - id: go.language.packages-modules
        role: teaches
  - item: go-a1-first-delivery#scene-3
    competencies:
      - id: go.language.packages-modules
        role: reinforces
  - item: go-module-basics
    competencies:
      - id: go.language.packages-modules
        role: assesses
`)
	result := validateMapping(t, App{}, mapping)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	report := decodeReport(t, result)
	if report.Valid || report.Summary.Errors != 2 || report.Summary.Entries != 3 {
		t.Fatalf("report = %+v", report)
	}
	assertProblems(t, report, []problem{
		{Severity: "error", Code: "unstable-identifier", Path: "/entries/0/item", Item: "go-a1-first-delivery#project-file",
			Message: `item "go-a1-first-delivery#project-file" is not a declared-stable identifier of target "go"`},
		{Severity: "error", Code: "unstable-identifier", Path: "/entries/1/item", Item: "go-a1-first-delivery#scene-3",
			Message: "pylearn declares lesson, question, quiz, scene"},
	})
	if !strings.Contains(result.stderr, "unstable-identifier at /entries/0/item") {
		t.Fatalf("stderr = %q", result.stderr)
	}
}

// evolvedGoPlatforms registers the PyLearn adapter over a later revision of the
// real Go domain pack (0.2.0) whose migrations retire legacy competency IDs.
func evolvedGoPlatforms(t *testing.T) *platform.Registry {
	t.Helper()
	data, err := godomain.Files.ReadFile("competencies.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var pack domain.Pack
	if err := yaml.Unmarshal(data, &pack); err != nil {
		t.Fatal(err)
	}
	pack.Version = "0.2.0"
	pack.Migrations = append(pack.Migrations,
		domain.Migration{From: "go.legacy.context", To: []string{"go.runtime.context"}, Strategy: "rename"},
		domain.Migration{From: "go.legacy.concurrency", To: []string{"go.concurrency.goroutines", "go.concurrency.channels"}, Strategy: "split"},
		domain.Migration{From: "go.legacy.connection-pool", To: []string{"go.database.pooling"}, Strategy: "merge"},
		domain.Migration{From: "go.legacy.gopath", Strategy: "remove"},
	)
	evolved, err := yaml.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	registry := platform.NewRegistry(pylearn.Adapter{Domains: domain.NewRegistryFrom(map[string][]byte{"go": evolved})})
	return &registry
}

const evolvedPackHeader = `schemaVersion: 2
platform: pylearn
target: go
packs:
  - domain: go
    packVersion: ">=0.2.0 <0.3.0"
`

func TestPlatformMappingValidatePassesRenamedCompetencyWithUpdateWarning(t *testing.T) {
	mapping := writeMapping(t, evolvedPackHeader+`entries:
  - item: go-b1-goroutines
    competencies:
      - id: go.legacy.context
        role: teaches
`)
	result := validateMapping(t, App{Platforms: evolvedGoPlatforms(t)}, mapping)
	if result.code != 0 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	report := decodeReport(t, result)
	if !report.Valid || report.Summary.Warnings != 1 || report.Summary.Errors != 0 {
		t.Fatalf("report = %+v", report)
	}
	assertProblems(t, report, []problem{
		{Severity: "warning", Code: "competency-renamed", Path: "/entries/0/competencies/0/id", Item: "go-b1-goroutines",
			Competency: "go.legacy.context", Replacements: []string{"go.runtime.context"},
			Message: `competency "go.legacy.context" was renamed to "go.runtime.context" in domain pack "go" 0.2.0`},
	})
	if got := report.Entries[0].Competencies[0]; got.ID != "go.legacy.context" || got.ResolvedID != "go.runtime.context" || got.Role != "teaches" {
		t.Fatalf("resolved competency = %+v", got)
	}
}

func TestPlatformMappingValidateFailsSplitMergeAndRemovedCompetencies(t *testing.T) {
	mapping := writeMapping(t, evolvedPackHeader+`entries:
  - item: go-b1-goroutines
    competencies:
      - id: go.legacy.concurrency
        role: teaches
  - item: go-goroutine-lifetimes
    competencies:
      - id: go.legacy.connection-pool
        role: assesses
      - id: go.legacy.gopath
        role: reinforces
      - id: go.legacy.never-existed
        role: assesses
`)
	result := validateMapping(t, App{Platforms: evolvedGoPlatforms(t)}, mapping)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	report := decodeReport(t, result)
	if report.Valid || report.Summary.Errors != 4 {
		t.Fatalf("report = %+v", report)
	}
	assertProblems(t, report, []problem{
		{Severity: "error", Code: "competency-split", Path: "/entries/0/competencies/0/id", Item: "go-b1-goroutines",
			Competency: "go.legacy.concurrency", Replacements: []string{"go.concurrency.goroutines", "go.concurrency.channels"},
			Message: `was split into go.concurrency.goroutines, go.concurrency.channels in domain pack "go" 0.2.0; ALP will not guess`},
		{Severity: "error", Code: "competency-merged", Path: "/entries/1/competencies/0/id", Item: "go-goroutine-lifetimes",
			Competency: "go.legacy.connection-pool", Replacements: []string{"go.database.pooling"},
			Message: `was merged into "go.database.pooling" in domain pack "go" 0.2.0; merged competencies require reassessment`},
		{Severity: "error", Code: "competency-removed", Path: "/entries/1/competencies/1/id", Item: "go-goroutine-lifetimes",
			Competency: "go.legacy.gopath", Message: `competency "go.legacy.gopath" was removed from domain pack "go" 0.2.0`},
		{Severity: "error", Code: "unknown-competency", Path: "/entries/1/competencies/2/id", Item: "go-goroutine-lifetimes",
			Competency: "go.legacy.never-existed", Message: "no pack migration covers it"},
	})
	for _, entry := range report.Entries {
		for _, competency := range entry.Competencies {
			if competency.ResolvedID != "" {
				t.Fatalf("failed competency must not resolve: %+v", competency)
			}
		}
	}
}

func TestPlatformMappingValidateFailsOutOfRangePackVersion(t *testing.T) {
	mapping := writeMapping(t, `schemaVersion: 2
platform: pylearn
target: go
packs:
  - domain: go
    packVersion: ">=1.0.0 <2.0.0"
entries:
  - item: go-b1-goroutines
    competencies:
      - id: go.concurrency.goroutines
        role: teaches
`)
	result := validateMapping(t, App{}, mapping)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	report := decodeReport(t, result)
	assertProblems(t, report, []problem{
		{Severity: "error", Code: "pack-version-out-of-range", Path: "/packs/0/packVersion",
			Message: `mapping requires domain pack "go" >=1.0.0 <2.0.0, but ALP has version 0.1.0`},
		{Severity: "error", Code: "competency-not-resolved", Path: "/entries/0/competencies/0/id", Item: "go-b1-goroutines",
			Competency: "go.concurrency.goroutines", Message: `domain pack "go" is unavailable or incompatible`},
	})
}

func TestPlatformMappingValidateRejectsV1MappingWithRewriteInstruction(t *testing.T) {
	mapping := writeMapping(t, `schemaVersion: 1
platform: pylearn
mappings:
  - contentId: go-b1-goroutines
    domain: go
    packVersion: ">=0.1.0"
    competencies: [go.concurrency.goroutines]
`)
	result := validateMapping(t, App{}, mapping)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	report := decodeReport(t, result)
	assertProblems(t, report, []problem{
		{Severity: "error", Code: "unsupported-mapping-version", Path: "/schemaVersion",
			Message: "v1 lists competencies without Mapping Roles, and ALP will not infer roles. Rewrite it as schemaVersion 2"},
	})
}

func TestPlatformMappingValidateListsEverySchemaProblem(t *testing.T) {
	mapping := writeMapping(t, goPackHeader+`entries:
  - item: go-b1-goroutines
    strengthCeiling: overwhelming
    competencies:
      - id: go.concurrency.goroutines
        role: masters
  - item: go-module-basics
    competencies: []
    weight: 3
`)
	result := validateMapping(t, App{}, mapping)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	report := decodeReport(t, result)
	paths := map[string]bool{}
	for _, got := range report.Problems {
		if got.Code != "schema" || got.Severity != "error" || got.Message == "" {
			t.Fatalf("problem = %+v", got)
		}
		paths[got.Path] = true
	}
	for _, want := range []string{"/entries/0/strengthCeiling", "/entries/0/competencies/0/role", "/entries/1/competencies", "/entries/1"} {
		if !paths[want] {
			t.Fatalf("schema problems %+v miss %s", report.Problems, want)
		}
	}
	if report.Summary.Errors != len(report.Problems) {
		t.Fatalf("summary = %+v for %d problems", report.Summary, len(report.Problems))
	}
	// Schema evaluation order follows map iteration; output must not.
	for range 20 {
		if again := validateMapping(t, App{}, mapping); !bytes.Equal(again.stdout, result.stdout) {
			t.Fatalf("schema problems are not reported deterministically\n--- first ---\n%s\n--- again ---\n%s", result.stdout, again.stdout)
		}
	}
}

func TestPlatformMappingValidateReportsEveryStructuralMismatch(t *testing.T) {
	mapping := writeMapping(t, `schemaVersion: 2
platform: moodle
target: python
packs:
  - domain: go
    packVersion: ">=0.1.0 <0.2.0"
  - domain: go
    packVersion: ">=0.1.0"
  - domain: rust
    packVersion: ">=1.0.0"
entries:
  - item: go-b1-goroutines
    competencies:
      - id: go.concurrency.goroutines
        role: teaches
      - id: go.concurrency.goroutines
        role: assesses
      - id: python.basics.loops
        role: teaches
  - item: go-b1-goroutines
    competencies:
      - id: go.concurrency.channels
        role: teaches
`)
	result := validateMapping(t, App{}, mapping)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	assertProblems(t, decodeReport(t, result), []problem{
		{Severity: "error", Code: "platform-mismatch", Path: "/platform", Message: `mapping is for platform "moodle", adapter validates platform "pylearn"`},
		{Severity: "error", Code: "target-mismatch", Path: "/target", Message: `mapping is for target "python", but target "go" is being validated`},
		{Severity: "error", Code: "duplicate-pack", Path: "/packs/1/domain", Message: `domain pack "go" is declared more than once`},
		{Severity: "error", Code: "unknown-domain-pack", Path: "/packs/2/domain", Message: `domain pack "rust" cannot be loaded`},
		{Severity: "error", Code: "duplicate-competency", Path: "/entries/0/competencies/1/id", Item: "go-b1-goroutines",
			Competency: "go.concurrency.goroutines", Message: "also at /entries/0/competencies/0"},
		{Severity: "error", Code: "undeclared-domain", Path: "/entries/0/competencies/2/id", Item: "go-b1-goroutines",
			Competency: "python.basics.loops", Message: "declared: go, rust"},
		{Severity: "error", Code: "duplicate-item", Path: "/entries/1/item", Item: "go-b1-goroutines", Message: "already mapped by /entries/0"},
	})
}

// curriculumOnlyAdapter reads curricula but does not declare Content Mapper.
type curriculumOnlyAdapter struct{ pylearn.Adapter }

func (curriculumOnlyAdapter) ID() string { return "curriculum-only" }
func (curriculumOnlyAdapter) Capabilities() []platform.Capability {
	return []platform.Capability{platform.CurriculumReader}
}

func TestPlatformMappingValidateRefusesAdapterWithoutContentMapper(t *testing.T) {
	registry := platform.NewRegistry(curriculumOnlyAdapter{pylearn.NewAdapter()})
	result := runPlatform(t, App{Platforms: &registry}, "mapping", "validate", "--adapter", "curriculum-only", "--target", "go",
		"--curriculum", pylearnCurriculumFixture, "--mapping", pylearnGoMappingFixture)
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	if got := errorField(t, result); got["code"] != "capability-not-declared" || got["capability"] != "content-mapper" {
		t.Fatalf("error = %#v", got)
	}
}

func TestPlatformMappingValidateUsageErrorsExitTwo(t *testing.T) {
	cases := map[string][]string{
		"no mapping subcommand":   {"mapping"},
		"flag instead of command": {"mapping", "--adapter", "pylearn"},
		"unknown subcommand":      {"mapping", "rewrite", "--adapter", "pylearn", "--target", "go"},
		"missing mapping":         {"mapping", "validate", "--adapter", "pylearn", "--target", "go", "--curriculum", pylearnCurriculumFixture},
		"missing curriculum":      {"mapping", "validate", "--adapter", "pylearn", "--target", "go", "--mapping", pylearnGoMappingFixture},
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

func TestPlatformMappingValidateReportsUnparseableMapping(t *testing.T) {
	result := validateMapping(t, App{}, writeMapping(t, "schemaVersion: 2\nentries: [unclosed\n"))
	if result.code != 1 {
		t.Fatalf("exit code = %d, stdout = %s", result.code, result.stdout)
	}
	assertProblems(t, decodeReport(t, result), []problem{
		{Severity: "error", Code: "unreadable-mapping", Message: "mapping cannot be parsed: invalid YAML"},
	})
}
