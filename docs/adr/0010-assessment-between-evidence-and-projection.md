# ADR-0010: Separate evidence, assessment, and projection

**Status:** Accepted

## Context

Raw evidence does not deterministically imply a competency level. Two agents can interpret the same code/exercise evidence differently even if the projection algorithm itself is deterministic.

Evidence itself can be recorded deterministically; mapping it to competency is the semantic step. Calling projection rebuild "deterministic" without persisting the semantic judgment would be false, and letting an agent reinterpret raw evidence during rebuild would be inconsistent with that claim.

## Decision

Introduce **Assessment** as a canonical, append-only semantic judgment between evidence and projection.

Flow:

```
Evidence -> Assessment -> Deterministic Projection
```

An assessment records:

- assessment ID;
- evidence IDs;
- competency ID;
- rubric/domain-pack version;
- assessor identity/type;
- judgment;
- confidence;
- rationale;
- timestamp;
- optional supersession/correction relationship.

Derived competency projections fold accepted assessments using deterministic rules.

Evidence remains observational. Assessments remain judgments. Projections remain rebuildable views.

## Consequences

- different agents can disagree without silently corrupting state, and each model's judgment remains auditable through its assessor/rubric provenance;
- reassessment can supersede prior judgment without rewriting evidence;
- rubrics can evolve without rewriting evidence;
- projection rebuild becomes genuinely deterministic given canonical assessments;
- we need assessment schemas, conflict-resolution rules, and an explicit projection policy.
