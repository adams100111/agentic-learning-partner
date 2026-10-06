# Glossary

Canonical domain language for Agentic Learning Partner (ALP).

## Agentic Learning Partner (ALP)

The overall product: a portable learning control plane for AI agents. ALP owns learner modeling, evidence, adaptive planning, context generation, and domain learning workflows. It is not an LMS or a course repository.

## Learner Profile

Durable factual context about the learner: experience, goals, roles, and stable preferences. Profile facts are distinct from demonstrated competency.

## Persona

Structured teaching configuration derived from durable learner context and explicit preferences. Persona controls how an agent should teach; it does not itself prove competency.

## Global Persona

Cross-domain persona configuration shared by all domain packs.

## Domain Persona

A domain-specific overlay on the global persona, such as Go-specific analogy or teaching preferences. It must not duplicate the full learner identity.

## Effective Context

A task-specific, generated projection of the minimum learner/persona/state information an agent needs for the current job. It is derived, ephemeral, and non-authoritative.

## Evidence

A durable observation of learner behavior or output, with provenance. Examples include diagnostic answers, exercise attempts, project code, quiz answers, and learner self-report.

## Competency

A stable domain capability identified by a domain-scoped ID, such as `go.runtime.context`. Competency identity is independent of any course or platform lesson ID.

## Competency Projection

The current derived view of a learner's competency level, confidence, freshness, and gaps. It is rebuildable from canonical inputs and is not itself evidence.

## Domain Pack

A subject-specific package of competency taxonomy, diagnostic/teaching rubrics, transfer mappings, references, freshness policy, and production criteria. Go is the first domain pack; Rust is planned later.

## Platform Adapter

The umbrella integration boundary between ALP and an external learning platform or delivery environment. A Platform Adapter is capability-oriented rather than one mandatory bidirectional interface. An integration may expose activity ingestion, curriculum inspection, content mapping, authoring-target realization, and platform validation independently.

## Platform Capability

A concrete capability exposed by a Platform Adapter. Initial capability families are activity sourcing/normalization, curriculum reading, content-to-competency mapping, authoring-target realization, and platform validation. Integrations implement only capabilities they can support honestly.

## Learning Target

A platform-independent destination for a learner-facing learning experience. A target may be a course, track, workshop, lab series, project path, or another structured delivery surface. A platform-specific course ID is one implementation of a Learning Target, not a core ALP identity. Targets are shared and learner-free; per-learner variation lives in Target Adaptation Projections and Accepted Adaptation Decisions (ADR-0061).

## Target Adaptation Projection

A derived, rebuildable projection of learner/domain state for one Learning Target. It may contain sequencing, skip/skim/challenge/full decisions, reinforcement needs, uncovered competencies, and authoring requirements. It is not evidence and does not own competency truth.

## Curriculum Specification

A platform-neutral specification of the shape, coverage, dependencies, sequencing, and evidence expectations for a Learning Target. It is derived from learner/domain state and target constraints.

## Learning Unit Specification

A platform-neutral specification for one learner-facing unit. It describes competencies, objectives, prior-knowledge assumptions, adaptation mode, misconceptions, evidence requirements, analogies/transfer constraints, freshness/source requirements, dependencies, and done bars. It deliberately excludes platform-native rendering concepts such as PyLearn Reel MDX components.

## Authoring Plan

A validated proposal for realizing one or more Curriculum/Learning Unit Specifications in a target platform. It may be executed by an agent or adapter through a branch/worktree/PR workflow, but ALP core does not silently mutate platform production content.

## External Identity

An opaque, platform-owned identifier ALP references in a namespaced form (`{platform, target, item}`), such as a platform instance, Learning Target, or content/activity item. ALP never derives meaning from external identity strings. Curriculum/Learning Unit Specification IDs are ALP-owned instead. See ADR-0057.

## Declared-Stable Identifier

A platform identifier the adapter declares stable across ordinary authoring edits. Only declared-stable identifiers may appear in mappings, realization links, or evidence provenance. For PyLearn: lesson `id`, explicit Scene `id`, quiz/question IDs — not heading-derived section slugs or positional scene IDs.

## Mapping Role

The relationship a mapped content item has to a competency: `teaches`, `reinforces`, or `assesses`. Only `assesses` activity can yield assessment-grade evidence. See ADR-0058.

## Accepted Adaptation Decision

A canonical, append-only record of an explicit adaptation choice (for example, an accepted skip) that must survive Target Adaptation Projection rebuilds. It is an input to the projection, not part of it, and requires explicit learner confirmation; agents may only propose. See ADR-0060, ADR-0061.

## Synthetic Event Identity

An adapter-derived activity identity (row key plus content hash) for platforms that store only latest-state rows rather than event logs. It makes evidence import idempotent without treating a snapshot as many events. See ADR-0059.

## Realization Link

The record on a Learning Unit Specification of which declared-stable platform items realized it. It is the path by which platform activity is traced back to the specification that motivated the content.

## Platform Gate Result

Structured output of a Platform Validator: per-gate ID, status (pass/fail/warn/skipped), platform-native command/version provenance, checked artifacts, item-referenced diagnostics, and an overall `publishable` flag. ALP consumes it without interpreting platform-native formats.

## Authoring Intent

The hierarchical scope of an authoring request: target skeleton → curriculum → unit → activity → patch. The default is the smallest justified intent.

## Platform Account Link

A learner-confirmed workspace record linking a platform user (`{platform instance, platform user ID}`) to the workspace learner. Activity imports are refused for platform users without a link; learners are never matched by email or name.

## Curriculum Export

A versioned, platform-produced description of a Learning Target's structure — declared-stable items, phases, mapping, and content hash — consumed by a Curriculum Reader. ALP never parses platform-native content directly.

## PyLearn

ALP's primary reference integration and first validation environment, but not the only platform target. PyLearn is a source of learner activity, project evidence, and candidate course material; it is not an authoritative source for technical correctness. Its Reel MDX system is the first reference implementation of a platform-native authoring target.

## Context Bundle

The structured output produced by ALP's context builder for a specific agent task. It records included sources, relevant learner state, and optional token-cost metadata.

## Adaptation Proposal

An evidence-linked proposal to alter sequencing, reinforcement, or content for a Learning Target. It does not mutate learner truth. When content changes are justified, it can lead to a Target Adaptation Projection and Authoring Plan rather than direct core-owned platform mutation.

## Canonical State

Git-synchronized learner/profile/persona/evidence data from which derived views can be rebuilt. Generated Markdown/HTML views are not canonical state.


## Learner Workspace

A separate private workspace/repository containing one learner's profile, personas, evidence, assessments, projections, review state, and sessions. It has an independent lifecycle from the reusable ALP engine.

## Assessment

A durable semantic judgment about what one or more evidence records demonstrate for a competency under a specific rubric version. Assessments sit between evidence and deterministic competency projections.

## Workspace Revision

The canonical Git revision/commit that an ALP state mutation was computed against. It supports optimistic concurrency across multiple agents.


## Store

The persistence boundary through which ALP reads and mutates one learner workspace. Store semantics are provider-independent; provider-specific transport or synchronization behavior does not belong in the core learning model.

## Store Provider

An implementation of ALP workspace persistence capabilities. Providers may support different capabilities such as revisions, history, synchronization, offline access, or atomic checkpoints.

## Git Store

The production Git-backed Store Provider. A Git Store uses a local working tree as the runtime workspace, Git commits as durable revisions/checkpoints, and an ordinary Git remote for multi-device synchronization.

## Local Store

A local-filesystem Store Provider with no remote synchronization requirement. It uses the same learner-state contracts as other providers and is suitable for offline/private/local-only use.

## Sync

The provider-mediated process that reconciles local and remote learner workspace state while preserving ALP canonical-state invariants and rebuilding derived state when needed.

## Checkpoint

A durable Store revision representing one logical ALP state transaction, typically a completed learning/session mutation set rather than each individual record append.

## Production v0

The first production-ready ALP release line. Pre-1.0 denotes evolving compatibility guarantees, not reduced implementation quality or an MVP/prototype standard.
