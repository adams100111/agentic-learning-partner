# ADR-0044: Reconcile ALP state semantically before constructing Git history

**Status:** Accepted

## Context

Raw Git merge/rebase works on text, while ALP state has different semantic classes:

- append-only evidence and assessments;
- human-maintained profile/persona documents;
- derived projections and caches.

Textual merge conflict resolution is not sufficient to decide learner truth.

## Decision

Git Store reconciliation operates on the ALP semantic model first, then creates Git history.

Given a base, local state, and remote state:

1. fetch remote;
2. identify the base/common revision;
3. inspect canonical changes on both sides;
4. reconcile append-only records by stable identity;
5. merge non-overlapping profile/persona changes;
6. resolve profile/persona changes by provenance when unambiguous;
7. require learner resolution for competing incompatible explicit learner decisions;
8. discard conflicting derived state;
9. regenerate derived projections/caches;
10. validate the reconciled workspace;
11. create a checkpoint based on the current remote tip;
12. push with optimistic race detection.

Git textual merge is an implementation/recovery tool, not the authority for learner semantics.

Push races use bounded optimistic retry. Production v0 retries at most three reconciliation/push attempts before returning a concurrency error.

Force-push is never part of normal ALP synchronization.

## Consequences

- canonical learner semantics determine conflict resolution;
- derived-state conflicts disappear through deterministic rebuild;
- concurrent append-only learning events reconcile naturally;
- Git remains the transport/history substrate without becoming the domain authority.
