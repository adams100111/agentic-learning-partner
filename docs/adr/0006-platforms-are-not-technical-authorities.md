# ADR-0006: External learning platforms are validation targets, not technical authorities

**Status:** Accepted

## Context

PyLearn is ALP's first integration and contains substantial existing curriculum, learner profiles, tool choices, and "modernity" guidance. Some of that material is recent and correct; other parts may age, reflect old learner state, or encode choices that are no longer best for late-2026 practice.

Treating PyLearn as authoritative would let historical AI-generated content recursively validate itself.

## Decision

ALP MUST treat PyLearn and other learning platforms as:

- sources of learner/activity evidence;
- sources of candidate curriculum/material;
- integration/validation environments.

They are NOT authoritative sources for technical truth.

Version-sensitive technical claims and ecosystem choices must be verified against current authoritative upstream sources according to `docs/SOURCE_POLICY.md`.

Age alone does not make content wrong. Existing material is retained when verification supports it and revised when evidence shows drift.

## Consequences

- ALP can improve PyLearn rather than merely reproduce it.
- Course-generation agents must perform targeted current-source verification.
- We need freshness metadata and source provenance in domain packs/content proposals.
- Content authoring costs more than trusting existing docs, but avoids recursive stale guidance.
