# Assessments

## Why assessments exist

Evidence is an observation. Competency level is an interpretation.

If an LLM directly turns evidence into canonical competency state, "state rebuild" is not deterministic because a different model may interpret the same evidence differently.

ALP therefore separates:

```
Evidence -> Assessment -> Projection
```

## Evidence

Records what happened:

- code result;
- explanation;
- quiz answer;
- project diff;
- debug outcome;
- learner self-report.

Evidence should avoid claims such as "therefore the learner is strong".

## Assessment

A versioned judgment over one or more evidence records.

An assessment includes:

- competency;
- evidence IDs;
- judged level;
- confidence;
- rationale;
- rubric ID/version;
- assessor identity/model/harness;
- supersession/status metadata.

Assessments are append-only. Corrections/supercession create new records rather than silently rewriting history.

## Projection

Projection is deterministic over **accepted assessments**, according to a defined folding policy.

The first policy should be conservative and simple:

- newest accepted assessment from the current rubric generation is the primary level;
- contradictory accepted assessments reduce confidence or trigger reassessment according to policy;
- `production-ready` is rejected unless the domain rubric's hard evidence requirements are satisfied;
- superseded/rejected assessments do not contribute.

The exact folding algorithm must be implemented and tested, not left to a prompt.

## Reassessment

When a rubric improves:

1. old evidence remains untouched;
2. new assessments may reference old evidence;
3. new assessments supersede prior assessments where appropriate;
4. projections rebuild deterministically.

This preserves auditability while allowing ALP to get smarter.
