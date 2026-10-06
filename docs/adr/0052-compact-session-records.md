# ADR-0052: Persist compact session records without transcripts

**Status:** Accepted

## Context

Cross-device continuity and auditability benefit from knowing what a completed learning session changed.

Raw conversation transcripts are high-volume, privacy-sensitive, and not required for learner-state reconstruction.

## Decision

Production v0 retains compact canonical session records by default.

A session record may contain:

- session ID;
- start/close timestamps;
- harness;
- opaque device ID;
- base/checkpoint revisions;
- domains touched;
- evidence IDs;
- assessment IDs;
- concise summary.

Raw conversation transcripts are not stored by default.

Users may disable canonical session-record retention.

## Consequences

- sessions remain explainable/auditable;
- learner state avoids transcript bloat and unnecessary private content;
- cross-harness continuity can reference durable state transitions.
