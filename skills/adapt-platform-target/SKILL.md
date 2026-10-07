---
name: adapt-platform-target
description: Adapt a platform Learning Target (a course on a learning platform) to the learner — propose skip, skim, challenge or full per unit from their evidence, record the learner's confirmed Accepted Adaptation Decisions, revoke one on request, and explain the persona-derived teaching shape.
---

# Adapt Platform Target

ALP proposes; the learner decides. `alp platform plan` computes an **Adaptation Proposal** for every unit (`proposedMode`). It becomes binding only as an **Accepted Adaptation Decision**, which you record after the learner confirms it. Every action here goes through the `alp platform` CLI and its JSON output; the CLI owns all learner truth, so you read its JSON and pass values back to it.

## Inputs

Every command takes the same target inputs. Gather them once:

- `--adapter ID` and `--target ID`: the platform adapter and its Learning Target.
- `--curriculum FILE`: the platform's curriculum export for the target, produced by the platform. Ask the learner for it when you do not have it.
- `--mapping FILE`: the platform's content mapping for the target. `alp platform inspect` reports the mapping's path in the platform repository as `mapping.ref`.
- `--constraints FILE`: the target constraints, when the target has any.
- `--workspace PATH`: only when the learner workspace is not the default one.

When `alp` is not on `PATH`, tell the learner how to install it (`docs/INSTALL.md`) and stop.

## 1. Inspect the target

```bash
alp platform inspect --adapter <adapter> --target <target> --curriculum <curriculum>
```

Done when the command exits 0 and `capabilities` includes `content-mapper`, `curriculum-reader` and `authoring-target`. If one is missing, the platform cannot be adapted: tell the learner which capability it lacks and stop.

## 2. Plan

```bash
alp platform plan --adapter <adapter> --target <target> --curriculum <curriculum> --mapping <mapping> [--constraints <constraints>]
```

Keep the JSON. You need, from `projection`:

- `revision` (`tapr_…`): the **basis** every decision is confirmed against;
- `units[]`: `item.item` (the unit ID), `title`, `proposedMode`, `mode`, `rationale`, `competencies[]` (`id`, `status`, `assessmentId`) and `decision` (`id`, `mode`, `applied`, `reason`) when a decision already applies;
- `sequence`, `reinforcement` and `gaps`.

On `invalid-mapping`, run `alp platform mapping validate` with the same inputs and report its `problems[]` to the learner; the mapping belongs to the platform, so the fix is a platform change.

Done when you hold `projection.revision` and every unit's `proposedMode` and `mode`.

## 3. Present the proposals

Show one row per unit, in curriculum order: title, `proposedMode`, effective `mode`, and the `rationale` with the competency statuses it cites. Explain the modes in the learner's terms:

| Mode | Meaning |
|---|---|
| `skip` | every competency the unit teaches or assesses is already demonstrated |
| `challenge` | strong but unconfirmed: test out instead of re-learning |
| `skim` | functional: review quickly |
| `full` | unassessed or rusty, or the unit maps to no competencies |

Call out where `mode` differs from `proposedMode` (the target's constraints moved it, or a decision applies), any decision with `applied: false` and its `reason`, and the `gaps` the target does not cover.

Then explain the **teaching shape**, which personalizes the shared content without per-learner copies. `specifications.persona.documents` lists the profile and persona documents that shaped it and `specifications.persona.defaults` the defaults that stood in for missing ones; each `specifications.units[].path` is a unit specification in the learner workspace whose `teaching.shape` holds `pace`, `activityTypes`, `avoid`, `emphasis`, `feedback` and per-competency `analogies` (`basis`, `sources`), and whose `adaptation.persona` holds the private basis (`risks`, `analogyReasons`, `sourceExperience`). Tell the learner which of their persona choices produced the shape, and which fields fell back to a default because a persona layer is missing; that gap is filled by refining the persona, outside this skill.

Done when the learner has seen every unit's proposal with its rationale, and the teaching shape with its sources.

## 4. Record the learner's decisions

Ask which proposals the learner accepts, unit by unit; the learner may also choose a different mode. A decision is the learner's, so record exactly the unit and mode they named, after they answer yes to that unit and mode in this conversation. Silence, a general "looks good", or your own judgement leave the proposal a proposal.

For each confirmed unit:

```bash
alp platform decision accept --adapter <adapter> --target <target> --curriculum <curriculum> --mapping <mapping> [--constraints <constraints>] \
  --unit <unit item ID> --mode skip|challenge|skim|full --basis <projection.revision> --confirm [--reason "<the learner's words>"]
```

Read `status`: `accepted`, or `already-accepted` (nothing written). Each accepted decision changes the projection, so take the next `--basis` from this output's `projection.revision`. Handle refusals:

| `error.code` | Action |
|---|---|
| `stale-projection-revision` | Learner state, the target or the decisions changed: re-run step 2, show what changed, and confirm again. |
| `mode-not-allowed` | The target constraints forbid the mode: offer an allowed one. |
| `unknown-unit` | Use `projection.units[].item.item`. |

Done when every unit the learner confirmed reports `accepted` or `already-accepted`, and nothing else was recorded.

## 5. Revoke on request

When the learner asks to undo a decision, take its ID from `projection.units[].decision.id`, confirm it with them, then:

```bash
alp platform decision revoke --adapter <adapter> --target <target> --curriculum <curriculum> --mapping <mapping> [--constraints <constraints>] \
  --decision <aad_…> --basis <projection.revision> --confirm [--reason "<the learner's words>"]
```

`status: revoked` appends a superseding record; the original stays as audit trail. `unknown-decision` means it is not active for this target.

Done when the output reports `revoked` and the unit in its `projection.units[]` carries no `decision`.

## After adapting

The plan run in step 2 also recorded an Authoring Plan (`authoringPlan`, with `authoring.status` and `authoring.reason`) when the target needs new or changed content. Realizing it on the platform is the `orchestrate-platform-authoring` workflow; tell the learner it is ready, with the `authoringPlan.id`.

Read `docs/TARGET_ADAPTATION.md` when a field or refusal above needs its full definition.
