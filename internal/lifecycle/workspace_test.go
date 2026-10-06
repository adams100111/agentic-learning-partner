package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/gitexec"
	"github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestInitListUseAndStatusNamedLocalWorkspace(t *testing.T) {
	validator, _ := workspace.NewValidator()
	manager := New(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir(), validator)
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "personal")

	status, err := manager.Init(ctx, "personal", "local", root, "learner", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if status.Provider != "local" || !status.Default || status.WorkspaceID == "" {
		t.Fatalf("status = %#v", status)
	}
	items, err := manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "personal" {
		t.Fatalf("items = %#v", items)
	}
	if err := manager.Use("personal"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := manager.Status(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Name != "personal" || reloaded.Path != root {
		t.Fatalf("reloaded = %#v", reloaded)
	}
}

func TestLocalWorkspaceSyncReturnsActionableCapabilityError(t *testing.T) {
	validator, _ := workspace.NewValidator()
	manager := New(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir(), validator)
	if _, err := manager.Init(context.Background(), "local", "local", filepath.Join(t.TempDir(), "local"), "learner", "", "", false); err != nil {
		t.Fatal(err)
	}
	_, err := manager.Sync(context.Background(), "local", "sync")
	if err == nil || !strings.Contains(err.Error(), "move this workspace to provider git") {
		t.Fatalf("sync error = %v", err)
	}
}

func TestCloneRequiresPrivacyAcknowledgementAndReportsGitStatus(t *testing.T) {
	remote := lifecycleRemote(t)
	validator, _ := workspace.NewValidator()
	manager := New(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir(), validator)
	destination := filepath.Join(t.TempDir(), "clone")

	if _, err := manager.Clone(context.Background(), "synced", remote, destination, "main", false); err == nil {
		t.Fatal("clone should require explicit privacy acknowledgement for unverifiable remote")
	}
	status, err := manager.Clone(context.Background(), "synced", remote, destination, "main", true)
	if err != nil {
		t.Fatal(err)
	}
	if status.Provider != "git" || status.SyncMode != "session" || status.Branch != "main" || !status.PrivacyAck {
		t.Fatalf("status = %#v", status)
	}
}

func TestMoveLocalToGitSwitchesConfigOnlyAfterSuccessfulConversion(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "ALP Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "alp@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "ALP Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "alp@example.invalid")

	validator, _ := workspace.NewValidator()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	manager := New(configPath, t.TempDir(), validator)
	ctx := context.Background()
	source := filepath.Join(t.TempDir(), "source")
	initial, err := manager.Init(ctx, "personal", "local", source, "learner", "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	active, _, _, _, err := manager.open("personal")
	if err != nil {
		t.Fatal(err)
	}
	base, _ := active.Revision(ctx)
	if _, err := active.Commit(ctx, base, store.ChangeSet{Mutations: []store.Mutation{{
		Path: "profile/profile.yaml",
		Data: []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n"),
	}}}); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(t.TempDir(), "git")
	moved, err := manager.Move(ctx, "personal", "git", destination, "", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	if moved.Provider != "git" || moved.WorkspaceID != initial.WorkspaceID {
		t.Fatalf("moved = %#v initial = %#v", moved, initial)
	}
	config, err := workspace.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if config.Workspaces["personal"].Path != destination || config.Workspaces["personal"].Provider != "git" {
		t.Fatalf("config = %#v", config.Workspaces["personal"])
	}
	if _, err := os.Stat(filepath.Join(destination, "profile", "profile.yaml")); err != nil {
		t.Fatalf("profile missing after conversion: %v", err)
	}
}

func lifecycleRemote(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	runLifecycleGit(t, source, "init", "-b", "main")
	runLifecycleGit(t, source, "config", "user.email", "alp@example.invalid")
	runLifecycleGit(t, source, "config", "user.name", "ALP Test")
	if err := os.WriteFile(filepath.Join(source, "workspace.yaml"), []byte("schemaVersion: 2\nworkspaceId: ws_remote\nlearnerId: learner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runLifecycleGit(t, source, "add", "workspace.yaml")
	runLifecycleGit(t, source, "commit", "-m", "init")
	remote := filepath.Join(t.TempDir(), "state.git")
	cmd := gitexec.Command(context.Background(), "", "clone", "--bare", source, remote)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bare clone: %v\n%s", err, out)
	}
	return remote
}

func runLifecycleGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := gitexec.Command(context.Background(), "", append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestConnectExistingLocalWorkspaceWithoutCopyingState(t *testing.T) {
	validator, _ := workspace.NewValidator()
	root := makeArchiveCompatibleLocalWorkspace(t, "ws_existing", "learner")
	profilePath := filepath.Join(root, "profile", "profile.yaml")
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		t.Fatal(err)
	}
	profile := []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")
	if err := os.WriteFile(profilePath, profile, 0o644); err != nil {
		t.Fatal(err)
	}

	manager := New(filepath.Join(t.TempDir(), "config.yaml"), t.TempDir(), validator)
	status, err := manager.Connect(context.Background(), "existing", "local", root, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if status.WorkspaceID != "ws_existing" || status.Provider != "local" {
		t.Fatalf("status = %#v", status)
	}
	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(profile) {
		t.Fatal("connecting existing state must not rewrite learner data")
	}
}

func makeArchiveCompatibleLocalWorkspace(t *testing.T, id, learner string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 2\nworkspaceId: "+id+"\nlearnerId: "+learner+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}
