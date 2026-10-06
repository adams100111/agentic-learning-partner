package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/adams100111/agentic-learning-partner/internal/store"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
	"go.yaml.in/yaml/v3"
)

type SyncMode string

const (
	SyncSession SyncMode = "session"
	SyncManual  SyncMode = "manual"
	SyncEager   SyncMode = "eager"
)

type Manager struct {
	Store      store.Store
	Validator  *workspace.Validator
	RuntimeDir string
	Rebuild    func(root string) error
	Now        func() time.Time
}

type BeginOptions struct {
	ID           string
	Harness      string
	DeviceID     string
	Mode         SyncMode
	RetainRecord *bool
	Sync         store.SyncOptions
}

type CloseInput struct {
	Domains     []string
	Evidence    []string
	Assessments []string
	Summary     string
}

type Record struct {
	SchemaVersion int      `yaml:"schemaVersion" json:"schemaVersion"`
	ID            string   `yaml:"id" json:"id"`
	StartedAt     string   `yaml:"startedAt" json:"startedAt"`
	ClosedAt      string   `yaml:"closedAt" json:"closedAt"`
	Harness       string   `yaml:"harness,omitempty" json:"harness,omitempty"`
	DeviceID      string   `yaml:"deviceId,omitempty" json:"deviceId,omitempty"`
	BaseRevision  string   `yaml:"baseRevision" json:"baseRevision"`
	SyncMode      SyncMode `yaml:"syncMode" json:"syncMode"`
	Offline       bool     `yaml:"offline,omitempty" json:"offline,omitempty"`
	Domains       []string `yaml:"domains,omitempty" json:"domains,omitempty"`
	Evidence      []string `yaml:"evidence,omitempty" json:"evidence,omitempty"`
	Assessments   []string `yaml:"assessments,omitempty" json:"assessments,omitempty"`
	Summary       string   `yaml:"summary,omitempty" json:"summary,omitempty"`
}

type CloseResult struct {
	Revision    store.Revision
	SyncPending bool
	SyncResult  store.SyncResult
}

type persistedSession struct {
	ID              string   `json:"id"`
	Harness         string   `json:"harness,omitempty"`
	DeviceID        string   `json:"deviceId,omitempty"`
	Mode            SyncMode `json:"mode"`
	RetainRecord    bool     `json:"retainRecord"`
	StartedAt       string   `json:"startedAt"`
	InitialRevision string   `json:"initialRevision"`
	Offline         bool     `json:"offline"`
	Remote          string   `json:"remote,omitempty"`
	Branch          string   `json:"branch,omitempty"`
	MaxRetries      int      `json:"maxRetries,omitempty"`
}

type Session struct {
	manager         *Manager
	coordinator     *store.Coordinator
	tx              *store.Transaction
	id              string
	harness         string
	deviceID        string
	mode            SyncMode
	retainRecord    bool
	startedAt       time.Time
	baseRevision    store.Revision
	initialRevision store.Revision
	offline         bool
	syncOptions     store.SyncOptions
	closed          bool
	pendingSync     bool
}

func (m *Manager) RecoverPendingSync(ctx context.Context, options store.SyncOptions) (store.SyncResult, error) {
	if m.Store == nil || m.Validator == nil {
		return store.SyncResult{}, errors.New("store and validator are required")
	}
	if !m.Store.Capabilities().Has(store.CapabilitySync) {
		return store.SyncResult{}, store.Require(m.Store, store.CapabilitySync)
	}
	manifest, err := workspace.ReadManifest(m.Store.Root())
	if err != nil {
		return store.SyncResult{}, err
	}
	coordinator := store.NewCoordinator(m.Store, m.Validator, m.RuntimeDir)
	recovery, err := coordinator.InspectRecovery(manifest.WorkspaceID)
	if err != nil {
		return store.SyncResult{}, err
	}
	if recovery.Status != store.RecoverySyncPending {
		return store.SyncResult{}, fmt.Errorf("workspace %s recovery is %s, not sync-pending", manifest.WorkspaceID, recovery.Status)
	}
	syncer, ok := m.Store.(store.Syncer)
	if !ok {
		return store.SyncResult{}, fmt.Errorf("store provider %q advertises sync but does not implement Syncer", m.Store.Provider())
	}
	result, err := syncer.Sync(ctx, options)
	if err != nil {
		return result, err
	}
	if result.Pending {
		return result, nil
	}
	if err := coordinator.CompleteSync(manifest.WorkspaceID); err != nil {
		return result, err
	}
	if meta, metaErr := m.readSessionMetadata(manifest.WorkspaceID); metaErr == nil {
		revision, err := m.Store.Revision(ctx)
		if err != nil {
			return result, err
		}
		if _, err := coordinator.Begin(ctx, meta.ID, revision); err != nil {
			return result, fmt.Errorf("resume eager session after synchronization: %w", err)
		}
	} else if !errors.Is(metaErr, os.ErrNotExist) {
		return result, metaErr
	}
	return result, nil
}

func (m *Manager) Begin(ctx context.Context, options BeginOptions) (*Session, error) {
	if m.Store == nil || m.Validator == nil {
		return nil, errors.New("store and validator are required")
	}
	if m.RuntimeDir == "" {
		return nil, errors.New("runtime directory is required")
	}
	mode := options.Mode
	if mode == "" {
		mode = SyncSession
	}
	if mode != SyncSession && mode != SyncManual && mode != SyncEager {
		return nil, fmt.Errorf("unsupported sync mode %q", mode)
	}
	id := options.ID
	if id == "" {
		var err error
		id, err = newID()
		if err != nil {
			return nil, err
		}
	}
	retain := true
	if options.RetainRecord != nil {
		retain = *options.RetainRecord
	}
	now := time.Now
	if m.Now != nil {
		now = m.Now
	}

	offline := false
	if mode != SyncManual && m.Store.Capabilities().Has(store.CapabilitySync) {
		syncer, ok := m.Store.(store.Syncer)
		if !ok {
			return nil, fmt.Errorf("store provider %q advertises sync but does not implement Syncer", m.Store.Provider())
		}
		result, err := syncer.Sync(ctx, options.Sync)
		if err != nil {
			return nil, err
		}
		offline = result.Pending
	}

	base, err := m.Store.Revision(ctx)
	if err != nil {
		return nil, err
	}
	coordinator := store.NewCoordinator(m.Store, m.Validator, m.RuntimeDir)
	tx, err := coordinator.Begin(ctx, id, base)
	if err != nil {
		return nil, err
	}
	session := &Session{
		manager: m, coordinator: coordinator, tx: tx,
		id: id, harness: options.Harness, deviceID: options.DeviceID,
		mode: mode, retainRecord: retain, startedAt: now().UTC(),
		baseRevision: base, initialRevision: base, offline: offline, syncOptions: options.Sync,
	}
	manifest, err := workspace.ReadManifest(m.Store.Root())
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := m.writeSessionMetadata(manifest.WorkspaceID, session); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return session, nil
}

func (m *Manager) Resume(ctx context.Context) (*Session, error) {
	if m.Store == nil || m.Validator == nil {
		return nil, errors.New("store and validator are required")
	}
	manifest, err := workspace.ReadManifest(m.Store.Root())
	if err != nil {
		return nil, err
	}
	meta, err := m.readSessionMetadata(manifest.WorkspaceID)
	if err != nil {
		return nil, err
	}
	coordinator := store.NewCoordinator(m.Store, m.Validator, m.RuntimeDir)
	tx, err := coordinator.Resume(manifest.WorkspaceID)
	if err != nil {
		return nil, err
	}
	startedAt, err := time.Parse(time.RFC3339, meta.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("parse persisted session start: %w", err)
	}
	initial := store.Revision(meta.InitialRevision)
	if initial == "" {
		initial = tx.BaseRevision()
	}
	return &Session{
		manager:         m,
		coordinator:     coordinator,
		tx:              tx,
		id:              meta.ID,
		harness:         meta.Harness,
		deviceID:        meta.DeviceID,
		mode:            meta.Mode,
		retainRecord:    meta.RetainRecord,
		startedAt:       startedAt,
		baseRevision:    tx.BaseRevision(),
		initialRevision: initial,
		offline:         meta.Offline,
		syncOptions:     store.SyncOptions{Remote: meta.Remote, Branch: meta.Branch, MaxRetries: meta.MaxRetries},
	}, nil
}

func (s *Session) ID() string                   { return s.id }
func (s *Session) BaseRevision() store.Revision { return s.baseRevision }
func (s *Session) Offline() bool                { return s.offline }

func (s *Session) StageRoot() string {
	if s.tx == nil {
		return ""
	}
	return s.tx.StageRoot()
}

func (s *Session) Abort() error {
	if s.closed {
		return errors.New("session is closed")
	}
	if s.tx == nil {
		return errors.New("session has no abortable staged transaction")
	}
	if err := s.tx.Rollback(); err != nil {
		return err
	}
	s.closed = true
	manifest, err := workspace.ReadManifest(s.manager.Store.Root())
	if err != nil {
		return err
	}
	return s.manager.deleteSessionMetadata(manifest.WorkspaceID)
}

func (s *Session) Put(path string, data []byte) error {
	if s.closed {
		return errors.New("session is closed")
	}
	if err := s.ensureTransaction(context.Background()); err != nil {
		return err
	}
	return s.tx.Put(path, data)
}

func (s *Session) Delete(path string) error {
	if s.closed {
		return errors.New("session is closed")
	}
	if err := s.ensureTransaction(context.Background()); err != nil {
		return err
	}
	return s.tx.Delete(path)
}

func (s *Session) Flush(ctx context.Context) (CloseResult, error) {
	if s.mode != SyncEager {
		return CloseResult{}, errors.New("session flush is only available in eager sync mode")
	}
	if s.closed {
		return CloseResult{}, errors.New("session is closed")
	}
	result, err := s.checkpoint(ctx, false, CloseInput{})
	if err != nil {
		return CloseResult{}, err
	}
	if result.SyncPending {
		s.pendingSync = true
		s.tx = nil
		if err := s.persistMetadata(); err != nil {
			return result, err
		}
		return result, nil
	}
	next, err := s.manager.Store.Revision(ctx)
	if err != nil {
		return CloseResult{}, err
	}
	tx, err := s.coordinator.Begin(ctx, s.id, next)
	if err != nil {
		return CloseResult{}, err
	}
	s.tx = tx
	s.baseRevision = next
	if err := s.persistMetadata(); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Session) Close(ctx context.Context, input CloseInput) (CloseResult, error) {
	if s.closed {
		return CloseResult{}, errors.New("session is already closed")
	}
	if err := s.ensureTransaction(ctx); err != nil {
		return CloseResult{}, err
	}
	result, err := s.checkpoint(ctx, true, input)
	if err != nil {
		return CloseResult{}, err
	}
	s.closed = true
	manifest, err := workspace.ReadManifest(s.manager.Store.Root())
	if err != nil {
		return result, err
	}
	if err := s.manager.deleteSessionMetadata(manifest.WorkspaceID); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Session) checkpoint(ctx context.Context, closing bool, input CloseInput) (CloseResult, error) {
	if s.tx == nil {
		return CloseResult{}, errors.New("session has no active transaction")
	}
	if closing && s.retainRecord {
		record := Record{
			SchemaVersion: 1,
			ID:            s.id,
			StartedAt:     s.startedAt.Format(time.RFC3339),
			ClosedAt:      s.now().Format(time.RFC3339),
			Harness:       s.harness,
			DeviceID:      s.deviceID,
			BaseRevision:  string(s.initialRevision),
			SyncMode:      s.mode,
			Offline:       s.offline,
			Domains:       uniqueSorted(input.Domains),
			Evidence:      uniqueSorted(input.Evidence),
			Assessments:   uniqueSorted(input.Assessments),
			Summary:       input.Summary,
		}
		data, err := yaml.Marshal(record)
		if err != nil {
			return CloseResult{}, err
		}
		if err := s.tx.Put(filepath.ToSlash(filepath.Join("sessions", s.id+".yaml")), data); err != nil {
			return CloseResult{}, err
		}
	}
	if s.manager.Rebuild != nil {
		stage := s.tx.StageRoot()
		if stage == "" {
			return CloseResult{}, errors.New("session staging workspace is unavailable")
		}
		if err := s.manager.Rebuild(stage); err != nil {
			return CloseResult{}, fmt.Errorf("rebuild staged derived state: %w", err)
		}
	}

	shouldSync := s.mode != SyncManual && s.manager.Store.Capabilities().Has(store.CapabilitySync)
	revision, err := s.tx.CheckpointWithMessage(ctx, shouldSync, "alp: session "+s.id)
	if err != nil {
		return CloseResult{}, err
	}
	s.tx = nil
	result := CloseResult{Revision: revision, SyncPending: shouldSync}
	if !shouldSync {
		return result, nil
	}

	syncer, ok := s.manager.Store.(store.Syncer)
	if !ok {
		return CloseResult{}, fmt.Errorf("store provider %q advertises sync but does not implement Syncer", s.manager.Store.Provider())
	}
	syncResult, syncErr := syncer.Sync(ctx, s.syncOptions)
	result.SyncResult = syncResult
	if syncErr != nil {
		result.SyncPending = true
		return result, syncErr
	}
	result.SyncPending = syncResult.Pending
	if syncResult.Pending {
		s.pendingSync = true
		return result, nil
	}
	manifest, err := workspace.ReadManifest(s.manager.Store.Root())
	if err != nil {
		return result, err
	}
	if err := s.coordinator.CompleteSync(manifest.WorkspaceID); err != nil {
		return result, err
	}
	s.pendingSync = false
	return result, nil
}

func (s *Session) ensureTransaction(ctx context.Context) error {
	if s.tx != nil {
		return nil
	}
	if !s.pendingSync {
		return errors.New("session transaction is unavailable")
	}
	syncer, ok := s.manager.Store.(store.Syncer)
	if !ok {
		return errors.New("pending synchronization cannot be resumed by this provider")
	}
	result, err := syncer.Sync(ctx, s.syncOptions)
	if err != nil {
		return err
	}
	if result.Pending {
		return errors.New("eager session is waiting for pending synchronization before accepting more changes")
	}
	manifest, err := workspace.ReadManifest(s.manager.Store.Root())
	if err != nil {
		return err
	}
	if err := s.coordinator.CompleteSync(manifest.WorkspaceID); err != nil {
		return err
	}
	revision, err := s.manager.Store.Revision(ctx)
	if err != nil {
		return err
	}
	tx, err := s.coordinator.Begin(ctx, s.id, revision)
	if err != nil {
		return err
	}
	s.tx = tx
	s.baseRevision = revision
	s.pendingSync = false
	return s.persistMetadata()
}

func (s *Session) persistMetadata() error {
	manifest, err := workspace.ReadManifest(s.manager.Store.Root())
	if err != nil {
		return err
	}
	return s.manager.writeSessionMetadata(manifest.WorkspaceID, s)
}

func (m *Manager) writeSessionMetadata(workspaceID string, session *Session) error {
	if workspaceID == "" {
		return errors.New("workspace id is required for persisted session metadata")
	}
	dir := filepath.Join(m.RuntimeDir, "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	meta := persistedSession{
		ID:              session.id,
		Harness:         session.harness,
		DeviceID:        session.deviceID,
		Mode:            session.mode,
		RetainRecord:    session.retainRecord,
		StartedAt:       session.startedAt.UTC().Format(time.RFC3339),
		InitialRevision: string(session.initialRevision),
		Offline:         session.offline,
		Remote:          session.syncOptions.Remote,
		Branch:          session.syncOptions.Branch,
		MaxRetries:      session.syncOptions.MaxRetries,
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	path := m.sessionMetadataPath(workspaceID)
	temp, err := os.CreateTemp(dir, ".session-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err := temp.Write(append(data, '\n')); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func (m *Manager) readSessionMetadata(workspaceID string) (persistedSession, error) {
	data, err := os.ReadFile(m.sessionMetadataPath(workspaceID))
	if err != nil {
		return persistedSession{}, err
	}
	var meta persistedSession
	if err := json.Unmarshal(data, &meta); err != nil {
		return persistedSession{}, fmt.Errorf("parse persisted session metadata: %w", err)
	}
	if meta.ID == "" {
		return persistedSession{}, errors.New("persisted session metadata is missing id")
	}
	return meta, nil
}

func (m *Manager) deleteSessionMetadata(workspaceID string) error {
	if err := os.Remove(m.sessionMetadataPath(workspaceID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *Manager) sessionMetadataPath(workspaceID string) string {
	return filepath.Join(m.RuntimeDir, "sessions", workspaceID+".json")
}

func (s *Session) now() time.Time {
	if s.manager.Now != nil {
		return s.manager.Now().UTC()
	}
	return time.Now().UTC()
}

func newID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "sess_" + hex.EncodeToString(raw[:]), nil
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
