# ADR-0055: Keep competency truth domain-scoped and course/platform scope as provenance/adaptation context

**Status:** Accepted

## Context

A learner may encounter the same competency through multiple courses, platforms, projects, diagnostics, and harnesses.

For example, `go.concurrency.channels` may be demonstrated in:

- PyLearn's existing Go course;
- a new ALP-generated PyLearn Go target;
- another LMS;
- a real project;
- a Claude/Codex diagnostic or debugging session.

If competency projections are course-scoped, ALP fragments learner truth and cannot reason consistently across learning surfaces.

## Decision

Competency identity and projection remain scoped to **learner × domain competency**, independent of course/platform.

Platform/course/target identity belongs in:

- evidence provenance;
- content mappings;
- Target Adaptation Projections;
- platform activity records.

Course/platform-specific planning is represented as a derived Target Adaptation Projection, not as a separate competency state.

## Consequences

- evidence from multiple platforms accumulates against one competency model;
- course migration does not reset learner knowledge;
- platform-specific adaptation can vary without duplicating learner truth;
- a future LMS or project-based learning surface can contribute evidence to the same domain state;
- target-specific sequencing/reinforcement remains possible through derived adaptation projections.
