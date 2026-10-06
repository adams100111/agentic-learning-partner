# ADR-0047: Give every learner workspace an immutable provider-independent identity

**Status:** Accepted

## Context

One learner may use multiple workspaces on one device, and a workspace may move between Store providers.

Using `learnerId` as storage identity is ambiguous because learner identity and workspace identity are different concepts.

## Decision

Every workspace has an immutable provider-independent `workspaceId`.

Production-v0 workspace metadata evolves to:

```yaml
schemaVersion: 2
workspaceId: ws_...
learnerId: ...
```

Named workspace aliases live in machine-local ALP configuration and point to workspace/provider configuration.

One workspace still represents exactly one learner.

A v1 -> v2 workspace migration creates a stable workspace ID without changing learner identity.

## Consequences

- recovery journals, exports, device provenance, Store operations, and config can identify a workspace independently of its learner;
- one learner can intentionally maintain multiple workspaces;
- moving between providers does not change workspace identity.
