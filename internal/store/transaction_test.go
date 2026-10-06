package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestCoordinatorStagesValidatesAndPublishesOneWriter(t *testing.T) {
	root := t.TempDir()
	runtimeDir := t.TempDir()
	writeTxnFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_txn\nlearnerId: learner\n")

	validator, err := workspace.NewValidator()
	if err != nil { t.Fatal(err) }
	local, err := OpenLocal(root, validator)
	if err != nil { t.Fatal(err) }
	base, err := local.Revision(context.Background())
	if err != nil { t.Fatal(err) }

	coordinator := NewCoordinator(local, validator, runtimeDir)
	tx, err := coordinator.Begin(context.Background(), "session-one", base)
	if err != nil { t.Fatal(err) }

	if _, err := coordinator.Begin(context.Background(), "session-two", base); err == nil || !strings.Contains(err.Error(), "already has a writer") {
		t.Fatalf("second writer error = %v", err)
	}

	profile := []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")
	if err := tx.Put("profile/profile.yaml", profile); err != nil { t.Fatal(err) }
	next, err := tx.Commit(context.Background())
	if err != nil { t.Fatal(err) }
	if next == base { t.Fatal("checkpoint must advance revision") }

	data, err := os.ReadFile(filepath.Join(root, "profile", "profile.yaml"))
	if err != nil { t.Fatal(err) }
	if string(data) != string(profile) { t.Fatalf("profile = %q", data) }

	if _, err := coordinator.InspectRecovery("ws_txn"); !os.IsNotExist(err) {
		t.Fatalf("completed local transaction should clear recovery journal, got %v", err)
	}
}

func TestInvalidStagedWorkspaceDoesNotPublishAndCanBeDiscarded(t *testing.T) {
	root := t.TempDir()
	runtimeDir := t.TempDir()
	writeTxnFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_txn\nlearnerId: learner\n")
	validator, _ := workspace.NewValidator()
	local, _ := OpenLocal(root, validator)
	base, _ := local.Revision(context.Background())
	coordinator := NewCoordinator(local, validator, runtimeDir)

	tx, err := coordinator.Begin(context.Background(), "session-bad", base)
	if err != nil { t.Fatal(err) }
	if err := tx.Put("profile/profile.yaml", []byte("schemaVersion: 1\nlearner: {}\nexperience: {}\ngoals: []\npreferences: {}\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(context.Background()); err == nil {
		t.Fatal("expected invalid staged workspace to fail")
	}
	if _, err := os.Stat(filepath.Join(root, "profile", "profile.yaml")); !os.IsNotExist(err) {
		t.Fatal("invalid staged profile must not be published")
	}

	recovery, err := coordinator.InspectRecovery("ws_txn")
	if err != nil { t.Fatal(err) }
	if recovery.Status != RecoveryStaged || recovery.SessionID != "session-bad" {
		t.Fatalf("recovery = %#v", recovery)
	}
	if err := coordinator.DiscardRecovery("ws_txn"); err != nil { t.Fatal(err) }
	if _, err := coordinator.Begin(context.Background(), "session-next", base); err != nil {
		t.Fatalf("discard should release writer lock: %v", err)
	}
}

func TestCheckpointedPendingSyncRecoveryIsPreserved(t *testing.T) {
	root := t.TempDir()
	runtimeDir := t.TempDir()
	writeTxnFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_txn\nlearnerId: learner\n")
	validator, _ := workspace.NewValidator()
	local, _ := OpenLocal(root, validator)
	base, _ := local.Revision(context.Background())
	coordinator := NewCoordinator(local, validator, runtimeDir)

	tx, err := coordinator.Begin(context.Background(), "session-sync", base)
	if err != nil { t.Fatal(err) }
	if err := tx.Put("profile/profile.yaml", []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")); err != nil { t.Fatal(err) }
	next, err := tx.Checkpoint(context.Background(), true)
	if err != nil { t.Fatal(err) }
	if next == base { t.Fatal("revision did not advance") }

	recovery, err := coordinator.InspectRecovery("ws_txn")
	if err != nil { t.Fatal(err) }
	if recovery.Status != RecoverySyncPending || recovery.CheckpointRevision != next {
		t.Fatalf("recovery = %#v", recovery)
	}
}

func writeTxnFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { t.Fatal(err) }
}
