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

func Require(store Store, capability Capability) error {
	if store == nil {
		return fmt.Errorf("store is required")
	}
	if !store.Capabilities().Has(capability) {
		return fmt.Errorf("store provider %q does not support capability %q", store.Provider(), capability)
	}
	return nil
}
