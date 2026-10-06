# ADR-0012: Portable skills must not depend on hidden cross-skill invocation

**Status:** Accepted

## Context

Skill composition behavior varies across agent harnesses. A wrapper skill that relies on the model/runtime to discover and invoke other skills can fail when dependencies are missing or invocation semantics differ.

ALP targets multiple harnesses.

## Decision

Core invariants live in deterministic CLI/contracts and shared references/scripts. Shared references are loaded progressively, only when a skill needs them.

Each user-facing ALP skill must be independently executable with explicit prerequisites. Orchestration skills may describe higher-level workflows, but correctness must not rely on an implicit runtime chain such as "invoke skill X, then skill Y."

Harness-specific adapters may improve UX but cannot own unique learning semantics.

## Consequences

- somewhat more explicit skill definitions;
- some workflow guidance is repeated minimally across skills;
- better Claude/Codex portability;
- shared references/scripts become important reuse mechanisms;
- skill dependency validation can be static.
