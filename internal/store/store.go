package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
)

type Revision string

func (r Revision) String() string { return string(r) }
func (r Revision) Empty() bool    { return r == "" }

type Capability string

const (
	CapabilityPersistence            Capability = "persistence"
	CapabilityOptimisticConcurrency  Capability = "optimistic-concurrency"
	CapabilityAtomicCheckpoint       Capability = "atomic-checkpoint"
	CapabilityRevisions              Capability = "revisions"
	CapabilityHistory                Capability = "history"
	CapabilityOffline                Capability = "offline"
	CapabilitySync                   Capability = "sync"
	CapabilityMultiDevice            Capability = "multi-device"
	CapabilityRemoteManagement       Capability = "remote-management"
)

type Capabilities map[Capability]bool

func (c Capabilities) Has(capability Capability) bool {
	return c[capability]
}

type CapabilityError struct {
	Provider   string
	Capability Capability
	Alternative string
}

func (e CapabilityError) Error() string {
	message := fmt.Sprintf("store provider %q does not support capability %q", e.Provider, e.Capability)
	if e.Alternative != "" {
		message += "; " + e.Alternative
	}
	return message
}

type Workspace struct {
	ID        string
	LearnerID string
	Path      string
}

type Entry struct {
	Path string
	Mode fs.FileMode
}

type Change struct {
	Path   string
	Data   []byte
	Delete bool
}

type ChangeSet struct {
	Expected Revision
	Changes  []Change
}

type Checkpoint struct {
	Revision Revision
	Summary  string
}

var ErrConflict = errors.New("store revision conflict")

type Store interface {
	Provider() string
	Capabilities() Capabilities
	Workspace(context.Context) (Workspace, error)
	Revision(context.Context) (Revision, error)
	Read(context.Context, string) ([]byte, error)
	List(context.Context, string) ([]Entry, error)
	Begin(context.Context, Revision) (Transaction, error)
}

type Transaction interface {
	Put(path string, data []byte) error
	Delete(path string) error
	Changes() []Change
	Commit(context.Context, string) (Checkpoint, error)
	Rollback() error
}

type SyncResult struct {
	Before  Revision
	After   Revision
	Changed bool
	Pending bool
}

type Syncer interface {
	Pull(context.Context) (SyncResult, error)
	Push(context.Context) (SyncResult, error)
	Sync(context.Context) (SyncResult, error)
}

type HistoryEntry struct {
	Revision Revision
	Summary  string
}

type HistoryReader interface {
	History(context.Context, int) ([]HistoryEntry, error)
}

func Require(provider string, capabilities Capabilities, required ...Capability) error {
	for _, capability := range required {
		if !capabilities.Has(capability) {
			alternative := ""
			if capability == CapabilitySync {
				alternative = "continue local-only or configure a sync-capable provider such as git"
			}
			return CapabilityError{Provider: provider, Capability: capability, Alternative: alternative}
		}
	}
	return nil
}
