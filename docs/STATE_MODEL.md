# Learner State and Evidence Model

## Why evidence is first-class

A learner-state system becomes unreliable when an agent can simply write "channels: strong". ALP therefore persists observations before conclusions.

## Three categories

### Profile

Slow-changing context about the learner:

- prior stacks and relative strength;
- architecture experience;
- goals;
- preferred comparison order;
- teaching constraints;
- environments/platforms used;
- learning preferences.

Profile assertions should include origin where useful: learner-stated, platform-derived, or evaluated.

### Evidence

Append-only records of demonstrated behavior.

Examples:

- diagnostic answer;
- code-reading result;
- code-writing exercise;
- failed/passed test;
- quiz answer;
- project commit;
- code review;
- production incident analysis;
- learner self-report;
- platform progress event.

Each evidence record must contain:

- unique ID;
- timestamp;
- domain;
- competency path(s);
- evidence type;
- source;
- result/observation;
- strength;
- provenance;
- optional contradiction/replacement link.

Evidence is never silently deleted.

### Projections

Derived current views:

- competency level;
- confidence;
- last verified;
- current phase/focus;
- gaps;
- reinforcement queue;
- review due dates;
- next recommended activity.

Projections are rebuildable from evidence + rules.

## Competency levels

Use ordinal levels, not fake percentages:

1. `unknown`
2. `rusty`
3. `functional`
4. `strong`
5. `production-ready`

Promotion is conservative. A normal single exercise should not jump several levels. Production-ready requires real project evidence and the domain rubric.

Downgrade is allowed when later evidence contradicts the current projection.

## Evidence strength

Default ordering:

```
self-report
< recognition/multiple-choice
< explanation/code-reading
< constrained code writing
< debugging
< independent project implementation
< production-quality project/incident evidence
```

This is not an absolute numeric formula; domain rubrics may override it.

## Multiple dimensions

Never collapse everything into one "Go level".

Example domain dimensions:

- syntax recall;
- language mental model;
- idiomatic usage;
- engineering/design judgment;
- runtime/concurrency;
- ecosystem/tooling;
- production execution;
- verification/testing.

This matters for Adams: weak Go syntax recall must not erase strong architecture reasoning.

## Canonical state layout

```
learner/
├── profile/
│   ├── learner.yaml
│   ├── experience.yaml
│   ├── goals.yaml
│   └── preferences.yaml
└── state/
    ├── current.yaml
    ├── competencies.yaml
    ├── review-queue.yaml
    ├── domains/
    │   └── go/
    │       └── roadmap.yaml
    ├── evidence/
    │   └── <uuid>.yaml
    └── sessions/
        └── <timestamp>.md
```

## State mutation protocol

Only the state-update workflow may mutate canonical projections.

Other skills:

1. read state;
2. perform learning/review;
3. produce proposed evidence;
4. invoke validation/update tooling.

The updater:

1. validates schema;
2. validates competency path;
3. persists append-only evidence;
4. recomputes affected projections;
5. updates reinforcement/review state;
6. validates invariants;
7. emits a change summary.

## Platform telemetry

Platform telemetry is evidence, not truth.

For example PyLearn's `concept_mastery` counts are informative, but ALP should not mechanically equate "three passes" with "strong". A generated exercise may be too easy, repeated answers may reflect recognition rather than transfer, and source-code evidence may contradict the rollup.

## Content-adaptation provenance

When learner evidence causes a content change proposal, record:

- evidence IDs;
- affected competency;
- affected content IDs/files;
- reason;
- proposed adaptation type;
- confidence;
- whether human review is required.

This gives us an auditable chain:

```
learner behavior -> evidence -> competency/gap -> content proposal -> merged revision
```
