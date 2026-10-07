---
name: record-learning-evidence
description: Convert a learning observation or project artifact into a minimal, append-only ALP evidence record.
---

# Record Learning Evidence

Use this when a learning activity, project change, diagnostic answer, platform event, or learner self-report should become durable evidence.

Prerequisite: the `alp` CLI must match the plugin version. When `alp` is missing or errors, follow `docs/CLI_PREREQUISITE.md` in the plugin root first.

## Workflow

0. Prefer an active ALP learning session. Check with `alp session status`; when no session is active and this evidence belongs to an interactive learning workflow, start one with `alp session begin --harness <harness>`. Do not open/close a separate session for every evidence record.

1. Identify the exact competency IDs the observation is relevant to.
2. Record only what happened; do not embed the competency judgment in the observation.
3. Choose the narrowest evidence type and strength justified.
4. If the result is negative/contradictory, classify the cause.
5. For repository evidence reference immutable repository/commit/path coordinates rather than copying source code.
6. Write a schema-valid evidence document.
7. Persist it with `alp evidence add --file <path>`. When a session is active, the command stages the evidence inside that session instead of checkpointing immediately.

Do not edit existing evidence records. Corrections use new evidence/supersession semantics.

Read `docs/EXTERNAL_EVIDENCE.md` and `docs/SELF_REPORT.md` only when those evidence classes are involved.
