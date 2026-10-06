# Harness Adapters

## Portable core

The portable core consists of:

- root `plugin.json`;
- root `skills/`;
- `alp` CLI;
- learner workspace contracts.

## OpenAI / Codex

OpenAI/Codex consumes the portable root package directly. There is no separate state semantics layer.

The root portable manifest is the canonical plugin identity.

## Claude Code

Claude Code uses `.claude-plugin/plugin.json` for plugin identity and loads the same root `skills/` directory through its standard plugin layout.

Plugin skills are namespaced by Claude Code automatically.

## No shadow state

Harness adapters may provide discovery/ergonomics only.

They must not create:

- alternate competency levels;
- alternate persona copies;
- hidden assessment history;
- harness-only review queues.

## CLI dependency

All durable state actions route through the installed `alp` CLI or canonical schema/file contracts.

If `alp` is unavailable, skills should explain how to install it rather than reproducing state logic inside the prompt.
