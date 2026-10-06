package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/gitexec"
	"github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestPlanRequiresExplicitEveryStep(t *testing.T) {
	root, _ := migrationWorkspace(t, 1)
	m := Migrator{
		TargetVersion: 3,
		Steps: map[int]Step{
			1: {From: 1, To: 2, Description: "one to two", Apply: func(string) error { return nil }},
		},
	}
	_, err := m.Plan(root)
	if err == nil || !strings.Contains(err.Error(), "no migration defined") {
		t.Fatalf("error = %v", err)
	}
}

func TestApplyUsesCleanRevisionAndRebuild(t *testing.T) {
	root, revision := migrationWorkspace(t, 1)
	rebuilt := false
	m := Migrator{
		TargetVersion: 2,
		Steps: map[int]Step{
			1: {
				From: 1, To: 2, Description: "upgrade",
				Apply: func(root string) error {
					return os.WriteFile(filepath.Join(root, "migrated.txt"), []byte("ok"), 0o644)
				},
			},
		},
		Rebuild: func(string) error { rebuilt = true; return nil },
	}
	plan, err := m.Apply(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) != 1 || !rebuilt {
		t.Fatalf("plan=%#v rebuilt=%v", plan, rebuilt)
	}
	version, err := workspaceSchemaVersion(root)
	if err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Fatalf("version = %d", version)
	}
}

func TestApplyRejectsDirtyWorkspace(t *testing.T) {
	root, revision := migrationWorkspace(t, 1)
	if err := os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Migrator{TargetVersion: 1}
	_, err := m.Apply(root, revision)
	if err == nil || !strings.Contains(err.Error(), "must be clean") {
		t.Fatalf("error = %v", err)
	}
}

func migrationWorkspace(t *testing.T, version int) (string, string) {
	t.Helper()
	root := t.TempDir()
	runMigrationGit(t, root, "init")
	runMigrationGit(t, root, "config", "user.email", "alp@example.invalid")
	runMigrationGit(t, root, "config", "user.name", "ALP Test")
	content := []byte("schemaVersion: " + strconv.Itoa(version) + "\nlearnerId: test\n")
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	runMigrationGit(t, root, "add", ".")
	runMigrationGit(t, root, "commit", "-m", "init")
	command := gitexec.Command(context.Background(), "", "-C", root, "rev-parse", "HEAD")
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	return root, strings.TrimSpace(string(output))
}

func runMigrationGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := gitexec.Command(context.Background(), "", append([]string{"-C", dir}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func TestApplyStagedMigratesV1LocalStoreToStableWorkspaceIdentity(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("schemaVersion: 1\nlearnerId: learner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	local, err := store.OpenLocal(root, validator)
	if err != nil {
		t.Fatal(err)
	}
	base, err := local.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	coordinator := store.NewCoordinator(local, validator, t.TempDir())
	tx, err := coordinator.Begin(context.Background(), "migrate-local", base)
	if err != nil {
		t.Fatal(err)
	}
	migrator := NewWorkspaceMigrator(validator)
	plan, err := migrator.ApplyStaged(tx.StageRoot())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Current != 1 || plan.Target != 2 {
		t.Fatalf("plan = %#v", plan)
	}
	if _, err := tx.CheckpointWithMessage(context.Background(), false, "alp: migrate local workspace"); err != nil {
		t.Fatal(err)
	}
	manifest, err := workspace.ReadManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 2 || manifest.WorkspaceID == "" || manifest.LearnerID != "learner" {
		t.Fatalf("manifest = %#v", manifest)
	}
}

func TestApplyStagedMigratesV1GitStoreAndCheckpointsIdentity(t *testing.T) {
	root, _ := migrationWorkspace(t, 1)
	validator, err := workspace.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	gitStore, err := store.OpenGit(root, validator)
	if err != nil {
		t.Fatal(err)
	}
	base, err := gitStore.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	coordinator := store.NewCoordinator(gitStore, validator, t.TempDir())
	tx, err := coordinator.Begin(context.Background(), "migrate-git", base)
	if err != nil {
		t.Fatal(err)
	}
	migrator := NewWorkspaceMigrator(validator)
	if _, err := migrator.ApplyStaged(tx.StageRoot()); err != nil {
		t.Fatal(err)
	}
	next, err := tx.CheckpointWithMessage(context.Background(), false, "alp: migrate git workspace")
	if err != nil {
		t.Fatal(err)
	}
	if next == base {
		t.Fatal("Git Store migration must advance the checkpoint revision")
	}
	manifest, err := workspace.ReadManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 2 || manifest.WorkspaceID == "" {
		t.Fatalf("manifest = %#v", manifest)
	}
}
