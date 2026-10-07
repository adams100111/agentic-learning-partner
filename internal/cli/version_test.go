package cli

import (
	"bytes"
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

func runVersion(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	app := App{Out: &out, ErrOut: &errOut, Getwd: func() (string, error) { return t.TempDir(), nil }}
	code := app.Run(append([]string{"version"}, args...))
	return code, out.String(), errOut.String()
}

func TestVersionPrintsBuildInfoDefaults(t *testing.T) {
	code, out, errOut := runVersion(t)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, errOut)
	}
	for _, want := range []string{
		"alp dev",
		"commit: unknown",
		"built: unknown",
		"go: " + runtime.Version(),
		"platform: " + runtime.GOOS + "/" + runtime.GOARCH,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestVersionJSON(t *testing.T) {
	code, out, errOut := runVersion(t, "--json")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, errOut)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	want := map[string]string{
		"version": "dev",
		"commit":  "unknown",
		"date":    "unknown",
		"go":      runtime.Version(),
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("json[%q] = %q, want %q (full: %s)", k, got[k], v, out)
		}
	}
}

func TestVersionRejectsUnknownFlag(t *testing.T) {
	if code, _, _ := runVersion(t, "--bogus"); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestUsageListsVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	app := App{Out: &out, ErrOut: &errOut, Getwd: func() (string, error) { return "", nil }}
	if code := app.Run(nil); code != 2 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(errOut.String(), "version") {
		t.Fatalf("usage lacks version: %s", errOut.String())
	}
}
