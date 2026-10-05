# Skill Composition

## Goal

Keep ALP portable across agent harnesses even when their skill-discovery/composition behavior differs.

## Rule

Core correctness must not depend on one skill successfully invoking another skill by name.

Skills may orchestrate conceptually, but shared invariants live in:

- schemas;
- deterministic CLI operations;
- shared reference files;
- explicit file/path contracts.

## Self-contained entry skills

A user-facing skill should contain enough routing/workflow instruction to complete its goal using the CLI and references available to it.

Examples:

- persona discovery;
- assessment;
- learning planning;
- evidence recording;
- progress review;
- content adaptation.

## Shared references

Do not duplicate large policies across skills. Reference stable common files and load them progressively.

## Harness adapters

Claude/Codex adapters may add ergonomic commands, agents, or hooks, but cannot create alternate state semantics.

## Rationale

Cross-skill invocation is an orchestration convenience, not a correctness boundary.
