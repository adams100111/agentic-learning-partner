# ADR-0051: Retain compact session records without conversation transcripts

**Status:** Accepted

## Context

Cross-device continuity, auditability, and debugging benefit from knowing which learning session produced evidence/assessments.

Persisting raw chat transcripts would create unnecessary privacy and storage costs.

## Decision

Production v0 retains compact canonical session records by default.

A session record may contain:

- session ID;
- start/close timestamps;
- harness identifier;
- opaque device ID;
- base Workspace Revision;
- checkpoint correlation/session ID; the resulting checkpoint revision is resolved from Store history/recovery metadata rather than embedded self-referentially;
- domains/competencies touched;
- evidence IDs;
- assessment IDs;
- concise session summary.

Raw conversation transcripts are not retained by default.

Users may disable canonical session-record retention.

Operational crash/recovery journals remain machine-local and are not canonical session records.

## Consequences

- learning history remains inspectable across harnesses/devices;
- state provenance improves without mirroring private chats;
- session history stays compact;
- privacy minimization remains compatible with useful auditability.
