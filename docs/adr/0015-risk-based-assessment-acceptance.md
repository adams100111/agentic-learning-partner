# ADR-0015: Use risk-based assessment acceptance

**Status:** Accepted

## Context

Manual approval for every assessment creates unusable friction; unconditional agent authority creates unsafe learner-state drift.

## Decision

Routine rubric-valid assessments may auto-accept. Higher-consequence transitions, especially production-ready and materially consequential persona inferences, use stronger gates/confirmation.

## Consequences

- low-friction ongoing learning;
- hard promotion rules remain enforceable;
- acceptance policy becomes part of deterministic state semantics.
