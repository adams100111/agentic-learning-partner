# ADR-0011: Merge-friendly canonical events and optimistic state updates

**Status:** Accepted

## Context

Claude Code, Codex, ChatGPT, or multiple machines may concurrently append learner evidence/assessments and rebuild derived state.

Append-only records merge well; mutable generated projections do not.

## Decision

- canonical evidence and assessments use collision-resistant sortable IDs such as UUIDv7 or ULID;
- canonical events are append-only except explicit correction/supersession records;
- derived projections are never manually conflict-merged;
- after Git reconciliation, projections are regenerated;
- update transactions carry an expected workspace revision/commit and fail cleanly on stale writes;
- retry requires re-reading the new canonical head before recomputing.

## Consequences

- concurrent agents can safely contribute independent evidence;
- Git conflicts are concentrated in true mutable configuration rather than event history;
- the CLI must implement optimistic concurrency and rebuild semantics.
