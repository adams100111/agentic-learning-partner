# Platform Adaptation Grill — Record (closed)

**Status:** closed 2026-10-07. This file is the decision record for Q1–Q38.

- Spec: adams100111/agentic-learning-partner#66 (tickets are its sub-issues, with native blocked-by edges).
- Hard-to-reverse decisions: ADR-0054 through ADR-0061.
- Verified PyLearn facts and gap resolutions: `docs/integrations/PYLEARN_REFERENCE_TARGET.md`.

The sections below preserve the original handoff and every round's settled decisions.

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

## Settled grill round 2

Q13, Q15–Q24, Q26, Q27, Q29, Q30 were explicitly accepted by the user (2026-10-07), including amendments made after PyLearn fact-finding.

### Additional verified PyLearn facts (round 2)

- ORM is Drizzle on SQLite/libSQL (`apps/web/db/schema.ts`), not Prisma.
- Learner activity tables (`progress`, `quiz_answers`, `concept_mastery`, `attempts`) are upserted latest-state rows; no event log, no event IDs. No `attempts` writer exists. No export endpoint/script/CLI exists.
- Stable IDs: lesson front-matter `id`, explicit Scene `id`, quiz/question IDs. Unstable: section IDs (slugified H2), positional scene IDs.
- Only content→concept link is free-text `concept="..."` on Quiz/SectionQuiz (143 tags, 142 unique, no registry); it is also the quiz ID and `concept_mastery` key.
- Gates (`lint:lessons`, `gate:reels`) emit text and exit codes; an internal `GateReport` type exists but is not emitted as JSON. No CI.
- New course needs `COURSE_CONFIG`, `KNOWN_COURSES`, and a hard-coded `syncLessons` insert; unknown/missing `course:` tags fall back to `pylearn`; phases outside global `PHASE_ORDER` are silently dropped.
- All content is static public MDX; no per-user content mechanism.

### Decisions

- **Q13 Identities (ADR-0057):** platform instance, Learning Target, content/activity item IDs are platform-owned opaque External Identities, namespaced `{platform, target, item}`. Curriculum/Unit Specification IDs are ALP-owned. Realized specs record Realization Links. Only Declared-Stable Identifiers may be referenced; adapter validators reject unstable ones.
- **Q15 Adaptation persistence (ADR-0060):** Target Adaptation Projection is a generated, rebuildable workspace file. Inputs: learner state, target content/mapping snapshot, target constraints, Accepted Adaptation Decisions (canonical, append-only, separate record).
- **Q16 Spec persistence (ADR-0060):** Curriculum/Unit Specifications are immutable versioned Store-owned artifacts in the private learner workspace; content PRs cite spec ID/hash.
- **Q17 Mapping semantics (ADR-0058):** mapping schema v2 with Mapping Roles `teaches|reinforces|assesses`; only `assesses` yields assessment-grade evidence; no static strength, optional strength ceiling. PyLearn `concept` tags are not the mapping; a separate mapping file in the PyLearn repo maps stable item IDs to ALP competencies.
- **Q18 Compatibility (ADR-0058):** semver `packVersion` range; validation resolves through pack migrations (rename → warn; split/merge → specific fail; removed/out-of-range → fail).
- **Q19 Idempotency (ADR-0059):** evidence ID = hash(platform instance, target, learner, item, event identity, event revision). State-only platforms use Synthetic Event Identity (row key + content hash). Re-import skips; higher revision supersedes; unmapped activity is reported, never dropped.
- **Q20 Transport (ADR-0059):** Activity Source contract is "records since cursor"; transport is an adapter detail; never direct platform DB access. Fix export doc/schema `courses` mismatch → `targets`.
- **Q21 Authoring granularity:** hierarchical Authoring Intent (target skeleton → curriculum → unit → activity → patch); default smallest justified; whole-course only on explicit request.
- **Q22 Validation:** structured Platform Gate Result (per-gate id/status/provenance/artifacts/diagnostics + `publishable`), stored with the Authoring Plan.
- **Q23 PyLearn course registration:** one declarative file per course (e.g. `content/courses/<id>.yaml`) generating config and DB rows; per-course phase lists; unknown/missing `course:` is a hard error.
- **Q24 Authoring ownership:** ALP skill → adaptation + unit spec; PyLearn-local skill → Reel MDX realization + PyLearn gates; thin ALP orchestration skill dispatches to the platform-declared authoring target skill and collects gate results.
- **Q26 Personalization/privacy (ADR-0060):** `go-alp` content is reusable and learner-free; personalization shapes selection/sequence/difficulty/examples/analogies; specs (with rationale) stay private; per-learner variants out of scope.
- **Q27 Freshness:** pack/spec declare version-sensitive claims; platform authoring re-verifies and records provenance in the PR; ALP refuses to mark a unit realized without required provenance.
- **Q29 PyLearn export (ADR-0059):** versioned export CLI producing `pylearn-export` v2 (`targets`, cursor) from current tables using synthetic identities; append-only activity log optional later; add an `attempts` writer or exclude attempt evidence from v0.
- **Q30 PyLearn gate output:** `--json` on `gate:reels`/`lint:lessons` emitting the existing `GateReport` mapped to Platform Gate Result; the PyLearn Platform Validator composes these plus `typecheck` and `compile:go`.

### ADRs created from round 2

- `docs/adr/0057-opaque-stable-external-identities.md`
- `docs/adr/0058-mapping-roles-and-migration-aware-validation.md`
- `docs/adr/0059-deterministic-activity-import-identity.md`
- `docs/adr/0060-versioned-private-specs-and-learner-free-content.md`

Glossary gained: External Identity, Declared-Stable Identifier, Mapping Role, Accepted Adaptation Decision, Synthetic Event Identity, Realization Link, Platform Gate Result, Authoring Intent.

## Settled grill round 3

Q14, Q25, Q28, Q31, Q35 were explicitly accepted by the user (2026-10-07).

- **Q14 Target lifecycle (ADR-0061):** Learning Targets are shared, learner-free platform identities; each learner has their own Target Adaptation Projection over a shared target. Realization Links live in the motivating learner's specs; other learners trace via platform mapping.
- **Q25 Progressive regeneration:** recompute projections freely; new spec version / content PR only when an evidence-linked adaptation materially changes a realized unit's spec (competencies, prerequisites, adaptation mode, misconceptions, required evidence). No PR for cosmetic/no-op recomputes. Unrealized units are re-specified freely.
- **Q28 Acceptance:** capability-tiered. Read-only: inspect, mapping validate, idempotent import (re-run → zero new evidence; unmapped reported). Authoring: + spec → branch → `publishable` Platform Gate Result → PR on a fresh target. Closed loop: scripted run where synthetic activity on the new unit changes the projection when warranted and the next adaptation derives from it. PyLearn `go-alp` must pass all three locally (no CI). Real Claude/Codex harness smoke reported separately, never assumed.
- **Q31 Decision authority (ADR-0061):** Accepted Adaptation Decisions only with explicit learner confirmation; agents propose via Adaptation Proposals; records confirmer/time/basis projection revision; revocable by superseding record.
- **Q35 ALP CLI:** generic `alp platform` family over an adapter registry, gated by declared capabilities: `inspect`, `mapping validate`, `import`, `plan` (spec + Authoring Plan), `gates record`; `--adapter <id> --target <id>`; deterministic JSON output. No platform-specific commands.

ADR created: `docs/adr/0061-shared-targets-learner-projections.md`.

## Settled grill round 4

Q33, Q34 were explicitly accepted by the user (2026-10-07).

- **Q33 Cross-repo mapping validation:** PyLearn lint (no ALP dependency) checks mapped IDs are declared-stable and exist. New PyLearn gate `gate:alp-mapping` runs `alp platform mapping validate --adapter pylearn` against the mapping's pinned pack version; required for ALP-authored targets and PRs touching the mapping file; `skipped` for unmapped courses; missing `alp` fails with an install hint.
- **Q34 `go-alp` structure:** phases derived from the Curriculum Specification at target-skeleton authoring; PyLearn authoring skill emits `content/courses/go-alp.yaml`; phase IDs freeze once any unit in them is realized; later versions may add/insert phases but never renumber/rename realized ones; manual `go` course untouched.

## Settled grill round 5

Q36–Q38 were explicitly accepted by the user (2026-10-07), after spec #66 and tickets were drafted.

- **Q36 Platform account link:** learner-confirmed workspace record linking `{platform instance, platform user ID}` → workspace `learnerId`; import refuses unlinked users; no email/name matching.
- **Q37 Curriculum Reader:** platforms expose target structure via a versioned curriculum export (declared-stable items, phases, mapping, content hash); PyLearn provides `export:curriculum`; ALP never parses MDX or platform repos.
- **Q38 Closed-loop synthetic activity:** seeded into a throwaway PyLearn libSQL DB through PyLearn's real persistence code, then exported with the real `export:activity`.

## Notes after the grill

- **2026-10-07 — Q35's command list was extended by Q31 and Q36.** Q35 listed `inspect`, `mapping validate`, `import`, `plan` and `gates record`. Q31 (learner-confirmed Accepted Adaptation Decisions) added `alp platform decision accept|revoke`, and Q36 (learner-confirmed Platform Account Links) added `alp platform account link`, which acts on a platform account rather than a Learning Target and so takes `--adapter` but no `--target`. The full contract, including which flags each command takes, is in `docs/TARGET_ADAPTATION.md` (`alp platform` command reference).

## Grill status

Grill closed. Spec: adams100111/agentic-learning-partner#66. Tickets: ALP #68–#75, PyLearn #43–#47 (sub-issues of #66 with native blocked-by edges). Next: `implement-spec`.

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
