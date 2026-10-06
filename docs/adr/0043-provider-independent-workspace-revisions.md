# ADR-0043: Make Workspace Revision provider-independent and opaque

**Status:** Accepted

## Context

Git-backed workspaces have commit SHAs, while Local Store and future providers may not have a Git history.

If ALP defines Workspace Revision as a Git SHA, Git becomes part of the learning engine contract and non-Git providers must emulate Git semantics artificially.

## Decision

Workspace Revision is an opaque Store-owned value.

The learning engine may:

- compare revisions for equality;
- persist the expected/base revision with a logical transaction;
- pass an expected revision back to the Store for optimistic concurrency.

The learning engine must not interpret the internal structure of a revision.

Git Store may encode a Git commit SHA in its revision value.

Local Store may derive a revision from its committed canonical snapshot/transaction state without retaining historical snapshots indefinitely.

## Consequences

- optimistic concurrency remains provider-independent;
- Git-specific history does not leak into the learning model;
- providers can choose honest revision mechanisms.
