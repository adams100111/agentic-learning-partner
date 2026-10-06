# Implementation Workflow

## Decision

ALP will use a spec-driven, ticket-graph implementation workflow inspired by Matt Pocock's engineering skills.

Recommended main flow:

```
grill-with-docs
  -> to-spec
  -> to-tickets
  -> implement-spec
  -> code-review

retro is a separate human-in-the-loop follow-up, not an automatic workflow stage. (human-in-the-loop, optional)
```

The architecture grill is now complete, so ALP does not need `wayfinder` for the current implementation phase.

## Why this fits ALP

ALP is a multi-slice build with clear blocking relationships:

- schema/runtime foundation;
- workspace/state primitives;
- context generation;
- persona workflow;
- domain-pack loading;
- Go diagnostics;
- platform adapters;
- harness adapters.

These are well suited to a task graph rather than one giant sequential implementation prompt.

## Use of implement-spec

`implement-spec` is appropriate when:

- a settled spec exists;
- tickets are independently scoped vertical slices;
- blocking edges are explicit;
- implementers can use isolated worktrees/branches;
- one integration branch owns convergence.

The implementation orchestrator should communicate through pointers to:

- the spec;
- ticket;
- ADRs;
- glossary;
- schema contracts;
- prior commits.

Do not duplicate large design context in every subagent prompt.

## TDD

Use TDD at pre-agreed public seams.

Initial high-value seams are expected to include:

- workspace discovery/configuration;
- schema validation;
- evidence/assessment append;
- deterministic projection rebuild;
- context bundle generation;
- domain-pack loading/compatibility;
- migration dry-run/apply.

Tests should verify behavior through those interfaces rather than internal helpers.

## Codebase design

Before implementing each major module, prefer a deep-module design:

- small public interface;
- substantial behavior hidden behind it;
- explicit seams;
- dependencies accepted rather than internally constructed;
- avoid an interface until something actually varies at that seam.

This is especially important in Go, where framework-style abstraction can easily overcomplicate the codebase.

## Code review

Review the completed integration branch along two separate axes:

1. repository standards / code quality;
2. fidelity to the implementation spec.

Do not let a clean implementation hide missing requirements, or a complete feature hide poor architecture.

## Retro

Retro is human-in-the-loop and is not an automatic implementation-loop step.

Run a retrospective only when the user explicitly invokes it or approves a recommendation to do so, typically after a substantial tranche, repeated review failures, or an agent-environment problem. Focus on:

- missing navigation pointers;
- missing deterministic checks;
- repeated mistakes that should become CI/lint rules;
- token-expensive exploration;
- steering docs that are too large or ineffective.

## When not to use implement-spec

Use per-ticket implementation instead when:

- tickets touch the same unstable seam and would create merge contention;
- implementation feedback is likely to invalidate later ticket assumptions;
- the task graph is mostly linear;
- concurrency would cost more coordination than it saves.

Parallelism is an optimization, not a goal.

## Prototype escape hatch

Use a throwaway prototype only when a remaining design question requires runnable evidence.

Do not prototype already-settled architecture merely because implementation has started.

## Agent-facing writing

Skills, AGENTS/CLAUDE steering, and referenced docs should follow progressive disclosure:

- tiny routing/steering files;
- pointers to deeper docs;
- no duplicated architecture prose;
- procedures written as deterministic steps where possible.

## Current next step

For the production-v0 Store/sync tranche, the architecture grill is complete. The next step is to synthesize the settled Store/provider/sync/onboarding/recovery architecture into one implementation spec, then break it into tracer-bullet tickets with blocking edges before starting code.
