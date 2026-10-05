# State Representation and Token Efficiency

## Decision

Use different representations for different jobs:

- YAML for canonical human/agent-editable profile and state.
- JSON Schema for validation.
- JSON for interchange/API export where useful.
- Markdown for narrative sessions, evaluations, and ADRs.
- HTML as a generated human-facing view only.
- Compact task-specific projections for normal LLM consumption.

## Core rule

Agents should not consume canonical state wholesale.

Canonical YAML -> deterministic projection -> task-specific compact context -> LLM.

The main token optimization is selective loading, not serialization punctuation.

## Progressive context loading

Load in this order:

1. tiny routing/index context;
2. relevant global persona subset;
3. relevant domain persona subset;
4. relevant competency state;
5. recent evidence only when needed;
6. full evidence/history only for reassessment or audit.

Normal teaching interactions should not load historical session files.

## Avoid duplication

Do not duplicate:

- global persona inside every domain persona;
- domain taxonomy inside learner state;
- source policy inside every skill;
- historical evidence inside competency snapshots;
- full curricula inside session context.

Stable instructions belong in skills and references. Dynamic context contains only learner/session differences.

## YAML vs JSON

YAML is preferred for canonical state because it is readable, diff-friendly, comment-friendly, and usually less visually noisy.

It is not inherently always more token-efficient than JSON. Long keys, comments, repeated mappings, and deep nesting can make YAML expensive too.

For LLM context, choose a compact generated representation and measure actual token usage.

## HTML

HTML is for humans, not agents. Generate it from canonical state for dashboards/persona views. Do not use it as authoritative state or feed it back to the model unless explicitly necessary.

## Context diagnostics

The CLI should eventually provide an inspection command that shows which sources were included in a generated context, why, and an estimated token cost.
