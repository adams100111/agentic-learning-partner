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
- `platform-gate-result.schema.json`: a Platform Validator's per-gate results (status, provenance, artifacts, diagnostics) and overall `publishable` flag. An example lives in `testdata/`.

Canonical learner files may be YAML; validation operates on the parsed data model.

Schema changes follow semantic intent:

- additive compatible fields may evolve within the schema version while pre-1.0;
- breaking persisted-state changes require an explicit migration plan;
- once v1 storage is used by released tooling, breaking format changes require a new schema version.

Schemas define structure, not teaching truth. Domain rubrics and deterministic projection rules define semantics.
