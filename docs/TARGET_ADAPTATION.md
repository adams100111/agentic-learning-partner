# Target adaptation: `alp platform plan`, specifications, Authoring Plans and Accepted Adaptation Decisions

ALP adapts a shared Learning Target (for example PyLearn `go-alp`) to one learner without forking the target or the learner's competency state (ADR-0055, ADR-0060, ADR-0061).

## Target Adaptation Projection

```
alp platform plan --adapter pylearn --target go-alp \
  --curriculum curriculum-export.json --mapping go-alp.mapping.yaml \
  [--constraints go-alp.constraints.yaml] \
  [--intent target-skeleton|curriculum|unit|activity|patch] [--unit UNIT] [--workspace PATH]
```

`plan` requires the Curriculum Reader, Content Mapper and Authoring Target capabilities and refuses an invalid mapping. It writes the learner's projection to `state/target-adaptations/<id>.json` (`schemas/target-adaptation-projection.schema.json`), reconciles the Curriculum and Learning Unit Specifications and records an Authoring Plan (see below), all in one Store transaction, and prints deterministic JSON: `{schemaVersion, adapter, target, path, projection, specifications, authoring, authoringPlan}`. `specifications.persona` reports the persona documents that shaped unit teaching and the documented defaults that stood in for missing ones (see [Teaching shape](#teaching-shape)).

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
| `provenance` | `learnerStateRevision` (`lsr_…`, a hash of the competency projection), `projectionRevision`, the target snapshot (`curriculum`, `mapping`, `constraints` content hashes), domain `packs` and the `sources` they were verified against, and `personas`: the `profile`, `global` and `domains` persona documents (path and `sha256:` content hash; `null` or absent when missing) |
| `authoringIntent` | the smallest intent the unit justified: `unit` (proposed, not skipped), `activity` (existing, not skipped, and something it requires evidence for is assessed by nothing in its item tree), or `none` |
| `unit` | `platformItem` (absent for a proposed unit) and `group` |
| `teaching` | learner-free teaching intent: `title`, `objectives`, `competencies` with Mapping Roles, `prerequisites`, `dependsOn` (unit spec IDs), `requiredEvidence` (assessment-grade, `minimumLevel: functional`, for each taught or assessed competency unless skipped), `claims`, and the persona-derived `shape` |
| `adaptation` | private learner basis: effective and proposed `mode`, applied `decision`, `rationale`, competency statuses citing assessments, `misconceptions` (assessment-recorded gaps), and the private `persona` basis of the teaching shape |

`claims` come from pack freshness classes: `version-sensitive-language-runtime`, `operational-platform` and `security-sensitive` are version-sensitive and source-required; `ecosystem-choice` is source-required; `stable-concept` declares no claim. The authoring target re-verifies these at authoring time and records source provenance (Q27).

### Teaching shape

Personalization shapes examples, analogies and difficulty even in shared content (ADR-0060, Q26). Each unit specification's `teaching.shape` is derived deterministically from the learner's Learner Profile (`profile/profile.yaml`), Global Persona (`personas/global.yaml`) and the Domain Persona of each of the unit's domains (`personas/domains/<domain>.yaml`):

| `teaching.shape` field | Derivation | Default when nothing sets it |
|---|---|---|
| `pace` | the Domain Persona's `teaching.pace`, else the Global Persona's, else the profile's `preferences.teachingPace` | `standard` |
| `activityTypes` | Global then Domain Persona `teaching.preferredActivityTypes`, deduplicated in order | `[]` |
| `avoid`, `emphasis` | Global then Domain Persona `teaching.avoid` / `teaching.emphasis` | `[]` |
| `feedback` | Global then Domain Persona `teaching.feedback`, then the profile's `preferences.feedbackStyle` | `[]` |
| `analogies` | one entry per unit competency: `{domain, competency, basis, concepts, sources}` (below) | `basis: none`, no sources |

**Analogy sources.** A persona's `analogyPolicy.semanticOverrides` are keyed by concept. A concept matches a competency when it names (allowing a plural) a segment of the competency ID after the domain, the leaf of one of its pack `sharedScaffolds`, or a hyphen-separated part of either (`go.runtime.context` has `context`, and `cancellation` through `shared.concurrency.cancellation`). The Domain Persona's overrides come first and shadow the Global Persona's for the same concept; within a layer, concepts are in name order. `sources` are the matching overrides' `prefer` stacks in order, restricted to analogy sources the profile lists: an `experience` key (`typescript`) or a framework under one (`dotnet` under `csharp`). `concepts` names the overrides that contributed a source and `basis` is `semantic-override`. When no override contributes, the default priority applies (`basis: default-priority`): the most specific `analogyPolicy.defaultPriority`, else the profile's stacks by `relativeRank`, again restricted to listed stacks. With no profile, or no listed stack, `basis` is `none`.

**Private persona basis** (`adaptation.persona`): `risks` (every risk of the unit's Global and Domain Personas, each with the unit `concepts` it names, so general risks name none), `analogyReasons` (the persona's `reason` for each applied override and the layer it came from), and `sourceExperience` (the profile stack, `level` and `relativeRank` behind each analogy source).

**Public and private (ADR-0060).** `teaching.shape` holds only analogy stacks and teaching constraints, so it is teaching intent and appears in the Authoring Plan's `public.units[*].teachingIntent`. Learner identity, experience levels and ranks, risks, override reasons and the persona/profile documents never do.

**Missing personas** are not a failure. A missing profile or persona layer is skipped and the defaults in the table apply. `plan` reports `specifications.persona`: `documents` (each path, `domain` for a Domain Persona, `present`, and `contentHash` when present) and `defaults` (each `{domain, field, value, reason}` applied, for example `pace` → `standard`, `analogies` → `none` when the profile lists no stacks, or `analogyPolicy.defaultPriority` → the profile's stacks by rank when no persona sets one). A present document must be schema-valid.

The curriculum specification records the same provenance plus `goal`, `groups`, `units` (each unit spec ID, version, hash, title, group, mode) and the `sequence` of units that are not skipped. Its `authoringIntent` is `target-skeleton` for a target with no content, otherwise the smallest intent any unit justifies.

Specifications contain no platform-native rendering concepts.

### Regeneration policy

Projections are recomputed freely. A new unit specification version is written only when a **material** field changes: competencies (IDs and roles), prerequisites, adaptation mode, misconceptions, required evidence, or the teaching shape (`teachingShape`: `teaching.shape` plus the private `risks`). Anything else (titles, objectives, claims, provenance including persona document hashes, override reasons, experience levels, a new projection revision from evidence that changes nothing material) keeps the stored version, which `plan` reports as `unchanged`. So a persona or profile edit is a material change only when it changes the derived teaching shape; it is not evidence-linked. A version created before teaching shapes has none, so the next `plan` writes a new version with `materialFields: [teachingShape]`. A new version carries `supersedes: {version, contentHash}` and `change: {materialFields, evidenceLinked}`, where `evidenceLinked` is true when the learner state revision changed with it. The curriculum specification is re-versioned only when its goal, groups or referenced unit versions change.

Once a unit is realized (it has a Realization Link, see below), its content exists on the platform, so a material change produces a new version (and so a content change) only when it is **evidence-linked**. A material change to a realized unit without new learner state is **held**: `plan` keeps the stored version and reports `status: held` with the `heldFields` it held back; since a persona or profile edit is not evidence-linked, a teaching shape change alone is held for a realized unit. Unrealized units are re-specified freely.

A realized unit keeps its specification ID. When a proposed unit (`uspec_…` keyed by its competency) is realized as a new unit-level platform item, later plans recognise that item through its Realization Link and keep the proposed unit's ID instead of minting an item-derived one. Until a new version is written, `plan` shows a realized unit at its realized item (`platformItem`, `realized: true`) and, so that realized content is never authored again, a kept version justifies only what the realized unit justifies now (for example `activity`, or nothing); a kept curriculum version likewise justifies what its units justify now, so a realized skeleton is no longer a new target.

## Authoring Plans and Authoring Intent

An Authoring Plan (`authoring-plans/<apl_id>.json`, `schemas/authoring-plan.schema.json`) is an Authoring Intent over specific specification versions, realized by the authoring target skill the adapter declares (`authoringTarget.skill`; PyLearn declares `pylearn-alp-authoring`). `alp platform inspect` reports the same declaration as `authoringTarget: {platform, skill}`, or `null` when the adapter declares no Authoring Target, so an orchestrator resolves the skill from the platform rather than knowing it. Its ID hashes its content, so the same intent over the same versions is the same plan (`authoringPlan.status: existing`).

Intents, largest to smallest: `target-skeleton` → `curriculum` → `unit` → `activity` → `patch`.

- **Default** (no `--intent`): the smallest justified intent (`activity` before `unit`), for the first unit in curriculum order that justifies it. `--unit UNIT` (a `uspec_…` ID or a unit item ID) selects the unit and uses its justified intent. Nothing justified → `authoring.status: nothing-to-author`, no plan.
- **New target** (no items): the default never authors; it reports `explicit-intent-required` with intent `target-skeleton`.
- `--intent target-skeleton`: explicit only, and only for a target with no content (`intent-not-applicable` otherwise). Covers every unit.
- `--intent curriculum` (whole course): explicit only. Covers every unit that justifies authoring.
- `--intent unit|activity|patch [--unit UNIT]`: `unit` applies only to proposed units; `activity` and `patch` only to existing units (`intent-not-applicable` otherwise). Without `--unit`, the first unit that justifies the intent.

`authoring` reports `{status, intent, selection: default|explicit, reason}`.

### Public face

Only `authoringPlan.public` may appear in platform content, branches or PRs (ADR-0060): `intent`, the curriculum citation (`id`, `version`, `contentHash`, `title`, `goal`, `groups` with unit IDs, titles and platform items) and, per unit in scope, its citation, `platformItem`, `group` and `teachingIntent` (exactly the spec's `teaching`, including its persona-derived `shape`). It carries no learner ID, evidence, assessment, status, misconception, adaptation mode, rationale, justification, risk, experience level or persona text.

### What the authoring target skill receives

The orchestration skill hands the platform's authoring skill the Authoring Plan JSON (`authoringPlan` from `plan`, or the recorded file). The skill realizes `public.units[*].teachingIntent` (and, for `target-skeleton`, `public.curriculum.groups`) natively, re-verifies every `claims` entry, and cites spec ID/version/hash in its PR.

The authoring skill (PyLearn: `pylearn-alp-authoring`) must apply `teachingIntent.shape` when it writes content:

- `analogies`: for each competency, draw analogies and transfer contrasts from `sources` in order (the first is the strongest scaffold). With `basis: semantic-override`, `concepts` names what the scaffold is for (for example `context` → `.NET CancellationToken`); contrast where the source's semantics differ, never translate mechanically. With `basis: none`, use no cross-stack analogies.
- `pace` sets density and difficulty (`senior-dense`: no beginner restatement, dense examples; `gentle`: more scaffolding); `activityTypes` picks the exercise forms (prefer the earlier ones); `avoid` lists forms and framings the content must not use; `emphasis` lists themes the content must foreground; `feedback` sets the voice of hints, explanations and grading feedback.

Shared content is authored once for every learner, so the shape biases choices (examples, analogies, difficulty, exercise mix) without per-learner variants. It may read the full unit specifications in the learner workspace (for example `adaptation.misconceptions` and `adaptation.mode` to shape difficulty and examples, or `adaptation.persona.risks` to design exercises that surface them) but must never copy them, or anything else outside `public`, into content or PRs.

## Platform Gate Results and Realization Links

```
alp platform gates record --adapter pylearn --target go-alp --curriculum curriculum-export.json \
  --plan apl_… --result gate-result.json [--realization realization.json] [--workspace PATH]
```

`gates record` requires the Platform Validator, Authoring Target and Curriculum Reader capabilities. It attaches a Platform Gate Result (`schemas/platform-gate-result.schema.json`) to a recorded Authoring Plan and decides, for every unit in the plan's scope, whether that unit specification version is **realized** (ADR-0057, Q22, Q27).

**Inputs.**

- `--result`: the platform validator's own JSON, recorded verbatim. PyLearn's is the stdout of `bun run validate:platform --target <id>` (it composes `compile:go`, `gate:reels --json`, `lint:lessons --json`, `typecheck` and `gate:alp-mapping --json` and decides `publishable`; it exits 1 when not publishable, which still prints a result to record). The result must be schema-valid, for the adapter's platform and `--target`, with unique gate IDs (`invalid-gate-result` otherwise). ALP never interprets gate commands, artifacts or diagnostics.
- `--curriculum`: the curriculum export **after** authoring (for PyLearn, `export:curriculum --course <id>` on the authoring branch), the only source of which items are declared-stable.
- `--realization` (`schemas/platform-realization-report.schema.json`): what the authoring target skill realized. Optional; without it no unit can be realized.

```json
{
  "schemaVersion": 1,
  "plan": "apl_…",
  "units": [
    {
      "unit": "uspec_…",
      "items": ["go-alp-s1-scheduler", "go-alp-s1-scheduler#quiz:preemption"],
      "claims": [
        {"domain": "go", "competency": "go.runtime.scheduler",
         "sources": [{"url": "https://go.dev/doc/go1.27", "version": "go1.27", "verifiedAt": "2026-10-07"}]}
      ]
    }
  ]
}
```

`items` are the target's opaque item IDs. They must all be items of the curriculum export (so declared-stable; heading slugs and positional scene IDs never are), exactly one must be a unit-level item (placed in a phase) and the rest must descend from it. An existing unit must be realized by its own platform item, a realized unit keeps its unit-level item, and an item realizes at most one unit specification (`invalid-realization-link`, `realization-conflict`). `claims` give the source provenance of the unit's `teaching.claims` (Q27): every claim needs at least one source with a `url` and `verifiedAt` date (`YYYY-MM-DD`); a version-sensitive claim's sources must also name the `version` verified. A report for another plan, a unit outside the plan, or a claim the unit does not declare is refused (`invalid-realization-report`). A refused input records nothing.

**Realization rule.** A unit is `realized` only when the result is `publishable`, the report lists its items, and every claim has the required provenance. Otherwise it is `unrealized` with every reason: `not-publishable` (naming the failed and skipped gates), `realization-not-reported`, or `provenance-missing` (per claim). Recording an unpublishable result is not an error: the record is the audit trail.

**Records** (canonical, append-only, one Store transaction):

- `authoring-plans/<apl_id>/gate-results/<pgr_id>.json` (`schemas/platform-gate-record.schema.json`): `learnerId`, `target`, `plan {id, intent}`, `recordedAt`, the verbatim `result`, and per unit `{unit {id, version, contentHash}, status, reasons, realization}`. Its ID hashes the plan, result and report, so recording the same inputs again reports `status: already-recorded` and writes nothing.
- `specifications/realizations/<rlz_id>.json` (`schemas/realization-link.schema.json`), one per realized unit version: the Realization Link with `unit`, `plan`, `gateRecord`, `root` (the unit-level item), every realizing `items` entry as a namespaced `{platform, target, item}`, and the unit's `claims` with their `sources`. Specification versions stay immutable; the link is the record on the specification of what realized it.

Output: `{schemaVersion, adapter, target, status: recorded|already-recorded, path, record, realizations}` (each realization with its `path`).

## Skills

Two portable skills drive this workflow through the `alp platform` JSON contracts only (ADR-0012, ADR-0056, Q24):

- `skills/adapt-platform-target`: inspect → plan → present each unit's `proposedMode` with its rationale and the persona-derived teaching shape → `decision accept` only for what the learner explicitly confirmed (`--confirm` with the current `--basis`) → `decision revoke` on request.
- `skills/orchestrate-platform-authoring`: take an Authoring Plan, resolve the authoring target skill from `inspect`'s `authoringTarget.skill`, hand it the plan, `gates record` the platform validator's result with the realization report, and report each unit realized or unrealized with its reasons; plus the activity loop: `account link` (learner-confirmed) → `import` (the caller keeps `cursor.next`) → re-plan.

`internal/cli` tests parse every `alp platform` invocation in `skills/*/SKILL.md` and fail when a skill names a command, flag or enumerated value the CLI does not take, or shows a decision or account link without `--confirm`.
