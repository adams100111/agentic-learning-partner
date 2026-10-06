package gitstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	storepkg "github.com/adams100111/agentic-learning-partner/internal/store"
	"go.yaml.in/yaml/v3"
)

type SemanticConflictError struct {
	Paths []string
}

func (e SemanticConflictError) Error() string {
	return "semantic learner-state conflict requires resolution: " + strings.Join(e.Paths, ", ")
}

func (s *Store) Pull(ctx context.Context) (storepkg.SyncResult, error) {
	before, err := s.Revision(ctx)
	if err != nil {
		return storepkg.SyncResult{}, err
	}
	if err := s.fetch(); err != nil {
		return storepkg.SyncResult{Before: before, After: before, Pending: true}, err
	}
	remoteRevision, err := s.remoteRevision()
	if err != nil {
		return storepkg.SyncResult{}, err
	}
	if remoteRevision == before {
		return storepkg.SyncResult{Before: before, After: before}, nil
	}

	base, err := git(s.root, "merge-base", before.String(), remoteRevision.String())
	if err != nil {
		return storepkg.SyncResult{}, err
	}
	baseRevision := storepkg.Revision(strings.TrimSpace(base))

	switch {
	case baseRevision == before:
		if err := s.ensureNoOwnedWorkingChanges(); err != nil {
			return storepkg.SyncResult{}, err
		}
		if _, err := git(s.root, "merge", "--ff-only", remoteRevision.String()); err != nil {
			return storepkg.SyncResult{}, err
		}
	case baseRevision == remoteRevision:
		// Local history is already ahead. Push will publish it.
	default:
		if err := s.reconcileDivergence(baseRevision, before, remoteRevision); err != nil {
			return storepkg.SyncResult{}, err
		}
	}

	after, err := s.Revision(ctx)
	if err != nil {
		return storepkg.SyncResult{}, err
	}
	return storepkg.SyncResult{Before: before, After: after, Changed: before != after}, nil
}

func (s *Store) Push(ctx context.Context) (storepkg.SyncResult, error) {
	before, err := s.Revision(ctx)
	if err != nil {
		return storepkg.SyncResult{}, err
	}
	_, err = git(s.root, "push", s.remote, "HEAD:"+s.branch)
	if err != nil {
		return storepkg.SyncResult{Before: before, After: before, Pending: true}, err
	}
	return storepkg.SyncResult{Before: before, After: before}, nil
}

func (s *Store) Sync(ctx context.Context) (storepkg.SyncResult, error) {
	initial, err := s.Revision(ctx)
	if err != nil {
		return storepkg.SyncResult{}, err
	}
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := s.Pull(ctx); err != nil {
			return storepkg.SyncResult{Before: initial, After: initial, Pending: true}, err
		}
		if _, err := s.Push(ctx); err == nil {
			after, revisionErr := s.Revision(ctx)
			if revisionErr != nil {
				return storepkg.SyncResult{}, revisionErr
			}
			return storepkg.SyncResult{Before: initial, After: after, Changed: initial != after}, nil
		} else if !isPushRace(err) {
			return storepkg.SyncResult{Before: initial, After: initial, Pending: true}, err
		}
	}
	after, _ := s.Revision(ctx)
	return storepkg.SyncResult{Before: initial, After: after, Pending: true},
		fmt.Errorf("git synchronization exceeded 3 optimistic retry attempts")
}

func (s *Store) fetch() error {
	if strings.TrimSpace(s.remote) == "" {
		return errors.New("git store remote is not configured")
	}
	if _, err := git(s.root, "remote", "get-url", s.remote); err != nil {
		return fmt.Errorf("git store remote %q is not configured: %w", s.remote, err)
	}
	_, err := git(s.root, "fetch", "--prune", s.remote, s.branch)
	return err
}

func (s *Store) remoteRevision() (storepkg.Revision, error) {
	value, err := git(s.root, "rev-parse", "refs/remotes/"+s.remote+"/"+s.branch)
	if err != nil {
		return "", err
	}
	return storepkg.Revision(strings.TrimSpace(value)), nil
}

func (s *Store) reconcileDivergence(base, local, remote storepkg.Revision) error {
	baseSnapshot, err := s.snapshot(base)
	if err != nil {
		return err
	}
	localSnapshot, err := s.snapshot(local)
	if err != nil {
		return err
	}
	localSnapshot, err = s.overlayWorkingOwned(localSnapshot)
	if err != nil {
		return err
	}
	remoteSnapshot, err := s.snapshot(remote)
	if err != nil {
		return err
	}

	merged, conflicts, err := reconcileSnapshots(baseSnapshot, localSnapshot, remoteSnapshot)
	if err != nil {
		return err
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return SemanticConflictError{Paths: conflicts}
	}

	if _, err := git(s.root, "merge", "-s", "ours", "--no-commit", remote.String()); err != nil {
		return err
	}
	if err := s.writeOwnedSnapshot(merged); err != nil {
		_, _ = git(s.root, "merge", "--abort")
		return err
	}
	if s.rebuild != nil {
		if err := s.rebuild(s.root); err != nil {
			_, _ = git(s.root, "merge", "--abort")
			return fmt.Errorf("rebuild derived state during sync: %w", err)
		}
	}
	if s.validator != nil {
		if err := s.validator(s.root); err != nil {
			_, _ = git(s.root, "merge", "--abort")
			return err
		}
	}
	paths, err := s.ownedWorkingPaths()
	if err != nil {
		_, _ = git(s.root, "merge", "--abort")
		return err
	}
	if len(paths) > 0 {
		args := append([]string{"add", "-A", "--"}, paths...)
		if _, err := git(s.root, args...); err != nil {
			_, _ = git(s.root, "merge", "--abort")
			return err
		}
	}
	if _, err := git(s.root, "commit", "-m", "state: reconcile ALP learner workspace"); err != nil {
		_, _ = git(s.root, "merge", "--abort")
		return err
	}
	return nil
}

func (s *Store) snapshot(revision storepkg.Revision) (map[string][]byte, error) {
	output, err := git(s.root, "ls-tree", "-r", "--name-only", revision.String())
	if err != nil {
		return nil, err
	}
	result := map[string][]byte{}
	for _, path := range strings.Split(strings.TrimSpace(output), "\n") {
		if path == "" || !IsOwnedPath(path) {
			continue
		}
		value, err := git(s.root, "show", revision.String()+":"+path)
		if err != nil {
			return nil, err
		}
		result[filepath.ToSlash(path)] = []byte(value)
	}
	return result, nil
}

func (s *Store) overlayWorkingOwned(snapshot map[string][]byte) (map[string][]byte, error) {
	result := cloneSnapshot(snapshot)
	paths, err := s.ownedWorkingPaths()
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		full := filepath.Join(s.root, filepath.FromSlash(path))
		data, err := os.ReadFile(full)
		if errors.Is(err, os.ErrNotExist) {
			delete(result, path)
			continue
		}
		if err != nil {
			return nil, err
		}
		result[path] = data
	}
	return result, nil
}

func (s *Store) ownedWorkingPaths() ([]string, error) {
	output, err := git(s.root, "status", "--porcelain", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	parts := bytes.Split([]byte(output), []byte{0})
	for _, part := range parts {
		if len(part) < 4 {
			continue
		}
		path := string(part[3:])
		if strings.Contains(path, " -> ") {
			path = strings.SplitN(path, " -> ", 2)[1]
		}
		path = filepath.ToSlash(path)
		if IsOwnedPath(path) {
			seen[path] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for path := range seen {
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

func (s *Store) ensureNoOwnedWorkingChanges() error {
	paths, err := s.ownedWorkingPaths()
	if err != nil {
		return err
	}
	if len(paths) != 0 {
		return fmt.Errorf("owned learner-state paths have uncheckpointed external-local changes: %s", strings.Join(paths, ", "))
	}
	return nil
}

func (s *Store) writeOwnedSnapshot(snapshot map[string][]byte) error {
	current, err := s.List(context.Background(), "")
	if err != nil {
		return err
	}
	for _, entry := range current {
		if strings.HasPrefix(entry.Path, "state/") {
			continue
		}
		if _, exists := snapshot[entry.Path]; !exists {
			if err := os.RemoveAll(filepath.Join(s.root, filepath.FromSlash(entry.Path))); err != nil {
				return err
			}
		}
	}
	for path, data := range snapshot {
		if strings.HasPrefix(path, "state/") {
			continue
		}
		target := filepath.Join(s.root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	stateRoot := filepath.Join(s.root, "state")
	if err := os.RemoveAll(stateRoot); err != nil {
		return err
	}
	return nil
}

func reconcileSnapshots(base, local, remote map[string][]byte) (map[string][]byte, []string, error) {
	paths := map[string]struct{}{}
	for path := range base {
		paths[path] = struct{}{}
	}
	for path := range local {
		paths[path] = struct{}{}
	}
	for path := range remote {
		paths[path] = struct{}{}
	}
	result := map[string][]byte{}
	var conflicts []string
	for path := range paths {
		if strings.HasPrefix(path, "state/") {
			continue
		}
		b, bok := base[path]
		l, lok := local[path]
		r, rok := remote[path]

		if isAppendOnly(path) {
			switch {
			case lok && rok && !bytes.Equal(l, r):
				conflicts = append(conflicts, path)
			case lok:
				result[path] = l
			case rok:
				result[path] = r
			}
			continue
		}
		switch {
		case lok == rok && (!lok || bytes.Equal(l, r)):
			if lok {
				result[path] = l
			}
		case bok && lok && bytes.Equal(b, l):
			if rok {
				result[path] = r
			}
		case bok && rok && bytes.Equal(b, r):
			if lok {
				result[path] = l
			}
		case !bok && lok && !rok:
			result[path] = l
		case !bok && !lok && rok:
			result[path] = r
		default:
			merged, ok, err := mergeStructuredDocument(b, l, r, bok, lok, rok)
			if err != nil {
				return nil, nil, err
			}
			if ok {
				result[path] = merged
			} else {
				conflicts = append(conflicts, path)
			}
		}
	}
	return result, conflicts, nil
}

func isAppendOnly(path string) bool {
	for _, prefix := range []string{"evidence/", "assessments/", "sessions/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func mergeStructuredDocument(base, local, remote []byte, baseOK, localOK, remoteOK bool) ([]byte, bool, error) {
	if !localOK || !remoteOK {
		return nil, false, nil
	}
	var b, l, r any
	if baseOK {
		if err := yaml.Unmarshal(base, &b); err != nil {
			return nil, false, nil
		}
	}
	if err := yaml.Unmarshal(local, &l); err != nil {
		return nil, false, nil
	}
	if err := yaml.Unmarshal(remote, &r); err != nil {
		return nil, false, nil
	}
	merged, ok := mergeValue(b, l, r)
	if !ok {
		return nil, false, nil
	}
	data, err := yaml.Marshal(merged)
	return data, err == nil, err
}

func mergeValue(base, local, remote any) (any, bool) {
	if equalYAML(local, remote) {
		return local, true
	}
	if equalYAML(base, local) {
		return remote, true
	}
	if equalYAML(base, remote) {
		return local, true
	}

	lm, lok := stringMap(local)
	rm, rok := stringMap(remote)
	bm, _ := stringMap(base)
	if !lok || !rok {
		return nil, false
	}
	if winner, ok := provenanceWinner(lm, rm); ok {
		return winner, true
	}

	result := map[string]any{}
	keys := map[string]struct{}{}
	for key := range bm {
		keys[key] = struct{}{}
	}
	for key := range lm {
		keys[key] = struct{}{}
	}
	for key := range rm {
		keys[key] = struct{}{}
	}
	for key := range keys {
		value, ok := mergeValue(bm[key], lm[key], rm[key])
		if !ok {
			return nil, false
		}
		if value != nil {
			result[key] = value
		}
	}
	return result, true
}

func provenanceWinner(local, remote map[string]any) (any, bool) {
	localRank, lok := provenanceRank(local)
	remoteRank, rok := provenanceRank(remote)
	if !lok || !rok || localRank == remoteRank {
		return nil, false
	}
	if localRank > remoteRank {
		return local, true
	}
	return remote, true
}

func provenanceRank(value map[string]any) (int, bool) {
	raw, ok := value["provenance"]
	if !ok {
		return 0, false
	}
	provenance, ok := stringMap(raw)
	if !ok {
		return 0, false
	}
	source, _ := provenance["sourceType"].(string)
	switch source {
	case "learner-correction":
		return 4, true
	case "learner-stated":
		return 3, true
	case "observed", "evaluated":
		return 2, true
	case "agent-inferred", "agent-proposed":
		return 1, true
	default:
		return 0, false
	}
}

func stringMap(value any) (map[string]any, bool) {
	switch mapping := value.(type) {
	case map[string]any:
		return mapping, true
	case map[any]any:
		result := make(map[string]any, len(mapping))
		for key, value := range mapping {
			text, ok := key.(string)
			if !ok {
				return nil, false
			}
			result[text] = value
		}
		return result, true
	default:
		return nil, false
	}
}

func equalYAML(left, right any) bool {
	return fmt.Sprintf("%#v", left) == fmt.Sprintf("%#v", right)
}

func cloneSnapshot(source map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(source))
	for path, data := range source {
		result[path] = append([]byte(nil), data...)
	}
	return result
}

func isPushRace(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "non-fast-forward") ||
		strings.Contains(message, "fetch first") ||
		strings.Contains(message, "rejected")
}
