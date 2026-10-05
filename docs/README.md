# Documentation Index

Read in this order:

1. [Product](PRODUCT.md) — what ALP is and is not.
2. [Architecture](ARCHITECTURE.md) — component boundaries.
3. [Personas](PERSONAS.md) — global/domain persona model and conversational updates.
4. [Persona Wizard](PERSONA_WIZARD.md) — adaptive discovery / grill-with-docs workflow.
5. [State Model](STATE_MODEL.md) — profile, evidence, projections, mutation rules.
6. [Representation](REPRESENTATION.md) — YAML/JSON/Markdown/HTML and token-efficiency rules.
7. [Roadmap](ROADMAP.md) — delivery sequence.
8. [Source Policy](SOURCE_POLICY.md) — authority, provenance, and freshness rules.
9. [PyLearn Integration](integrations/PYLEARN.md) — first real use case.
10. [Domain Packs](DOMAIN_PACKS.md) — Go now, Rust later.
11. [ADRs](adr/README.md) — decisions and rationale.

Implementation-specific specifications should be added under `docs/specs/` before substantial features are built.

## Documentation ownership

- Product invariants belong in `PRODUCT.md`.
- Cross-cutting architecture belongs in `ARCHITECTURE.md`.
- Decisions that close meaningful alternatives require an ADR.
- Domain-specific teaching policy belongs under the domain pack, not in core docs.
- Platform-specific behavior belongs under `docs/integrations/`.
- Generated learner state never belongs in architecture docs.

When documents disagree: accepted ADRs define the decision, then product invariants, then architecture, then feature specs.
