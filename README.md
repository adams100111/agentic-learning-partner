# Agentic Learning Partner

Agentic Learning Partner (ALP) is a portable, stateful learning system for AI coding agents.

It combines:

- a durable learner profile,
- evidence-based competency state,
- adaptive planning,
- domain packs (Go first; Rust later),
- project and platform telemetry,
- reusable Agent Skills,
- deterministic state tooling,
- and harness adapters for Codex, Claude Code, and future agent runtimes.

The repository is intentionally documentation-first. Architecture, state semantics, integration contracts, and ADRs are defined before implementation.

## First use case

The first integration target is [PyLearn](https://github.com/adams100111/pylearn), an existing personalized engineering-learning platform. PyLearn already contains useful learner-profile, reinforcement, project-evidence, exercise-attempt, quiz, and concept-mastery signals. ALP will generalize those ideas into a reusable learning engine rather than replace PyLearn.

## Status

Foundation / architecture phase.

See `docs/ROADMAP.md` once the documentation branch is merged.
