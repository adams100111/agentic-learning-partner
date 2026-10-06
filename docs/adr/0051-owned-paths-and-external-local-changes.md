# ADR-0051: Restrict Store checkpoints to ALP-owned paths and preserve external local edits

**Status:** Accepted

## Context

A dedicated learner workspace repository may still contain benign repository metadata or human edits.

Blind Git staging risks committing unrelated files. Conversely, manually edited ALP-owned canonical documents may contain legitimate learner changes that must not be silently discarded.

## Decision

Production v0 defines an ALP-owned path policy covering canonical/derived learner-state paths such as:

- workspace manifest;
- profile;
- personas;
- evidence;
- assessments;
- sessions;
- derived state.

Git Store checkpoints stage only ALP-owned/registered portable paths. Normal Store code never uses broad repository staging such as `git add -A`.

At session start, dirty ALP-owned paths are treated as uncheckpointed external local learner-state changes:

1. validate;
2. classify provenance as external local change;
3. include them in semantic reconciliation;
4. require human resolution when semantic intent is ambiguous.

Unrelated dirty repository paths are never absorbed into ALP state.

## Consequences

- agents cannot accidentally commit unrelated repository files;
- power-user canonical edits are preserved;
- dedicated workspace repositories can still carry README/license/gitignore metadata safely.
