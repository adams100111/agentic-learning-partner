# ADR-0041: Make Git Store synchronization an ALP-managed semantic workflow

**Status:** Accepted

## Context

A learner may use Claude Code, Codex, or other ALP harnesses from multiple devices against one private learner workspace.

Raw Git commands are insufficient as the product API because ALP state has semantic classes:

- append-only evidence and assessments;
- human-maintained profile/persona state;
- rebuildable derived projections.

Agents also must not receive or manage Git credentials directly.

## Decision

The Git Store always operates on a local checkout and uses ordinary Git remotes for synchronization.

Normal agent workflows use constrained ALP operations rather than arbitrary Git commands.

Default synchronization mode is **session**:

1. session start fetches and reconciles remote state;
2. the session records a base Store revision;
3. learning mutations accumulate locally as one logical ChangeSet;
4. session close validates canonical state;
5. derived state is regenerated;
6. ALP creates one checkpoint;
7. ALP pushes the checkpoint.

Also support explicit `manual` and `eager` synchronization policies.

Conflict handling is semantic:

- append-only records are reconciled by stable collision-resistant identity;
- derived state is discarded and regenerated;
- profile/persona conflicts are provenance-aware where unambiguous;
- conflicting explicit learner decisions require learner resolution;
- unsupported conflicts stop safely.

Git Store synchronization uses the host's existing Git/SSH credential infrastructure. ALP does not store credentials.

The ALP Git surface does not expose force-push, history rewriting, destructive reset, remote branch deletion, or arbitrary ref mutation to agents.

## Consequences

- Git becomes an implementation of Store synchronization rather than the ALP domain model;
- multi-device concurrency uses existing workspace-revision semantics;
- session history is readable rather than one commit per small mutation;
- agents cannot silently bypass state invariants with raw Git operations;
- humans may still use ordinary Git outside ALP for recovery or advanced workflows.
