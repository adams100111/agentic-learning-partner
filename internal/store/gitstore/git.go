package gitstore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

type Validator func(root string) error

type Store struct {
	root      string
	branch    string
	validator Validator
}

type fileBackup struct {
	path   string
	data   []byte
	mode   fs.FileMode
	exists bool
}

func Open(root, branch string, validator Validator) (*Store, error) {
	if branch == "" {
		branch = "main"
	}
	manifest, err := workspace.ReadManifest(root)
	if err != nil {
		return nil, err
	}
	if manifest.SchemaVersion != workspace.CurrentSchemaVersion {
		return nil, fmt.Errorf("workspace schema %d is not current schema %d", manifest.SchemaVersion, workspace.CurrentSchemaVersion)
	}
	top, err := git(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("open git store: %w", err)
	}
	absoluteRoot, _ := filepath.Abs(root)
	absoluteTop, _ := filepath.Abs(strings.TrimSpace(top))
	if filepath.Clean(absoluteRoot) != filepath.Clean(absoluteTop) {
		return nil, fmt.Errorf("git store workspace must be repository root")
	}
	s := &Store{root: root, branch: branch, validator: validator}
	if validator != nil {
		if err := validator(root); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func Initialize(root, learnerID, branch string, validator Validator) (*Store, error) {
	if branch == "" {
		branch = "main"
	}
	if strings.TrimSpace(learnerID) == "" {
		return nil, errors.New("learner id is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	if len(entries) != 0 {
		return nil, fmt.Errorf("workspace directory %s is not empty", root)
	}
	if _, err := git(root, "init", "-b", branch); err != nil {
		return nil, err
	}
	id, err := workspace.NewWorkspaceID()
	if err != nil {
		return nil, err
	}
	if err := workspace.WriteManifest(root, workspace.Manifest{
		SchemaVersion: workspace.CurrentSchemaVersion,
		WorkspaceID: id,
		LearnerID: learnerID,
	}); err != nil {
		return nil, err
	}
	if validator != nil {
		if err := validator(root); err != nil {
			return nil, err
		}
	}
	if _, err := git(root, "add", "--", "workspace.yaml"); err != nil {
		return nil, err
	}
	if _, err := git(root, "commit", "-m", "state: initialize ALP workspace"); err != nil {
		return nil, err
	}
	return Open(root, branch, validator)
}

func Clone(remote, destination, branch string, validator Validator) (*Store, error) {
	if branch == "" {
		branch = "main"
	}
	if strings.TrimSpace(remote) == "" {
		return nil, errors.New("git remote is required")
	}
	args := []string{"clone", "--branch", branch, "--single-branch", remote, destination}
	command := exec.Command("git", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git clone: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return Open(destination, branch, validator)
}

func (s *Store) Provider() string { return "git" }

func (s *Store) Capabilities() storepkg.Capabilities {
	return storepkg.Capabilities{
		storepkg.CapabilityPersistence:           true,
		storepkg.CapabilityOptimisticConcurrency: true,
		storepkg.CapabilityAtomicCheckpoint:      true,
		storepkg.CapabilityRevisions:             true,
		storepkg.CapabilityHistory:               true,
		storepkg.CapabilityOffline:               true,
		storepkg.CapabilitySync:                  true,
		storepkg.CapabilityMultiDevice:           true,
		storepkg.CapabilityRemoteManagement:      true,
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
	value, err := git(s.root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return storepkg.Revision(strings.TrimSpace(value)), nil
}

func (s *Store) Read(_ context.Context, path string) ([]byte, error) {
	clean, err := safeOwnedPath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(s.root, clean))
}

func (s *Store) List(_ context.Context, prefix string) ([]storepkg.Entry, error) {
	clean := filepath.Clean(filepath.FromSlash(prefix))
	if prefix == "" {
		clean = "."
	}
	base := filepath.Join(s.root, clean)
	var result []storepkg.Entry
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		if !IsOwnedPath(rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		result = append(result, storepkg.Entry{Path: filepath.ToSlash(rel), Mode: info.Mode()})
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, err
}

func (s *Store) Begin(ctx context.Context, expected storepkg.Revision) (storepkg.Transaction, error) {
	current, err := s.Revision(ctx)
	if err != nil {
		return nil, err
	}
	if expected != "" && current != expected {
		return nil, fmt.Errorf("%w: expected %s, found %s", storepkg.ErrConflict, expected, current)
	}
	stage, err := os.MkdirTemp("", "alp-git-tx-*")
	if err != nil {
		return nil, err
	}
	if err := copyOwned(s.root, stage); err != nil {
		os.RemoveAll(stage)
		return nil, err
	}
	return &transaction{store: s, expected: current, stage: stage}, nil
}

func (s *Store) History(_ context.Context, limit int) ([]storepkg.HistoryEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	output, err := git(s.root, "log", fmt.Sprintf("-%d", limit), "--format=%H%x00%s")
	if err != nil {
		return nil, err
	}
	var result []storepkg.HistoryEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 2)
		entry := storepkg.HistoryEntry{Revision: storepkg.Revision(parts[0])}
		if len(parts) == 2 {
			entry.Summary = parts[1]
		}
		result = append(result, entry)
	}
	return result, nil
}

type transaction struct {
	store    *Store
	expected storepkg.Revision
	stage    string
	changes  []storepkg.Change
	finished bool
}

func (t *transaction) Put(path string, data []byte) error {
	if t.finished {
		return errors.New("transaction already finished")
	}
	clean, err := safeOwnedPath(path)
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
	clean, err := safeOwnedPath(path)
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
	if len(t.changes) == 0 {
		return storepkg.Checkpoint{Revision: current, Summary: summary}, nil
	}

	backups := make(map[string]fileBackup, len(t.changes))
	var paths []string
	for _, change := range t.changes {
		clean, _ := safeOwnedPath(change.Path)
		target := filepath.Join(t.store.root, clean)
		info, statErr := os.Stat(target)
		if statErr == nil && !info.IsDir() {
			data, readErr := os.ReadFile(target)
			if readErr != nil {
				return storepkg.Checkpoint{}, readErr
			}
			backups[clean] = fileBackup{path: target, data: data, mode: info.Mode(), exists: true}
		} else {
			backups[clean] = fileBackup{path: target}
		}
		paths = append(paths, filepath.ToSlash(clean))
		if change.Delete {
			if err := os.RemoveAll(target); err != nil {
				restore(backups)
				return storepkg.Checkpoint{}, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			restore(backups)
			return storepkg.Checkpoint{}, err
		}
		if err := os.WriteFile(target, change.Data, 0o644); err != nil {
			restore(backups)
			return storepkg.Checkpoint{}, err
		}
	}
	sort.Strings(paths)
	args := append([]string{"add", "-A", "--"}, paths...)
	if _, err := git(t.store.root, args...); err != nil {
		restore(backups)
		return storepkg.Checkpoint{}, err
	}
	if strings.TrimSpace(summary) == "" {
		summary = "state: ALP checkpoint"
	}
	if _, err := git(t.store.root, "commit", "-m", summary); err != nil {
		_, _ = git(t.store.root, "reset", "--mixed", "HEAD")
		restore(backups)
		return storepkg.Checkpoint{}, err
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
}

func IsOwnedPath(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "workspace.yaml" {
		return true
	}
	for _, prefix := range []string{"profile/", "personas/", "evidence/", "assessments/", "sessions/", "state/"} {
		if strings.HasPrefix(clean, prefix) {
			return true
		}
	}
	return false
}

func safeOwnedPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path is required")
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace", path)
	}
	if !IsOwnedPath(clean) {
		return "", fmt.Errorf("path %q is not owned by ALP Git Store", path)
	}
	return clean, nil
}

func copyOwned(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if !IsOwnedPath(rel) {
			return nil
		}
		target := filepath.Join(destination, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
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

func restore(backups map[string]fileBackup) {
	for _, item := range backups {
		if item.exists {
			_ = os.MkdirAll(filepath.Dir(item.path), 0o755)
			_ = os.WriteFile(item.path, item.data, item.mode.Perm())
		} else {
			_ = os.RemoveAll(item.path)
		}
	}
}

func git(dir string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func (s *Store) ConfigureRemote(name, url string) error {
	if strings.TrimSpace(name) == "" {
		name = "origin"
	}
	if strings.TrimSpace(url) == "" {
		return errors.New("git remote URL is required")
	}
	if _, err := git(s.root, "remote", "get-url", name); err == nil {
		if _, err := git(s.root, "remote", "set-url", name, url); err != nil {
			return err
		}
	} else {
		if _, err := git(s.root, "remote", "add", name, url); err != nil {
			return err
		}
	}
	s.remote = name
	return nil
}
