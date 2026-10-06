package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

type Validator func(root string) error

type Store struct {
	root      string
	validator Validator
	mu        sync.Mutex
}

func Open(root string, validator Validator) (*Store, error) {
	manifest, err := workspace.ReadManifest(root)
	if err != nil {
		return nil, err
	}
	if manifest.SchemaVersion != workspace.CurrentSchemaVersion {
		return nil, fmt.Errorf("workspace schema %d is not current schema %d", manifest.SchemaVersion, workspace.CurrentSchemaVersion)
	}
	s := &Store{root: root, validator: validator}
	if validator != nil {
		if err := validator(root); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func Initialize(root, learnerID string, validator Validator) (*Store, error) {
	if strings.TrimSpace(learnerID) == "" {
		return nil, errors.New("learner id is required")
	}
	id, err := workspace.NewWorkspaceID()
	if err != nil {
		return nil, err
	}
	return InitializeWithManifest(root, workspace.Manifest{
		SchemaVersion: workspace.CurrentSchemaVersion,
		WorkspaceID: id,
		LearnerID: learnerID,
	}, validator)
}

func InitializeWithManifest(root string, manifest workspace.Manifest, validator Validator) (*Store, error) {
	if manifest.SchemaVersion != workspace.CurrentSchemaVersion || manifest.WorkspaceID == "" || manifest.LearnerID == "" {
		return nil, errors.New("valid current workspace manifest is required")
	}
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		entries, readErr := os.ReadDir(root)
		if readErr != nil {
			return nil, readErr
		}
		if len(entries) != 0 {
			return nil, fmt.Errorf("workspace directory %s is not empty", root)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := workspace.WriteManifest(root, manifest); err != nil {
		return nil, err
	}
	return Open(root, validator)
}

func (s *Store) Provider() string { return "local" }

func (s *Store) Capabilities() storepkg.Capabilities {
	return storepkg.Capabilities{
		storepkg.CapabilityPersistence:           true,
		storepkg.CapabilityOptimisticConcurrency: true,
		storepkg.CapabilityAtomicCheckpoint:      true,
		storepkg.CapabilityOffline:               true,
	}
}

func (s *Store) Workspace(context.Context) (storepkg.Workspace, error) {
	manifest, err := workspace.ReadManifest(s.root)
	if err != nil {
		return storepkg.Workspace{}, err
	}
	return storepkg.Workspace{ID: manifest.WorkspaceID, LearnerID: manifest.LearnerID, Path: s.root}, nil
}

func (s *Store) Revision(context.Context) (storepkg.Revision, error) {
	revision, err := snapshotRevision(s.root)
	if err != nil {
		return "", err
	}
	return storepkg.Revision("local:" + revision), nil
}

func (s *Store) Read(_ context.Context, path string) ([]byte, error) {
	clean, err := safeRelative(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(s.root, clean))
}

func (s *Store) List(_ context.Context, prefix string) ([]storepkg.Entry, error) {
	clean, err := safeRelativeAllowEmpty(prefix)
	if err != nil {
		return nil, err
	}
	base := filepath.Join(s.root, clean)
	var entries []storepkg.Entry
	err = filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		if isRuntimePath(rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		entries = append(entries, storepkg.Entry{Path: filepath.ToSlash(rel), Mode: info.Mode()})
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, err
}

func (s *Store) Begin(ctx context.Context, expected storepkg.Revision) (storepkg.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.Revision(ctx)
	if err != nil {
		return nil, err
	}
	if expected != "" && current != expected {
		return nil, fmt.Errorf("%w: expected %s, found %s", storepkg.ErrConflict, expected, current)
	}

	lock, err := acquireLock(s.root)
	if err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(filepath.Dir(s.root), "."+filepath.Base(s.root)+".alp-tx-*")
	if err != nil {
		lock.Release()
		return nil, fmt.Errorf("create transaction stage: %w", err)
	}
	if err := copyTree(s.root, stage); err != nil {
		os.RemoveAll(stage)
		lock.Release()
		return nil, fmt.Errorf("stage workspace: %w", err)
	}
	return &transaction{
		store:    s,
		expected: current,
		stage:    stage,
		lock:     lock,
	}, nil
}

type transaction struct {
	store     *Store
	expected  storepkg.Revision
	stage     string
	lock      *writeLock
	changes   []storepkg.Change
	finished  bool
}

func (t *transaction) Put(path string, data []byte) error {
	if t.finished {
		return errors.New("transaction already finished")
	}
	clean, err := safeRelative(path)
	if err != nil {
		return err
	}
	target := filepath.Join(t.stage, clean)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return err
	}
	t.changes = append(t.changes, storepkg.Change{Path: filepath.ToSlash(clean), Data: append([]byte(nil), data...)})
	return nil
}

func (t *transaction) Delete(path string) error {
	if t.finished {
		return errors.New("transaction already finished")
	}
	clean, err := safeRelative(path)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(t.stage, clean)); err != nil {
		return err
	}
	t.changes = append(t.changes, storepkg.Change{Path: filepath.ToSlash(clean), Delete: true})
	return nil
}

func (t *transaction) Changes() []storepkg.Change {
	return append([]storepkg.Change(nil), t.changes...)
}

func (t *transaction) Commit(ctx context.Context, summary string) (storepkg.Checkpoint, error) {
	if t.finished {
		return storepkg.Checkpoint{}, errors.New("transaction already finished")
	}
	defer t.finish()

	current, err := t.store.Revision(ctx)
	if err != nil {
		return storepkg.Checkpoint{}, err
	}
	if current != t.expected {
		return storepkg.Checkpoint{}, fmt.Errorf("%w: expected %s, found %s", storepkg.ErrConflict, t.expected, current)
	}
	if t.store.validator != nil {
		if err := t.store.validator(t.stage); err != nil {
			return storepkg.Checkpoint{}, err
		}
	}

	backup := t.store.root + ".alp-backup-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.Rename(t.store.root, backup); err != nil {
		return storepkg.Checkpoint{}, fmt.Errorf("move current workspace aside: %w", err)
	}
	if err := os.Rename(t.stage, t.store.root); err != nil {
		_ = os.Rename(backup, t.store.root)
		return storepkg.Checkpoint{}, fmt.Errorf("publish transaction: %w", err)
	}
	t.stage = ""
	if err := os.RemoveAll(backup); err != nil {
		return storepkg.Checkpoint{}, fmt.Errorf("remove transaction backup: %w", err)
	}
	revision, err := t.store.Revision(ctx)
	if err != nil {
		return storepkg.Checkpoint{}, err
	}
	return storepkg.Checkpoint{Revision: revision, Summary: summary}, nil
}

func (t *transaction) Rollback() error {
	if t.finished {
		return nil
	}
	t.finish()
	return nil
}

func (t *transaction) finish() {
	t.finished = true
	if t.stage != "" {
		_ = os.RemoveAll(t.stage)
		t.stage = ""
	}
	if t.lock != nil {
		_ = t.lock.Release()
		t.lock = nil
	}
}

type LockInfo struct {
	PID       int
	StartedAt time.Time
	Path      string
}

type LockError struct {
	Info LockInfo
}

func (e LockError) Error() string {
	return fmt.Sprintf("workspace is locked by writer pid=%d startedAt=%s", e.Info.PID, e.Info.StartedAt.UTC().Format(time.RFC3339))
}

func (e LockError) RecoverableAfter(maxAge time.Duration, now time.Time) bool {
	if e.Info.StartedAt.IsZero() {
		return false
	}
	return now.Sub(e.Info.StartedAt) > maxAge
}

type writeLock struct {
	path string
}

func acquireLock(root string) (*writeLock, error) {
	manifest, err := workspace.ReadManifest(root)
	if err != nil {
		return nil, err
	}
	runtimeDir := filepath.Join(filepath.Dir(root), ".alp-runtime", manifest.WorkspaceID)
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(runtimeDir, "write.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			info := LockInfo{Path: path}
			data, _ := os.ReadFile(path)
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "pid=") {
					info.PID, _ = strconv.Atoi(strings.TrimPrefix(line, "pid="))
				}
				if strings.HasPrefix(line, "startedAt=") {
					info.StartedAt, _ = time.Parse(time.RFC3339, strings.TrimPrefix(line, "startedAt="))
				}
			}
			return nil, LockError{Info: info}
		}
		return nil, err
	}
	_, writeErr := fmt.Fprintf(file, "pid=%d\nstartedAt=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(path)
		return nil, writeErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return nil, closeErr
	}
	return &writeLock{path: path}, nil
}

func (l *writeLock) Release() error {
	if l == nil || l.path == "" {
		return nil
	}
	return os.Remove(l.path)
}

func snapshotRevision(root string) (string, error) {
	h := sha256.New()
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if isRuntimePath(rel) {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return "", err
		}
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write([]byte{0})
		h.Write(data)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." || isRuntimePath(rel) {
			return nil
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

func safeRelative(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is required")
	}
	return safeRelativeAllowEmpty(path)
}

func safeRelativeAllowEmpty(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace", path)
	}
	return clean, nil
}

func isRuntimePath(path string) bool {
	clean := filepath.ToSlash(path)
	return clean == ".alp-runtime" || strings.HasPrefix(clean, ".alp-runtime/")
}
