# ADR-0052: Convert providers through verified canonical export and make restore intent explicit

**Status:** Accepted

## Context

A learner may begin with Local Store, later adopt Git Store, or eventually move to another provider.

Restore can also mean recovery, duplication, or semantic combination, which are materially different operations.

## Decision

Provider conversion is a first-class production-v0 workflow implemented through provider-independent canonical export/restore semantics.

A provider move:

1. verifies the source workspace;
2. exports canonical learner state;
3. initializes the destination provider;
4. restores and validates canonical state;
5. rebuilds derived state;
6. switches the named workspace only after destination success.

Restore requires an explicit mode:

- **recover** — restore the same `workspaceId` to repair/replace a damaged copy;
- **clone** — create a new `workspaceId` from exported learner state;
- **merge** — semantically reconcile exported state with an existing same-learner workspace.

ALP never guesses restore intent.

## Consequences

- Store provider choice is reversible;
- Local Store can graduate cleanly to Git Store;
- backup restore does not accidentally duplicate workspace identity;
- future providers inherit one portable migration path.
