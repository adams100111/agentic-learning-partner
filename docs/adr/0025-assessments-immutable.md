# ADR-0025: Accepted assessments are immutable and superseded, not rewritten

**Status:** Accepted

## Decision

Corrections create new assessments that reference/supersede older ones. Existing accepted assessment history is not silently mutated.

## Consequences

- complete audit trail;
- rubric/model changes remain explainable.
