# ADR-0059: Derive deterministic evidence identity from platform activity, including synthetic identities for state-only platforms

**Status:** Accepted

## Context

PyLearn evidence IDs are currently `sha256(pylearn|kind|sourceID|contentID)`: no learner, no target, no event revision. Re-import hits append-only `O_EXCL` and errors instead of skipping. Unmapped activity is dropped silently. There is no cursor.

PyLearn has no activity event log: `progress`, `quiz_answers`, `concept_mastery`, and `attempts` are upserted latest-state rows with no event IDs.

## Decision

- Evidence identity = `hash(platform instance, target, learner, item, event identity, event revision)`. Timestamps alone never define identity.
- Platforms with real event IDs supply them. Platforms that only hold latest-state rows get a **synthetic event identity** from the adapter: row key plus a content hash of the evidence-relevant row fields. An unchanged row re-exported is a no-op; a changed row produces new evidence; a snapshot is never treated as many events.
- Import is idempotent: identical evidence is skipped and reported as already imported; a higher revision of the same event supersedes prior evidence.
- Activity on unmapped items is reported as `unmapped` in the import result, never silently dropped.
- The Activity Source contract is "records since cursor"; transport (file, CLI, API) is an adapter detail. ALP never reads platform databases directly.
- PyLearn v0 adds a versioned export CLI emitting `pylearn-export` v2 (with `targets` and cursor) from current tables. An append-only PyLearn activity log is an optional later improvement.

## Consequences

- PyLearn evidence identity changes; existing v0 fixtures need migration;
- state-only platforms lose intermediate history but cannot double-count;
- import results become diagnosable (imported / skipped / superseded / unmapped).

## Alternatives considered

- Require every platform to add an event log first: blocks integration on platform redesign.
- Timestamp-based identity: non-deterministic across re-exports.

## Notes

- **2026-10-07 — "higher revision" is ordered by `observedAt`.** A synthetic event revision is a content hash, so revisions have no intrinsic order. "A higher revision of the same event supersedes prior evidence" is decided by the record's `observedAt`, the time the platform row last changed: a different revision with a later `observedAt` supersedes the event's active evidence, an earlier one is skipped as `stale-revision`, and an equal `observedAt` is skipped as `revision-conflict` (ALP does not guess). `observedAt` orders revisions but is never part of identity. Platforms with real, ordered event revisions are ordered the same way through `observedAt`. See `docs/integrations/PYLEARN_EXPORT.md`. This note clarifies the decision; it does not change it.
