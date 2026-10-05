# ADR-0024: Classify negative evidence before state effect

**Status:** Accepted

## Decision

Failures are assessed for cause before affecting competency. Raw failure counts never directly downgrade state.

## Consequences

- syntax/tooling/transient mistakes do not incorrectly erase conceptual competence;
- repeated conceptual failure can still drive downgrade/reassessment.
