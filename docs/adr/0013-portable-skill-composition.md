# ADR-0013: Do not depend on cross-skill invocation for correctness

**Status:** Accepted

## Context

Agent harnesses differ in skill invocation/composition behavior. Wrapper skills that require other skills to be invoked by name can fail or behave inconsistently.

## Decision

Make user-facing skills independently executable using deterministic CLI operations and progressively loaded shared references. Cross-skill invocation may improve orchestration but is not a correctness dependency.

## Consequences

- stronger Claude/Codex portability;
- some workflow guidance is repeated minimally;
- shared policies must live in stable references/CLI contracts rather than hidden prompt chains.
