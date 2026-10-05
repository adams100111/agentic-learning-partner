# Personas

## Purpose

ALP treats persona as structured learner configuration, not as a giant prompt.

A persona captures the durable context that helps an agent teach effectively:

- what the learner already knows;
- relative strength across stacks;
- architecture and production experience;
- preferred teaching style;
- known failure modes;
- useful analogy sources;
- feedback preferences;
- domain-specific teaching overrides.

Persona is distinct from competency state. A learner may strongly prefer project-based learning while still being rusty in a specific language feature.

## Persona layers

### Global learner persona

Cross-domain traits that should not be duplicated per language:

- professional level;
- strongest stacks and relative ordering;
- architecture/system-design experience;
- infrastructure/operations experience;
- learning preferences;
- feedback preferences;
- common disengagement triggers;
- agentic-development workflow;
- preferred project style.

### Domain persona

A domain persona specializes the global persona for a subject such as Go or Rust.

It may define:

- prior exposure;
- domain-specific goals;
- domain-specific teaching emphasis;
- analogy-selection preferences;
- anti-patterns likely to transfer from prior stacks;
- domain-specific project preferences.

It MUST NOT duplicate the full global persona.

### Effective session persona

Agents normally consume a generated, task-specific effective persona rather than every persona/state file.

Conceptually:

```
relevant global persona
+ relevant domain persona
+ current competency subset
+ current goal
+ due reinforcement
+ current project
= effective session context
```

This projection is usually ephemeral.

## Initial learner profile

The initial learner is a senior software architect/backend engineer.

Current relative stack ordering:

1. PHP/Laravel — strongest/deepest;
2. TypeScript/NestJS — strong, with very strong NestJS experience;
3. C#/.NET — strong;
4. Python/FastAPI — strong but newest backend stack, not yet expert.

The learner also has substantial architecture, PostgreSQL, Docker/Linux, self-hosting, cloud/platform, and agentic-development experience.

These facts seed the first learner profile. They MUST remain data, not hard-coded core behavior.

## Analogy selection

A global stack ordering is a useful default but not a universal rule.

The best analogy is selected from:

- learner familiarity;
- semantic closeness;
- risk of importing the wrong abstraction;
- prior analogy exposure.

Examples:

- Laravel may be the default application/backend comparison.
- .NET may be better for `context.Context` because `CancellationToken` is semantically close.
- TypeScript may be better for Go structural interfaces.
- Go may later become a useful analogy source for Rust error/concurrency concepts.

## Persona provenance

Each durable assertion should support provenance such as:

- `learner-stated`;
- `observed`;
- `evaluated`;
- `imported`;
- `agent-inferred`;
- `agent-proposed`.

Explicit learner statements and repeated demonstrated evidence carry more authority than weak inference.

## Conversational updates

The learner should be able to talk naturally to the agent:

- "Laravel is still my strongest stack, but NestJS is stronger than .NET for me."
- "Stop using Python analogies unless they are clearly better."
- "I prefer debugging exercises to quizzes."
- "Show me why you think I am weak in this area."
- "Update my persona based on this project."

The agent should:

1. inspect existing state and evidence;
2. avoid asking already-answered questions;
3. resolve contradictions when needed;
4. propose structured changes;
5. show a concise material diff;
6. persist approved/configured changes with provenance;
7. rebuild the effective context.

## Human-readable persona

Canonical persona remains structured YAML.

ALP should generate human-readable views in Markdown and optionally HTML so learners can inspect:

- what ALP believes;
- why it believes it;
- provenance/confidence;
- domain-specific overrides;
- recent changes.

Generated views are not authoritative.
