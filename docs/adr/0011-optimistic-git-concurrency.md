# ADR-0011: Optimistic concurrency for Git-backed learner state

**Status:** Accepted

## Context

Multiple agents/machines may append learner state concurrently. Git history alone does not prevent stale-base projection writes.

## Decision

Use collision-resistant canonical record IDs, expected workspace revisions, append-friendly records, idempotent imports, and post-reconciliation projection rebuilds.

Derived projections are never manually merged.

## Consequences

- concurrent sessions can reconcile safely;
- mutation operations need base-revision checks;
- import/event IDs need stable uniqueness semantics.
