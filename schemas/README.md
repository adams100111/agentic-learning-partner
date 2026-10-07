# Schemas

These JSON Schemas are executable contracts for canonical YAML/JSON data.

Current v1 contracts:

- `profile.schema.json`
- `persona.schema.json`
- `evidence.schema.json`
- `assessment.schema.json`
- `context-bundle.schema.json`
- `competency-taxonomy.schema.json`
- `adaptation-proposal.schema.json`

Platform adaptation contracts (spec #66, ADR-0054, ADR-0057):

- `platform-curriculum-export.schema.json`: a platform's versioned description of its Learning Targets (declared-stable items, phases, mapping reference, content hash), read by `alp platform inspect --curriculum FILE` (Q37). PyLearn emits it with `export:curriculum`.
- `platform-mapping.schema.json`: platform mapping v2 (ADR-0058). A platform-owned mapping from one Learning Target's declared-stable items to competencies with Mapping Roles, an optional strength ceiling, and semver pack ranges; validated by `alp platform mapping validate` (see `docs/PLATFORM_MAPPING.md`). Schema version 1 is rejected, not migrated.
- `pylearn-export.schema.json`: `pylearn-export` v2 (ADR-0059), the activity PyLearn emits with `export:activity` for one platform user since a cursor: records grouped by `targets` (replacing v1 `courses`), each with a declared-stable item and a Synthetic Event Identity. Read by `alp platform import --adapter pylearn`; v1 is rejected. See `docs/integrations/PYLEARN_EXPORT.md`.
- `platform-account-link.schema.json`: the learner-confirmed workspace record (`platform-accounts/<id>.yaml`) linking `{platform, instance, platform user ID}` to the workspace `learnerId` (Q36), written by `alp platform account link --confirm`. Import refuses unlinked platform users.
- `target-constraints.schema.json`: platform-neutral goal of and limits on adapting one Learning Target (allowed adaptation modes; units that may never be skipped; goal competencies that specifications propose units for), an optional `--constraints` input of `alp platform plan`.
- `target-adaptation-projection.schema.json`: the generated, rebuildable Target Adaptation Projection (`state/target-adaptations/<id>.json`, ADR-0060, ADR-0061) written by `alp platform plan`: one learner's per-unit skip/challenge/skim/full modes, sequence, reinforcement and prerequisite gaps over a shared target, with a content-hash `revision`. See `docs/TARGET_ADAPTATION.md`.
- `accepted-adaptation-decision.schema.json`: the canonical, append-only Accepted Adaptation Decision (`adaptation-decisions/<id>.yaml`) written only with the learner's confirmation by `alp platform decision accept|revoke --confirm`; it records the confirmer, time and basis projection revision, and is revoked by a superseding record.
- `learning-unit-specification.schema.json`: one immutable version (`specifications/units/<id>-v<n>.json`) of a platform-neutral Learning Unit Specification written by `alp platform plan` (ADR-0056, ADR-0060): provenance (learner state revision, projection revision, target snapshot, mapping/pack versions, pack sources), the justified Authoring Intent, learner-free `teaching` intent (competencies with roles, objectives, prerequisites, dependencies, required evidence, version-sensitive/source-required claims) and the private `adaptation` basis (mode, statuses, misconceptions). See `docs/TARGET_ADAPTATION.md`.
- `curriculum-specification.schema.json`: one immutable version (`specifications/curricula/<id>-v<n>.json`) of a learner's Curriculum Specification for a shared target: goal, groups and the unit specification versions in sequence order.
- `authoring-plan.schema.json`: an immutable, content-addressed Authoring Plan (`authoring-plans/<id>.json`): an Authoring Intent over specific specification versions, the platform-declared authoring target skill, and the learner-free `public` face (spec IDs/versions/hashes and teaching intent) that is the only part allowed in platform content or PRs.
- `platform-gate-result.schema.json`: a Platform Validator's per-gate results (status, provenance, artifacts, diagnostics) and overall `publishable` flag. An example lives in `testdata/`.

Canonical learner files may be YAML; validation operates on the parsed data model.

Schema changes follow semantic intent:

- additive compatible fields may evolve within the schema version while pre-1.0;
- breaking persisted-state changes require an explicit migration plan;
- once v1 storage is used by released tooling, breaking format changes require a new schema version.

Schemas define structure, not teaching truth. Domain rubrics and deterministic projection rules define semantics.
