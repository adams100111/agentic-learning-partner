# Agentic Learning Partner — Roadmap

> Status: foundation draft. This roadmap is product-level; domain curricula live under their domain packs.

## Product goal

Build a portable agentic learning system that helps an experienced learner acquire and retain engineering skills through AI agents without depending on one harness, one chat history, or one LMS.

ALP must be able to:

1. understand who the learner already is;
2. diagnose domain-specific gaps without re-teaching transferable knowledge;
3. generate or revise learning content appropriate to that learner;
4. ingest evidence from exercises, quizzes, source code, project work, conversations, and external platforms;
5. update competency state conservatively and audibly;
6. adapt the next activity and existing content over time;
7. synchronize durable state through Git;
8. work from Claude Code, Codex, and future Agent Skills-compatible harnesses.

## Phase 0 — Documentation and contracts

Deliver before implementation:

- architecture and terminology;
- learner/evidence/state model;
- global/domain persona model;
- persona discovery/refinement workflow;
- canonical/derived representation and token-context policy;
- domain-pack contract;
- platform-integration contract;
- state mutation rules;
- Git ownership and synchronization model;
- initial ADR set;
- PyLearn integration analysis;
- initial Go domain requirements;
- test strategy and acceptance criteria.

### Foundation artifacts now defined

- product/architecture/state/source policies;
- structured global + domain personas;
- adaptive persona wizard;
- canonical YAML / schema / derived-view strategy;
- JSON Schema contracts for profile, persona, evidence, context, taxonomy, and adaptation proposals;
- initial learner seed;
- independent Go competency taxonomy and diagnostic contract;
- PyLearn export/evidence mapping contract;
- implementation specs 001-005.

**Exit:** implementation can begin once the foundation PR is reviewed/merged and any material contract objections are resolved.

## Phase 1 — Portable core

Implement the harness-neutral core:

- portable root `plugin.json`;
- core Agent Skills;
- learner profile/persona schemas;
- persona wizard/refinement skill;
- context projection builder;
- evidence schema;
- competency schema;
- projection/state schema;
- deterministic state validation/rebuild CLI;
- append-only evidence store;
- current-state projection;
- human-readable persona/state views;
- context inspection/token diagnostics;
- adaptive planner contract;
- review/reinforcement queue.

No MCP server is required in this phase.

**Exit:** a local cloned repo can ingest evidence, rebuild state deterministically, and tell an agent what to do next.

## Phase 2 — Go domain pack

Go is the first domain pack.

Deliver:

- Go competency taxonomy;
- transfer mapping from Laravel/PHP, TypeScript/NestJS, .NET, Python/FastAPI, architecture, DevOps;
- diagnostic rubric;
- idiomatic-Go review rubric;
- production readiness rubric;
- concurrency-specific evidence rules;
- backend/database/tooling references;
- modernity verification protocol;
- adaptive roadmap generation.

Target current stable Go at lesson-authoring time; never hard-code stale ecosystem advice without a verification date.

**Exit:** ALP can diagnose Adams's Go state and produce a justified learning plan without a fixed beginner course.

## Phase 3 — Harness adapters

### Codex / OpenAI

- portable Agent Plugins layout;
- optional `.codex-plugin/plugin.json` compatibility manifest;
- Codex-specific hooks only where deterministic validation benefits;
- repository-local installation documentation.

### Claude Code

- `.claude-plugin/plugin.json`;
- skills reuse;
- optional commands/subagents/hooks as thin adapters;
- no Claude-specific learning truth.

**Exit:** the same learner state and evidence are usable from both harnesses.

## Phase 4 — PyLearn adapter: read-only intelligence

PyLearn is the first external learning-platform integration.

Build an adapter that can normalize:

- learner profile docs;
- timeline/evaluation docs;
- reinforcement backlog;
- lesson progress;
- exercise attempts;
- concept mastery;
- quizzes;
- reflections/bookmarks where useful;
- source-code/project evidence.

Initially the adapter MUST NOT rewrite PyLearn content automatically.

**Exit:** ALP can analyze PyLearn and recommend learner-state and content changes with provenance.

## Phase 5 — PyLearn adaptive authoring

Add controlled write workflows:

- generate new lesson/reel proposals;
- revise existing lesson sections;
- add reinforcement based on actual weaknesses;
- identify stale or redundant content;
- re-verify modernity-sensitive material;
- open changes through Git branches/PRs;
- require deterministic/platform quality gates before merge.

Content changes should be proposals with evidence links, not invisible in-place mutations.

**Exit:** PyLearn becomes continuously adaptive rather than one-shot AI-generated.

## Phase 6 — Closed-loop learning

Establish the loop:

```
platform activity
  -> evidence normalization
  -> learner-state update
  -> gap/reinforcement analysis
  -> curriculum/content proposal
  -> human/agent review
  -> platform update
  -> new learner activity
```

Add decay/review scheduling, contradictory evidence handling, confidence, and retrospective evaluation.

## Phase 7 — Rust domain pack

Add Rust without duplicating the learning engine:

- ownership/borrowing/lifetimes;
- traits/type-system reasoning;
- Send/Sync;
- async/Tokio;
- Axum/Tower/SQLx;
- systems/performance;
- Rust production rubric.

Reuse shared engineering competencies and learning state while keeping Rust-specific competence separate.

## Phase 8 — Generalized platform/plugin ecosystem

Only after PyLearn proves the contract:

- generic filesystem platform adapter;
- optional MCP for remote/live systems where local tools are insufficient;
- additional learning platforms;
- additional engineering domain packs;
- publishable plugin packaging if desired.

## Non-goals for early versions

- replacing PyLearn's UI;
- building a traditional LMS;
- central hosted learner database;
- automatic production content mutation without review;
- an MCP server merely to read/write local files;
- universal support for every language before Go proves the model.
