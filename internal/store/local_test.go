package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

func TestLocalStoreRevisionAndOptimisticCommit(t *testing.T) {
	root := t.TempDir()
	writeStoreFile(t, root, "workspace.yaml", "schemaVersion: 2
workspaceId: ws_test
learnerId: learner
")
	validator, err := workspace.NewValidator()
	if err != nil { t.Fatal(err) }
	s, err := OpenLocal(root, validator)
	if err != nil { t.Fatal(err) }

	ctx := context.Background()
	base, err := s.Revision(ctx)
	if err != nil { t.Fatal(err) }

	next, err := s.Commit(ctx, base, ChangeSet{Mutations: []Mutation{{
		Path: "profile/profile.yaml",
		Data: []byte("schemaVersion: 1
learner:
  id: learner
  professionalLevel: senior
experience: {}
goals: []
preferences: {}
"),
	}}})
	if err != nil { t.Fatal(err) }
	if next == base { t.Fatal("revision must change after canonical commit") }

	_, err = s.Commit(ctx, base, ChangeSet{Mutations: []Mutation{{
		Path: "profile/profile.yaml",
		Data: []byte("stale"),
	}}})
	if err == nil || !strings.Contains(err.Error(), "revision changed") {
		t.Fatalf("stale commit error = %v", err)
	}
}

func TestLocalStoreCapabilitiesAreHonest(t *testing.T) {
	root := t.TempDir()
	writeStoreFile(t, root, "workspace.yaml", "schemaVersion: 2
workspaceId: ws_test
learnerId: learner
")
	validator, _ := workspace.NewValidator()
	s, err := OpenLocal(root, validator)
	if err != nil { t.Fatal(err) }

	for _, capability := range []Capability{CapabilityPersistence, CapabilityOptimisticConcurrency, CapabilityAtomicCheckpoint, CapabilityRevisions, CapabilityOffline} {
		if !s.Capabilities().Has(capability) {
			t.Fatalf("missing capability %s", capability)
		}
	}
	for _, capability := range []Capability{CapabilitySync, CapabilityHistory, CapabilityMultiDevice, CapabilityRemoteManagement} {
		if s.Capabilities().Has(capability) {
			t.Fatalf("local store must not claim %s", capability)
		}
	}
	if err := Require(s, CapabilitySync); err == nil || !strings.Contains(err.Error(), "sync") {
		t.Fatalf("expected explicit capability error, got %v", err)
	}
}

func TestLocalStoreRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	writeStoreFile(t, root, "workspace.yaml", "schemaVersion: 2
workspaceId: ws_test
learnerId: learner
")
	validator, _ := workspace.NewValidator()
	s, _ := OpenLocal(root, validator)
	rev, _ := s.Revision(context.Background())
	_, err := s.Commit(context.Background(), rev, ChangeSet{Mutations: []Mutation{{Path: "../escape", Data: []byte("bad")}}})
	if err == nil {
		t.Fatal("expected unsafe path rejection")
	}
}

func writeStoreFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { t.Fatal(err) }
}

func TestLocalStoreRollsBackInvalidCheckpoint(t *testing.T) {
	root := t.TempDir()
	writeStoreFile(t, root, "workspace.yaml", "schemaVersion: 2\nworkspaceId: ws_test\nlearnerId: learner\n")
	writeStoreFile(t, root, "profile/profile.yaml", "schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")
	validator, _ := workspace.NewValidator()
	s, err := OpenLocal(root, validator)
	if err != nil { t.Fatal(err) }
	base, _ := s.Revision(context.Background())

	_, err = s.Commit(context.Background(), base, ChangeSet{Mutations: []Mutation{{
		Path: "profile/profile.yaml",
		Data: []byte("schemaVersion: 1\nlearner: {}\nexperience: {}\ngoals: []\npreferences: {}\n"),
	}}})
	if err == nil { t.Fatal("expected invalid checkpoint") }

	data, readErr := os.ReadFile(filepath.Join(root, "profile", "profile.yaml"))
	if readErr != nil { t.Fatal(readErr) }
	if !strings.Contains(string(data), "id: learner") {
		t.Fatalf("invalid checkpoint was not rolled back:\n%s", data)
	}
}
