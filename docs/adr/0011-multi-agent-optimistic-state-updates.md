# ADR-0011: Merge-friendly canonical events and optimistic state updates

**Status:** Accepted

## Context

Claude Code, Codex, ChatGPT, or multiple machines may concurrently append learner evidence/assessments and rebuild derived state.

Append-only records merge well; mutable generated projections do not. Git history alone does not prevent stale-base projection writes.

## Decision

- canonical evidence and assessments use collision-resistant sortable IDs such as UUIDv7 or ULID;
- canonical events are append-only except explicit correction/supersession records;
- imports are idempotent, so re-importing the same source activity does not create duplicate canonical records (see ADR-0059 for activity import identity);
- derived projections are never manually conflict-merged;
- after Git reconciliation, projections are regenerated;
- update transactions carry an expected workspace revision/commit and fail cleanly on stale writes;
- retry requires re-reading the new canonical head before recomputing.

## Consequences

- concurrent agents can safely contribute independent evidence;
- Git conflicts are concentrated in true mutable configuration rather than event history;
- the CLI must implement optimistic concurrency and rebuild semantics;
- import/event IDs need stable uniqueness semantics.
