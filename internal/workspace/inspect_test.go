package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectReturnsGitRevision(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "alp@example.invalid")
	runGit(t, root, "config", "user.name", "ALP Test")
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 1\nlearnerId: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "workspace.yaml")
	runGit(t, root, "commit", "-m", "init")

	info, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if info.Path != root {
		t.Fatalf("path = %q", info.Path)
	}
	if len(info.Revision) < 7 {
		t.Fatalf("revision = %q", info.Revision)
	}
}

func TestInspectRejectsNonGitWorkspace(t *testing.T) {
	_, err := Inspect(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "Git repository") {
		t.Fatalf("error = %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
