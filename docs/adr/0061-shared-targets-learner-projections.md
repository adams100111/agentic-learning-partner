# ADR-0061: Learning Targets are shared; adaptation is per learner

**Status:** Accepted

## Context

A Learning Target such as PyLearn `go-alp` is platform content that, per ADR-0060, must be reusable and learner-free. Yet its curriculum is motivated by one learner's state, and other learners may later take it. Target identity could be per-learner (one course per learner) or shared.

## Decision

- A **Learning Target** is a shared platform-owned identity (ADR-0057) backed by reusable, learner-free content.
- Each learner gets their own **Target Adaptation Projection** over a shared target. Per-learner variation lives in projections and Accepted Adaptation Decisions, not in duplicated targets.
- Realization Links live in the specs of the learner whose evidence motivated the content. Other learners trace activity to competencies through the platform mapping, never through another learner's spec.
- Accepted Adaptation Decisions require explicit learner confirmation; agents may only propose them. Each decision records confirmer, time, and the projection revision it was based on, and is revocable by a superseding record.

## Consequences

- no per-learner course proliferation in platforms;
- content authored for one learner benefits others without leaking their state;
- the motivating learner's spec history is the audit trail for why content exists.

## Alternatives considered

- Per-learner targets: duplicates content and pushes private adaptation into public platform state.
