package store

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/gitexec"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestGitStoreCommitsOnlyOwnedPaths(t *testing.T) {
	root := t.TempDir()
	initGitStoreRepo(t, root)
	writeGitStoreFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_git\nlearnerId: learner\n")
	runGitStore(t, root, "add", "workspace.yaml")
	runGitStore(t, root, "commit", "-m", "init")
	writeGitStoreFile(t, root, "notes.txt", "do not commit\n")

	validator, _ := workspace.NewValidator()
	s, err := OpenGit(root, validator)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := s.Revision(context.Background())
	profile := []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")
	next, err := s.Commit(context.Background(), base, ChangeSet{
		Message:   "learn: checkpoint",
		Mutations: []Mutation{{Path: "profile/profile.yaml", Data: profile}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if next == base {
		t.Fatal("git revision did not advance")
	}

	status := runGitStore(t, root, "status", "--porcelain")
	if !strings.Contains(status, "?? notes.txt") {
		t.Fatalf("unrelated file was absorbed by checkpoint: %q", status)
	}
	show := runGitStore(t, root, "show", "--name-only", "--format=", "HEAD")
	if !strings.Contains(show, "profile/profile.yaml") || strings.Contains(show, "notes.txt") {
		t.Fatalf("checkpoint paths = %q", show)
	}
}

func TestGitStoreRejectsNonOwnedMutation(t *testing.T) {
	root := t.TempDir()
	initGitStoreRepo(t, root)
	writeGitStoreFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_git\nlearnerId: learner\n")
	runGitStore(t, root, "add", ".")
	runGitStore(t, root, "commit", "-m", "init")
	validator, _ := workspace.NewValidator()
	s, _ := OpenGit(root, validator)
	base, _ := s.Revision(context.Background())
	_, err := s.Commit(context.Background(), base, ChangeSet{Mutations: []Mutation{{Path: "README.md", Data: []byte("no")}}})
	if err == nil || !strings.Contains(err.Error(), "not ALP-owned") {
		t.Fatalf("error = %v", err)
	}
}

func TestCloneGitStoreUsesLocalCheckout(t *testing.T) {
	source := t.TempDir()
	initGitStoreRepo(t, source)
	writeGitStoreFile(t, source, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_git\nlearnerId: learner\n")
	runGitStore(t, source, "add", ".")
	runGitStore(t, source, "commit", "-m", "init")

	bare := filepath.Join(t.TempDir(), "state.git")
	runCommandGitStore(t, "", "git", "clone", "--bare", source, bare)
	dest := filepath.Join(t.TempDir(), "clone")
	validator, _ := workspace.NewValidator()
	s, err := CloneGit(context.Background(), bare, dest, "main", validator)
	if err != nil {
		t.Fatal(err)
	}
	if s.Root() != dest {
		t.Fatalf("root = %s", s.Root())
	}
	if !s.Capabilities().Has(CapabilitySync) || !s.Capabilities().Has(CapabilityMultiDevice) {
		t.Fatalf("capabilities = %#v", s.Capabilities())
	}
}

func initGitStoreRepo(t *testing.T, root string) {
	t.Helper()
	runGitStore(t, root, "init", "-b", "main")
	runGitStore(t, root, "config", "user.email", "alp@example.invalid")
	runGitStore(t, root, "config", "user.name", "ALP Test")
}

func writeGitStoreFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGitStore(t *testing.T, root string, args ...string) string {
	t.Helper()
	return runCommandGitStore(t, root, "git", append([]string{"-C", root}, args...)...)
}

func runCommandGitStore(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Env = gitexec.Env()
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func TestInitializeGitCreatesOwnedInitialCheckpointOnly(t *testing.T) {
	root := t.TempDir()
	writeGitStoreFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_git\nlearnerId: learner\n")
	writeGitStoreFile(t, root, "README.md", "unrelated\n")
	runGitStore(t, root, "init", "-b", "main")
	runGitStore(t, root, "config", "user.email", "alp@example.invalid")
	runGitStore(t, root, "config", "user.name", "ALP Test")
	// Remove the unborn repository so InitializeGit exercises existing-git/no-HEAD behavior.
	validator, _ := workspace.NewValidator()
	s, err := InitializeGit(context.Background(), root, "main", "", validator)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Revision(context.Background()); err != nil {
		t.Fatalf("initialized store must have a revision: %v", err)
	}
	show := runGitStore(t, root, "show", "--name-only", "--format=", "HEAD")
	if !strings.Contains(show, "workspace.yaml") || strings.Contains(show, "README.md") {
		t.Fatalf("initial checkpoint paths = %q", show)
	}
}
