package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

type RecoveryStatus string

const (
	RecoveryStaged      RecoveryStatus = "staged"
	RecoveryCheckpointed RecoveryStatus = "checkpointed"
	RecoverySyncPending RecoveryStatus = "sync-pending"
)

type Recovery struct {
	WorkspaceID        string         `json:"workspaceId"`
	SessionID          string         `json:"sessionId"`
	BaseRevision       Revision       `json:"baseRevision"`
	CheckpointRevision Revision       `json:"checkpointRevision,omitempty"`
	Status             RecoveryStatus `json:"status"`
	StageDir           string         `json:"stageDir,omitempty"`
	StartedAt          string         `json:"startedAt"`
}

type lockMetadata struct {
	WorkspaceID string `json:"workspaceId"`
	SessionID   string `json:"sessionId"`
	PID         int    `json:"pid"`
	StartedAt   string `json:"startedAt"`
}

type Coordinator struct {
	store      Store
	validator  *workspace.Validator
	runtimeDir string
}

func NewCoordinator(store Store, validator *workspace.Validator, runtimeDir string) *Coordinator {
	return &Coordinator{store: store, validator: validator, runtimeDir: runtimeDir}
}

type Transaction struct {
	coordinator *Coordinator
	recovery    Recovery
	changes     map[string]Mutation
}

func (c *Coordinator) Begin(ctx context.Context, sessionID string, expected Revision) (*Transaction, error) {
	if c.store == nil || c.validator == nil {
		return nil, errors.New("store and workspace validator are required")
	}
	if sessionID == "" {
		return nil, errors.New("session id is required")
	}
	current, err := c.store.Revision(ctx)
	if err != nil {
		return nil, err
	}
	if expected == "" || current != expected {
		return nil, fmt.Errorf("workspace revision changed: expected %s, found %s", expected, current)
	}
	manifest, err := workspace.ReadManifest(c.store.Root())
	if err != nil {
		return nil, err
	}
	if manifest.WorkspaceID == "" {
		return nil, errors.New("workspace must be migrated to schema v2 before transactions")
	}
	if err := os.MkdirAll(c.lockDir(), 0o700); err != nil {
		return nil, err
	}
	lock := lockMetadata{
		WorkspaceID: manifest.WorkspaceID,
		SessionID: sessionID,
		PID: os.Getpid(),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := createExclusiveJSON(c.lockPath(manifest.WorkspaceID), lock); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("workspace %s already has a writer; inspect recovery before resolving the existing lock", manifest.WorkspaceID)
		}
		return nil, err
	}

	stageDir := filepath.Join(c.runtimeDir, "transactions", manifest.WorkspaceID, sessionID)
	if err := os.RemoveAll(stageDir); err != nil {
		_ = os.Remove(c.lockPath(manifest.WorkspaceID))
		return nil, err
	}
	stageWorkspace := filepath.Join(stageDir, "workspace")
	if err := copyWorkspace(c.store.Root(), stageWorkspace); err != nil {
		_ = os.Remove(c.lockPath(manifest.WorkspaceID))
		return nil, err
	}
	recovery := Recovery{
		WorkspaceID: manifest.WorkspaceID,
		SessionID: sessionID,
		BaseRevision: expected,
		Status: RecoveryStaged,
		StageDir: stageDir,
		StartedAt: lock.StartedAt,
	}
	if err := c.writeRecovery(recovery); err != nil {
		_ = os.RemoveAll(stageDir)
		_ = os.Remove(c.lockPath(manifest.WorkspaceID))
		return nil, err
	}
	return &Transaction{coordinator: c, recovery: recovery, changes: map[string]Mutation{}}, nil
}

func (t *Transaction) Put(path string, data []byte) error {
	target, err := safePath(filepath.Join(t.recovery.StageDir, "workspace"), path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return err
	}
	t.changes[path] = Mutation{Path: path, Data: append([]byte(nil), data...)}
	return t.persistChanges()
}

func (t *Transaction) Delete(path string) error {
	target, err := safePath(filepath.Join(t.recovery.StageDir, "workspace"), path)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	t.changes[path] = Mutation{Path: path, Delete: true}
	return t.persistChanges()
}

func (t *Transaction) Commit(ctx context.Context) (Revision, error) {
	return t.Checkpoint(ctx, false)
}

func (t *Transaction) Checkpoint(ctx context.Context, syncPending bool) (Revision, error) {
	stageWorkspace := filepath.Join(t.recovery.StageDir, "workspace")
	if issues := t.coordinator.validator.ValidateWorkspace(stageWorkspace); len(issues) > 0 {
		return "", fmt.Errorf("staged workspace is invalid: %s", issues[0].Error())
	}
	changeSet := ChangeSet{Mutations: make([]Mutation, 0, len(t.changes))}
	keys := make([]string, 0, len(t.changes))
	for key := range t.changes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		changeSet.Mutations = append(changeSet.Mutations, t.changes[key])
	}
	revision, err := t.coordinator.store.Commit(ctx, t.recovery.BaseRevision, changeSet)
	if err != nil {
		return "", err
	}
	t.recovery.CheckpointRevision = revision
	t.recovery.Status = RecoveryCheckpointed
	if syncPending {
		t.recovery.Status = RecoverySyncPending
	}
	t.recovery.StageDir = ""
	if err := t.coordinator.writeRecovery(t.recovery); err != nil {
		return "", err
	}
	_ = os.RemoveAll(filepath.Dir(stageWorkspace))
	_ = os.Remove(t.coordinator.lockPath(t.recovery.WorkspaceID))
	if !syncPending {
		_ = os.Remove(t.coordinator.journalPath(t.recovery.WorkspaceID))
	}
	return revision, nil
}

func (t *Transaction) persistChanges() error {
	path := filepath.Join(t.recovery.StageDir, "changes.json")
	keys := make([]string, 0, len(t.changes))
	for key := range t.changes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	changes := make([]Mutation, 0, len(keys))
	for _, key := range keys {
		changes = append(changes, t.changes[key])
	}
	data, err := json.MarshalIndent(ChangeSet{Mutations: changes}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '
'), 0o600)
}

func (c *Coordinator) Resume(workspaceID string) (*Transaction, error) {
	recovery, err := c.InspectRecovery(workspaceID)
	if err != nil {
		return nil, err
	}
	if recovery.Status != RecoveryStaged || recovery.StageDir == "" {
		return nil, fmt.Errorf("workspace %s has no resumable staged transaction", workspaceID)
	}
	if _, err := os.Stat(c.lockPath(workspaceID)); err != nil {
		return nil, fmt.Errorf("recovery lock is missing: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(recovery.StageDir, "changes.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	changes := map[string]Mutation{}
	if len(data) > 0 {
		var set ChangeSet
		if err := json.Unmarshal(data, &set); err != nil {
			return nil, fmt.Errorf("parse staged changes: %w", err)
		}
		for _, mutation := range set.Mutations {
			changes[mutation.Path] = mutation
		}
	}
	return &Transaction{coordinator: c, recovery: recovery, changes: changes}, nil
}

func (c *Coordinator) InspectLock(workspaceID string) (map[string]any, error) {
	data, err := os.ReadFile(c.lockPath(workspaceID))
	if err != nil {
		return nil, err
	}
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

func (c *Coordinator) InspectRecovery(workspaceID string) (Recovery, error) {
	data, err := os.ReadFile(c.journalPath(workspaceID))
	if err != nil {
		return Recovery{}, err
	}
	var recovery Recovery
	if err := json.Unmarshal(data, &recovery); err != nil {
		return Recovery{}, fmt.Errorf("parse recovery journal: %w", err)
	}
	return recovery, nil
}

func (c *Coordinator) DiscardRecovery(workspaceID string) error {
	recovery, err := c.InspectRecovery(workspaceID)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if recovery.StageDir != "" {
		if err := os.RemoveAll(recovery.StageDir); err != nil {
			return err
		}
	}
	_ = os.Remove(c.lockPath(workspaceID))
	if err := os.Remove(c.journalPath(workspaceID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (c *Coordinator) writeRecovery(recovery Recovery) error {
	if err := os.MkdirAll(c.journalDir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(recovery, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(c.journalDir(), ".journal-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err := temp.Write(append(data, '
')); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(name, c.journalPath(recovery.WorkspaceID))
}

func (c *Coordinator) lockDir() string    { return filepath.Join(c.runtimeDir, "locks") }
func (c *Coordinator) journalDir() string { return filepath.Join(c.runtimeDir, "journals") }
func (c *Coordinator) lockPath(id string) string { return filepath.Join(c.lockDir(), id+".lock") }
func (c *Coordinator) journalPath(id string) string { return filepath.Join(c.journalDir(), id+".json") }

func createExclusiveJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(data, '
'))
	return err
}

func copyWorkspace(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(destination, 0o755)
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == ".alp-runtime") {
			return filepath.SkipDir
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("workspace staging does not follow symlink %s", relative)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
