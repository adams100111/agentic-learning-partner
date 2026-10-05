# Concurrent Agents and Git State

## Problem

Claude Code, Codex, ChatGPT, or multiple machines may operate on the same learner workspace.

Append-only evidence reduces conflicts but does not eliminate stale-base writes or derived-state races.

## Rules

### Collision-resistant IDs

Evidence and assessment IDs must be generated independently without central coordination. Prefer UUIDv7 or ULID.

### Optimistic concurrency

Every state mutation transaction records/validates the workspace base revision (normally the Git commit SHA).

If the remote/base revision moved:

1. fetch/reconcile;
2. append independent records;
3. regenerate derived projections;
4. retry the commit.

Do not overwrite newer state blindly.

### Derived state is never hand-merged

After Git reconciliation, rebuild derived projections from canonical inputs.

### One logical transaction

A persona-wizard or learning-session close may modify several canonical files. Commit them as one logical transaction where possible.

### Idempotent ingestion

Platform imports need stable source identities so repeating the same import does not duplicate evidence.

### Cross-harness rule

No harness may maintain an authoritative private shadow state that cannot be reconstructed from the workspace.
