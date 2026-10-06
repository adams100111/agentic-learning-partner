package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

type Git struct {
	root      string
	validator *workspace.Validator
}

type GitStatus struct {
	Branch         string
	Remote         string
	DirtyOwned     []string
	DirtyUnrelated []string
}

func OpenGit(root string, validator *workspace.Validator) (*Git, error) {
	if validator == nil {
		return nil, errors.New("workspace validator is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	top, err := gitCommand(absolute, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("open Git Store: %w", err)
	}
	top = strings.TrimSpace(top)
	if filepath.Clean(top) != filepath.Clean(absolute) {
		return nil, fmt.Errorf("Git Store workspace must be repository root: %s", absolute)
	}
	if issues := validator.ValidateWorkspace(absolute); len(issues) > 0 {
		return nil, fmt.Errorf("open Git Store: %s", issues[0].Error())
	}
	return &Git{root: absolute, validator: validator}, nil
}

func (s *Git) Provider() string { return "git" }
func (s *Git) Root() string     { return s.root }
func (s *Git) Capabilities() Capabilities {
	return Capabilities{
		CapabilityPersistence:           true,
		CapabilityOptimisticConcurrency: true,
		CapabilityAtomicCheckpoint:      true,
		CapabilityRevisions:             true,
		CapabilityHistory:               true,
		CapabilityOffline:               true,
		CapabilitySync:                  true,
		CapabilityMultiDevice:           true,
		CapabilityRemoteManagement:      true,
	}
}

func (s *Git) Revision(ctx context.Context) (Revision, error) {
	_ = ctx
	value, err := gitCommand(s.root, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("Git Store revision: %w", err)
	}
	return Revision("git:" + strings.TrimSpace(value)), nil
}

func (s *Git) Read(ctx context.Context, key string) ([]byte, error) {
	_ = ctx
	path, err := safePath(s.root, key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func (s *Git) Commit(ctx context.Context, expected Revision, changes ChangeSet) (Revision, error) {
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
	if len(changes.Mutations) == 0 {
		return current, nil
	}

	for _, mutation := range changes.Mutations {
		if !IsOwnedPath(mutation.Path) {
			return "", fmt.Errorf("path %q is not ALP-owned", mutation.Path)
		}
		if _, err := safePath(s.root, mutation.Path); err != nil {
			return "", err
		}
	}

	type previous struct {
		path   string
		data   []byte
		mode   os.FileMode
		exists bool
	}
	var history []previous
	rollback := func() {
		for i := len(history) - 1; i >= 0; i-- {
			item := history[i]
			if !item.exists {
				_ = os.Remove(item.path)
				continue
			}
			_ = os.MkdirAll(filepath.Dir(item.path), 0o755)
			temp, err := os.CreateTemp(filepath.Dir(item.path), ".alp-git-rollback-*")
			if err != nil {
				continue
			}
			_, _ = temp.Write(item.data)
			_ = temp.Chmod(item.mode)
			_ = temp.Close()
			_ = os.Rename(temp.Name(), item.path)
		}
	}

	paths := make([]string, 0, len(changes.Mutations))
	for _, mutation := range changes.Mutations {
		target, _ := safePath(s.root, mutation.Path)
		prior := previous{path: target}
		if info, statErr := os.Stat(target); statErr == nil {
			prior.exists = true
			prior.mode = info.Mode().Perm()
			prior.data, err = os.ReadFile(target)
			if err != nil {
				rollback()
				return "", err
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			rollback()
			return "", statErr
		}
		history = append(history, prior)

		if mutation.Delete {
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				rollback()
				return "", err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				rollback()
				return "", err
			}
			temp, err := os.CreateTemp(filepath.Dir(target), ".alp-git-write-*")
			if err != nil {
				rollback()
				return "", err
			}
			if _, err := temp.Write(mutation.Data); err != nil {
				temp.Close()
				rollback()
				return "", err
			}
			if err := temp.Close(); err != nil {
				rollback()
				return "", err
			}
			if err := os.Rename(temp.Name(), target); err != nil {
				rollback()
				return "", err
			}
		}
		paths = append(paths, filepath.ToSlash(filepath.Clean(filepath.FromSlash(mutation.Path))))
	}

	if issues := s.validator.ValidateWorkspace(s.root); len(issues) > 0 {
		rollback()
		return "", fmt.Errorf("committed workspace is invalid: %s", issues[0].Error())
	}

	sort.Strings(paths)
	unique := paths[:0]
	for i, path := range paths {
		if i == 0 || path != paths[i-1] {
			unique = append(unique, path)
		}
	}
	paths = unique

	// Intent-to-add lets path-limited commit include new files while preserving
	// unrelated staged/untracked work in the real index.
	addArgs := append([]string{"add", "-N", "--"}, paths...)
	if _, err := gitCommand(s.root, addArgs...); err != nil {
		rollback()
		return "", fmt.Errorf("prepare Git Store checkpoint: %w", err)
	}

	message := strings.TrimSpace(changes.Message)
	if message == "" {
		message = "alp: checkpoint learner state"
	}
	commitArgs := []string{"commit", "--only", "-m", message, "--"}
	commitArgs = append(commitArgs, paths...)
	if _, err := gitCommand(s.root, commitArgs...); err != nil {
		rollback()
		_, _ = gitCommand(s.root, append([]string{"reset", "--"}, paths...)...)
		return "", fmt.Errorf("create Git Store checkpoint: %w", err)
	}
	return s.Revision(ctx)
}

func (s *Git) Status(ctx context.Context) (GitStatus, error) {
	_ = ctx
	branch, err := gitCommand(s.root, "branch", "--show-current")
	if err != nil {
		return GitStatus{}, err
	}
	remote, _ := gitCommand(s.root, "remote", "get-url", "origin")
	raw, err := gitCommand(s.root, "status", "--porcelain")
	if err != nil {
		return GitStatus{}, err
	}
	status := GitStatus{Branch: strings.TrimSpace(branch), Remote: strings.TrimSpace(remote)}
	for _, line := range strings.Split(raw, "
") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[len(parts)-1]
		}
		if IsOwnedPath(path) {
			status.DirtyOwned = append(status.DirtyOwned, path)
		} else {
			status.DirtyUnrelated = append(status.DirtyUnrelated, path)
		}
	}
	sort.Strings(status.DirtyOwned)
	sort.Strings(status.DirtyUnrelated)
	return status, nil
}

func CloneGit(ctx context.Context, remote, destination, branch string, validator *workspace.Validator) (*Git, error) {
	if strings.TrimSpace(remote) == "" {
		return nil, errors.New("Git Store remote is required")
	}
	if branch == "" {
		branch = "main"
	}
	if _, err := os.Stat(destination); err == nil {
		return nil, fmt.Errorf("clone destination already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	args := []string{"clone", "--branch", branch, "--single-branch", remote, destination}
	command := exec.CommandContext(ctx, "git", args...)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("clone Git Store: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return OpenGit(destination, validator)
}

func InitializeGit(ctx context.Context, root, branch, remote string, validator *workspace.Validator) (*Git, error) {
	if branch == "" {
		branch = "main"
	}
	if _, err := gitCommandContext(ctx, root, "rev-parse", "--git-dir"); err != nil {
		if _, err := gitCommandContext(ctx, root, "init", "-b", branch); err != nil {
			return nil, fmt.Errorf("initialize Git Store: %w", err)
		}
	}
	store, err := OpenGit(root, validator)
	if err != nil {
		return nil, err
	}
	if remote != "" {
		if current, remoteErr := gitCommand(root, "remote", "get-url", "origin"); remoteErr == nil {
			if strings.TrimSpace(current) != remote {
				return nil, fmt.Errorf("origin already configured as %q", strings.TrimSpace(current))
			}
		} else if _, err := gitCommand(root, "remote", "add", "origin", remote); err != nil {
			return nil, fmt.Errorf("configure Git Store remote: %w", err)
		}
	}
	return store, nil
}

func gitCommand(dir string, args ...string) (string, error) {
	return gitCommandContext(context.Background(), dir, args...)
}

func gitCommandContext(ctx context.Context, dir string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}
