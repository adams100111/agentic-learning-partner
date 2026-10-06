# ADR-0049: Make provider conversion and restore semantics explicit

**Status:** Accepted

## Context

Provider-independent export enables Local Store, Git Store, and future providers to exchange canonical learner state.

Restoring into an existing workspace can mean recovery, cloning, or semantic merge, which are materially different operations.

## Decision

Production v0 supports first-class provider conversion using verified export/restore semantics.

Provider conversion conceptually performs:

1. verify source Store;
2. export canonical learner state;
3. initialize destination Store;
4. restore and validate canonical state;
5. rebuild derived state;
6. switch active workspace only after success.

Restore modes are explicit:

- **recover**: restore the same workspace identity after damage/loss;
- **clone**: create a new workspace identity from exported learner state;
- **merge**: semantically reconcile exported state with an existing compatible learner workspace.

ALP never guesses the intended mode.

## Consequences

- Store providers remain replaceable;
- migration between Local and Git Store is a supported product workflow;
- destructive restore mistakes are reduced by explicit semantics.
