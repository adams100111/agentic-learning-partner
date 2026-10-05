# ADR-0014: Commit derived projections as generated caches

**Status:** Accepted

## Context

Rebuilding all projections on every harness startup can become expensive as evidence grows, but derived state must not become a second authority.

## Decision

Commit selected derived projections/materialized views to the learner workspace as generated caches.

They are non-authoritative and must be regenerated after canonical-input merges or conflicts.

## Consequences

- fast Claude/Codex startup;
- readable current state in Git;
- CI can detect projection drift;
- humans/agents must never resolve generated-state conflicts manually.
