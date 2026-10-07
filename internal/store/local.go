package store

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
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

type Local struct {
	root      string
	validator *workspace.Validator
}

func OpenLocal(root string, validator *workspace.Validator) (*Local, error) {
	if validator == nil {
		return nil, errors.New("workspace validator is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(absolute); err != nil || !info.IsDir() {
		if err != nil {
			return nil, fmt.Errorf("open local store: %w", err)
		}
		return nil, fmt.Errorf("open local store: %s is not a directory", absolute)
	}
	if issues := validator.ValidateWorkspace(absolute); len(issues) > 0 {
		return nil, fmt.Errorf("open local store: %s", issues[0].Error())
	}
	return &Local{root: absolute, validator: validator}, nil
}

func (s *Local) Provider() string { return "local" }
func (s *Local) Root() string     { return s.root }
func (s *Local) Capabilities() Capabilities {
	return Capabilities{
		CapabilityPersistence:           true,
		CapabilityOptimisticConcurrency: true,
		CapabilityAtomicCheckpoint:      true,
		CapabilityRevisions:             true,
		CapabilityOffline:               true,
	}
}

func (s *Local) Revision(ctx context.Context) (Revision, error) {
	_ = ctx
	paths, err := canonicalPaths(s.root)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	for _, relative := range paths {
		data, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		hash.Write([]byte(relative))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
	}
	return Revision("local:" + hex.EncodeToString(hash.Sum(nil))), nil
}

func (s *Local) Read(ctx context.Context, key string) ([]byte, error) {
	_ = ctx
	path, err := safePath(s.root, key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func (s *Local) Commit(ctx context.Context, expected Revision, changes ChangeSet) (Revision, error) {
	current, err := s.Revision(ctx)
	if err != nil {
		return "", err
	}
	if expected == "" {
		return "", errors.New("expected workspace revision is required")
	}
	if current != expected {
		return "", fmt.Errorf("workspace revision changed: expected %s, found %s", expected, current)
	}

	type staged struct {
		target string
		temp   string
		remove bool
	}
	var stagedFiles []staged
	cleanup := func() {
		for _, item := range stagedFiles {
			if item.temp != "" {
				_ = os.Remove(item.temp)
			}
		}
	}
	defer cleanup()

	for _, mutation := range changes.Mutations {
		if !IsOwnedPath(mutation.Path) {
			return "", fmt.Errorf("path %q is not ALP-owned", mutation.Path)
		}
		target, err := safePath(s.root, mutation.Path)
		if err != nil {
			return "", err
		}
		if mutation.Delete {
			stagedFiles = append(stagedFiles, staged{target: target, remove: true})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		temp, err := os.CreateTemp(filepath.Dir(target), ".alp-commit-*")
		if err != nil {
			return "", err
		}
		if _, err := temp.Write(mutation.Data); err != nil {
			temp.Close()
			return "", err
		}
		if err := temp.Close(); err != nil {
			return "", err
		}
		stagedFiles = append(stagedFiles, staged{target: target, temp: temp.Name()})
	}

	type previous struct {
		path   string
		data   []byte
		mode   os.FileMode
		exists bool
	}
	var history []previous
	rollback := func() {
		for index := len(history) - 1; index >= 0; index-- {
			item := history[index]
			if !item.exists {
				_ = os.Remove(item.path)
				continue
			}
			_ = os.MkdirAll(filepath.Dir(item.path), 0o755)
			temp, err := os.CreateTemp(filepath.Dir(item.path), ".alp-rollback-*")
			if err != nil {
				continue
			}
			_, _ = temp.Write(item.data)
			_ = temp.Chmod(item.mode)
			_ = temp.Close()
			_ = os.Rename(temp.Name(), item.path)
		}
	}

	for _, item := range stagedFiles {
		prior := previous{path: item.target}
		if info, err := os.Stat(item.target); err == nil {
			prior.exists = true
			prior.mode = info.Mode().Perm()
			prior.data, err = os.ReadFile(item.target)
			if err != nil {
				rollback()
				return "", err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			rollback()
			return "", err
		}
		history = append(history, prior)

		if item.remove {
			if err := os.Remove(item.target); err != nil && !errors.Is(err, os.ErrNotExist) {
				rollback()
				return "", err
			}
			continue
		}
		if err := os.Rename(item.temp, item.target); err != nil {
			rollback()
			return "", err
		}
		item.temp = ""
	}
	if issues := s.validator.ValidateWorkspace(s.root); len(issues) > 0 {
		rollback()
		return "", fmt.Errorf("committed workspace is invalid: %s", issues[0].Error())
	}
	return s.Revision(ctx)
}

func canonicalPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || strings.HasPrefix(name, ".alp-") {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if IsCanonicalRevisionPath(relative) {
			paths = append(paths, relative)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func IsCanonicalRevisionPath(path string) bool {
	if path == "workspace.yaml" || path == "workspace.json" {
		return true
	}
	for _, prefix := range []string{"profile/", "personas/", "evidence/", "assessments/", "sessions/", "platform-accounts/", "adaptation-decisions/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func safePath(root, key string) (string, error) {
	if filepath.IsAbs(key) {
		return "", fmt.Errorf("store path must be relative: %q", key)
	}
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("store path escapes workspace: %q", key)
	}
	target := filepath.Join(root, clean)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("store path escapes workspace: %q", key)
	}
	return target, nil
}

func IsOwnedPath(path string) bool {
	path = filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if path == "workspace.yaml" || path == "workspace.json" {
		return true
	}
	for _, prefix := range []string{"profile/", "personas/", "evidence/", "assessments/", "sessions/", "platform-accounts/", "adaptation-decisions/", "state/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
