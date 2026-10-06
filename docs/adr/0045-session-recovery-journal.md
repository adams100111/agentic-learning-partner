# ADR-0045: Keep interrupted-session recovery state machine-local

**Status:** Accepted

## Context

A learning session can be interrupted after creating provisional evidence/assessment changes but before completing its canonical checkpoint, or after checkpointing but before pushing.

Incomplete operational state should not become synchronized learner truth merely because a process crashed.

## Decision

In-progress session recovery metadata lives outside canonical synchronized learner state in a machine-local recovery journal.

The journal contains only operational recovery information such as:

- workspace identity;
- session ID;
- base Workspace Revision;
- staged ChangeSet references;
- checkpoint progress;
- sync-pending state.

On recovery:

- unfinished uncheckpointed learning changes require resume/review or explicit discard;
- ALP must not silently promote them to canonical learner truth;
- if the journal proves the canonical checkpoint already completed and only remote push remains, ALP may continue the pending synchronization safely.

Successfully completed sessions may persist a canonical session record according to retention policy, but the crash journal itself remains local operational state.

## Consequences

- interrupted processes do not create accidental shared truth;
- users can resume meaningful work without losing staged changes;
- sync-only recovery can complete automatically when the canonical checkpoint already exists;
- operational recovery details do not pollute learner-state history.
