# ADR-0032: Reference external project evidence instead of copying by default

**Status:** Accepted

## Decision

Evidence normally points to immutable repository commits/paths. Copy only minimal excerpts/hashes when needed.

## Consequences

- smaller private workspace;
- clearer provenance;
- less duplicated source code.
