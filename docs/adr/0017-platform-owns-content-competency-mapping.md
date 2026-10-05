# ADR-0017: Learning platforms own content-to-competency mappings

**Status:** Accepted

## Context

ALP defines stable competencies, but only a platform knows which of its lessons/exercises currently teach or assess them.

## Decision

Store content-ID -> competency-ID mappings with the platform/content repository. ALP validates and consumes them.

## Consequences

- mappings evolve with content;
- ALP remains platform-independent;
- learner workspace remains free of platform metadata.
