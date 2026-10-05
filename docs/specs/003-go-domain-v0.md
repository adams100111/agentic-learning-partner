# Spec 003 — Go Domain Pack v0

**Status:** Draft-ready for implementation planning

## Objective

Create the first domain pack proving that ALP can diagnose and teach Go independently of any learning platform's curriculum structure.

## Baseline

Target Go 1.27.x as verified on 2026-10-05 against official Go sources.

Language/runtime claims use official Go sources.

Ecosystem choices are intentionally deferred until separately verified.

## Inputs

- `domains/go/competencies.yaml`;
- `domains/go/DIAGNOSTIC.md`;
- `domains/go/SOURCES.md`;
- global learner profile/persona;
- Go domain persona;
- evidence/state.

## Required pack capabilities

### Taxonomy

Expose stable competency IDs independent of lesson/platform IDs.

### Diagnostic

Generate adaptive activities and map observations to competency evidence.

### Transfer-aware teaching

Use prior stacks as scaffolding based on semantic closeness, not a rigid universal comparison order.

### Idiomatic review

Review code for:

- unnecessary interfaces/layers;
- package ownership;
- error handling;
- context propagation;
- goroutine ownership;
- synchronization;
- resource lifetime;
- testing;
- production lifecycle.

### Modernity

Mark references/choices with freshness class and verification metadata.

### Planning

Create a learning sequence from evidence rather than assuming every learner starts at taxonomy item 1.

## Initial diagnostic outcome

The initial learner begins with:

- historical Go exposure;
- very rusty self-reported recall;
- no claimed production Go experience.

All actual competency levels remain unassessed until evidence is collected.

## Acceptance tests

1. forgotten syntax does not downgrade shared architecture ability;
2. strong architecture does not auto-promote Go idiomatic competencies;
3. a learner who demonstrates slice mastery can skip remedial slice teaching;
4. concurrency weakness can become the next focus even if backend concepts are already familiar;
5. production-ready cannot be awarded by quiz/diagnostic evidence alone;
6. Go planning does not depend on PyLearn phase names;
7. version-sensitive claims cite/resolve to current source metadata.
