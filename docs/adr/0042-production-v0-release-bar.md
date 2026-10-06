# ADR-0042: Treat v0 as production-ready pre-1.0, not as an MVP

**Status:** Accepted

## Context

ALP already has an executable core, but installation and multi-device lifecycle still require product-level work.

Calling the current core "v0 complete" would incorrectly equate a pre-1.0 version number with prototype or MVP quality.

## Decision

ALP v0 is the first production-ready release line. Pre-1.0 means the compatibility contract may still evolve; it does not relax reliability, security, migration, recovery, or lifecycle requirements.

Do not tag the first production release until a normal user can:

- install ALP;
- initialize or connect a learner workspace;
- use Local Store or Git Store;
- securely synchronize Git-backed state;
- recover interrupted sessions/sync;
- reconcile supported multi-device conflicts;
- migrate workspace state;
- export, verify, and restore provider-independent state;
- use multiple named one-learner workspaces on one device;
- learn on one device/harness and continue on another from canonical state;
- complete normal workflows without manually editing Git or YAML.

The current implementation is therefore described as the **ALP Core Foundation** until the production-v0 acceptance bar passes.

## Consequences

- v0 scope includes lifecycle/security/recovery work, not only learning-engine features;
- production acceptance tests become part of release readiness;
- later v1.0 primarily strengthens public compatibility guarantees rather than introducing basic production quality.
