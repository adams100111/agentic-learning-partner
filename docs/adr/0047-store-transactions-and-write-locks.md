# ADR-0047: Stage Store transactions and serialize writers per workspace

**Status:** Accepted

## Context

Multiple harness processes may read or mutate the same local learner workspace. Direct in-place writes can expose partial state after crashes, and optimistic revision checks alone do not prevent two local writers from racing while both mutate files.

## Decision

Store mutations use an isolated transaction staging area where practical.

A transaction:

1. begins from a pinned Workspace Revision;
2. stages canonical mutations outside the published workspace state;
3. validates canonical documents;
4. rebuilds derived state;
5. validates the final staged workspace;
6. atomically publishes/checkpoints the transaction.

The recovery journal points to the staged transaction until commit completes.

Local mutation concurrency is:

- concurrent readers allowed;
- one writer transaction per workspace per device.

Use an OS/process-safe workspace write lock containing enough metadata to diagnose/recover stale locks, including workspace ID, session ID, process identity, and start time.

A stale lock is never discarded blindly; recovery first determines whether an interrupted transaction exists.

## Consequences

- partial writes are not exposed as canonical state;
- same-device write races are prevented before mutation;
- crash recovery has a concrete staged transaction to resume or discard;
- Store providers may implement atomic publish differently while preserving the transaction contract.
