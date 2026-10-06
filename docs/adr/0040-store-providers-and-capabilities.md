# ADR-0040: Model learner persistence as Store providers with explicit capabilities

**Status:** Accepted

## Context

ALP must remain portable across devices, harnesses, and persistence backends. Git is the first production synchronization mechanism, but Git-specific operations must not become the learning engine's persistence contract.

A single broad interface would either leak Git concepts into every provider or force providers to implement operations they cannot support honestly.

## Decision

Model learner persistence through a small provider-independent **Store** boundary plus optional capability interfaces.

The core Store owns provider-independent learner-workspace persistence semantics. Workflows that need additional behavior require explicit capabilities such as:

- revision access;
- checkpointing;
- synchronization;
- history;
- remote management.

Production v0 implements:

- **Git Store** as the default production, revisioned, multi-device provider;
- **Local Store** as a fully functional local-only provider.

The provider contract is also specified for future implementations without shipping them in v0:

- S3-compatible object storage;
- WebDAV;
- remote ALP API;
- PostgreSQL-backed server storage.

GitHub, GitLab, Gitea, and Forgejo are Git remote/onboarding integrations, not separate Store providers.

## Consequences

- the learning model remains independent of Git;
- providers advertise only capabilities they actually support;
- Local Store does not fake history or remote synchronization;
- future storage backends can reuse canonical ALP state semantics;
- provider-specific transport logic stays below the Store boundary.
