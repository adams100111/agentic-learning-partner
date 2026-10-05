# Domain Pack Versioning and Migrations

## Semantic versioning

Every domain pack declares a semantic version.

```yaml
domain: go
version: 1.2.0
```

## Breaking taxonomy changes

Renaming, splitting, merging, or removing competency IDs requires explicit migration metadata.

Example:

```yaml
migrations:
  - from: go.concurrency.sync
    to:
      - go.concurrency.mutexes
      - go.concurrency.coordination
    strategy: reassess
```

Historical evidence and assessments keep their original references for auditability.

Projection/migration tooling resolves old IDs according to explicit migration rules.

## Compatibility

Learning platforms that map content to competencies declare a compatible domain-pack version range.

Breaking pack upgrades require platform mapping migration.
