# ADR-0012: Persist only learning-relevant personal data

**Status:** Accepted

## Context

Persona discovery and connected repositories/platforms could easily collect more personal data than is useful.

## Decision

Persist only data that materially improves learning, assessment, planning, or continuity. Adapters use purpose-limited allowlists. Secrets and unrelated personal data are excluded.

## Consequences

- better privacy even in private workspaces;
- adapter/export designs must be explicit;
- human views must avoid unnecessary raw sensitive provenance.
