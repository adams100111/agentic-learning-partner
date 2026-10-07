package schemas_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

const gateResultSchema = "platform-gate-result.schema.json"

func loadGateExample(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile("testdata/platform-gate-result.pylearn-go-alp.json")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func validateGateResult(t *testing.T, document map[string]any) string {
	t.Helper()
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if issue := validator.ValidateDocument(gateResultSchema, "gate-result.json", data); issue != nil {
		return issue.Error()
	}
	return ""
}

func firstGate(document map[string]any) map[string]any {
	return document["gates"].([]any)[0].(map[string]any)
}

func TestPlatformGateResultSchemaAcceptsPerGateResults(t *testing.T) {
	if problem := validateGateResult(t, loadGateExample(t)); problem != "" {
		t.Fatalf("example gate result rejected: %s", problem)
	}
}

func TestPlatformGateResultSchemaRejectsMalformedResults(t *testing.T) {
	cases := map[string]struct {
		mutate func(document map[string]any)
		field  string
	}{
		"missing publishable": {func(d map[string]any) { delete(d, "publishable") }, "publishable"},
		"unknown status":      {func(d map[string]any) { firstGate(d)["status"] = "ok" }, "status"},
		"missing provenance":  {func(d map[string]any) { delete(firstGate(d), "provenance") }, "provenance"},
		"provenance without version": {func(d map[string]any) {
			delete(firstGate(d)["provenance"].(map[string]any), "version")
		}, "version"},
		"missing artifacts":   {func(d map[string]any) { delete(firstGate(d), "artifacts") }, "artifacts"},
		"missing diagnostics": {func(d map[string]any) { delete(firstGate(d), "diagnostics") }, "diagnostics"},
		"diagnostic item without namespace": {func(d map[string]any) {
			gate := d["gates"].([]any)[1].(map[string]any)
			gate["diagnostics"].([]any)[0].(map[string]any)["item"] = map[string]any{"item": "go-alp-a1-modules#init"}
		}, "item"},
		"publishable despite a failing gate": {func(d map[string]any) { d["publishable"] = true }, "status"},
		"no gates":                           {func(d map[string]any) { d["gates"] = []any{} }, "gates"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			document := loadGateExample(t)
			tc.mutate(document)
			problem := validateGateResult(t, document)
			if problem == "" {
				t.Fatal("malformed gate result was accepted")
			}
			if !strings.Contains(problem, tc.field) {
				t.Fatalf("problem %q does not identify %q", problem, tc.field)
			}
		})
	}
}
