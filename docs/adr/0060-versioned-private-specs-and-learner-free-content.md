# ADR-0060: Keep specifications as versioned private artifacts and authored platform content learner-free

**Status:** Accepted

## Context

Curriculum/Learning Unit Specifications and Target Adaptation Projections drive platform content PRs. Platform content (PyLearn Reel MDX) is static, public, and shared by all users. Specifications are derived from private learner evidence.

## Decision

- Target Adaptation Projections are generated, rebuildable files in the learner workspace. Their canonical inputs are learner state, target content/mapping snapshot, target constraints, and **Accepted Adaptation Decisions** — a separate canonical append-only record of choices that must survive rebuild.
- Curriculum/Learning Unit Specifications are immutable, versioned artifacts in the private learner workspace, recording learner revision, target snapshot, mapping/pack versions, sources, and authoring intent. Changes create new versions.
- Content PRs cite spec ID/hash and teaching intent only. Authored content and PRs never contain evidence, scores, attempt code, or learner-identifying data.
- Personalization may shape which units exist, their sequence, difficulty, examples, and analogies. Per-learner content variants are out of scope until a platform has a per-user content mechanism.

## Consequences

- specs are auditable and reproducible without leaking learner state;
- a public content repo can carry ALP-driven content safely;
- generated content is reusable by other learners.

## Alternatives considered

- Ephemeral prompts: content PRs become unauditable.
- Specs in the platform repo: leaks private learner reasoning.
