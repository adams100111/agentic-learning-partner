# Architecture

## Architectural thesis

ALP is not a course repository and not a traditional LMS. It is a reusable **learning control plane** for AI agents.

The system separates five concerns:

1. **Learner model** — durable facts and preferences about the learner.
2. **Evidence** — append-only observations of demonstrated behavior.
3. **State projections** — current competencies, focus, reinforcement/review needs.
4. **Domain packs** — language/domain-specific knowledge, rubrics, diagnostics, and teaching workflows.
5. **Platform adapters** — translation between ALP and a concrete learning surface such as PyLearn.

## Layers

```
                 Harness adapters
             Codex | Claude Code | ...
                        |
                    Agent Skills
                        |
       +----------------+----------------+
       |                                 |
   Learning core                    Domain packs
       |                            Go | Rust | ...
       +----------------+----------------+
                        |
                 Evidence/state API
                        |
          deterministic local tooling
                        |
                Git-backed storage
                        |
                 Platform adapters
                        |
                  PyLearn first
```

## Portable core

The core owns mechanisms, not Go content:

- learner-profile contract;
- evidence envelope;
- competency/projection rules;
- assessment workflow;
- planning workflow;
- reinforcement/review workflow;
- session protocol;
- state validation/rebuild;
- provenance;
- Git synchronization guidance.

## Domain packs

A domain pack owns domain semantics:

- competency taxonomy;
- prerequisite graph;
- transfer mappings;
- diagnostic activities;
- grading rubrics;
- teaching references;
- production-quality criteria;
- modernity/source policy;
- domain-specific content review skills.

Go ships first. Rust is added later against the same core contract.

## Shared engineering competencies

Some knowledge is portable and should not be duplicated per language:

- architecture;
- HTTP/API design;
- relational databases;
- transactions;
- distributed-systems concepts;
- testing strategy;
- observability;
- security;
- containers;
- cloud/deployment;
- general concurrency concepts.

Language-specific implementation ability remains separate.

Example:

```
shared.concurrency.race_conditions = strong
go.concurrency.race_detector = functional
rust.concurrency.send_sync = unknown
```

## Platform adapters

A platform adapter does NOT own learner truth. It translates platform signals to and from ALP.

For PyLearn:

- DB exercise attempts become structured evidence;
- quiz answers become structured evidence;
- concept-mastery rows are input signals, not authoritative ALP competency;
- learner Markdown evaluations become imported evidence/profile facts;
- project/source code becomes high-value evidence;
- ALP recommendations can become proposed PyLearn content changes.

## Persistence

Git is the durable synchronization and audit mechanism.

Human-authored/static:
- profile seed;
- goals/preferences;
- domain taxonomies;
- rubrics;
- skills;
- ADRs.

Append-only:
- evidence records;
- session records.

Derived/rebuildable:
- current competency snapshot;
- current focus;
- reinforcement/review queue;
- planner projection.

No harness-specific chat memory is authoritative.

## Model vs deterministic tooling

LLMs should:
- interpret work;
- teach;
- assess;
- propose evidence;
- plan;
- author/revise content.

Deterministic code should:
- validate schemas;
- enforce IDs;
- enforce allowed competency paths/levels;
- append evidence;
- rebuild projections;
- detect dangling provenance;
- detect projection drift;
- produce machine-readable status.

Principle: **models reason; deterministic tooling maintains truth.**

## Plugin packaging

The portable package uses the current Agent Plugins format with root `plugin.json` and root `skills/`. OpenAI-specific compatibility may also include `.codex-plugin/plugin.json`.

Claude Code gets a thin `.claude-plugin/plugin.json` adapter plus optional commands/agents/hooks. Claude-specific components MUST delegate to portable skills rather than fork learning policy.

## Repository target shape

```
agentic-learning-partner/
├── plugin.json
├── README.md
├── docs/
│   ├── ARCHITECTURE.md
│   ├── ROADMAP.md
│   ├── STATE_MODEL.md
│   ├── integrations/
│   └── adr/
├── skills/
│   ├── assess-learning/
│   ├── plan-learning/
│   ├── update-learning-state/
│   ├── review-learning/
│   └── domains/
├── domains/
│   └── go/
├── schemas/
├── cmd/alp/
├── internal/
├── learner/
│   ├── profile/
│   └── state/
├── adapters/
│   ├── pylearn/
│   ├── claude-code/
│   └── codex/
├── .claude-plugin/
├── .codex-plugin/
└── .github/workflows/
```
