---
name: orchestrate-platform-authoring
description: Realize an ALP Authoring Plan on a learning platform — hand the plan to the authoring skill the platform declares, record the platform's gate result, and report which units are realized. Also runs the activity loop - link the learner's platform account, import platform activity as evidence, re-plan.
---

# Orchestrate Platform Authoring

A thin conductor. ALP says *what* a unit must teach (an Authoring Plan); the platform's own **authoring target skill** decides *how* in the platform's native format and proves it with the platform's own gates. You resolve that skill from the platform's declaration, hand it the plan, and record what came back. Platform authoring knowledge lives in that skill; everything ALP knows comes from `alp platform` JSON.

Prerequisite: the `alp` CLI must match the plugin version. When `alp` is missing or errors, follow `docs/CLI_PREREQUISITE.md` in the plugin root first.

## Inputs

- `--adapter ID` and `--target ID`: the platform adapter and its Learning Target.
- `--curriculum FILE`: the platform's curriculum export for the target. Ask the learner for it when you do not have it.
- `--mapping FILE`: the platform's content mapping (`alp platform inspect` reports its path in the platform repository as `mapping.ref`).
- `--constraints FILE`: the target constraints, when the target has any.
- `--workspace PATH`: only when the learner workspace is not the default one.

When `alp` is not on `PATH`, tell the learner how to install it (`docs/INSTALL.md`) and stop.

## Authoring

### 1. Get the Authoring Plan

Take the plan from the learner (an `apl_…` ID, whose file is `authoring-plans/<apl_id>.json` in the learner workspace) or from a fresh plan:

```bash
alp platform plan --adapter <adapter> --target <target> --curriculum <curriculum> --mapping <mapping> [--constraints <constraints>]
```

Read `authoring`:

| `authoring.status` | Action |
|---|---|
| `planned` | Use `authoringPlan` (`id`, `path`, `intent`, `authoringTarget.skill`). |
| `nothing-to-author` | Report `authoring.reason` and stop. |
| `explicit-intent-required` | A new target with no content. Ask the learner whether to author its skeleton; on yes, re-run plan with `--intent target-skeleton`. |

Whole-course authoring (`--intent curriculum`) and a chosen unit (`--unit <uspec_… or unit item ID>`) are explicit learner requests; the default is the smallest justified intent.

Done when you hold one Authoring Plan: its `id`, its file path and its `authoringTarget.skill`.

### 2. Resolve the authoring target skill

```bash
alp platform inspect --adapter <adapter> --target <target> --curriculum <curriculum>
```

`authoringTarget.skill` names the skill the platform declares. It must equal the plan's `authoringTarget.skill`. When `authoringTarget` is `null` the platform declares no authoring target: report that and stop. When the two names differ, the plan predates a platform change: re-run step 1.

Done when one skill name comes from the platform's declaration and matches the plan.

### 3. Hand the plan to that skill

Invoke the declared skill with the Authoring Plan file and, for each unit in the plan, its private persona view from the `plan` output (`specifications.units[].persona`; re-run step 1's command when you started from an `apl_…` ID — the same inputs give the same plan). It works in the platform's own repository: it realizes the plan's `public` face, re-verifies the plan's claims, runs the platform's gates and opens the platform's normal review. Only `public` may appear in platform content or PRs; the rest of the plan and the persona view are the learner's private record: do not write the persona view to a file in the platform repository.

The skill is the platform's, installed from the platform's repository, so this harness may not have it. When you cannot invoke it, give the learner its name, the plan path and the hand-back below, and stop here; step 4 resumes in any harness once the hand-back exists.

The authoring target skill returns exactly three files, its hand-back:

- **gate result**: the platform validator's JSON output, verbatim (`schemas/platform-gate-result.schema.json`), kept even when it is not publishable;
- **curriculum after authoring**: the platform's curriculum export taken on the authoring branch;
- **realization report**: per realized unit, its declared-stable items and the source provenance of its claims (`schemas/platform-realization-report.schema.json`).

This skill is the sole recorder of gate results (Q24): the authoring target skill returns the hand-back and never runs `alp platform gates record` or writes to the learner workspace. When the platform skill offers to record, decline and record it yourself in step 4, so each result is recorded once, against the plan you handed over.

Done when you hold all three files, or the learner has the hand-off.

### 4. Record the gate result

```bash
alp platform gates record --adapter <adapter> --target <target> --curriculum <curriculum after authoring> \
  --plan <apl_id> --result <gate result> --realization <realization report>
```

Record every result, publishable or not: the record is the audit trail. `status` is `recorded`, or `already-recorded` when the same result and report were recorded before (nothing written). A refusal records nothing; report its `error.code` and `error.message` (`invalid-gate-result`, `invalid-realization-report`, `invalid-realization-link`, `realization-conflict`, `unknown-authoring-plan`) to whoever produced the file.

Done when the output reports `recorded` or `already-recorded`.

### 5. Report per unit

For each entry of `record.units[]` report `unit.id`, `unit.version` and `status`:

- `realized`: name its Realization Link from `realizations[]` (`root` item and `path`).
- `unrealized`: give every `reasons[]` entry (`code`, `message`) and the fix it points to: `not-publishable` → the platform gates named in the message must pass; `realization-not-reported` → the realization report must list the unit's items; `provenance-missing` → the named claim needs a verified source. The fix is another pass of steps 3–4 on the same plan.

Done when every unit in `record.units[]` is reported realized or unrealized with its reasons.

## Activity loop

Learner activity on the platform flows back as evidence, and evidence changes the next plan.

### A. Link the learner's platform account

Import needs a learner-confirmed link from the platform account to this workspace. The platform's activity export names the platform instance and user ID; an import refused with `platform-account-not-linked` quotes both in `error.message`. Show them to the learner and ask whether that account is theirs. Only after they say yes:

```bash
alp platform account link --adapter <adapter> --instance <instance> --user <platform user ID> --confirm
```

`status` is `linked` or `already-linked`. `platform-account-linked-to-another-learner` means the account belongs to a different workspace: stop and tell the learner.

Done when the account reports `linked` or `already-linked`.

### B. Import activity

The cursor belongs to the caller: pass the `cursor.next` the previous import returned, or omit `--cursor` on the first import.

```bash
alp platform import --adapter <adapter> --target <target> --curriculum <curriculum> --mapping <mapping> \
  --export <activity export> [--cursor <previous cursor.next>]
```

Report `counts` (`records`, `imported`, `skipped`, `superseded`, `unmapped`) and list the `unmapped` records with their `reason`: activity the mapping does not cover yet, a platform mapping change. Hand `cursor.next` back to the learner or calling workflow to keep for the next import; ALP does not store it. `cursor-mismatch` means the export does not continue from the given cursor: get an export since that cursor, or a full snapshot.

Done when the import exits 0 and the caller holds the new `cursor.next`.

### C. Re-plan

Run `alp platform plan` (step 1) again. Compare each unit's `proposedMode` with the previous plan and report the changes with their `rationale`. Changed proposals stay proposals until the learner confirms them with `alp platform decision accept` (the `adapt-platform-target` workflow). A new `authoringPlan` starts the Authoring steps again.

Done when the learner has the changed proposals and, if any, the new Authoring Plan ID.

Read `docs/TARGET_ADAPTATION.md` (plans, gates, realization) and `docs/PLATFORM_MAPPING.md` (import grading) when a field or refusal above needs its full definition.
