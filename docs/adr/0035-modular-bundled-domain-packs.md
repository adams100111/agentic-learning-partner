# ADR-0035: Domain packs are modular internally and bundled initially

**Status:** Accepted

## Decision

Ship Go/Rust packs inside the main repository initially behind a formal pack contract. Do not split into separate packages/repos until independent distribution/versioning pressure exists.

## Consequences

- simpler v1 development;
- future external packs remain possible without redesign.
