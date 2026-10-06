# ADR-0057: Reference platform things through opaque, declared-stable external identities

**Status:** Accepted

## Context

Platform adaptation needs identities for platforms, Learning Targets, curricula, units, and platform content/activity items. Today only competency IDs have a defined format. Platform IDs are free strings, content IDs are opaque `contentId`s, and Learning Target/Curriculum/Learning Unit have no identity at all.

PyLearn shows that not every platform identifier is stable: lesson front-matter `id`, explicit Scene `id`, and quiz/question IDs are stable by convention, but section IDs are slugified headings and unnamed scenes get positional IDs (`scene-N`). Mapping or evidence keyed to those would silently break when an author edits a heading or reorders scenes.

## Decision

Identity ownership follows who creates the thing:

- **Platform-owned, opaque:** platform instance, Learning Target, and content/activity item IDs. ALP stores them as namespaced tuples (`{platform, target, item}`) and never derives meaning from the strings.
- **ALP-owned:** Curriculum Specification and Learning Unit Specification IDs, because ALP generates those artifacts.

A realized unit specification records the platform item IDs it was realized as; this is the bidirectional link evidence flows back through.

Only **declared-stable** platform identifiers may appear in mappings, realization links, or evidence provenance. Each adapter's validator defines which of its identifiers are stable and rejects references to unstable ones (for PyLearn: no section slugs, no positional scene IDs).

The competency ID domain-prefix check remains the only place ALP interprets an identifier's structure.

## Consequences

- platform authoring renames become explicit migrations instead of silent evidence orphaning;
- platforms may need to add explicit IDs where they currently rely on derived ones;
- ALP gains schema fields for target and spec identity.

## Alternatives considered

- ALP-minted IDs for everything: requires ALP to track platform-internal objects it does not own.
- Accept any platform ID: fragile under ordinary authoring edits.
