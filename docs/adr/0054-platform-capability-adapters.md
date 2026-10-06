# ADR-0054: Model platform integration as capability-oriented adapters

**Status:** Accepted

## Context

ALP must integrate with rich learning platforms without making any one platform's content/runtime model part of the learning core.

PyLearn is the primary reference integration and currently needs more than read-side telemetry normalization: curriculum inspection, content mapping, platform-native authoring, and platform validation are all relevant. Other future platforms may support only a subset.

A single giant bidirectional Platform Adapter would either force fake capabilities or leak platform-specific concepts into ALP core.

## Decision

Keep **Platform Adapter** as the umbrella term and model platform integration through independent **Platform Capabilities**.

Initial capability families are:

- activity source / evidence normalization;
- curriculum reader;
- content-to-competency mapping;
- authoring-target realization;
- platform validation / quality gates.

An integration implements only the capabilities it actually supports.

ALP core owns learner modeling, evidence interpretation, competency assessment/projection, adaptive planning, and platform-neutral learning intent.

The platform owns presentation/runtime/content representation and its own application/activity records.

PyLearn is the first reference implementation of this contract, not a special case in ALP core.

## Consequences

- ALP remains portable beyond PyLearn;
- read-only integrations remain valid;
- authoring-capable platforms can expose richer capabilities without changing learner semantics;
- PyLearn Reel/Scene/MDX concepts stay outside ALP core;
- capability mismatch is explicit rather than silently emulated.
