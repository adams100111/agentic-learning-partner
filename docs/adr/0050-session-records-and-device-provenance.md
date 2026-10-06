# ADR-0050: Retain compact session records and machine-local device identity

**Status:** Accepted

## Context

Cross-device and cross-harness learning needs enough provenance to explain where a checkpoint came from without storing raw conversations.

## Decision

Production v0 creates a machine-local opaque device identity.

Device configuration is not synchronized.

Completed ALP sessions retain a compact canonical session record by default containing only operational/learning provenance such as:

- session ID;
- start/close timestamps;
- harness;
- opaque device ID;
- base Workspace Revision;
- checkpoint Workspace Revision;
- domains touched;
- evidence IDs;
- assessment IDs;
- compact summary.

Raw conversation transcripts are not retained by default.

Users may disable canonical session-record retention.

## Consequences

- audit and continuity work across devices;
- learner state does not become a chat archive;
- device provenance is informative rather than an authentication credential.
