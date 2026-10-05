# Schemas

These JSON Schemas are executable contracts for canonical YAML/JSON data.

Current v1 contracts:

- `profile.schema.json`
- `persona.schema.json`
- `evidence.schema.json`
- `context-bundle.schema.json`
- `competency-taxonomy.schema.json`
- `adaptation-proposal.schema.json`

Canonical learner files may be YAML; validation operates on the parsed data model.

Schema changes follow semantic intent:

- additive compatible fields may evolve within the schema version while pre-1.0;
- breaking persisted-state changes require an explicit migration plan;
- once v1 storage is used by released tooling, breaking format changes require a new schema version.

Schemas define structure, not teaching truth. Domain rubrics and deterministic projection rules define semantics.
