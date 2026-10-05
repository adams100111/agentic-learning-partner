# Architecture

## Architectural thesis

ALP is not a course repository and not a traditional LMS. It is a reusable **learning control plane** for AI agents.

The system separates five concerns:

1. **Learner profile/personas** — durable facts, preferences, and domain-specific learning configuration.
2. **Evidence** — append-only observations of demonstrated behavior.
3. **Assessments** — versioned semantic judgments over evidence under explicit rubrics.
4. **State projections** — deterministic current competencies, focus, reinforcement/review needs derived from accepted assessments.
5. **Context projection** — compact task-specific learner context for agents.
6. **Domain packs** — language/domain-specific knowledge, rubrics, diagnostics, and teaching workflows.
7. **Platform adapters** — translation between ALP and a concrete learning surface such as PyLearn.

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
                 Evidence/assessment/state API
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

- learner-profile and persona contracts;
- persona discovery/refinement workflow;
- task-specific context projection;
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

## Engine vs learner workspace

The reusable ALP engine and learner-specific state have separate lifecycles.

```
agentic-learning-partner
  schemas / skills / domain packs / CLI / adapters / docs

agentic-learning-state (private learner workspace)
  profile / personas / evidence / assessments / projections / sessions

pylearn
  learning platform / content / learner activity
```

ALP operates against a configured learner workspace. Synthetic fixtures/examples may live in the engine repository; real learner state must not.

## Engine / workspace separation

The reusable engine/plugin does not own personal learner state.

- `agentic-learning-partner` contains reusable engine code, schemas, skills, domain packs, rubrics, and adapters.
- a learner workspace such as `agentic-learning-state` contains private profile/persona/evidence/assessment/state.
- learning platforms such as PyLearn remain separate systems.

See `WORKSPACE.md` and ADR-0009.

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
- assessment records;
- session records.

Derived/rebuildable:
- current competency snapshot;
- current focus;
- reinforcement/review queue;
- planner projection;
- effective session persona/context;
- human-readable Markdown/HTML views.

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
- rebuild projections from canonical accepted assessments;
- detect dangling provenance;
- detect projection drift;
- produce machine-readable status.

Principle: **models reason; deterministic tooling maintains truth.**

Semantic competency judgment is explicitly persisted as an **assessment**. Projection rebuild does not ask an LLM to reinterpret raw evidence. It deterministically folds accepted, versioned assessments.

## Persona and context boundary

Canonical learner/profile/persona data is stored as structured YAML with schema validation. Agents normally receive a compact generated context, not the complete canonical learner state.

The context builder selects only the information required for the current task. Historical evidence is loaded on demand for reassessment, contradiction resolution, or audit.

Generated Markdown/HTML views exist for humans and are never authoritative.

## Token-efficiency invariant

ALP treats context cost as an architectural constraint. Optimize primarily by progressive disclosure and eliminating duplicated knowledge, not merely by choosing one serialization format over another.

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
│   ├── PERSONAS.md
│   ├── PERSONA_WIZARD.md
│   ├── REPRESENTATION.md
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
├── examples/
│   └── synthetic-workspace/
├── adapters/
│   ├── pylearn/
│   ├── claude-code/
│   └── codex/
├── .claude-plugin/
├── .codex-plugin/
└── .github/workflows/
```
