# ADR-0002: One learning core with domain packs

**Status:** Accepted

## Context

Go is the first learning target, with Rust expected later. Learning mechanics overlap heavily, while language/runtime competencies do not.

## Decision

Build one portable learning core and separate domain packs.

The core owns learner/evidence/state/planning mechanisms. A domain pack owns competency taxonomy, diagnostics, rubrics, teaching references, transfer mappings, and production criteria.

Shared engineering competencies are modeled separately from language-specific competencies.

## Consequences

- Rust can be added without copying the plugin.
- Go-specific assumptions cannot leak into the core.
- Domain packs require a stable contract and validation.
- Some workflows remain specialized rather than over-generalized.
