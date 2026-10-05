# Platform-to-Competency Mapping

## Ownership

The learning platform owns the mapping from its own content IDs to ALP competency IDs.

Example in PyLearn:

```yaml
go-b5-context-debugging:
  competencies:
    - go.runtime.context
    - go.concurrency.goroutines
```

## Why platform-owned

The platform controls:

- content identity;
- lesson/exercise changes;
- what a piece of content actually teaches/tests.

Therefore the mapping should change in the same PR as the content it describes.

## ALP responsibility

ALP:

- defines competency IDs in domain packs;
- validates that mapped IDs exist;
- normalizes platform events using the mapping;
- never requires PyLearn phase IDs inside the competency taxonomy.

## Learner workspace

The learner workspace does not own platform content mappings.
