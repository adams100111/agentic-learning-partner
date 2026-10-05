# External Project Evidence

## Default strategy

Reference immutable source locations rather than copying external project content into learner state.

Example:

```yaml
source:
  kind: repository
  repository: adams100111/example
  commit: abc123
  paths:
    - internal/worker/pool.go
```

## Why

- avoids duplicating private code;
- keeps workspace small;
- preserves provenance;
- lowers token/storage cost.

## Optional excerpt

Store only a minimal immutable excerpt/hash when needed for:

- durable auditability;
- source expected to disappear;
- small evidence artifact required by rubric.

## Unavailable sources

If a referenced source later becomes inaccessible:

- keep historical evidence/assessment;
- mark source availability as unavailable;
- do not silently invalidate history;
- flag re-verification if consequential competency claims no longer have verifiable supporting evidence.
