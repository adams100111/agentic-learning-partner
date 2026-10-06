# ADR-0048: Give workspaces and devices distinct stable identities

**Status:** Accepted

## Context

Learner identity is not sufficient to identify storage or operational provenance.

One learner may have multiple workspaces, and the same workspace may be used from multiple devices/harnesses.

## Decision

Every learner workspace has an immutable provider-independent `workspaceId` distinct from `learnerId`.

The workspace manifest advances to schema version 2 and includes:

- `workspaceId`;
- `learnerId`.

A v1 -> v2 migration assigns a new stable workspace ID without changing learner identity.

Each device also has a machine-local opaque random device ID, with an optional user-chosen name.

Device identity is stored outside synchronized learner state.

Compact session/evidence provenance may reference device ID and harness identity.

## Consequences

- journals/config/export can identify the workspace independently of the learner;
- one learner can safely use multiple workspaces;
- device provenance is available without turning devices into credentials;
- workspace migration is required before production v0 release.
