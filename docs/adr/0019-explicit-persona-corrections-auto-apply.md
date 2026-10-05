# ADR-0019: Explicit learner persona corrections may auto-apply

**Status:** Accepted

## Context

Requiring confirmation after an explicit correction such as "NestJS is stronger for me than .NET" adds pointless friction. Inferred durable traits are different.

## Decision

Explicit learner corrections may be persisted immediately with provenance and a concise change summary.

Material inferred/wizard-synthesized persona changes require confirmation according to risk.

## Consequences

- natural conversational UX;
- explicit learner authority remains primary;
- agents cannot silently redefine durable persona from weak inference.
