# Target adaptation: `alp platform plan` and Accepted Adaptation Decisions

ALP adapts a shared Learning Target (for example PyLearn `go-alp`) to one learner without forking the target or the learner's competency state (ADR-0055, ADR-0060, ADR-0061).

## Target Adaptation Projection

```
alp platform plan --adapter pylearn --target go-alp \
  --curriculum curriculum-export.json --mapping go-alp.mapping.yaml \
  [--constraints go-alp.constraints.yaml] [--workspace PATH]
```

`plan` requires the Curriculum Reader and Content Mapper capabilities and refuses an invalid mapping. It writes the learner's projection to `state/target-adaptations/<id>.json` (`schemas/target-adaptation-projection.schema.json`) and prints it as deterministic JSON (`{schemaVersion, adapter, target, path, projection}`).

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
```

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

## Extension point

Curriculum and Learning Unit Specifications and Authoring Plans (spec #66, ticket #72) are layered on `plan`: they are derived from the projection and its revision, and will add their own output fields and Authoring Target capability requirements.
