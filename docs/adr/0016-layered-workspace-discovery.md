# ADR-0016: Layered learner-workspace discovery

**Status:** Accepted

## Context

ALP may run from PyLearn, arbitrary project repos, Claude Code, Codex, or a terminal.

## Decision

Resolve workspace via explicit option, project-local config, environment variable, then user-level config. Fail clearly if unresolved.

## Consequences

- predictable UX across harnesses;
- projects can opt into a learner workspace without copying state;
- no hidden engine-repo coupling.
