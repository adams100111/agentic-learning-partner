package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
)

func TestLocalStoreTransactionPublishesAtomically(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	store, err := Initialize(root, "learner", func(root string) error {
		if _, err := os.Stat(filepath.Join(root, "invalid")); err == nil {
			return errors.New("invalid staged workspace")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	before, err := store.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := store.Begin(ctx, before)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Put("profile/profile.yaml", []byte("schemaVersion: 1\nlearner:\n  id: learner\nexperience: {}\ngoals: []\npreferences: {}\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "profile", "profile.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staged write leaked before commit: %v", err)
	}
	checkpoint, err := tx.Commit(ctx, "initialize profile")
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Revision == before {
		t.Fatal("revision did not change")
	}
	if _, err := os.Stat(filepath.Join(root, "profile", "profile.yaml")); err != nil {
		t.Fatal(err)
	}
}

func TestLocalStoreRejectsStaleRevision(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	store, err := Initialize(root, "learner", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Begin(context.Background(), storepkg.Revision("local:stale"))
	if err == nil || !errors.Is(err, storepkg.ErrConflict) {
		t.Fatalf("error = %v", err)
	}
}

func TestLocalStoreOneWriterLock(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	store, err := Initialize(root, "learner", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	revision, _ := store.Revision(ctx)
	first, err := store.Begin(ctx, revision)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Rollback()

	secondStore, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = secondStore.Begin(ctx, revision)
	if err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("second writer error = %v", err)
	}
}

func TestLocalStoreDoesNotAdvertiseSync(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	store, err := Initialize(root, "learner", nil)
	if err != nil {
		t.Fatal(err)
	}
	if store.Capabilities().Has(storepkg.CapabilitySync) {
		t.Fatal("local store must not claim sync")
	}
}
