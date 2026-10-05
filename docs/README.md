# Documentation Index

Read in this order:

1. [Product](PRODUCT.md) — what ALP is and is not.
2. [Architecture](ARCHITECTURE.md) — component boundaries.
3. [State Model](STATE_MODEL.md) — profile, evidence, projections, mutation rules.
4. [Roadmap](ROADMAP.md) — delivery sequence.
5. [Source Policy](SOURCE_POLICY.md) — authority, provenance, and freshness rules.
6. [PyLearn Integration](integrations/PYLEARN.md) — first real use case.
7. [Domain Packs](DOMAIN_PACKS.md) — Go now, Rust later.
8. [ADRs](adr/README.md) — decisions and rationale.

Implementation-specific specifications should be added under `docs/specs/` before substantial features are built.

## Documentation ownership

- Product invariants belong in `PRODUCT.md`.
- Cross-cutting architecture belongs in `ARCHITECTURE.md`.
- Decisions that close meaningful alternatives require an ADR.
- Domain-specific teaching policy belongs under the domain pack, not in core docs.
- Platform-specific behavior belongs under `docs/integrations/`.
- Generated learner state never belongs in architecture docs.

When documents disagree: accepted ADRs define the decision, then product invariants, then architecture, then feature specs.
