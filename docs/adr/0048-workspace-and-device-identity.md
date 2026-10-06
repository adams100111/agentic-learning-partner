# ADR-0048: Give workspaces stable provider-independent identity and devices local provenance identity

**Status:** Accepted

## Context

Learner identity is not sufficient to identify a storage workspace when one learner may use multiple workspaces, exports, restores, devices, and providers.

Operational provenance also benefits from identifying the device that produced a session without treating device identity as authentication.

## Decision

Production v0 introduces an immutable provider-independent `workspaceId`.

The workspace manifest evolves to schema version 2 conceptually containing:

```yaml
schemaVersion: 2
workspaceId: ws_...
learnerId: adams
```

A proper workspace schema migration upgrades existing v1 workspaces.

ALP also creates a machine-local opaque `deviceId` and optional user-chosen device name.

Device identity:

- is provenance/debug metadata;
- is not authentication;
- does not contain credentials;
- remains machine-local configuration;
- may be referenced by canonical compact session/evidence provenance.

## Consequences

- journals, exports, Store operations, named-workspace config, and restore semantics can refer to stable workspace identity;
- one learner may safely own multiple independent workspaces;
- device provenance is visible without coupling trust to a machine identifier.
