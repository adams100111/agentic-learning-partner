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

A boundary that translates between an external learning platform and ALP. A platform adapter normalizes learner/activity signals and adaptation outputs without making the platform ALP's technical source of truth.

## PyLearn

ALP's first integration and validation environment. PyLearn is a source of learner activity, project evidence, and candidate course material; it is not an authoritative source for technical correctness.

## Context Bundle

The structured output produced by ALP's context builder for a specific agent task. It records included sources, relevant learner state, and optional token-cost metadata.

## Adaptation Proposal

An evidence-linked proposal to alter learning content or sequencing. It is not an automatic content mutation.

## Canonical State

Git-synchronized learner/profile/persona/evidence data from which derived views can be rebuilt. Generated Markdown/HTML views are not canonical state.
