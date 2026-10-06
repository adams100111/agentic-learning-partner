package workspacearchive

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	localstore "github.com/adams100111/agentic-learning-partner/internal/store/local"
)

func TestExportIsDeterministicAndExcludesDerivedState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "source")
	source, err := localstore.Initialize(root, "learner", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	revision, _ := source.Revision(ctx)
	tx, err := source.Begin(ctx, revision)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Put("profile/profile.yaml", []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")); err != nil {
		t.Fatal(err)
	}
	if err := tx.Put("state/competencies.yaml", []byte("schemaVersion: 1\ncompetencies: []\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(ctx, "seed"); err != nil {
		t.Fatal(err)
	}

	first := filepath.Join(t.TempDir(), "first.alp")
	second := filepath.Join(t.TempDir(), "second.alp")
	stamp := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	if _, err := Export(ctx, source, first, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err := Export(ctx, source, second, stamp); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(first)
	b, _ := os.ReadFile(second)
	if !bytes.Equal(a, b) {
		t.Fatal("exports with the same state/timestamp must be byte-for-byte deterministic")
	}
	verified, err := Verify(first, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := verified.Files["state/competencies.yaml"]; ok {
		t.Fatal("derived state must be excluded from canonical export")
	}
}

func TestVerifyRejectsCorruptedArchive(t *testing.T) {
	root := filepath.Join(t.TempDir(), "source")
	source, err := localstore.Initialize(root, "learner", nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "workspace.alp")
	if _, err := Export(context.Background(), source, path, time.Unix(0, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 16 {
		t.Fatal("archive unexpectedly small")
	}
	data[len(data)/2] ^= 0xff
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(path, nil); err == nil {
		t.Fatal("expected corrupted archive verification failure")
	}
}

func TestCloneRestoreCreatesNewWorkspaceIdentity(t *testing.T) {
	ctx := context.Background()
	sourceRoot := filepath.Join(t.TempDir(), "source")
	source, err := localstore.Initialize(sourceRoot, "learner", nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "workspace.alp")
	manifest, err := Export(ctx, source, path, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	verified, err := Verify(path, nil)
	if err != nil {
		t.Fatal(err)
	}

	targetRoot := filepath.Join(t.TempDir(), "target")
	target, err := localstore.Initialize(targetRoot, "learner", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Restore(ctx, verified, target, RestoreClone); err != nil {
		t.Fatal(err)
	}
	identity, err := target.Workspace(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ID == manifest.WorkspaceID {
		t.Fatal("clone restore must create a new workspaceId")
	}
}

func TestConvertLocalStoreToGitPreservesWorkspaceIdentity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := context.Background()
	sourceRoot := filepath.Join(t.TempDir(), "source")
	source, err := localstore.Initialize(sourceRoot, "learner", nil)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdentity, _ := source.Workspace(ctx)

	destination := filepath.Join(t.TempDir(), "git-workspace")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	runArchiveGit(t, destination, "config", "--global", "--add", "user.email", "alp-test@example.invalid")
	runArchiveGit(t, destination, "config", "--global", "--add", "user.name", "ALP Test")
	target, err := ConvertToGit(ctx, source, destination, "main", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	targetIdentity, err := target.Workspace(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if targetIdentity.ID != sourceIdentity.ID {
		t.Fatalf("converted workspace id = %s, want %s", targetIdentity.ID, sourceIdentity.ID)
	}
	if err := storepkg.Require(target.Provider(), target.Capabilities(), storepkg.CapabilitySync); err != nil {
		t.Fatal(err)
	}
}

func runArchiveGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
