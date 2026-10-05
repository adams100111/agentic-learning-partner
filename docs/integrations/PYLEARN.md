# PyLearn Integration — First Use Case

Repository: `adams100111/pylearn`

## Why PyLearn is an excellent first use case

PyLearn is already much more than static course content. It contains several pieces of the learning system ALP intends to generalize:

### Learner profile and evaluation

- `docs/learner/PROFILE.md` is a living learner snapshot.
- `docs/learner/timeline/` preserves evaluation history.
- `docs/learner/REINFORCEMENT.md` maps observed weaknesses back into existing lessons.

This is effectively a manually maintained early version of ALP's profile + evidence + projection model.

### Runtime learning telemetry

The application schema already tracks:

- lesson progress;
- exercise attempts and test status;
- failure messages/stdout;
- concept mastery;
- quiz answers;
- reflections;
- bookmarks.

This gives ALP behavioral evidence that ordinary repository-only tutors lack.

### Project evidence

PyLearn also evaluates real learner-built code such as the Python anchor project. This is higher-value evidence than completion status.

### AI-authored curriculum

PyLearn is already designed for AI-assisted course creation, with:

- a constitution;
- curriculum maps;
- source policies;
- modernity guardrails;
- content audits;
- SpecKit workflows;
- Claude/Codex project guides;
- Go and Rust course planning.

Therefore ALP does not need to prove that agents can generate course text. It needs to improve **selection, personalization, evidence grounding, and continuous revision**.

## Current limitation ALP should solve

Today these systems are partly disconnected:

```
learner DB telemetry
learner evaluation Markdown
source-code evidence
curriculum docs
content audits
agent memory
```

A human/agent manually interprets them and updates course materials.

ALP should provide the missing control loop.

## Target closed loop

```
PyLearn learner activity
        |
        v
PyLearn adapter / evidence normalization
        |
        v
ALP evidence store
        |
        v
competency + reinforcement projections
        |
        v
adaptive planner/content reviewer
        |
        +----> next activity recommendation
        |
        +----> lesson reinforcement proposal
        |
        +----> lesson rewrite proposal
        |
        +----> new lesson proposal (high bar)
        |
        v
branch / PR in PyLearn
        |
 quality gates + review
        |
        v
updated PyLearn content
```

## Read-side integration

The first adapter should read and normalize:

### Repository-authored signals

- `docs/CONSTITUTION.md`
- `docs/learner/PROFILE.md`
- `docs/learner/REINFORCEMENT.md`
- `docs/learner/timeline/*`
- course constitutions/curriculum overviews;
- content front matter;
- relevant project repositories/paths;
- content audit documents where useful.

### Application telemetry

From PyLearn's database:

- progress;
- attempts;
- concept mastery;
- quiz answers;
- reflections;
- optionally bookmarks.

The initial implementation can use an explicit export command/file rather than direct database coupling. This keeps ALP portable and makes the adapter testable.

## Write-side integration

ALP should NOT directly mutate course content during the first integration.

Preferred write flow:

1. produce a structured adaptation proposal;
2. create a branch in PyLearn;
3. apply content/doc changes;
4. run PyLearn's existing gates;
5. show evidence/provenance in the PR;
6. merge only after review.

## Adaptation types

Prefer the smallest intervention justified by evidence:

1. no content change; schedule review;
2. swap/reframe analogy;
3. add a trip-wire/callout;
4. add a targeted micro-exercise;
5. add a worked example;
6. revise an existing section;
7. split/restructure a lesson;
8. add a new lesson only when the competency map shows a real uncovered requirement.

This aligns with PyLearn's current reinforcement philosophy and avoids curriculum churn.

## Content generation value

Claude Code or Codex should improve substantially with ALP because the agent will no longer begin authoring from only the static constitution. It can receive a compact, evidence-grounded context such as:

- learner profile;
- relevant competencies only;
- recent contradictory evidence;
- due reinforcement;
- analogies already seen;
- target lesson's competency contract;
- verified source policy;
- current platform constraints.

That context is both **smaller and more accurate** than loading every historical course document.

## Ongoing analysis

A suggested cadence:

- after meaningful exercise/project evidence: update learner state;
- at session close: recompute reinforcement/review queue;
- before authoring a lesson: load current learner projection;
- periodically: audit stale technology claims;
- at phase/milestone boundaries: deeper competency reassessment and curriculum-gap analysis.

ALP must not rewrite content merely because time passed. Changes need evidence or a verified external modernity change.

## Important architectural consequence

PyLearn's current Go and Rust documents are valuable **domain-content seeds**, but they should not become ALP core state.

ALP should extract/generalize:
- learner mechanics;
- evidence semantics;
- authoring/review workflows.

It should reference/import domain-specific material into the Go pack where appropriate, keeping PyLearn free to remain a presentation/application surface.
