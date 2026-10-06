# ADR-0052: Separate deterministic Store acceptance tests from real-harness release smoke tests

**Status:** Accepted

## Context

Production-v0 must prove multi-device and cross-harness behavior without making normal automated tests depend on external Git hosting or SaaS availability.

## Decision

Production-v0 verification has two levels.

### Deterministic automated acceptance

Use Local Store and local bare Git remotes to prove:

- initialization;
- clone/connect;
- named workspace switching;
- capability failures;
- session lifecycle;
- offline work;
- checkpointing;
- semantic reconciliation;
- push races;
- same-device write locking;
- interrupted transaction recovery;
- provider conversion;
- export/verify/restore;
- v1 -> v2 workspace migration;
- two independent local clones representing separate devices.

### Release smoke tests

Before production-v0 tagging, manually/agentically verify:

- real private GitHub remote;
- current Claude Code plugin installation;
- current Codex/OpenAI plugin installation;
- device A -> device B continuation;
- cross-harness continuation without shared conversation memory.

## Consequences

- the core suite remains deterministic and fast;
- real ecosystem integration is still a release gate;
- hosted-provider outages do not destabilize ordinary development tests.
