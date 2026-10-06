# ADR-0053: Fail explicitly on missing Store capabilities and keep unimplemented providers informative

**Status:** Accepted

## Context

Different Store providers support different capabilities.

Silently degrading a workflow can change durability or synchronization guarantees without the learner realizing it. At the same time, freezing speculative implementation details for providers that do not yet exist creates unnecessary compatibility burden.

## Decision

When a workflow requires an unavailable Store capability, ALP fails explicitly, explains the missing capability, and offers the nearest valid alternative.

ALP does not silently emulate or degrade provider guarantees.

The production-v0 Store capability/logical contract is normative.

Descriptions of unimplemented S3-compatible, WebDAV, remote API, and PostgreSQL-backed providers are informative design guidance only until an implementation is shipped.

## Consequences

- provider behavior is honest and inspectable;
- workflows do not accidentally lose sync/history guarantees;
- future providers can evolve without violating speculative wire/storage formats.
