# ADR-0018: One learner per workspace in v1

**Status:** Accepted

## Context

Supporting many learners in one Git workspace would complicate identity, privacy, context routing, and merge semantics before there is a proven need.

## Decision

One learner workspace represents exactly one learner in v1.

## Consequences

- simple context/state model;
- clear privacy boundary;
- multi-user/SaaS persistence is explicitly deferred.
