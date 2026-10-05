# ADR-0010: Separate evidence, assessment, and projection

**Status:** Accepted

## Context

Evidence can be recorded deterministically, but mapping evidence to competency is semantic judgment. Calling projections deterministic while allowing an agent to reinterpret raw evidence during rebuild is inconsistent.

## Decision

Use three layers:

1. evidence: factual observation;
2. assessment: versioned judgment tied to evidence and rubric;
3. projection: deterministic fold of accepted assessments.

Assessments are append-only/supersedable and include assessor/rubric provenance.

## Consequences

- state rebuild is actually deterministic;
- different models' judgments remain auditable;
- rubrics can evolve without rewriting evidence;
- the engine needs an assessment schema and explicit projection policy.
