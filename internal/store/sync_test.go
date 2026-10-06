package store

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestGitSyncReconcilesIndependentAppendOnlyRecords(t *testing.T) {
	remote, a, b := makeTwoGitDevices(t)
	_ = remote
	ctx := context.Background()

	revA, _ := a.Revision(ctx)
	if _, err := a.Commit(ctx, revA, ChangeSet{Message: "session a", Mutations: []Mutation{{
		Path: "sessions/session-a.yaml", Data: []byte("id: session-a\n"),
	}}}); err != nil { t.Fatal(err) }
	if result, err := a.Push(ctx, SyncOptions{}); err != nil || result.Pending {
		t.Fatalf("push A result=%#v err=%v", result, err)
	}

	revB, _ := b.Revision(ctx)
	if _, err := b.Commit(ctx, revB, ChangeSet{Message: "session b", Mutations: []Mutation{{
		Path: "sessions/session-b.yaml", Data: []byte("id: session-b\n"),
	}}}); err != nil { t.Fatal(err) }

	result, err := b.Sync(ctx, SyncOptions{})
	if err != nil { t.Fatal(err) }
	if result.Pending { t.Fatalf("sync pending: %#v", result) }
	for _, name := range []string{"session-a.yaml", "session-b.yaml"} {
		if _, err := os.Stat(filepath.Join(b.Root(), "sessions", name)); err != nil {
			t.Fatalf("missing reconciled %s: %v", name, err)
		}
	}

	if _, err := a.Sync(ctx, SyncOptions{}); err != nil { t.Fatal(err) }
	for _, name := range []string{"session-a.yaml", "session-b.yaml"} {
		if _, err := os.Stat(filepath.Join(a.Root(), "sessions", name)); err != nil {
			t.Fatalf("device A missing %s after continuation sync: %v", name, err)
		}
	}
}

func TestGitSyncDiscardsDerivedConflictAndRebuilds(t *testing.T) {
	_, a, b := makeTwoGitDevices(t)
	ctx := context.Background()
	projectionA := []byte("schemaVersion: 1\ncompetencies: []\n")
	projectionB := []byte("schemaVersion: 1\ncompetencies:\n  - id: go.runtime.context\n    domain: go\n    level: functional\n    confidence: medium\n    assessmentId: asmt_x\n    lastVerified: 2026-10-06T00:00:00Z\n")

	revA, _ := a.Revision(ctx)
	if _, err := a.Commit(ctx, revA, ChangeSet{Message: "derived a", Mutations: []Mutation{{Path: "state/competencies.yaml", Data: projectionA}}}); err != nil { t.Fatal(err) }
	if _, err := a.Push(ctx, SyncOptions{}); err != nil { t.Fatal(err) }

	revB, _ := b.Revision(ctx)
	if _, err := b.Commit(ctx, revB, ChangeSet{Message: "derived b", Mutations: []Mutation{{Path: "state/competencies.yaml", Data: projectionB}}}); err != nil { t.Fatal(err) }

	rebuilt := []byte("schemaVersion: 1\ncompetencies: []\n")
	_, err := b.Sync(ctx, SyncOptions{Rebuild: func(root string) error {
		path := filepath.Join(root, "state", "competencies.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
		return os.WriteFile(path, rebuilt, 0o644)
	}})
	if err != nil { t.Fatal(err) }
	data, err := os.ReadFile(filepath.Join(b.Root(), "state", "competencies.yaml"))
	if err != nil { t.Fatal(err) }
	if string(data) != string(rebuilt) {
		t.Fatalf("derived state was not rebuilt:\n%s", data)
	}
}

func TestGitSyncStopsOnCompetingExplicitLearnerClaims(t *testing.T) {
	_, a, b := makeTwoGitDevicesWithProfile(t)
	ctx := context.Background()

	profileA := validProfile("strong")
	profileB := validProfile("expert")
	revA, _ := a.Revision(ctx)
	if _, err := a.Commit(ctx, revA, ChangeSet{Message: "profile a", Mutations: []Mutation{{Path: "profile/profile.yaml", Data: profileA}}}); err != nil { t.Fatal(err) }
	if _, err := a.Push(ctx, SyncOptions{}); err != nil { t.Fatal(err) }

	revB, _ := b.Revision(ctx)
	if _, err := b.Commit(ctx, revB, ChangeSet{Message: "profile b", Mutations: []Mutation{{Path: "profile/profile.yaml", Data: profileB}}}); err != nil { t.Fatal(err) }

	_, err := b.Sync(ctx, SyncOptions{})
	if err == nil || !strings.Contains(err.Error(), "learner resolution") {
		t.Fatalf("expected explicit learner conflict, got %v", err)
	}
}

func TestGitSyncNetworkFailureLeavesCheckpointPending(t *testing.T) {
	remote, a, _ := makeTwoGitDevices(t)
	ctx := context.Background()
	rev, _ := a.Revision(ctx)
	if _, err := a.Commit(ctx, rev, ChangeSet{Message: "offline", Mutations: []Mutation{{
		Path: "sessions/offline.yaml", Data: []byte("id: offline\n"),
	}}}); err != nil { t.Fatal(err) }
	if err := os.Rename(remote, remote+".offline"); err != nil { t.Fatal(err) }

	result, err := a.Sync(ctx, SyncOptions{})
	if err != nil { t.Fatal(err) }
	if !result.Pending {
		t.Fatalf("expected pending sync, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(a.Root(), "sessions", "offline.yaml")); err != nil {
		t.Fatal("offline checkpoint was lost")
	}
}

func makeTwoGitDevices(t *testing.T) (string, *Git, *Git) {
	t.Helper()
	return makeTwoGitDevicesFromProfile(t, nil)
}

func makeTwoGitDevicesWithProfile(t *testing.T) (string, *Git, *Git) {
	t.Helper()
	return makeTwoGitDevicesFromProfile(t, validProfile("functional"))
}

func makeTwoGitDevicesFromProfile(t *testing.T, profile []byte) (string, *Git, *Git) {
	t.Helper()
	source := t.TempDir()
	initGitStoreRepo(t, source)
	writeGitStoreFile(t, source, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_sync\nlearnerId: learner\n")
	if profile != nil {
		path := filepath.Join(source, "profile", "profile.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
		if err := os.WriteFile(path, profile, 0o644); err != nil { t.Fatal(err) }
	}
	runGitStore(t, source, "add", ".")
	runGitStore(t, source, "commit", "-m", "init")

	remote := filepath.Join(t.TempDir(), "state.git")
	cmd := exec.Command("git", "clone", "--bare", source, remote)
	if out, err := cmd.CombinedOutput(); err != nil { t.Fatalf("bare clone: %v\n%s", err, out) }

	validator, _ := workspace.NewValidator()
	aRoot := filepath.Join(t.TempDir(), "a")
	bRoot := filepath.Join(t.TempDir(), "b")
	a, err := CloneGit(context.Background(), remote, aRoot, "main", validator)
	if err != nil { t.Fatal(err) }
	b, err := CloneGit(context.Background(), remote, bRoot, "main", validator)
	if err != nil { t.Fatal(err) }
	for _, root := range []string{aRoot, bRoot} {
		runGitStore(t, root, "config", "user.email", "alp@example.invalid")
		runGitStore(t, root, "config", "user.name", "ALP Test")
	}
	return remote, a, b
}

func validProfile(level string) []byte {
	return []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience:\n  php:\n    level: " + level + "\n    provenance:\n      sourceType: learner-stated\n      intent: statement\ngoals: []\npreferences: {}\n")
}
