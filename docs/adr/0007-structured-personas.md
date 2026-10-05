# ADR-0007: Structured global and domain personas

**Status:** Accepted

## Context

A giant persona prompt is expensive, hard to keep synchronized, and likely to drift across languages. Fully separate personas per language would duplicate the learner's identity. A single generic persona cannot capture domain-specific learning behavior.

## Decision

Represent persona as structured, versioned learner configuration with:

- one global learner persona;
- optional domain-specific personas;
- provenance for durable assertions;
- generated task-specific effective personas.

Domain personas specialize but never duplicate global learner identity.

Persona and competency state remain separate concepts.

## Consequences

- one synchronized learner identity across Go, Rust, and future domains;
- language-specific teaching can adapt independently;
- persona updates can be diffed and audited in Git;
- context usage stays bounded through derived projections;
- the CLI/plugin needs persona inspection, editing, and context-building workflows.
