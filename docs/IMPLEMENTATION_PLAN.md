# Implementation Plan

This document turns the product roadmap into buildable increments. Each increment should become a feature spec before code is written.

## Contract status

Documentation/specification now exists for:

- I1 foundation contracts: `docs/specs/001-foundation-contracts.md`;
- persona wizard: `docs/specs/002-persona-wizard.md`;
- Go domain v0: `docs/specs/003-go-domain-v0.md`;
- human views: `docs/specs/004-human-views.md`;
- PyLearn read adapter: `docs/specs/005-pylearn-read-adapter.md`.

The next implementation step after review is I1: schema validation + CLI skeleton, not more free-form architecture expansion.

## I1 — Repository and schema foundation

Deliver:

- portable `plugin.json` skeleton;
- Go module for the ALP CLI;
- `schemas/` for profile, persona, evidence, competency taxonomy, projection, context bundle, adaptation proposal;
- fixtures;
- schema validation tests;
- docs CI.

No learner inference yet. Canonical YAML + schema validation is established here; HTML/Markdown views remain derived.

## I2 — Persona, context, evidence store and projections

Deliver CLI capabilities:

- `alp validate`;
- `alp persona show`;
- `alp persona diff`;
- `alp context build`;
- `alp context inspect`;
- `alp evidence add`;
- `alp state rebuild`;
- `alp status`.

Properties:

- append-only evidence;
- deterministic projection rebuild;
- provenance validation;
- no duplicate IDs;
- no unknown competency paths.

## I3 — Core learning skills

Initial portable skills:

- `discover-learner-persona`;
- `refine-learner-persona`;
- `assess-learning`;
- `plan-learning`;
- `record-learning-evidence`;
- `review-learning-progress`;
- `adapt-learning-content`.

Keep instructions goal-oriented and delegate state mutation to CLI. Persona discovery uses adaptive rounds, inspects existing docs/repositories first, asks only high-information follow-ups, and writes structured multi-file updates with provenance.

## I4 — Go pack v0

Deliver:

- taxonomy;
- transfer map;
- diagnostic;
- initial production rubric;
- modernity source map;
- code-review rubric;
- concurrency rubric.

Use the existing PyLearn Go curriculum as input evidence/reference, but author the ALP taxonomy independently so it is not coupled to PyLearn phase IDs.

## I5 — Harness adapters

OpenAI/Codex:

- root portable manifest;
- compatibility manifest if required by tested local flow;
- optional validation hooks.

Claude Code:

- manifest;
- thin commands/agents only where they improve UX;
- shared skill directories/reference reuse where supported.

Test that both harnesses read the same cloned learner state.

## I6 — PyLearn exporter/read adapter

Prefer a versioned export contract over reading SQLite internals directly.

Example high-level export:

```json
{
  "schemaVersion": 1,
  "learner": {},
  "progress": [],
  "attempts": [],
  "conceptMastery": [],
  "quizAnswers": [],
  "reflections": []
}
```

Normalize export + repository evidence into ALP evidence records.

## I7 — PyLearn analysis skill

Produce:

- learner-state deltas;
- reinforcement recommendations;
- content gaps;
- stale-content flags;
- evidence-linked adaptation proposals.

Read-only against course content.

## I8 — PyLearn controlled authoring

Apply approved proposals on a branch/PR.

PR description should include:

- evidence;
- learner-state interpretation;
- affected competencies;
- changed content;
- quality gates run;
- risks.

## I9 — Closed-loop validation

Run real Go-learning sessions through PyLearn and ALP.

Evaluate:

- whether state matches expert review;
- whether adaptation reduces redundancy;
- whether agents choose better analogies/content depth;
- false-positive curriculum changes;
- context/token efficiency;
- cross-harness continuity.

## I10 — Rust readiness review

Before adding Rust, review ADR-0002 with actual Go evidence.

Only then implement Rust domain pack.
