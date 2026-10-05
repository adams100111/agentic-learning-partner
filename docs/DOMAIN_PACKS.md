# Domain Pack Contract

## Purpose

A domain pack adds subject-specific intelligence to the shared learning core.

The first pack is Go. Rust follows only after the Go/core boundary proves stable.

## A domain pack MUST define

### Identity

- stable domain ID;
- display name;
- version of the pack schema.

### Competency taxonomy

Hierarchical competency paths with descriptions and evidence expectations.

Example:

```
go.language.errors
go.language.interfaces
go.runtime.context
go.concurrency.channels
go.backend.http
go.database.pgx
go.production.observability
```

### Transfer mapping

What shared/prior skills may accelerate teaching without being treated as domain proof.

For the initial Go learner, reference priority should reflect actual strength:

1. PHP/Laravel for deepest application/backend intuition where the analogy fits;
2. TypeScript/NestJS;
3. C#/.NET, especially concurrency/runtime/static-language comparisons;
4. Python/FastAPI selectively because it is newer;
5. shared software-architecture knowledge directly.

A specific lesson may choose a lower-priority stack when its semantic mapping is materially better.

### Diagnostic rubric

Activities and scoring guidance capable of distinguishing:

- forgotten syntax;
- missing language mental model;
- unidiomatic transfer from another ecosystem;
- real architecture/engineering weakness.

### Teaching policy

- preferred project/application contexts;
- concepts that should not be re-taught;
- anti-patterns imported from prior stacks;
- active-recall expectations;
- production-quality bar.

### Modernity policy

- authoritative sources;
- last-verified metadata;
- which ecosystem choices must be re-checked before authoring/revising;
- legacy-only callout policy.

### Production rubric

Criteria for `production-ready` competencies.

## A domain pack MUST NOT own

- learner identity;
- global evidence persistence;
- Git synchronization;
- generic state update mechanics;
- platform database schemas;
- harness-specific memory.

## Shared competency links

Domain packs may reference shared competencies as prerequisites/scaffolding but must maintain separate domain evidence.

Example:

```
shared.architecture.dependency_direction = strong
go.idioms.interface_placement = unknown
```

The first can shorten explanation; it cannot promote the second.
