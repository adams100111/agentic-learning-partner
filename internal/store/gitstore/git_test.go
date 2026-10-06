package gitstore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitStoreCheckpointStagesOnlyOwnedPaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "init", "-b", "main")
	run(t, root, "git", "config", "user.email", "alp@example.invalid")
	run(t, root, "git", "config", "user.name", "ALP Test")
	id := "ws_0123456789abcdef0123456789abcdef"
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 2\nworkspaceId: "+id+"\nlearnerId: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "add", "workspace.yaml")
	run(t, root, "git", "commit", "-m", "init")

	store, err := Open(root, "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("unrelated"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	revision, _ := store.Revision(ctx)
	tx, err := store.Begin(ctx, revision)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Put("profile/profile.yaml", []byte("schemaVersion: 1\nlearner:\n  id: test\nexperience: {}\ngoals: []\npreferences: {}\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(ctx, "state: add profile"); err != nil {
		t.Fatal(err)
	}

	files := strings.Fields(runOutput(t, root, "git", "show", "--pretty=", "--name-only", "HEAD"))
	if len(files) != 1 || files[0] != "profile/profile.yaml" {
		t.Fatalf("checkpoint files = %#v", files)
	}
	status := runOutput(t, root, "git", "status", "--porcelain")
	if !strings.Contains(status, "?? notes.txt") {
		t.Fatalf("unrelated dirty file was not preserved: %q", status)
	}
}

func TestGitStoreRejectsUnownedMutation(t *testing.T) {
	root := makeStore(t)
	store, err := Open(root, "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	revision, _ := store.Revision(context.Background())
	tx, err := store.Begin(context.Background(), revision)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := tx.Put("README.md", []byte("no")); err == nil {
		t.Fatal("expected unowned path rejection")
	}
}

func TestGitStoreHistoryUsesCommitRevisions(t *testing.T) {
	root := makeStore(t)
	store, err := Open(root, "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	history, err := store.History(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Revision == "" {
		t.Fatalf("history = %#v", history)
	}
}

func makeStore(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "init", "-b", "main")
	run(t, root, "git", "config", "user.email", "alp@example.invalid")
	run(t, root, "git", "config", "user.name", "ALP Test")
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 2\nworkspaceId: ws_0123456789abcdef0123456789abcdef\nlearnerId: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "add", "workspace.yaml")
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

func runOutput(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return string(output)
}
