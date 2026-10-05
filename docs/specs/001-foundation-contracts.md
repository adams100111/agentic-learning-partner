# Spec 001 — Foundation Contracts

**Status:** Draft-ready for implementation planning

## Objective

Make ALP's learner/profile/persona/evidence/context contracts executable and deterministic before adding teaching behavior.

## Scope

Implement validation and projection primitives for:

- learner profile;
- global/domain persona;
- evidence records;
- competency taxonomy;
- task-specific context bundle;
- content-adaptation proposals.

## Required behavior

### Validation

The CLI must validate YAML/JSON instances against the versioned JSON Schemas.

Failures must identify:

- file;
- schema;
- field/path;
- reason.

### Persona composition

Given:

- learner profile;
- global persona;
- optional domain persona;
- current state;
- current task;

build a compact effective context without mutating canonical files.

### Context selection

The builder must support progressive disclosure.

For ordinary tasks it must not load:

- entire evidence history;
- unrelated domain state;
- full curricula;
- generated HTML views.

### Context inspection

Provide an inspect mode that reports:

- included source paths;
- reason each source was included;
- omitted high-level source groups;
- estimated token cost when available.

### Evidence append

Evidence records are append-only.

The system must reject:

- duplicate evidence IDs;
- malformed competency IDs;
- unsupported evidence types;
- invalid timestamps;
- unknown domain competencies where validation is configured.

### Projection rebuild

Derived state must be rebuildable from canonical profile/persona/evidence/domain rules.

No harness conversation memory may be required.

## Acceptance tests

1. valid initial learner YAML passes;
2. malformed persona fails with a useful path error;
3. adding one evidence record changes only relevant projections;
4. deleting a derived snapshot and rebuilding produces the same result;
5. context for a Go concurrency task excludes unrelated Python/history material;
6. Claude/Codex adapters can consume the same generated context bundle;
7. Markdown/HTML views are reproducible from canonical state and are not read as source truth.

## Out of scope

- final promotion algorithms;
- PyLearn database ingestion;
- Go teaching content;
- MCP server;
- hosted service.
