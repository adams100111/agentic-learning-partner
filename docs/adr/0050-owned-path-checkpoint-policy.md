# ADR-0050: Checkpoint only ALP-owned workspace paths and treat external canonical edits as learner-state changes

**Status:** Accepted

## Context

A dedicated Git Store workspace may still contain repository metadata or user-edited ALP files.

Blind staging risks committing unrelated files. Conversely, discarding human edits to ALP canonical documents would violate user ownership of learner state.

## Decision

Git Store checkpoints stage only ALP-owned paths and explicitly registered portable workspace metadata.

Owned paths include canonical and generated ALP workspace paths such as:

- workspace manifest;
- profile;
- personas;
- evidence;
- assessments;
- retained compact sessions;
- derived state.

Git Store never uses blind whole-repository staging such as `git add -A`.

Unrelated repository files remain outside ALP checkpoint scope.

Human edits to ALP-owned canonical files are treated as uncheckpointed learner-state changes:

1. detect dirty owned paths;
2. validate them;
3. classify provenance as an external local change where appropriate;
4. reconcile against remote canonical state;
5. require human resolution when semantic intent is ambiguous.

Profile/persona reconciliation operates at semantic claim/field granularity where schemas provide identifiable structure. Free-form conflicting fields stop safely rather than being guessed.

## Consequences

- agents cannot accidentally commit unrelated repository contents;
- advanced users retain the ability to edit canonical learner state directly;
- semantic merge behavior can evolve with richer profile/persona schemas;
- checkpoints remain scoped to ALP-owned state.
