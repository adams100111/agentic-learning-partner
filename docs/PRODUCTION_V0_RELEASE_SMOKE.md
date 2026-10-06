# Production-v0 Real Harness Release Smoke

This is the external release gate for ALP production v0.

It is intentionally separate from the deterministic automated suite. Do not mark this smoke as passed unless the steps are actually executed with a real private Git remote, Claude Code, and Codex.

## Preconditions

- ALP production-v0 candidate CLI/plugin is installed in both harnesses.
- Git authentication works through the host SSH agent/credential helper.
- A private GitHub repository is available for learner state.
- Two independent local checkout locations represent device A and device B.
- The learner workspace contains bootstrap profile/persona state but no fabricated competency evidence.

## Device A — Claude Code

1. Install/load the ALP plugin.
2. Run the setup workflow:
   - create or connect a Git Store workspace;
   - use the private GitHub remote;
   - use session synchronization.
3. Verify:

       alp workspace status
       alp validate

4. Ask Claude Code:

   > Use Agentic Learning Partner to begin my Go learning journey. Inspect my existing learner state first. Do not assume competency from historical exposure. Run the minimum high-information first diagnostic round and persist only evidence I actually demonstrate.

5. Complete at least one assessed diagnostic activity.
6. Verify that ALP created canonical evidence/assessment state and rebuilt projections.
7. Close/checkpoint the session and synchronize.
8. Record the resulting remote commit SHA for the smoke report.

## Device B — Codex

1. Install/load the same ALP production-v0 candidate.
2. Clone/connect the same private learner-state repository into a separate local path.
3. Verify:

       alp workspace status
       alp validate
       alp status

4. Ask Codex:

   > Continue my Go learning from my current ALP state. First explain what ALP believes, cite the persisted evidence/assessment rationale, and choose the smallest next activity. Do not rely on prior chat memory.

5. Confirm Codex:
   - sees the evidence created on device A;
   - does not restart onboarding;
   - explains the same competency projection from canonical state;
   - selects the next activity from current gaps/review state.

## Concurrent append-only reconciliation

With both devices starting from the same remote revision:

1. device A records one independent evidence/session checkpoint locally;
2. device B records another independent evidence/session checkpoint locally;
3. push A;
4. synchronize B;
5. confirm both append-only records exist and derived state is rebuilt;
6. synchronize A and confirm the same canonical result.

## Explicit canonical conflict

Create incompatible explicit learner edits to the same profile claim on A and B.

Expected:

- ALP must not silently choose one;
- synchronization stops with learner-resolution-required behavior;
- after explicit resolution, the canonical profile validates and synchronizes.

## Offline recovery

On one device:

1. make the remote unavailable after session start;
2. complete a legitimate learning session;
3. confirm local checkpoint succeeds with sync pending;
4. restore network/remote;
5. resume synchronization;
6. confirm pending recovery journal clears only after successful sync.

## Export / restore smoke

1. export the synchronized workspace to a `.alp` archive;
2. verify the archive;
3. restore as a clone into a new Local Store;
4. confirm learner identity is preserved and workspace identity follows clone semantics;
5. confirm derived state rebuilds.

## Pass criteria

The release smoke passes only when all of the following are true:

- Claude Code creates genuine persisted learner evidence on device A;
- Codex continues from that state on device B without shared conversation memory;
- normal session sync requires no raw Git commands;
- no Git credentials appear in ALP state or chat;
- concurrent append-only changes reconcile;
- conflicting explicit learner claims stop for resolution;
- offline work checkpoints and later synchronizes;
- export/verify/restore succeeds;
- projections rebuild deterministically from canonical state.

## Reporting

Record:

- ALP commit/tag under test;
- Claude Code version;
- Codex version;
- Git version;
- operating systems/devices;
- private remote provider;
- device A starting/ending revision;
- device B starting/ending revision;
- pass/fail for each section;
- any manual intervention required.

Until this document has a real completed smoke report, ALP may be a production-v0 **candidate**, but the production-v0 release tag must not be claimed as validated.
