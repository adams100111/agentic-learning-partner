# ADR-0003: Evidence-first, Git-backed durable state

**Status:** Accepted

## Context

Harness chat memory is not portable or auditable. Agent-written summary scores can drift or become overconfident.

## Decision

Persist append-only evidence in Git. Derive current competency, focus, and reinforcement/review projections from evidence and rubrics using deterministic tooling wherever possible.

Git is the synchronization and audit transport. Chat memory is convenience only.

## Consequences

- State can move between Claude Code, Codex, and machines.
- Historical reasoning remains inspectable.
- Merge conflicts are reduced by append-only evidence.
- We must build schemas, validation, projection rebuild, and provenance.
