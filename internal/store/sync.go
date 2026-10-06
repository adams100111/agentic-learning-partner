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

)

type SyncOptions struct {
	Remote     string
	Branch     string
	MaxRetries int
	Rebuild    func(root string) error
}

type SyncResult struct {
	Changed  bool
	Pending  bool
	Attempts int
	Revision Revision
	Message  string
}

func (s *Git) Pull(ctx context.Context, options SyncOptions) (SyncResult, error) {
	remote, branch, err := s.syncTarget(options)
	if err != nil {
		return SyncResult{}, err
	}
	remoteSHA, fetchErr := s.fetchRemote(ctx, remote, branch)
	if fetchErr != nil {
		revision, _ := s.Revision(ctx)
		return SyncResult{Pending: true, Revision: revision, Message: fetchErr.Error()}, nil
	}
	changed, revision, err := s.reconcileRemote(ctx, remoteSHA, options)
	if err != nil {
		return SyncResult{}, err
	}
	return SyncResult{Changed: changed, Revision: revision}, nil
}

func (s *Git) Push(ctx context.Context, options SyncOptions) (SyncResult, error) {
	remote, branch, err := s.syncTarget(options)
	if err != nil {
		return SyncResult{}, err
	}
	revision, _ := s.Revision(ctx)
	output, pushErr := s.pushRemote(ctx, remote, branch)
	if pushErr != nil {
		return SyncResult{Pending: true, Revision: revision, Message: output}, nil
	}
	return SyncResult{Revision: revision}, nil
}

func (s *Git) Sync(ctx context.Context, options SyncOptions) (SyncResult, error) {
	remote, branch, err := s.syncTarget(options)
	if err != nil {
		return SyncResult{}, err
	}
	maxRetries := options.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	changed := false
	for attempt := 1; attempt <= maxRetries; attempt++ {
		remoteSHA, fetchErr := s.fetchRemote(ctx, remote, branch)
		if fetchErr != nil {
			revision, _ := s.Revision(ctx)
			return SyncResult{Changed: changed, Pending: true, Attempts: attempt, Revision: revision, Message: fetchErr.Error()}, nil
		}
		pulled, revision, reconcileErr := s.reconcileRemote(ctx, remoteSHA, options)
		if reconcileErr != nil {
			return SyncResult{}, reconcileErr
		}
		changed = changed || pulled

		output, pushErr := s.pushRemote(ctx, remote, branch)
		if pushErr == nil {
			finalRevision, _ := s.Revision(ctx)
			return SyncResult{Changed: changed, Attempts: attempt, Revision: finalRevision}, nil
		}
		if isPushRace(output) {
			if attempt == maxRetries {
				return SyncResult{}, fmt.Errorf("Git Store concurrency retry limit reached after %d attempts", maxRetries)
			}
			continue
		}
		return SyncResult{Changed: changed, Pending: true, Attempts: attempt, Revision: revision, Message: output}, nil
	}
	return SyncResult{}, fmt.Errorf("Git Store synchronization failed")
}

func (s *Git) syncTarget(options SyncOptions) (string, string, error) {
	remote := strings.TrimSpace(options.Remote)
	if remote == "" {
		remote = "origin"
	}
	branch := strings.TrimSpace(options.Branch)
	if branch == "" {
		value, err := gitCommand(s.root, "branch", "--show-current")
		if err != nil {
			return "", "", err
		}
		branch = strings.TrimSpace(value)
	}
	if branch == "" {
		return "", "", errors.New("Git Store synchronization requires a checked-out branch")
	}
	return remote, branch, nil
}

func (s *Git) fetchRemote(ctx context.Context, remote, branch string) (string, error) {
	if _, err := gitCommandContext(ctx, s.root, "fetch", "--no-tags", remote, branch); err != nil {
		return "", fmt.Errorf("fetch Git Store remote: %w", err)
	}
	value, err := gitCommandContext(ctx, s.root, "rev-parse", "FETCH_HEAD")
	if err != nil {
		return "", fmt.Errorf("resolve fetched Git Store revision: %w", err)
	}
	return strings.TrimSpace(value), nil
}

func (s *Git) pushRemote(ctx context.Context, remote, branch string) (string, error) {
	command := execGit(ctx, s.root, "push", remote, "HEAD:refs/heads/"+branch)
	output, err := command.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func (s *Git) reconcileRemote(ctx context.Context, remoteSHA string, options SyncOptions) (bool, Revision, error) {
	localRevision, err := s.Revision(ctx)
	if err != nil {
		return false, "", err
	}
	localSHA := gitSHA(localRevision)
	if localSHA == remoteSHA {
		return false, localRevision, nil
	}

	status, err := s.Status(ctx)
	if err != nil {
		return false, "", err
	}
	baseSHA, err := gitCommand(s.root, "merge-base", localSHA, remoteSHA)
	if err != nil {
		return false, "", fmt.Errorf("Git Store histories do not share a base: %w", err)
	}
	baseSHA = strings.TrimSpace(baseSHA)

	if baseSHA == localSHA && len(status.DirtyOwned) == 0 {
		remoteSnapshot, err := s.snapshotAt(remoteSHA)
		if err != nil {
			return false, "", err
		}
		if err := s.advanceBranch(localSHA, remoteSHA, remoteSnapshot); err != nil {
			return false, "", err
		}
		return true, Revision("git:" + remoteSHA), nil
	}
	if baseSHA == remoteSHA {
		// Local canonical commits are already ahead. Dirty owned changes remain
		// uncheckpointed for the session/transaction layer to own.
		return false, localRevision, nil
	}

	if changes, err := s.nonOwnedCommittedChanges(baseSHA, localSHA); err != nil {
		return false, "", err
	} else if len(changes) > 0 {
		return false, "", fmt.Errorf("cannot semantically reconcile committed non-ALP paths: %s", strings.Join(changes, ", "))
	}

	baseSnapshot, err := s.snapshotAt(baseSHA)
	if err != nil {
		return false, "", err
	}
	localSnapshot, err := s.snapshotWorking()
	if err != nil {
		return false, "", err
	}
	remoteSnapshot, err := s.snapshotAt(remoteSHA)
	if err != nil {
		return false, "", err
	}
	merged, err := reconcileSnapshots(baseSnapshot, localSnapshot, remoteSnapshot)
	if err != nil {
		return false, "", err
	}

	temp, err := os.MkdirTemp(filepath.Dir(s.root), ".alp-reconcile-*")
	if err != nil {
		return false, "", err
	}
	_ = os.RemoveAll(temp)
	if _, err := gitCommand(s.root, "worktree", "add", "--detach", temp, remoteSHA); err != nil {
		return false, "", fmt.Errorf("create reconciliation worktree: %w", err)
	}
	defer func() {
		_, _ = gitCommand(s.root, "worktree", "remove", "--force", temp)
		_ = os.RemoveAll(temp)
	}()

	if err := replaceOwnedSnapshot(temp, merged); err != nil {
		return false, "", err
	}
	if options.Rebuild != nil {
		if err := options.Rebuild(temp); err != nil {
			return false, "", fmt.Errorf("rebuild reconciled derived state: %w", err)
		}
	}
	if issues := s.validator.ValidateWorkspace(temp); len(issues) > 0 {
		return false, "", fmt.Errorf("reconciled workspace is invalid: %s", issues[0].Error())
	}
	finalSnapshot, err := snapshotDirectory(temp)
	if err != nil {
		return false, "", err
	}

	changedPaths := changedSnapshotPaths(remoteSnapshot, finalSnapshot)
	if len(changedPaths) > 0 {
		args := append([]string{"add", "-A", "--"}, changedPaths...)
		if _, err := gitCommand(temp, args...); err != nil {
			return false, "", fmt.Errorf("stage reconciled owned paths: %w", err)
		}
	}
	tree, err := gitCommand(temp, "write-tree")
	if err != nil {
		return false, "", fmt.Errorf("write reconciled Git tree: %w", err)
	}
	tree = strings.TrimSpace(tree)
	remoteTree, err := gitCommand(s.root, "rev-parse", remoteSHA+"^{tree}")
	if err != nil {
		return false, "", err
	}
	remoteTree = strings.TrimSpace(remoteTree)

	newSHA := remoteSHA
	if tree != remoteTree {
		commit, err := gitCommand(s.root, "commit-tree", tree, "-p", remoteSHA, "-p", localSHA, "-m", "alp: reconcile learner state")
		if err != nil {
			return false, "", fmt.Errorf("create semantic reconciliation commit: %w", err)
		}
		newSHA = strings.TrimSpace(commit)
	}
	if err := s.advanceBranch(localSHA, newSHA, finalSnapshot); err != nil {
		return false, "", err
	}
	return true, Revision("git:" + newSHA), nil
}

func (s *Git) advanceBranch(expectedSHA, newSHA string, snapshot map[string][]byte) error {
	branchRef, err := gitCommand(s.root, "symbolic-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve Git Store branch ref: %w", err)
	}
	branchRef = strings.TrimSpace(branchRef)
	previous, err := s.snapshotWorking()
	if err != nil {
		return err
	}
	if err := replaceOwnedSnapshot(s.root, snapshot); err != nil {
		return err
	}
	if issues := s.validator.ValidateWorkspace(s.root); len(issues) > 0 {
		_ = replaceOwnedSnapshot(s.root, previous)
		return fmt.Errorf("advanced workspace is invalid: %s", issues[0].Error())
	}
	if _, err := gitCommand(s.root, "update-ref", branchRef, newSHA, expectedSHA); err != nil {
		_ = replaceOwnedSnapshot(s.root, previous)
		return fmt.Errorf("advance Git Store branch: %w", err)
	}
	paths := unionSnapshotPaths(previous, snapshot)
	if len(paths) > 0 {
		args := append([]string{"reset", "--quiet", newSHA, "--"}, paths...)
		if _, err := gitCommand(s.root, args...); err != nil {
			return fmt.Errorf("refresh Git Store index after checkpoint: %w", err)
		}
	}
	return nil
}

func (s *Git) snapshotAt(revision string) (map[string][]byte, error) {
	raw, err := gitCommand(s.root, "ls-tree", "-r", "--name-only", "-z", revision)
	if err != nil {
		return nil, err
	}
	snapshot := map[string][]byte{}
	for _, path := range strings.Split(raw, " ") {
		if path == "" || !IsOwnedPath(path) {
			continue
		}
		data, err := gitCommandBytes(s.root, "show", revision+":"+path)
		if err != nil {
			return nil, fmt.Errorf("read %s at %s: %w", path, revision, err)
		}
		snapshot[path] = data
	}
	return snapshot, nil
}

func (s *Git) snapshotWorking() (map[string][]byte, error) {
	revision, err := s.Revision(context.Background())
	if err != nil {
		return nil, err
	}
	snapshot, err := s.snapshotAt(gitSHA(revision))
	if err != nil {
		return nil, err
	}
	status, err := s.Status(context.Background())
	if err != nil {
		return nil, err
	}
	for _, path := range status.DirtyOwned {
		full, err := safePath(s.root, path)
		if err != nil {
			return nil, err
		}
		data, readErr := os.ReadFile(full)
		if errors.Is(readErr, os.ErrNotExist) {
			delete(snapshot, path)
			continue
		}
		if readErr != nil {
			return nil, readErr
		}
		snapshot[path] = data
	}
	return snapshot, nil
}

func snapshotDirectory(root string) (map[string][]byte, error) {
	snapshot := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !IsOwnedPath(relative) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[relative] = data
		return nil
	})
	return snapshot, err
}

func replaceOwnedSnapshot(root string, snapshot map[string][]byte) error {
	current, err := snapshotDirectory(root)
	if err != nil {
		return err
	}
	for path := range current {
		if _, keep := snapshot[path]; keep {
			continue
		}
		full, err := safePath(root, path)
		if err != nil {
			return err
		}
		if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	paths := make([]string, 0, len(snapshot))
	for path := range snapshot {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		full, err := safePath(root, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		temp, err := os.CreateTemp(filepath.Dir(full), ".alp-sync-*")
		if err != nil {
			return err
		}
		if _, err := temp.Write(snapshot[path]); err != nil {
			temp.Close()
			return err
		}
		if err := temp.Close(); err != nil {
			return err
		}
		if err := os.Rename(temp.Name(), full); err != nil {
			return err
		}
	}
	return nil
}

func (s *Git) nonOwnedCommittedChanges(base, revision string) ([]string, error) {
	raw, err := gitCommand(s.root, "diff", "--name-only", "-z", base+".."+revision)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, path := range strings.Split(raw, " ") {
		if path != "" && !IsOwnedPath(path) {
			result = append(result, path)
		}
	}
	sort.Strings(result)
	return result, nil
}

func gitSHA(revision Revision) string {
	return strings.TrimPrefix(string(revision), "git:")
}

func isPushRace(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "non-fast-forward") ||
		strings.Contains(lower, "fetch first") ||
		strings.Contains(lower, "[rejected]")
}

func execGit(ctx context.Context, dir string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
}

func gitCommandBytes(dir string, args ...string) ([]byte, error) {
	command := execGit(context.Background(), dir, args...)
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func changedSnapshotPaths(before, after map[string][]byte) []string {
	var result []string
	for _, path := range unionSnapshotPaths(before, after) {
		left, leftOK := before[path]
		right, rightOK := after[path]
		if leftOK != rightOK || (leftOK && string(left) != string(right)) {
			result = append(result, path)
		}
	}
	return result
}
