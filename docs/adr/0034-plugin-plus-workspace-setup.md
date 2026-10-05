# ADR-0034: Setup is plugin/CLI plus learner workspace

**Status:** Accepted

## Decision

ALP setup configures the reusable plugin/CLI and a separate learner workspace using existing Git credentials.

No custom credential store.

## Consequences

- simpler security model;
- consistent Claude/Codex/terminal setup.
