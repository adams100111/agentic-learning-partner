package gitstore

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
)

func TestSyncReconcilesIndependentAppendOnlyRecordsAcrossClones(t *testing.T) {
	remote, first, second := syncFixture(t)
	ctx := context.Background()

	storeA, err := Open(first, "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	storeB, err := Open(second, "main", nil)
	if err != nil {
		t.Fatal(err)
	}

	revA, _ := storeA.Revision(ctx)
	txA, err := storeA.Begin(ctx, revA)
	if err != nil {
		t.Fatal(err)
	}
	if err := txA.Put("evidence/ev_a.yaml", []byte("id: ev_a\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := txA.Commit(ctx, "state: evidence A"); err != nil {
		t.Fatal(err)
	}
	if _, err := storeA.Push(ctx); err != nil {
		t.Fatal(err)
	}

	revB, _ := storeB.Revision(ctx)
	txB, err := storeB.Begin(ctx, revB)
	if err != nil {
		t.Fatal(err)
	}
	if err := txB.Put("evidence/ev_b.yaml", []byte("id: ev_b\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := txB.Commit(ctx, "state: evidence B"); err != nil {
		t.Fatal(err)
	}
	if _, err := storeB.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	verify := filepath.Join(t.TempDir(), "verify")
	runSync(t, "", "git", "clone", remote, verify)
	for _, name := range []string{"ev_a.yaml", "ev_b.yaml"} {
		if _, err := os.Stat(filepath.Join(verify, "evidence", name)); err != nil {
			t.Fatalf("missing %s after reconciliation: %v", name, err)
		}
	}
}

func TestSyncSurfacesCompetingExplicitLearnerChanges(t *testing.T) {
	_, first, second := syncFixture(t)
	ctx := context.Background()

	baseProfile := []byte(`schemaVersion: 1
learner:
  id: test
experience:
  go:
    level: rusty
    provenance:
      sourceType: learner-stated
goals: []
preferences: {}
`)
	for _, root := range []string{first, second} {
		if err := os.MkdirAll(filepath.Join(root, "profile"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "profile", "profile.yaml"), baseProfile, 0o644); err != nil {
			t.Fatal(err)
		}
		runSync(t, root, "git", "add", "profile/profile.yaml")
		runSync(t, root, "git", "commit", "-m", "state: add profile")
	}
	// Make both clones share the same committed base.
	runSync(t, first, "git", "push", "origin", "main")
	runSync(t, second, "git", "fetch", "origin", "main")
	runSync(t, second, "git", "reset", "--hard", "origin/main")

	writeProfile := func(root, level string) {
		data := strings.Replace(string(baseProfile), "level: rusty", "level: "+level, 1)
		if err := os.WriteFile(filepath.Join(root, "profile", "profile.yaml"), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		runSync(t, root, "git", "add", "profile/profile.yaml")
		runSync(t, root, "git", "commit", "-m", "state: explicit learner correction")
	}
	writeProfile(first, "strong")
	writeProfile(second, "functional")

	storeA, _ := Open(first, "main", nil)
	storeB, _ := Open(second, "main", nil)
	if _, err := storeA.Push(ctx); err != nil {
		t.Fatal(err)
	}
	_, err := storeB.Sync(ctx)
	var conflict SemanticConflictError
	if err == nil || !errors.As(err, &conflict) {
		t.Fatalf("expected semantic conflict, got %v", err)
	}
}

func TestGitStoreAdvertisesSyncCapability(t *testing.T) {
	_, first, _ := syncFixture(t)
	store, err := Open(first, "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := storepkg.Require(store.Provider(), store.Capabilities(), storepkg.CapabilitySync, storepkg.CapabilityMultiDevice); err != nil {
		t.Fatal(err)
	}
}

func syncFixture(t *testing.T) (remote, first, second string) {
	t.Helper()
	root := t.TempDir()
	remote = filepath.Join(root, "remote.git")
	runSync(t, "", "git", "init", "--bare", "--initial-branch=main", remote)

	seed := filepath.Join(root, "seed")
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	runSync(t, seed, "git", "init", "-b", "main")
	runSync(t, seed, "git", "config", "user.email", "alp@example.invalid")
	runSync(t, seed, "git", "config", "user.name", "ALP Test")
	if err := os.WriteFile(filepath.Join(seed, "workspace.yaml"), []byte("schemaVersion: 2\nworkspaceId: ws_0123456789abcdef0123456789abcdef\nlearnerId: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runSync(t, seed, "git", "add", "workspace.yaml")
	runSync(t, seed, "git", "commit", "-m", "init")
	runSync(t, seed, "git", "remote", "add", "origin", remote)
	runSync(t, seed, "git", "push", "-u", "origin", "main")

	first = filepath.Join(root, "first")
	second = filepath.Join(root, "second")
	runSync(t, "", "git", "clone", remote, first)
	runSync(t, "", "git", "clone", remote, second)
	for _, clone := range []string{first, second} {
		runSync(t, clone, "git", "config", "user.email", "alp@example.invalid")
		runSync(t, clone, "git", "config", "user.name", "ALP Test")
	}
	return remote, first, second
}

func runSync(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	if dir != "" {
		command.Dir = dir
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return string(output)
}
