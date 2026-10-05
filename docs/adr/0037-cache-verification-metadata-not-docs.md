# ADR-0037: Cache source verification metadata, not full external docs

**Status:** Accepted

## Decision

Persist compact source/version/hash/claim metadata and re-fetch authoritative content when freshness policy requires it.

## Consequences

- lower token/storage cost;
- avoids stale documentation mirrors.
