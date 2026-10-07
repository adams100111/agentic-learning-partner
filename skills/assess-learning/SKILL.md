---
name: assess-learning
description: Assess demonstrated learner evidence against domain competencies and produce auditable assessment records.
---

# Assess Learning

Use this for diagnostics, code review, project checkpoints, debugging exercises, or explicit reassessment.

Prerequisite: the `alp` CLI must match the plugin version. When `alp` is missing or errors, follow `docs/CLI_PREREQUISITE.md` in the plugin root first.

## Workflow

0. Use one active ALP session for the assessment round. Start it with `alp session begin --harness <harness>` when necessary; reuse an existing active session rather than creating per-record checkpoints.

1. Resolve the target domain and competency IDs from the installed domain pack.
2. Inspect only evidence needed for the assessment.
3. Separate the observation from the judgment:
   - evidence records what happened;
   - assessment interprets what it demonstrates.
4. Classify negative evidence before changing competency judgment.
5. Use the domain rubric and evidence-strength hierarchy. Self-report is weak evidence for high competence.
6. Write one targeted assessment per competency, even when one project supports several competencies.
7. Persist evidence with `alp evidence add --file ...`.
8. Persist assessments with `alp assessment add --file ...`.
9. Rebuild projections with `alp state rebuild`.
10. Rebuild derived state with `alp state rebuild`, then close the logical learning transaction with `alp session close --summary "..."`. After close/sync, explain consequential changes with `alp competency show <id>`.

Production-ready cannot be established unless the runtime/domain gate accepts the hard evidence requirements.

Read `docs/ASSESSMENTS.md`, `docs/NEGATIVE_EVIDENCE.md`, and the target domain's diagnostic/rubric references when deeper policy is needed.
