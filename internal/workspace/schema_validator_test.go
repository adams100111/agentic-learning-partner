package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatorAcceptsMinimalWorkspace(t *testing.T) {
	root := t.TempDir()
	write(t, root, "workspace.yaml", "schemaVersion: 1\nlearnerId: adams\n")
	write(t, root, "profile/profile.yaml", `schemaVersion: 1
learner:
  id: adams
experience: {}
goals: []
preferences: {}
`)

	validator, err := NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	if issues := validator.ValidateWorkspace(root); len(issues) != 0 {
		t.Fatalf("unexpected validation issues: %#v", issues)
	}
}

func TestValidatorReportsFilePathAndReason(t *testing.T) {
	root := t.TempDir()
	write(t, root, "workspace.yaml", "schemaVersion: 9\nlearnerId: adams\n")

	validator, err := NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	issues := validator.ValidateWorkspace(root)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v", issues)
	}
	message := issues[0].Error()
	if !strings.Contains(message, "workspace.yaml") || !strings.Contains(message, "schemaVersion") {
		t.Fatalf("validation message %q does not identify file and field", message)
	}
}

func TestValidatorRequiresManifest(t *testing.T) {
	validator, err := NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	issues := validator.ValidateWorkspace(t.TempDir())
	if len(issues) != 1 || !strings.Contains(issues[0].Error(), "workspace manifest is required") {
		t.Fatalf("issues = %#v", issues)
	}
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
