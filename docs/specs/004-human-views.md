# Spec 004 — Human Persona and State Views

**Status:** Draft-ready for implementation planning

## Objective

Make ALP's beliefs transparent to the learner without making generated presentation files part of canonical state.

## Required views

### Persona

Show:

- durable learner identity;
- stack strengths/order;
- teaching preferences;
- domain overrides;
- provenance/confidence for material assertions.

### Competency

Show:

- competency level;
- last verified;
- supporting/contradictory evidence summary;
- current gaps;
- production-ready requirements still missing.

### Current learning

Show:

- domain;
- current focus;
- current project/milestone;
- due review/reinforcement;
- recommended next activity and why.

### Evidence history

Allow drill-down from a competency to evidence records.

## Output formats

Initial:

- terminal/text;
- Markdown.

Later:

- generated static HTML dashboard.

HTML is derived and non-authoritative.

## Commands

Conceptual UX:

```
alp persona show
alp persona show --domain go
alp status
alp competency show go.runtime.context
alp evidence show ev_...
alp view build --format html
```

## Acceptance tests

- a learner can answer "why does ALP think this?" without reading raw YAML;
- changing canonical state changes regenerated views;
- editing generated HTML never changes canonical state;
- views do not expose irrelevant secrets/private platform data;
- domain persona overrides are visually distinguishable from global persona.
