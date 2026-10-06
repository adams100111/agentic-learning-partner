package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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
	Store       store.Store
	Validator   *workspace.Validator
	RuntimeDir  string
	Rebuild     func(root string) error
	Now         func() time.Time
}

type BeginOptions struct {
	ID             string
	Harness        string
	DeviceID       string
	Mode           SyncMode
	RetainRecord   *bool
	Sync           store.SyncOptions
}

type CloseInput struct {
	Domains      []string
	Evidence     []string
	Assessments  []string
	Summary      string
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

type Session struct {
	manager      *Manager
	coordinator  *store.Coordinator
	tx           *store.Transaction
	id           string
	harness      string
	deviceID     string
	mode         SyncMode
	retainRecord bool
	startedAt    time.Time
	baseRevision store.Revision
	offline      bool
	syncOptions  store.SyncOptions
	closed       bool
	pendingSync  bool
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
	return &Session{
		manager: m, coordinator: coordinator, tx: tx,
		id: id, harness: options.Harness, deviceID: options.DeviceID,
		mode: mode, retainRecord: retain, startedAt: now().UTC(),
		baseRevision: base, offline: offline, syncOptions: options.Sync,
	}, nil
}

func (s *Session) ID() string { return s.id }
func (s *Session) BaseRevision() store.Revision { return s.baseRevision }
func (s *Session) Offline() bool { return s.offline }

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
	return result, nil
}

func (s *Session) checkpoint(ctx context.Context, closing bool, input CloseInput) (CloseResult, error) {
	if s.tx == nil {
		return CloseResult{}, errors.New("session has no active transaction")
	}
	if closing && s.retainRecord {
		record := Record{
			SchemaVersion: 1,
			ID: s.id,
			StartedAt: s.startedAt.Format(time.RFC3339),
			ClosedAt: s.now().Format(time.RFC3339),
			Harness: s.harness,
			DeviceID: s.deviceID,
			BaseRevision: string(s.baseRevision),
			SyncMode: s.mode,
			Offline: s.offline,
			Domains: uniqueSorted(input.Domains),
			Evidence: uniqueSorted(input.Evidence),
			Assessments: uniqueSorted(input.Assessments),
			Summary: input.Summary,
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
	return nil
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
