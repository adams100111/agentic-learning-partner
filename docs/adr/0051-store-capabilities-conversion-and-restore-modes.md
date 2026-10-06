# ADR-0051: Fail explicitly on missing Store capabilities and make provider conversion/restore intent explicit

**Status:** Accepted

## Context

Store providers have intentionally different capabilities. Silent degradation would make workflows unpredictable.

Moving or restoring a workspace can also mean different things: recovery of the same workspace, creation of a distinct workspace, or semantic merge.

## Decision

A workflow that requires an unavailable Store capability fails explicitly and reports the nearest valid alternative.

Provider conversion is a first-class production-v0 workflow implemented through provider-independent export/restore semantics.

Restore intent is explicit:

- **recover**: restore the same `workspaceId` after validation and confirmation;
- **clone**: create a new `workspaceId` from exported learner state;
- **merge**: semantically reconcile compatible learner state.

ALP never guesses restore intent.

The Store capability/logical contract is normative in production v0.

Descriptions of future S3/WebDAV/remote-API/PostgreSQL providers are informative until those providers are implemented.

## Consequences

- Local Store never pretends to support synchronization/history;
- provider conversion is testable independently of Git;
- restore cannot silently overwrite or duplicate workspace identity.
