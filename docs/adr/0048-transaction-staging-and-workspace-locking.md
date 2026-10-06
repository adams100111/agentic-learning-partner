# ADR-0048: Stage Store transactions in isolation and allow one writer per workspace per device

**Status:** Accepted

## Context

Two harnesses may operate concurrently on one device, and a process may crash part-way through a logical state transaction.

Allowing direct partial writes to the canonical workspace makes recovery ambiguous and can expose invalid intermediate state.

## Decision

Store mutations execute through isolated transaction staging where practical.

A transaction:

1. pins a base Workspace Revision;
2. acquires the workspace write lock;
3. stages the logical ChangeSet outside the canonical published state;
4. validates canonical changes;
5. regenerates derived state;
6. validates the resulting workspace;
7. atomically publishes/checkpoints;
8. releases the lock.

Local Store may use a temporary directory plus atomic replacement semantics.

Git Store may use an internal temporary worktree/index strategy or equivalent isolation.

Reads may occur concurrently.

Only one writer transaction may be active for a workspace on one device.

The process-safe write lock records operational metadata such as workspace ID, session ID, process ID, and start time. Stale locks are handled through recovery logic rather than blindly deleted.

## Consequences

- crashes do not normally expose half-written canonical state;
- same-device agents cannot overwrite each other through concurrent mutation;
- recovery can reason about staged work and checkpoint state explicitly.
