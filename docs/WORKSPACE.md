# Learner Workspace

## Decision

The reusable ALP engine/plugin and learner-specific state have separate repositories/lifecycles.

### Engine repository

`agentic-learning-partner`

Owns:

- plugin manifests and skills;
- schemas;
- domain packs;
- rubrics;
- source/freshness policy;
- CLI/runtime implementation;
- platform adapters;
- reusable documentation.

### Learner workspace repository

`agentic-learning-state`

Owns:

- profile;
- global/domain personas;
- evidence;
- assessments;
- derived projections;
- sessions;
- review/reinforcement state;
- workspace configuration.

### Learning platform repository

For the first use case: `pylearn`.

Owns:

- platform UI/application;
- platform activity records;
- course content;
- platform-specific tests and quality gates.

## Why separate

The engine may be installed, published, versioned, or upgraded independently.

Learner state is private, personal, long-lived, and synchronized independently.

A learning platform may evolve or be replaced without moving learner identity into it.

## Workspace discovery

The CLI/plugin should resolve a workspace explicitly, in this priority order:

1. command/session explicit workspace;
2. project-local ALP config;
3. environment/configured default workspace;
4. fail with a clear setup instruction.

Do not silently create learner state inside the plugin repository.

## Compatibility

A workspace declares which ALP state schema versions it uses.

ALP must validate workspace compatibility before mutation and provide migrations for breaking persisted-state changes.

## Git behavior

Append-only canonical records:

- evidence;
- assessments;
- session records where retained.

Human-maintained canonical records:

- profile;
- personas;
- goals/preferences.

Derived:

- competency projections;
- current focus;
- review queue;
- generated views.

Derived files are regenerated after merge/reconciliation and should not be resolved by hand.
