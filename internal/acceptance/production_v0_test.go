package acceptance

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/gitexec"
	"github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"github.com/adams100111/agentic-learning-partner/internal/workspacearchive"
)

func TestProductionV0LocalToGitToSecondDeviceContinuation(t *testing.T) {
	ctx := context.Background()
	t.Setenv("GIT_AUTHOR_NAME", "ALP Acceptance")
	t.Setenv("GIT_AUTHOR_EMAIL", "alp-acceptance@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "ALP Acceptance")
	t.Setenv("GIT_COMMITTER_EMAIL", "alp-acceptance@example.invalid")

	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}

	localRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(localRoot, "workspace.yaml"), []byte("schemaVersion: 2\nworkspaceId: ws_acceptance\nlearnerId: learner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	local, err := store.OpenLocal(localRoot, validator)
	if err != nil {
		t.Fatal(err)
	}
	base, err := local.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	profile := []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")
	if _, err := local.Commit(ctx, base, store.ChangeSet{
		Mutations: []store.Mutation{{Path: "profile/profile.yaml", Data: profile}},
	}); err != nil {
		t.Fatal(err)
	}

	remote := filepath.Join(t.TempDir(), "state.git")
	runAcceptance(t, "", "git", "init", "--bare", remote)

	deviceAPath := filepath.Join(t.TempDir(), "device-a")
	deviceA, err := workspacearchive.ConvertToGit(ctx, local, validator, workspacearchive.ConvertToGitOptions{
		Destination: deviceAPath,
		Branch:      "main",
		Remote:      remote,
		RuntimeDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := deviceA.Push(ctx, store.SyncOptions{Branch: "main"}); err != nil || result.Pending {
		t.Fatalf("initial push result=%#v err=%v", result, err)
	}

	deviceBPath := filepath.Join(t.TempDir(), "device-b")
	deviceB, err := store.CloneGit(ctx, remote, deviceBPath, "main", validator)
	if err != nil {
		t.Fatal(err)
	}
	manifestB, err := workspace.ReadManifest(deviceB.Root())
	if err != nil {
		t.Fatal(err)
	}
	if manifestB.WorkspaceID != "ws_acceptance" || manifestB.LearnerID != "learner" {
		t.Fatalf("device B manifest = %#v", manifestB)
	}
	data, err := os.ReadFile(filepath.Join(deviceB.Root(), "profile", "profile.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(profile) {
		t.Fatalf("device B did not continue canonical learner state:\n%s", data)
	}

	runAcceptance(t, deviceB.Root(), "git", "config", "user.email", "alp-acceptance@example.invalid")
	runAcceptance(t, deviceB.Root(), "git", "config", "user.name", "ALP Acceptance")
	revB, err := deviceB.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	session := []byte("schemaVersion: 1\nid: sess_device_b\nstartedAt: 2026-10-06T12:00:00Z\nclosedAt: 2026-10-06T12:01:00Z\nbaseRevision: " + string(revB) + "\nsyncMode: session\nharness: codex\nsummary: Continued from synchronized learner state.\n")
	if _, err := deviceB.Commit(ctx, revB, store.ChangeSet{
		Message:   "alp: device b continuation",
		Mutations: []store.Mutation{{Path: "sessions/sess_device_b.yaml", Data: session}},
	}); err != nil {
		t.Fatal(err)
	}
	if result, err := deviceB.Sync(ctx, store.SyncOptions{Branch: "main"}); err != nil || result.Pending {
		t.Fatalf("device B sync result=%#v err=%v", result, err)
	}
	if _, err := deviceA.Sync(ctx, store.SyncOptions{Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(deviceA.Root(), "sessions", "sess_device_b.yaml")); err != nil {
		t.Fatalf("device A did not receive device B continuation: %v", err)
	}

	archive := filepath.Join(t.TempDir(), "workspace.alp")
	if _, err := workspacearchive.Export(ctx, deviceA, archive, time.Unix(0, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	verified, err := workspacearchive.Verify(archive, validator)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Manifest.WorkspaceID != "ws_acceptance" {
		t.Fatalf("archive workspace identity = %s", verified.Manifest.WorkspaceID)
	}
}

func runAcceptance(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	command := exec.Command(name, args...)
	command.Env = gitexec.Env()
	if dir != "" {
		command.Dir = dir
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
}
