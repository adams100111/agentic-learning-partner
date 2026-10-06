package store

import (
	"context"
	"fmt"
)

type Revision string

type Capability string

const (
	CapabilityPersistence           Capability = "persistence"
	CapabilityOptimisticConcurrency Capability = "optimistic-concurrency"
	CapabilityAtomicCheckpoint      Capability = "atomic-checkpoint"
	CapabilityRevisions             Capability = "revisions"
	CapabilityHistory               Capability = "history"
	CapabilityOffline               Capability = "offline"
	CapabilitySync                  Capability = "sync"
	CapabilityMultiDevice           Capability = "multi-device"
	CapabilityRemoteManagement      Capability = "remote-management"
)

type Capabilities map[Capability]bool

func (c Capabilities) Has(capability Capability) bool { return c[capability] }

type Mutation struct {
	Path   string
	Data   []byte
	Delete bool
}

type ChangeSet struct {
	Message   string     `json:"message,omitempty"`
	Mutations []Mutation `json:"mutations"`
}

type Store interface {
	Provider() string
	Root() string
	Capabilities() Capabilities
	Revision(context.Context) (Revision, error)
	Read(context.Context, string) ([]byte, error)
	Commit(context.Context, Revision, ChangeSet) (Revision, error)
}

type CapabilityError struct {
	Provider    string
	Capability  Capability
	Alternative string
}

func (e CapabilityError) Error() string {
	message := fmt.Sprintf("store provider %q does not support capability %q", e.Provider, e.Capability)
	if e.Alternative != "" {
		message += "; " + e.Alternative
	}
	return message
}

func Require(active Store, capability Capability) error {
	if active == nil {
		return fmt.Errorf("store is required")
	}
	if !active.Capabilities().Has(capability) {
		alternative := ""
		if capability == CapabilitySync {
			alternative = "continue local-only or move this workspace to provider git"
		}
		return CapabilityError{Provider: active.Provider(), Capability: capability, Alternative: alternative}
	}
	return nil
}

type Syncer interface {
	Pull(context.Context, SyncOptions) (SyncResult, error)
	Push(context.Context, SyncOptions) (SyncResult, error)
	Sync(context.Context, SyncOptions) (SyncResult, error)
}
