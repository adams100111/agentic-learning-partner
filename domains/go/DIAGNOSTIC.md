# Go Diagnostic Contract

## Purpose

Determine current Go ability without confusing forgotten syntax with general engineering maturity.

The initial learner has prior Go exposure but reports retaining little. Therefore the diagnostic begins at recall level but escalates quickly when reasoning returns.

## Dimensions

Assess separately:

1. language recall;
2. Go mental model;
3. idiomatic Go judgment;
4. runtime/concurrency reasoning;
5. backend/database execution;
6. testing/verification;
7. production/runtime operations;
8. architecture/abstraction judgment.

## Format

Default initial diagnostic: 10-15 adaptive activities, not 10-15 lectures.

Use a mix of:

- predict output/behavior;
- read code;
- find bug;
- repair code;
- tiny implementation;
- concurrency trace;
- API/package design choice;
- production failure scenario;
- AI-generated Go code review.

Perfect syntax is not required when the target is conceptual reasoning. Record syntax misses separately.

## Suggested first diagnostic sequence

1. slice aliasing / append behavior;
2. pointer vs value receiver;
3. implicit interface satisfaction and interface placement;
4. error wrapping + errors.Is/As;
5. defer/resource lifetime;
6. net/http handler/middleware reading;
7. context cancellation ownership;
8. transaction-boundary reasoning;
9. goroutine lifetime/capture;
10. channel blocking/close ownership;
11. select/timeout reasoning;
12. mutex vs channel choice;
13. race/leak diagnosis;
14. package/layer review of over-abstracted agent-generated Go;
15. graceful-shutdown/production design.

The sequence is adaptive: stop spending questions on a competency once evidence is strong enough for the current planning decision.

## Evidence rules

Each activity emits one or more evidence records.

Do not promote solely because of self-report.

Examples:

- remembering `errors.Is` syntax after a hint -> recall evidence;
- independently diagnosing a wrapped-error boundary -> stronger mental-model/idiomatic evidence;
- implementing a tested production error boundary in a project -> strong/project evidence.

## Production-ready bar

No diagnostic-only sequence can establish production-ready.

Production-ready requires repeated project evidence appropriate to the competency, including verification and explanation of design/operational choices.

## Initial learner-specific calibration

Known prior exposure may inform question selection:

- net/http;
- Chi;
- PostgreSQL;
- pgx;
- sqlx-style tooling;
- auth tokens;
- handlers/middleware;
- structs/interfaces;
- basic goroutines.

It does NOT pre-promote any competency.

## Agent behavior

- do not ask "what is middleware/DI/transaction?" unless evidence suggests actual conceptual confusion;
- use existing engineering knowledge as scaffolding;
- challenge framework-translated over-abstraction;
- distinguish architecture correctness from Go idiomaticity;
- ask for trade-offs, not trivia;
- prefer one adaptive question at a time during live tutoring, while batch diagnostics remain available for speed.
