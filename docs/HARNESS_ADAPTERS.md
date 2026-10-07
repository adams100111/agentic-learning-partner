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

## Platform authoring target skills

A platform's authoring target skill (ADR-0056) lives in the platform's repository and is installed with it, not with ALP. `orchestrate-platform-authoring` resolves its name from `alp platform inspect` (`authoringTarget.skill`). When the harness cannot invoke it, the orchestration skill hands the learner the plan and the expected hand-back files and resumes at `alp platform gates record`, so correctness never depends on cross-skill invocation (ADR-0012).

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
