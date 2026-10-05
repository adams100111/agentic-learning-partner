# Spec 002 — Persona Wizard

**Status:** Draft-ready for implementation planning

## Objective

Implement a conversational learner-discovery workflow that produces high-quality structured persona/profile updates with minimal questioning and explicit provenance.

## Entry modes

- initial;
- refine;
- domain-onboarding;
- review;
- docs-assisted;
- contradiction-resolution.

## Workflow

1. load compact existing learner context;
2. inspect available evidence/repositories/documents requested or configured for the session;
3. build an uncertainty list;
4. rank unresolved items by expected learning-content impact;
5. ask adaptive question round;
6. optionally run compact domain probes;
7. synthesize;
8. produce a proposed multi-file change set;
9. show material changes/rationale;
10. persist as one logical transaction;
11. emit any competency evidence separately;
12. rebuild effective persona.

## Question-selection rule

Never ask a question solely because it appears in a questionnaire template.

Ask when:

- the field is unknown or stale;
- confidence is too low;
- existing sources conflict;
- the answer would materially alter teaching/content generation.

## Grill behavior

The wizard may challenge assertions, but should prefer concrete evidence over rhetorical pressure.

Example:

"Strong in NestJS" -> ask about shipped scope only if that strength materially affects the current domain and lacks provenance.

## Docs-assisted mode

When artifacts are available:

- inspect first;
- identify stale dates;
- separate historical facts from current state;
- ask only unresolved follow-ups.

## Transaction semantics

A single wizard completion may modify:

- profile;
- global persona;
- domain persona;
- goals/preferences.

It must not directly alter competency projection based solely on persona answers. Demonstrated probes produce evidence records.

## Acceptance tests

- existing known stack ordering is not re-asked;
- contradictory profile/docs generate a targeted question;
- a one-off preference answer does not silently become high-confidence permanent persona;
- domain persona stores only domain-specific overrides;
- generated effective persona is materially smaller than canonical state;
- all persisted changes include provenance.
