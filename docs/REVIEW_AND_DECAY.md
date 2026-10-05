# Competency Freshness and Review Scheduling

## No time-based competency downgrade

Time alone does not reduce a demonstrated competency level.

A competency can remain:

```yaml
level: strong
freshness: stale
confidence: medium
reviewDue: true
```

without being downgraded to `functional`.

Downgrade requires contradictory evidence interpreted through an assessment.

## Freshness

Track freshness separately from level.

Freshness may consider:

- time since last relevant strong evidence;
- evidence type/strength;
- whether the learner has exercised the competency in recent project work;
- dependency importance;
- domain risk.

## Review scheduling

Do not use one universal spaced-repetition interval.

Review priority should consider:

- time since last strong evidence;
- competency importance;
- prior failures;
- evidence strength;
- current project relevance;
- dependency importance;
- production/security consequence.

Project usage can satisfy review when it genuinely exercises the competency.

## Review output

The review scheduler should produce a prioritized queue with rationale rather than only dates.
