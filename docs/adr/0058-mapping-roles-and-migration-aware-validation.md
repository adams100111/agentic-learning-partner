# ADR-0058: Content mappings declare competency roles and validate through pack migrations

**Status:** Accepted

## Context

The v1 platform mapping is a flat `competencies[]` list per `contentId` with `additionalProperties:false`. It cannot say whether content teaches, reinforces, or legitimately assesses a competency, so any activity becomes evidence of equal semantic weight. Evidence strength is hard-coded per signal kind in the PyLearn adapter.

`ValidateMapping` fails hard on any unknown competency and does not call `Pack.ResolveCompetency`, although domain packs already declare rename/merge/split migrations.

PyLearn's only existing content→concept link is the free-text `concept="..."` prop on quizzes, which doubles as quiz ID and `concept_mastery` key.

## Decision

- Platform mapping schema v2: each entry maps a declared-stable item ID (ADR-0057) to competencies, each with a **role**: `teaches`, `reinforces`, or `assesses`.
- Only activity on an `assesses` entry may yield assessment-grade evidence; `teaches`/`reinforces` activity yields exposure/practice evidence.
- Static mappings do not set evidence strength. Strength derives from the adapter's signal-kind policy plus the observed result. A mapping may declare a strength **ceiling**.
- Compatibility stays a semver `packVersion` range. Validation resolves IDs through pack migrations first: rename → pass with an update warning; split/merge → fail with a specific migration message; removed or out-of-range → fail. Nothing is silently dropped.
- Mappings remain owned by the platform/content repository (ADR-0017). Platform-local concept tags (e.g. PyLearn `concept`, `concept_mastery`) are not ALP competency IDs and are not the mapping.

## Consequences

- schema version bump and adapter changes;
- PyLearn needs a new mapping file rather than reusing quiz concept strings;
- domain pack upgrades with renames no longer break platform mappings.

## Alternatives considered

- Static evidence strength per mapping: ignores task design and observed outcome.
- Reusing PyLearn concept tags: 142 near-unique free-text strings with no registry.
