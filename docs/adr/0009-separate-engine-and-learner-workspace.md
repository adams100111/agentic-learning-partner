# ADR-0009: Separate ALP engine from learner workspace

**Status:** Accepted

## Context

The reusable plugin/domain-pack code and one learner's private profile/evidence/state have different ownership, privacy, release, and synchronization lifecycles.

Keeping them in one repository would make plugin installation/sharing carry learner data and would couple state history to engine releases.

## Decision

Use separate repositories/workspaces:

- **Agentic Learning Partner** owns reusable plugin code, schemas, domain packs, skills, adapters, and documentation.
- A **learner workspace** owns one learner's profile, personas, evidence, assessments, projections, review state, and sessions.
- PyLearn and other learning platforms remain separate integration repositories.

The initial learner workspace is the private `adams100111/agentic-learning-state` repository.

ALP operates against a configured workspace rather than assuming learner state is inside the plugin repository.

## Consequences

- learner data can remain private while ALP can later be shared/published;
- engine and learner state evolve independently;
- schema/domain compatibility must be declared by the workspace;
- install/config UX must locate a workspace;
- fixtures/examples in ALP must be synthetic, never copied personal state.
