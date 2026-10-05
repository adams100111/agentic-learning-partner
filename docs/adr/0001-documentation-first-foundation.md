# ADR-0001: Establish contracts before implementation

**Status:** Accepted

## Context

ALP spans learner state, AI-agent workflows, domain-specific teaching, platform integration, Git synchronization, and multiple harnesses. Starting from plugin files alone would make ownership boundaries emerge accidentally.

## Decision

Begin with roadmap, architecture, state/evidence model, integration contract, and ADRs before implementation code.

Implementation may begin once Phase 0's foundational ownership questions are documented.

## Consequences

- Slower first executable artifact.
- Much lower risk of coupling the state model to Go, PyLearn, Claude Code, or Codex.
- Decisions remain inspectable by future agents.
