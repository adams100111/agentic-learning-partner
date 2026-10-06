# Handoff — Platform Adaptation / PyLearn Reference Integration Grill

## Purpose

This file is the complete handoff for a **brand-new agent/session**.

Do not rely on prior conversation memory.

The next agent should resume the actual Matt Pocock `grill-with-docs` workflow from the settled state below.

## Repositories

Primary engine:

```text
adams100111/agentic-learning-partner
```

Primary reference platform:

```text
adams100111/pylearn
```

ALP is intentionally not PyLearn-specific.

PyLearn is the primary use case/reference integration, not the only target.

## Working branch

Documentation-only grill work is on:

```text
docs/platform-adaptation-grill
```

No platform-adaptation implementation should be started until the grill is complete and the resulting spec/tickets are approved.

## Workflow requirement

Use the real Matt Pocock skills/workflow:

1. `grill-with-docs`
2. `grilling`
3. `domain-modeling`
4. after grill closes: `to-spec`
5. then `to-tickets`
6. implementation only after ticket graph approval via `implement-spec`

Do **not** automatically run retro.

### Grill behavior

- facts are the agent's job: inspect both repositories instead of asking the user;
- decisions are the user's job;
- ask every currently unblocked decision in a frontier round;
- number questions;
- include a recommended answer for every question;
- when the user says `use your recs` / agrees, commit settled vocabulary/docs/ADRs and recompute the frontier;
- create ADRs only for hard-to-reverse/surprising/tradeoff decisions;
- keep the glossary current.

## Existing production-v0 status

ALP production-v0 Store/session/onboarding candidate is already implemented on `main`.

Do not reopen that work unless a platform-adaptation requirement demonstrates a real incompatibility.

Important existing capabilities:

- Local Store;
- Git Store;
- opaque revisions/capabilities;
- staged transactions;
- session/manual/eager policies;
- resumable sessions;
- semantic multi-device sync;
- workspace migration;
- export/verify/restore;
- provider conversion;
- portable harness skills;
- Go domain pack;
- diagnostics/planning/evidence/assessment/projection.

GitHub Actions are intentionally disabled because the user's quota is exhausted.

Do not claim the full Go suite or real Claude/Codex smoke passed unless actually executed.

## Core premise established before grill

The user clarified:

- ALP targets many future learning platforms/environments;
- PyLearn is the primary reference/use case;
- PyLearn content generation should target the real Reel MDX engine/components;
- learner competency state must not fragment per course.

## Settled grill round 1

All Q1–Q12 below were explicitly accepted by the user.

### Q1 — ALP/platform boundary

**Decision:** ALP remains the platform-agnostic learning control plane.

ALP owns:

- learner truth;
- evidence/assessment semantics;
- competency projections;
- reinforcement/review state;
- adaptive planning;
- platform-neutral curriculum/unit intent.

Platforms own:

- presentation;
- runtime;
- native content representation;
- platform application/activity records;
- platform quality gates.

### Q2 — Platform Adapter shape

**Decision:** retain Platform Adapter as the umbrella term but make it capability-oriented rather than one giant bidirectional interface.

Initial conceptual capability families:

- Activity Source;
- Curriculum Reader;
- Content Mapper;
- Authoring Target;
- Platform Validator.

Exact interface names/shapes remain open.

### Q3 — Competency identity scope

**Decision:** competency truth is one **learner × domain competency state**, not learner × platform × course.

`go.concurrency.channels` has one current projection regardless of where evidence came from.

Course/platform/target belong in provenance and adaptation context.

### Q4 — Course-specific adaptation

**Decision:** introduce a separate derived **Target Adaptation Projection** for one Learning Target.

It may describe:

- skip / skim / challenge / full;
- sequencing;
- reinforcement;
- coverage/gaps;
- authoring requirements.

It is not evidence or competency truth.

### Q5 — Generic authoring output

**Decision:** ALP produces platform-neutral Curriculum / Learning Unit Specifications.

These describe learning intent, not render syntax.

A unit specification can include:

- competencies;
- objective;
- prior knowledge assumptions;
- adaptation mode;
- required evidence;
- misconceptions;
- reinforcement;
- analogies/transfer constraints;
- freshness/source requirements;
- dependencies;
- done bar.

### Q6 — PyLearn authoring target

**Decision:** PyLearn generation targets the actual **Reel MDX engine and native component/runtime contract**.

Do not invent an ALP Markdown lesson format.

### Q7 — Content-to-competency mapping ownership

**Decision:** ADR-0017 remains correct.

The platform/content repository owns content/activity → ALP competency mappings.

ALP owns competency IDs and validation.

Mapping should eventually distinguish what content:

- teaches/reinforces;
- legitimately assesses/evidences.

Detailed schema is not yet decided.

### Q8 — Mutation boundary

**Decision:** ALP core does not silently mutate production platform content.

Write flow:

```text
adaptation
→ specification
→ Authoring Plan
→ platform branch/worktree
→ native content changes
→ platform gates
→ review/PR
```

Agents/adapters may automate the workflow while respecting platform policies.

### Q9 — PyLearn special casing

**Decision:** no PyLearn concepts in ALP core.

PyLearn is the reference/acceptance implementation.

Design test:

> If the generic contract cannot express PyLearn's Reel system cleanly, it is too weak. If ALP core contains Reel/Scene/MDX concepts, it is too PyLearn-specific.

### Q10 — First smoke target

**Decision:** do not overwrite the existing PyLearn `go` course.

Use a fresh experimental target, provisionally `go-alp`.

This creates a useful comparison:

```text
go
= manually designed pre-ALP curriculum

go-alp
= ALP-driven curriculum from canonical learner state
```

Exact target ID/naming still may be refined.

### Q11 — Whole-course vs progressive generation

**Decision:** ALP may derive/project the complete curriculum graph, but platform-native executable content should be authored progressively.

Do not eagerly generate every polished Reel if later learner evidence may change them.

### Q12 — Closed-loop invariant

**Decision:** a complete integration must close both directions:

```text
Learner State
→ Target Adaptation Projection
→ Curriculum / Learning Unit Specifications
→ Platform-native authored content
→ Learner activity
→ Platform export
→ ALP Evidence → Assessment → Projection
→ justified target adaptation
```

Read-only import or one-way generation alone is not a complete integration.

## ADRs created from round 1

On `docs/platform-adaptation-grill`:

- `docs/adr/0054-platform-capability-adapters.md`
- `docs/adr/0055-domain-scoped-competency-target-adaptation.md`
- `docs/adr/0056-platform-native-authoring-targets.md`

Glossary was also updated.

## Verified current ALP/PyLearn gaps

These are facts from repo inspection, not design guesses.

### ALP adapter reachability

PyLearn adapter implementation exists internally, including tests.

However current `alp` CLI does not expose a deterministic platform/PyLearn import/analyze command.

This is an implementation gap to design after the grill.

### Concrete platform mapping

ALP has:

- mapping schema/docs;
- ADR-0017;
- adapter validation logic.

No concrete PyLearn content/activity → Go competency mapping was found in the PyLearn repository.

This is required for closed-loop evidence.

### Write-side authoring

Current PyLearn integration docs explicitly remain adaptation-proposal/read-first.

No production generic authoring capability + PyLearn reference implementation exists yet.

### PyLearn course registration hard-coding

PyLearn's schema/routing is multi-course capable, but current sync/config code explicitly knows only the existing Python/default and Go courses.

Verified examples:

- `apps/web/lib/courses/config.ts` contains explicit `pylearn` and `go` configs;
- `apps/web/lib/courses/sync.ts` has `KNOWN_COURSES = new Set([DEFAULT_COURSE_ID, "go"])`;
- sync seeds only those rows;
- unknown Reel course tags currently fall back to the default course.

A fresh `go-alp` target therefore needs generic course registration work in PyLearn.

### Activity feedback loop

Conceptual export/evidence semantics exist in ALP docs/internal adapter, but the end-to-end agent-facing workflow is not complete.

## Files to read first in a new session

ALP:

```text
GLOSSARY.md
docs/PLATFORM_MAPPING.md
docs/integrations/PYLEARN.md
docs/integrations/PYLEARN_EXPORT.md
docs/adr/0017-platform-owns-content-competency-mapping.md
docs/adr/0054-platform-capability-adapters.md
docs/adr/0055-domain-scoped-competency-target-adaptation.md
docs/adr/0056-platform-native-authoring-targets.md
docs/PLATFORM_ADAPTATION_ARCHITECTURE.md
docs/integrations/PYLEARN_REFERENCE_TARGET.md
internal/platform/pylearn/adapter.go
internal/platform/pylearn/model.go
internal/platform/pylearn/adapter_test.go
internal/cli/app.go
domains/go/competencies.yaml
domains/go/diagnostic.yaml
```

PyLearn:

```text
docs/go/CONSTITUTION.md
docs/go/CURRICULUM_OVERVIEW.md
apps/web/lib/courses/config.ts
apps/web/lib/courses/index.ts
apps/web/lib/courses/context.ts
apps/web/lib/courses/sync.ts
apps/web/app/(app)/learn/[course]/page.tsx
apps/web/app/(app)/learn/[course]/reel/[slug]/page.tsx
```

Then inspect current Reel authoring/component/runtime contracts in PyLearn before asking any component-specific design question.

## Next grill frontier — NOT YET DECIDED

The next agent should recompute dependencies, but the following decisions are expected to be unblocked after round 1.

Do not treat the recommendations below as accepted until the user explicitly agrees.

### Candidate Q13 — Stable identities

How should ALP distinguish:

- Platform;
- Learning Target;
- Curriculum;
- Learning Unit;
- Activity/content item?

**Recommendation:** use provider/platform-owned stable IDs namespaced by adapter/target; ALP should reference them as opaque external identities and never synthesize meaning from ID strings.

### Candidate Q14 — Target lifecycle

Can a Learning Target be ephemeral/generated per learner, shared/template-based, or both?

**Recommendation:** support both. Separate target identity from a reusable curriculum/template identity. A target may instantiate/adapt a reusable curriculum for one learner/cohort without duplicating competency truth.

### Candidate Q15 — Adaptation persistence

Should Target Adaptation Projections be canonical learner state or derived caches?

**Recommendation:** derived/rebuildable projection; canonical inputs are learner state + target/content snapshot/mapping + explicit target constraints/accepted learner choices.

If an explicit accepted curriculum decision must survive rebuild, store that decision separately from the projection.

### Candidate Q16 — Curriculum/unit spec persistence

Are generated specifications canonical artifacts or ephemeral prompts?

**Recommendation:** versioned artifacts tied to learner revision, target snapshot, mapping/domain-pack versions, sources, and authoring intent. They are not learner truth but should be reproducible/auditable when they drive content PRs.

### Candidate Q17 — Mapping semantics

Should one content mapping distinguish:

- teaches;
- reinforces;
- assesses;
- evidence strength/type?

**Recommendation:** yes for teach/reinforce/assess; do not hard-code evidence strength solely in the static mapping. Assessment/evidence strength should also depend on actual activity/task design and observed result.

### Candidate Q18 — Mapping compatibility/versioning

How should platform mappings declare compatibility with ALP domain packs?

**Recommendation:** retain semantic compatibility ranges plus explicit migration/validation failure when referenced competency IDs disappear/change.

### Candidate Q19 — Activity identity/idempotency

How does ALP avoid importing the same platform event twice?

**Recommendation:** adapter supplies stable platform event/activity identity plus export/cursor metadata; ALP derives deterministic evidence identity/idempotency from platform + learner + event identity/version rather than timestamps alone.

### Candidate Q20 — Export transport

Should ALP require file export, direct DB access, API, or capability-specific transport?

**Recommendation:** generic Activity Source contract; transports are adapter implementation details. PyLearn v0 reference can keep explicit versioned export/file or CLI output because it is portable/testable.

### Candidate Q21 — Authoring granularity

Should authoring capability generate entire target, curriculum skeleton, unit, activity, or patch?

**Recommendation:** capability should support hierarchical intents, but normal execution is smallest justified change: patch/unit first, curriculum skeleton when creating a new target, whole-course materialization only when explicitly requested.

### Candidate Q22 — Platform-native validation

What does ALP need from Platform Validator?

**Recommendation:** structured gate results with platform-native command/provenance, not merely pass/fail text. ALP should know whether authoring is publishable without owning how the platform validates MDX/runtime/media.

### Candidate Q23 — PyLearn course registration

Should PyLearn derive course registry dynamically from content/database/config instead of a hard-coded set?

**Recommendation:** yes. Make course registration declarative/generic so `go-alp` is a normal target and unknown target IDs fail explicitly rather than silently falling back to Python/default.

### Candidate Q24 — PyLearn authoring ownership

Should the generic authoring skill live in ALP, PyLearn, or be split?

**Recommendation:** split composition:
- ALP skill owns learner/adaptation/specification decisions;
- PyLearn-local skill/instructions own Reel MDX realization and platform gates;
- an orchestration skill composes both without duplicating PyLearn authoring knowledge inside ALP.

### Candidate Q25 — Progressive materialization policy

When should downstream units be regenerated after new evidence?

**Recommendation:** target projection can recompute broadly, but regenerate platform content only when an evidence-linked adaptation materially changes an authored unit; never churn already-correct content merely because state was recomputed.

### Candidate Q26 — Learner-specific content vs shared content

Should generated `go-alp` content be personalized directly to one learner or remain reusable?

**Recommendation:** separate reusable technical/content core from learner-specific adaptation/selection where practical. Personalization can influence examples, analogy choices, sequencing, challenge level, and optional variants without embedding private learner state into public/shared course content.

This question needs careful privacy/product grilling.

### Candidate Q27 — Source freshness in generated content

Where is source verification enforced?

**Recommendation:** ALP/domain pack provides freshness/source requirements; platform authoring workflow must re-verify version-sensitive technical claims at author time and preserve relevant provenance in the spec/PR, not blindly trust historical PyLearn content.

### Candidate Q28 — Closed-loop release acceptance

What exact integration tests constitute “platform support”?

**Recommendation:** require both read and write loop for a full authoring-capable adapter:
- target inspect;
- mapping validation;
- adaptation/spec generation;
- platform-native authoring branch;
- platform gates;
- learner activity export;
- idempotent ALP evidence import;
- projection change when warranted;
- next adaptation derived from new state.

Read-only adapters may have a lower capability-specific acceptance bar.

## Important cautions for the next agent

- Do not collapse Learning Target into Course.
- Do not course-scope competency projections.
- Do not put Reel/Scene/MDX types in ALP core.
- Do not move platform mappings into learner workspace.
- Do not treat PyLearn concept_mastery as ALP competency truth.
- Do not directly write platform content from generic ALP core.
- Do not silently mutate existing `go` during the first closed-loop smoke.
- Do not assume current hard-coded PyLearn course registry is acceptable.
- Do not begin implementation before grill → spec → tickets.
- Do not claim production readiness solely because the Store/sync layer is ready.

## Intended end state after grill

The next implementation tranche should eventually produce a generic platform/adaptation contract validated by PyLearn, including:

- platform capability contracts;
- target/curriculum/unit identities;
- target adaptation projection;
- mapping contract/versioning;
- activity import/idempotency;
- authoring specification artifacts;
- PyLearn generic course registration;
- PyLearn Reel MDX realization workflow;
- platform validation contract;
- closed-loop smoke using a new Go target.

Exact scope must come from the completed grill/spec, not this handoff's candidate questions.
