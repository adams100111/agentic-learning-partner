# ADR-0047: Stage Store transactions and enforce one writer per local workspace

**Status:** Accepted

## Context

Multiple agent harnesses may read the same local learner workspace concurrently, and a process may crash part-way through a logical state transaction. Allowing multiple writers to modify canonical files directly risks partial writes, interleaved state, and recovery ambiguity, and can expose invalid intermediate state.

Revision checks alone detect stale bases too late if both processes have already mutated the working copy.

## Decision

Reads may run concurrently, but only one writer transaction may operate on a local workspace at a time.

Store mutations use an isolated transaction staging area where practical:

1. capture the base Workspace Revision;
2. acquire the workspace writer lock;
3. stage the ChangeSet outside the visible canonical workspace;
4. validate canonical changes;
5. rebuild derived state in staging;
6. validate the complete staged workspace;
7. atomically publish/checkpoint the transaction;
8. release the lock.

The writer lock is process-safe and records operational metadata such as:

- workspace ID;
- session ID;
- PID;
- started-at timestamp.

A stale lock is never discarded blindly. Recovery logic may classify a lock as stale only after determining the owning process is gone and reconciling any associated recovery journal, which points to the staged transaction until the checkpoint completes.

Local Store may use temporary directories plus atomic replacement.

Staging is provider-independent: a transaction writes into a staged copy under the runtime directory, never the canonical workspace. At checkpoint the Git Store publishes each staged file via temp-file-and-rename (rolling back on failure) and commits only ALP-owned paths with `git commit --only`, so partially-written canonical state never becomes visible. A temporary worktree is used only for sync reconciliation, not for staging.

## Consequences

- interrupted writes do not expose half-applied learner state;
- same-machine agent concurrency has a deterministic ownership rule;
- recovery can distinguish an active writer from an abandoned transaction, and has a concrete staged transaction to resume or discard;
- provider implementations remain free to choose the safest staging and atomic-publish mechanism available while preserving the transaction contract.
