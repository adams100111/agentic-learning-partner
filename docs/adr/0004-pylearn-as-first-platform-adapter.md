# ADR-0004: PyLearn is the first platform integration

**Status:** Accepted

## Context

PyLearn already contains AI-authored personalized curricula, learner evaluation documents, real exercise/quiz/mastery telemetry, and source-code learning evidence.

## Decision

Use PyLearn as ALP's first external learning-platform adapter and validation environment.

ALP will initially read/normalize PyLearn signals and propose changes. Automated content mutation is deferred until provenance, state, and quality gates are proven.

## Consequences

- ALP is validated against a real, non-trivial learning system immediately.
- Existing PyLearn learner mechanisms can seed ALP design.
- ALP must remain platform-neutral rather than becoming "PyLearn internals extracted".
