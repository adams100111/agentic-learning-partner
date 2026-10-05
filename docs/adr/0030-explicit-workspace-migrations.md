# ADR-0030: Workspace migrations are explicit and recoverable

**Status:** Accepted

## Decision

Breaking persisted-state migrations require check/dry-run/migrate workflow, Git checkpointing, validation, and separate migration commits.

No silent destructive upgrades.

## Consequences

- safer engine upgrades;
- migration tooling becomes a first-class CLI feature.
