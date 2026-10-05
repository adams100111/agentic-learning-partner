package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateWorkspace(t *testing.T) {
	root := makeWorkspace(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	app := App{
		Out:    &out,
		ErrOut: &errOut,
		Getwd:  func() (string, error) { return t.TempDir(), nil },
	}

	if code := app.Run([]string{"validate", "--workspace", root}); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "valid workspace:") || !strings.Contains(out.String(), "revision:") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestValidateMissingWorkspaceConfiguration(t *testing.T) {
	t.Setenv("ALP_WORKSPACE", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	var out bytes.Buffer
	var errOut bytes.Buffer
	app := App{
		Out:    &out,
		ErrOut: &errOut,
		Getwd:  func() (string, error) { return t.TempDir(), nil },
	}

	if code := app.Run([]string{"validate"}); code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(errOut.String(), "no ALP learner workspace configured") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func makeWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run(t, root, "git", "init")
	run(t, root, "git", "config", "user.email", "alp@example.invalid")
	run(t, root, "git", "config", "user.name", "ALP Test")

	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 1\nlearnerId: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "profile"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profile", "profile.yaml"), []byte("schemaVersion: 1\nlearner:\n  id: test\nexperience: {}\ngoals: []\npreferences: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "add", ".")
	run(t, root, "git", "commit", "-m", "init")
	return root
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
}
