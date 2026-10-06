# ADR-0056: Generate platform-neutral learning specifications and realize them through platform-native authoring

**Status:** Accepted

## Context

ALP needs to support adaptive content generation without becoming a presentation engine or inventing a lowest-common-denominator lesson format.

PyLearn already has a rich Reel MDX engine with native components, runtime behavior, narration/media contracts, course navigation, and quality gates.

Embedding Reel/Scene/MDX concepts in ALP core would couple ALP to PyLearn. Generating generic Markdown first would discard capabilities and create unnecessary translation.

## Decision

ALP produces platform-neutral:

- Curriculum Specifications;
- Learning Unit Specifications;
- Target Adaptation Projections;
- Authoring Plans.

A platform authoring capability realizes those specifications using the platform's native representation.

For PyLearn, the authoring target is the **Reel MDX engine and its actual component/runtime contract**.

ALP core does not directly mutate production platform content. Normal write-side execution is:

1. derive adaptation/specification from learner state;
2. create an Authoring Plan;
3. realize it in a platform branch/worktree;
4. run platform-native quality gates;
5. review through the platform repository's normal PR workflow.

A capable agent/adapter may automate this workflow, but it still respects the platform's repository and validation boundaries.

## Consequences

- ALP remains platform-independent;
- PyLearn gets first-class use of Reel MDX rather than generic Markdown;
- future platforms can realize the same learning intent differently;
- platform quality gates remain authoritative for platform content correctness;
- content generation and learner-state mutation remain separate concerns.
