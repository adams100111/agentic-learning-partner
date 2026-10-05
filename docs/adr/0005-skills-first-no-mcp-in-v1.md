# ADR-0005: Skills-first plugin; no custom MCP server in v1

**Status:** Accepted

## Context

Current OpenAI and Claude plugin systems support reusable skills, scripts, hooks, and local repository access. ALP's initial work is predominantly local reasoning + filesystem/Git state maintenance.

## Decision

Ship v1 as a skills-first portable plugin with deterministic local CLI/tooling and thin harness adapters.

Do not build a custom MCP server until a use case requires authenticated remote/live actions that local tools or existing connectors cannot supply.

## Consequences

- Smaller security and operational surface.
- Easier local adoption in Claude Code/Codex.
- Remote integrations may later justify MCP without redesigning the core.
