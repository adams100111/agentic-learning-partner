# Production v0

## Meaning

Production v0 is ALP's first production-ready release line.

The `0.x` version indicates that public compatibility guarantees may still evolve. It does not mean MVP, prototype, incomplete recovery semantics, or reduced reliability.

## Release acceptance

Production v0 is not complete until the following lifecycle works without manual Git/YAML intervention during normal operation.

### Installation and workspace lifecycle

- install the portable ALP plugin and CLI;
- initialize a new learner workspace;
- connect/clone an existing workspace;
- configure and switch among named workspaces;
- validate and migrate workspace state;
- discover the active workspace predictably.

### Store providers

- Local Store is fully functional;
- Git Store is fully functional;
- required capabilities are explicit;
- unsupported capability requests fail safely.

### Git synchronization

- session-start reconciliation;
- logical session checkpoint;
- push after successful checkpoint;
- explicit pull/push/sync commands;
- optimistic remote revision checks;
- append-only record reconciliation;
- derived-state regeneration;
- semantic profile/persona conflict handling;
- interrupted-sync recovery;
- no unsafe Git operations exposed to agents.

### Security

- ALP stores no Git credentials;
- host SSH/credential helpers provide authentication;
- state remotes use secure transport;
- normal Git Store setup prefers private remotes;
- state remains purpose-limited and excludes secrets;
- dangerous/destructive operations require an explicit recovery path outside normal agent APIs.

### Portability and recovery

- provider-independent workspace export;
- export integrity manifest/checksums;
- verification before restore;
- restore into a supported provider;
- workspace remains valid after restore/migration.

### Agentic onboarding

- setup wizard inspects existing environment/state first;
- new users can create a workspace;
- existing users can connect their state;
- Git remote creation may use a securely connected forge integration;
- the wizard asks only unresolved user decisions;
- deterministic CLI/runtime operations perform mutations.

### Multi-device / cross-harness acceptance

A production acceptance test must prove:

1. device A creates or connects state;
2. a real harness records genuine learning evidence;
3. state checkpoints and synchronizes;
4. device B clones/connects the same learner state;
5. a different harness continues without shared chat memory;
6. concurrent independent append-only changes reconcile;
7. canonical document conflicts stop or resolve according to policy;
8. derived state is reproducible after reconciliation;
9. interrupted session/sync can recover safely.

## Current status

The repository currently contains the ALP Core Foundation. It is not yet the production-v0 release under this definition.
