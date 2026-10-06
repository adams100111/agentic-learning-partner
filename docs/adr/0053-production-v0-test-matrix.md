# ADR-0053: Separate deterministic Store acceptance from real-harness release smoke tests

**Status:** Accepted

## Context

Production readiness requires testing multi-device Git behavior and real harness installation, but ordinary deterministic tests should not depend on GitHub availability, network quota, or external service health.

## Decision

Production v0 uses two acceptance levels.

### Deterministic automated suite

Use Local Store and local bare Git remotes to test:

- workspace initialization;
- clone/connect;
- named workspace configuration and switching;
- provider capabilities, including explicit capability failures;
- session lifecycle and checkpointing;
- optimistic revisions;
- atomic transaction staging;
- same-device writer locking;
- two-clone multi-device behavior;
- append-only reconciliation;
- profile/persona conflict behavior;
- push races and bounded retry;
- offline sessions and pending sync;
- interrupted transactions and recovery journals;
- workspace schema migration;
- export/verify/restore;
- Local Store to Git Store conversion.

### Release smoke tests

Before production-v0 release, execute real workflows using:

- a private GitHub remote;
- current Claude Code plugin installation;
- current Codex plugin installation;
- two independent device/workspace clones, proving device A -> device B continuation.

The smoke test proves cross-harness continuation without shared conversation memory.

Future provider sections for S3/WebDAV/remote API/PostgreSQL are informative until implemented. The normative contract is the Store logical/capability model.

## Consequences

- core tests remain fast and deterministic;
- external integration failures do not destabilize ordinary correctness checks;
- real release validation still proves actual GitHub/harness behavior, so real ecosystem integration remains a release gate;
- speculative future provider details do not become premature compatibility promises.
