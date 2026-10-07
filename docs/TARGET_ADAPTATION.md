# Target adaptation: `alp platform plan`, specifications, Authoring Plans and Accepted Adaptation Decisions

ALP adapts a shared Learning Target (for example PyLearn `go-alp`) to one learner without forking the target or the learner's competency state (ADR-0055, ADR-0060, ADR-0061).

## Target Adaptation Projection

```
alp platform plan --adapter pylearn --target go-alp \
  --curriculum curriculum-export.json --mapping go-alp.mapping.yaml \
  [--constraints go-alp.constraints.yaml] \
  [--intent target-skeleton|curriculum|unit|activity|patch] [--unit UNIT] [--workspace PATH]
```

`plan` requires the Curriculum Reader, Content Mapper and Authoring Target capabilities and refuses an invalid mapping. It writes the learner's projection to `state/target-adaptations/<id>.json` (`schemas/target-adaptation-projection.schema.json`), reconciles the Curriculum and Learning Unit Specifications and records an Authoring Plan (see below), all in one Store transaction, and prints deterministic JSON: `{schemaVersion, adapter, target, path, projection, specifications, authoring, authoringPlan}`.

The projection is a pure function of its canonical inputs, all recorded under `inputs`:

- learner state: the competency projection computed from accepted assessments (the same computation as `alp state rebuild`; `plan` never writes `state/competencies.yaml`);
- the target's curriculum export and mapping (content hashes) and the resolved domain-pack versions;
- optional target constraints (content hash);
- the active Accepted Adaptation Decisions for the target.

`id` (`tap_…`) is derived from `{platform, target}`, so each learner workspace has one projection per shared target. `revision` (`tapr_…`) hashes the projection content, so rebuilding from the same inputs reproduces the file byte for byte, and any change to learner state, target content, constraints or decisions changes it. The file is disposable: delete it and re-run `plan`.

### Units and modes

A unit is a top-level curriculum item (an item placed in a phase), in phase then export order. Its competencies are the mapped competencies of the unit and its descendants, each with the learner's status read from — and citing the `assessmentId` of — the competency projection. The projection never judges competency itself.

| Status | Competency projection |
|---|---|
| `demonstrated` | `strong` or `production-ready`, confidence medium/high, not flagged for reassessment |
| `unconfirmed` | `strong` or `production-ready` with low confidence or needing reassessment |
| `functional` | `functional` |
| `rusty` | `rusty` |
| `unassessed` | no projected competency, or `unknown` |

`proposedMode` is derived from the competencies the unit teaches or assesses (reinforced competencies count only when it does neither): any `unassessed` or `rusty` → `full`; else any `unconfirmed` → `challenge` (test out instead of re-learning); else any `functional` → `skim`; all `demonstrated` → `skip`. A unit with no mapped competencies is `full`.

`mode` is the effective mode: target constraints move a disallowed mode to the next allowed mode in the order skip → challenge → skim → full (`full` is always allowed), and an active Accepted Adaptation Decision then sets the mode if the constraints allow it (otherwise the decision is reported with `applied: false`). `sequence` lists the units that are not skipped. `reinforcement` lists target competencies that are `rusty`, `unconfirmed`, or have assessment gaps, with the items that reinforce or assess them. `gaps` lists unassessed or rusty prerequisites (from the domain pack) of target competencies that the target does not cover.

`proposedMode` is the Adaptation Proposal an agent may put to the learner. Agents never record decisions.

## Target constraints

`schemas/target-constraints.schema.json`:

```yaml
schemaVersion: 1
platform: pylearn
target: go-alp
allowedModes: [skip, challenge, full]   # must include full
requiredUnits: [go-alp-a1-context]      # never skipped
goal: [go.concurrency.channels]         # optional: competencies the target is for
```

`goal` competencies must belong to a domain pack the mapping declares. Specifications propose units for goal competencies the target's content does not cover yet, and for their prerequisites the learner has not shown (unassessed or rusty). A new target (no items yet) gets its whole skeleton from its goal.

## Accepted Adaptation Decisions

```
alp platform decision accept --adapter pylearn --target go-alp --curriculum … --mapping … [--constraints …] \
  --unit go-alp-a3-generics --mode skim --basis tapr_… --confirm [--reason TEXT]
alp platform decision revoke --adapter pylearn --target go-alp --curriculum … --mapping … [--constraints …] \
  --decision aad_… --basis tapr_… --confirm [--reason TEXT]
```

A decision is a canonical, append-only record in `adaptation-decisions/<id>.yaml` (`schemas/accepted-adaptation-decision.schema.json`). It is written only when:

- `--confirm` states the learner's explicit confirmation (otherwise `learner-confirmation-required`, nothing written);
- `--basis` equals the current projection revision rebuilt from the same inputs (otherwise `stale-projection-revision`);
- the unit is a unit of the target (`unknown-unit`) and the mode is allowed by the constraints (`mode-not-allowed`).

Each record carries `learnerId`, `confirmation: {confirmedBy: learner, confirmedAt}`, `recordedAt` and `basis.projectionRevision`. Accepting a different mode for a unit supersedes its active decision; accepting the same mode again reports `already-accepted` and writes nothing. `revoke` appends a record that supersedes the decision; the original stays. The decision record and the rebuilt projection are written in one Store transaction, and the command prints both.

Decisions are inputs to the projection, not part of it, so they survive any projection rebuild.

## Curriculum and Learning Unit Specifications

`plan` derives platform-neutral specifications from the projection (ADR-0056, ADR-0060). They are immutable, versioned, canonical records in the private learner workspace, never in a platform repository:

- `specifications/units/<uspec_id>-v<n>.json` (`schemas/learning-unit-specification.schema.json`);
- `specifications/curricula/<cspec_id>-v<n>.json` (`schemas/curriculum-specification.schema.json`).

IDs are ALP-owned (ADR-0057): `cspec_…` is derived from `{platform, target}`; `uspec_…` from the target plus the existing unit's declared-stable item, or, for a proposed unit, the competency it is proposed for. A version is written once and never replaced; `contentHash` hashes its canonical JSON without `contentHash`, and `plan` refuses (`specification-integrity`) a stored version whose content no longer matches its hash.

**Units.** The target's existing units (top-level items, in target order) plus proposed units: uncovered prerequisite `gaps` of the projection and uncovered `goal` competencies with their unshown prerequisites. Each proposed unit teaches one competency and is placed before the first existing unit that depends on it; the rest follow in prerequisite order. Existing units are grouped by their platform phase; proposed units by prerequisite depth (`stage-1`, `stage-2`, …), which the authoring target maps to its own structure.

**A unit specification records:**

| Field | Content |
|---|---|
| `provenance` | `learnerStateRevision` (`lsr_…`, a hash of the competency projection), `projectionRevision`, the target snapshot (`curriculum`, `mapping`, `constraints` content hashes), domain `packs` and the `sources` they were verified against |
| `authoringIntent` | the smallest intent the unit justified: `unit` (proposed, not skipped), `activity` (existing, not skipped, and something it requires evidence for is assessed by nothing in its item tree), or `none` |
| `unit` | `platformItem` (absent for a proposed unit) and `group` |
| `teaching` | learner-free teaching intent: `title`, `objectives`, `competencies` with Mapping Roles, `prerequisites`, `dependsOn` (unit spec IDs), `requiredEvidence` (assessment-grade, `minimumLevel: functional`, for each taught or assessed competency unless skipped), and `claims` |
| `adaptation` | private learner basis: effective and proposed `mode`, applied `decision`, `rationale`, competency statuses citing assessments, and `misconceptions` (assessment-recorded gaps) |

`claims` come from pack freshness classes: `version-sensitive-language-runtime`, `operational-platform` and `security-sensitive` are version-sensitive and source-required; `ecosystem-choice` is source-required; `stable-concept` declares no claim. The authoring target re-verifies these at authoring time and records source provenance (Q27).

The curriculum specification records the same provenance plus `goal`, `groups`, `units` (each unit spec ID, version, hash, title, group, mode) and the `sequence` of units that are not skipped. Its `authoringIntent` is `target-skeleton` for a target with no content, otherwise the smallest intent any unit justifies.

Specifications contain no platform-native rendering concepts.

### Regeneration policy

Projections are recomputed freely. A new unit specification version is written only when a **material** field changes: competencies (IDs and roles), prerequisites, adaptation mode, misconceptions, or required evidence. Anything else (titles, objectives, claims, provenance, a new projection revision from evidence that changes nothing material) keeps the stored version, which `plan` reports as `unchanged`. A new version carries `supersedes: {version, contentHash}` and `change: {materialFields, evidenceLinked}`, where `evidenceLinked` is true when the learner state revision changed with it. The curriculum specification is re-versioned only when its goal, groups or referenced unit versions change.

Once a unit is realized (Realization Links, ticket #73), a material change will also have to be evidence-linked before it produces a new version and so a content change; until then units are unrealized and are re-specified freely.

## Authoring Plans and Authoring Intent

An Authoring Plan (`authoring-plans/<apl_id>.json`, `schemas/authoring-plan.schema.json`) is an Authoring Intent over specific specification versions, realized by the authoring target skill the adapter declares (`authoringTarget.skill`; PyLearn declares `pylearn-alp-authoring`). Its ID hashes its content, so the same intent over the same versions is the same plan (`authoringPlan.status: existing`).

Intents, largest to smallest: `target-skeleton` → `curriculum` → `unit` → `activity` → `patch`.

- **Default** (no `--intent`): the smallest justified intent (`activity` before `unit`), for the first unit in curriculum order that justifies it. `--unit UNIT` (a `uspec_…` ID or a unit item ID) selects the unit and uses its justified intent. Nothing justified → `authoring.status: nothing-to-author`, no plan.
- **New target** (no items): the default never authors; it reports `explicit-intent-required` with intent `target-skeleton`.
- `--intent target-skeleton`: explicit only, and only for a target with no content (`intent-not-applicable` otherwise). Covers every unit.
- `--intent curriculum` (whole course): explicit only. Covers every unit that justifies authoring.
- `--intent unit|activity|patch [--unit UNIT]`: `unit` applies only to proposed units; `activity` and `patch` only to existing units (`intent-not-applicable` otherwise). Without `--unit`, the first unit that justifies the intent.

`authoring` reports `{status, intent, selection: default|explicit, reason}`.

### Public face

Only `authoringPlan.public` may appear in platform content, branches or PRs (ADR-0060): `intent`, the curriculum citation (`id`, `version`, `contentHash`, `title`, `goal`, `groups` with unit IDs, titles and platform items) and, per unit in scope, its citation, `platformItem`, `group` and `teachingIntent` (exactly the spec's `teaching`). It carries no learner ID, evidence, assessment, status, misconception, adaptation mode, rationale or justification.

### What the authoring target skill receives

The orchestration skill hands the platform's authoring skill the Authoring Plan JSON (`authoringPlan` from `plan`, or the recorded file). The skill realizes `public.units[*].teachingIntent` (and, for `target-skeleton`, `public.curriculum.groups`) natively, re-verifies every `claims` entry, and cites spec ID/version/hash in its PR. It may read the full unit specifications in the learner workspace (for example `adaptation.misconceptions` and `adaptation.mode` to shape difficulty and examples) but must never copy them, or anything else outside `public`, into content or PRs.
