package workspacearchive

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestExportIsDeterministicAndCanonicalOnly(t *testing.T) {
	ctx := context.Background()
	validator, _ := workspace.NewValidator()
	root := makeArchiveWorkspace(t, "ws_source", "learner")
	source, err := store.OpenLocal(root, validator)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := source.Revision(ctx)
	if _, err := source.Commit(ctx, base, store.ChangeSet{Mutations: []store.Mutation{
		{Path: "profile/profile.yaml", Data: validArchiveProfile()},
		{Path: "state/competencies.yaml", Data: []byte("schemaVersion: 1\ncompetencies: []\n")},
	}}); err != nil {
		t.Fatal(err)
	}

	stamp := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	first := filepath.Join(t.TempDir(), "first.alp")
	second := filepath.Join(t.TempDir(), "second.alp")
	if _, err := Export(ctx, source, first, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err := Export(ctx, source, second, stamp); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(first)
	b, _ := os.ReadFile(second)
	if !bytes.Equal(a, b) {
		t.Fatal("same canonical state and timestamp must produce identical archive bytes")
	}
	verified, err := Verify(first, validator)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := verified.Files["state/competencies.yaml"]; ok {
		t.Fatal("derived state must not be exported")
	}
	if _, ok := verified.Files["profile/profile.yaml"]; !ok {
		t.Fatal("canonical profile is missing from export")
	}
}

func TestVerifyRejectsCorruptedArchive(t *testing.T) {
	validator, _ := workspace.NewValidator()
	source, _ := store.OpenLocal(makeArchiveWorkspace(t, "ws_source", "learner"), validator)
	path := filepath.Join(t.TempDir(), "workspace.alp")
	if _, err := Export(context.Background(), source, path, time.Unix(0, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	// Corrupt entry content, not zip header bytes the reader never consults.
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	offset, err := reader.File[len(reader.File)-1].DataOffset()
	if err != nil {
		t.Fatal(err)
	}
	data[offset] ^= 0xff
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(path, validator); err == nil {
		t.Fatal("corrupted archive must fail verification")
	}
}

func TestCloneRestoreCreatesNewWorkspaceIdentityAndRebuilds(t *testing.T) {
	ctx := context.Background()
	validator, _ := workspace.NewValidator()
	sourceRoot := makeArchiveWorkspace(t, "ws_source", "learner")
	source, _ := store.OpenLocal(sourceRoot, validator)
	base, _ := source.Revision(ctx)
	if _, err := source.Commit(ctx, base, store.ChangeSet{Mutations: []store.Mutation{{Path: "profile/profile.yaml", Data: validArchiveProfile()}}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "workspace.alp")
	if _, err := Export(ctx, source, path, time.Unix(0, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	verified, err := Verify(path, validator)
	if err != nil {
		t.Fatal(err)
	}

	targetRoot := makeArchiveWorkspace(t, "ws_clone", "learner")
	target, _ := store.OpenLocal(targetRoot, validator)
	_, err = Restore(ctx, verified, target, validator, RestoreOptions{
		Mode: RestoreClone, RuntimeDir: t.TempDir(),
		Rebuild: func(root string) error {
			path := filepath.Join(root, "state", "competencies.yaml")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			return os.WriteFile(path, []byte("schemaVersion: 1\ncompetencies: []\n"), 0o644)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := workspace.ReadManifest(targetRoot)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.WorkspaceID != "ws_clone" {
		t.Fatalf("clone restore changed target workspace identity: %s", manifest.WorkspaceID)
	}
	if _, err := os.Stat(filepath.Join(targetRoot, "state", "competencies.yaml")); err != nil {
		t.Fatal("derived state was not rebuilt during restore")
	}
}

func TestMergeRestoreUsesSemanticReconciliation(t *testing.T) {
	ctx := context.Background()
	validator, _ := workspace.NewValidator()

	sourceRoot := makeArchiveWorkspace(t, "ws_source", "learner")
	source, _ := store.OpenLocal(sourceRoot, validator)
	base, _ := source.Revision(ctx)
	sourceProfile := []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience:\n  go:\n    level: functional\n    provenance:\n      sourceType: learner-stated\n      intent: statement\ngoals: []\npreferences: {}\n")
	if _, err := source.Commit(ctx, base, store.ChangeSet{Mutations: []store.Mutation{{Path: "profile/profile.yaml", Data: sourceProfile}}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "merge.alp")
	if _, err := Export(ctx, source, path, time.Unix(0, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	verified, _ := Verify(path, validator)

	targetRoot := makeArchiveWorkspace(t, "ws_target", "learner")
	target, _ := store.OpenLocal(targetRoot, validator)
	targetBase, _ := target.Revision(ctx)
	targetProfile := []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience:\n  php:\n    level: strong\n    provenance:\n      sourceType: learner-stated\n      intent: statement\ngoals: []\npreferences: {}\n")
	if _, err := target.Commit(ctx, targetBase, store.ChangeSet{Mutations: []store.Mutation{{Path: "profile/profile.yaml", Data: targetProfile}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Restore(ctx, verified, target, validator, RestoreOptions{Mode: RestoreMerge, RuntimeDir: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(targetRoot, "profile", "profile.yaml"))
	if !bytes.Contains(data, []byte("php:")) || !bytes.Contains(data, []byte("go:")) {
		t.Fatalf("semantic merge lost independent claims:\n%s", data)
	}
}

func TestConvertLocalToGitPreservesWorkspaceIdentity(t *testing.T) {
	ctx := context.Background()
	validator, _ := workspace.NewValidator()
	sourceRoot := makeArchiveWorkspace(t, "ws_convert", "learner")
	source, _ := store.OpenLocal(sourceRoot, validator)
	destination := filepath.Join(t.TempDir(), "git")
	t.Setenv("GIT_AUTHOR_NAME", "ALP Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "alp-test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "ALP Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "alp-test@example.invalid")
	target, err := ConvertToGit(ctx, source, validator, ConvertToGitOptions{
		Destination: destination, Branch: "main", RuntimeDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := workspace.ReadManifest(target.Root())
	if err != nil {
		t.Fatal(err)
	}
	if manifest.WorkspaceID != "ws_convert" {
		t.Fatalf("conversion changed workspace identity: %s", manifest.WorkspaceID)
	}
	if !target.Capabilities().Has(store.CapabilitySync) {
		t.Fatal("converted Git Store must support synchronization")
	}
}

func makeArchiveWorkspace(t *testing.T, id, learner string) string {
	t.Helper()
	root := t.TempDir()
	data := []byte("schemaVersion: 2\nworkspaceId: " + id + "\nlearnerId: " + learner + "\n")
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func validArchiveProfile() []byte {
	return []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")
}
