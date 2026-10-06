package session

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestLocalSessionPublishesRebuildAndCompactRecord(t *testing.T) {
	root := t.TempDir()
	runtimeDir := t.TempDir()
	writeSessionFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_session\nlearnerId: learner\n")
	validator, _ := workspace.NewValidator()
	local, err := store.OpenLocal(root, validator)
	if err != nil { t.Fatal(err) }
	fixed := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	manager := &Manager{
		Store: local, Validator: validator, RuntimeDir: runtimeDir,
		Now: func() time.Time { return fixed },
		Rebuild: func(stage string) error {
			return writeSessionFileErr(stage, "state/competencies.yaml", "schemaVersion: 1\ncompetencies: []\n")
		},
	}
	s, err := manager.Begin(context.Background(), BeginOptions{ID: "sess_local", Harness: "codex", DeviceID: "dev_test"})
	if err != nil { t.Fatal(err) }
	profile := []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")
	if err := s.Put("profile/profile.yaml", profile); err != nil { t.Fatal(err) }
	result, err := s.Close(context.Background(), CloseInput{
		Domains: []string{"go"}, Evidence: []string{}, Assessments: []string{}, Summary: "bootstrap session",
	})
	if err != nil { t.Fatal(err) }
	if result.SyncPending { t.Fatal("local store must not report remote sync pending") }
	for _, name := range []string{"profile/profile.yaml", "state/competencies.yaml", "sessions/sess_local.yaml"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Fatalf("missing published %s: %v", name, err)
		}
	}
	sessionData, _ := os.ReadFile(filepath.Join(root, "sessions", "sess_local.yaml"))
	if string(sessionData) == "" || containsText(string(sessionData), "checkpointRevision") {
		t.Fatalf("session record should be compact and non-self-referential:\n%s", sessionData)
	}
}

func TestSessionRetentionCanBeDisabled(t *testing.T) {
	root := t.TempDir()
	writeSessionFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_no_record\nlearnerId: learner\n")
	validator, _ := workspace.NewValidator()
	local, _ := store.OpenLocal(root, validator)
	retain := false
	manager := &Manager{Store: local, Validator: validator, RuntimeDir: t.TempDir()}
	s, err := manager.Begin(context.Background(), BeginOptions{ID: "sess_private", Mode: SyncManual, RetainRecord: &retain})
	if err != nil { t.Fatal(err) }
	if err := s.Put("profile/profile.yaml", []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")); err != nil { t.Fatal(err) }
	if _, err := s.Close(context.Background(), CloseInput{}); err != nil { t.Fatal(err) }
	if _, err := os.Stat(filepath.Join(root, "sessions", "sess_private.yaml")); !os.IsNotExist(err) {
		t.Fatalf("session record should be absent when retention is disabled: %v", err)
	}
}

func TestGitSessionOfflineCloseKeepsPendingRecoveryAndLaterCompletes(t *testing.T) {
	ctx := context.Background()
	remote, gitStore := sessionGitFixture(t)
	validator, _ := workspace.NewValidator()
	runtimeDir := t.TempDir()
	manager := &Manager{Store: gitStore, Validator: validator, RuntimeDir: runtimeDir}
	s, err := manager.Begin(ctx, BeginOptions{ID: "sess_offline"})
	if err != nil { t.Fatal(err) }
	if err := s.Put("profile/profile.yaml", []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")); err != nil { t.Fatal(err) }

	offline := remote + ".offline"
	if err := os.Rename(remote, offline); err != nil { t.Fatal(err) }
	result, err := s.Close(ctx, CloseInput{})
	if err != nil { t.Fatal(err) }
	if !result.SyncPending { t.Fatalf("expected sync pending: %#v", result) }

	coordinator := store.NewCoordinator(gitStore, validator, runtimeDir)
	recovery, err := coordinator.InspectRecovery("ws_git_session")
	if err != nil { t.Fatal(err) }
	if recovery.Status != store.RecoverySyncPending {
		t.Fatalf("recovery status = %s", recovery.Status)
	}

	if err := os.Rename(offline, remote); err != nil { t.Fatal(err) }
	recovered, err := manager.RecoverPendingSync(ctx, store.SyncOptions{})
	if err != nil { t.Fatal(err) }
	if recovered.Pending { t.Fatalf("sync should complete after remote returns: %#v", recovered) }
	if _, err := coordinator.InspectRecovery("ws_git_session"); !os.IsNotExist(err) {
		t.Fatalf("completed sync should clear recovery journal: %v", err)
	}
}

func sessionGitFixture(t *testing.T) (string, *store.Git) {
	t.Helper()
	source := t.TempDir()
	runSessionGit(t, source, "init", "-b", "main")
	runSessionGit(t, source, "config", "user.email", "alp@example.invalid")
	runSessionGit(t, source, "config", "user.name", "ALP Test")
	writeSessionFile(t, source, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_git_session\nlearnerId: learner\n")
	runSessionGit(t, source, "add", "workspace.yaml")
	runSessionGit(t, source, "commit", "-m", "init")
	remote := filepath.Join(t.TempDir(), "state.git")
	cmd := exec.Command("git", "clone", "--bare", source, remote)
	if out, err := cmd.CombinedOutput(); err != nil { t.Fatalf("bare clone: %v\n%s", err, out) }

	validator, _ := workspace.NewValidator()
	clone := filepath.Join(t.TempDir(), "device")
	gitStore, err := store.CloneGit(context.Background(), remote, clone, "main", validator)
	if err != nil { t.Fatal(err) }
	runSessionGit(t, clone, "config", "user.email", "alp@example.invalid")
	runSessionGit(t, clone, "config", "user.name", "ALP Test")
	return remote, gitStore
}

func writeSessionFile(t *testing.T, root, name, content string) {
	t.Helper()
	if err := writeSessionFileErr(root, name, content); err != nil { t.Fatal(err) }
}

func writeSessionFileErr(root, name, content string) error {
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
	return os.WriteFile(path, []byte(content), 0o644)
}

func runSessionGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil { t.Fatalf("git %v: %v\n%s", args, err, out) }
	return string(out)
}

func containsText(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle { return true }
	}
	return false
}

func TestPersistedEagerSessionResumesAcrossManagerInstances(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runtimeDir := t.TempDir()
	writeSessionFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_eager_resume\nlearnerId: learner\n")
	validator, _ := workspace.NewValidator()
	local, err := store.OpenLocal(root, validator)
	if err != nil { t.Fatal(err) }

	managerA := &Manager{Store: local, Validator: validator, RuntimeDir: runtimeDir}
	started, err := managerA.Begin(ctx, BeginOptions{ID: "sess_eager_resume", Mode: SyncEager})
	if err != nil { t.Fatal(err) }
	if err := started.Put("profile/profile.yaml", []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")); err != nil {
		t.Fatal(err)
	}

	managerB := &Manager{Store: local, Validator: validator, RuntimeDir: runtimeDir}
	resumed, err := managerB.Resume(ctx)
	if err != nil { t.Fatal(err) }
	if resumed.ID() != "sess_eager_resume" {
		t.Fatalf("resumed session id = %s", resumed.ID())
	}
	if _, err := resumed.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "profile", "profile.yaml")); err != nil {
		t.Fatalf("eager flush did not publish checkpoint: %v", err)
	}

	managerC := &Manager{Store: local, Validator: validator, RuntimeDir: runtimeDir}
	afterFlush, err := managerC.Resume(ctx)
	if err != nil { t.Fatal(err) }
	persona := []byte("schemaVersion: 1\nscope: global\ndomain: null\nteaching:\n  pace: senior-dense\n")
	if err := afterFlush.Put("personas/global.yaml", persona); err != nil {
		t.Fatal(err)
	}
	if _, err := afterFlush.Close(ctx, CloseInput{Summary: "eager resume acceptance"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "personas", "global.yaml")); err != nil {
		t.Fatalf("resumed eager session did not publish final changes: %v", err)
	}
	if _, err := managerC.Resume(ctx); !os.IsNotExist(err) {
		t.Fatalf("closed session metadata should be removed, got %v", err)
	}
}
