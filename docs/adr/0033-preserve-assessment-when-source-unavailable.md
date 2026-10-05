# ADR-0033: Preserve historical assessments when source becomes unavailable

**Status:** Accepted

## Decision

Loss of source access does not erase historical evidence/assessment. Mark availability and require re-verification where current high-consequence claims depend entirely on inaccessible evidence.

## Consequences

- audit history survives repo deletion/access changes;
- current confidence can still react to unverifiable evidence.
