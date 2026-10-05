# ADR-0008: YAML canonical state with compact derived agent context

**Status:** Accepted

## Context

ALP must be Git-friendly, human-readable, machine-valid, and efficient for AI agents. A single representation cannot optimize all four.

## Decision

Use YAML as the canonical editable state representation and JSON Schema as its validation contract. Generate JSON, Markdown, or HTML when those formats fit a specific consumer.

LLM agents normally receive compact task-specific projections instead of complete canonical state. Generated HTML is human-facing only and non-authoritative.

## Consequences

- readable Git diffs;
- deterministic validation;
- lower context usage through selective projection;
- representation can vary without changing the source of truth;
- token budgets can be measured independently of storage format.
