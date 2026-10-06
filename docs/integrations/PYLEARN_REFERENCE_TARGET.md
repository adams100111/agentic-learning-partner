# PyLearn as ALP Reference Platform

## Status

PyLearn is ALP's **primary reference platform and first closed-loop validation target**, but ALP must remain useful with other learning platforms and delivery environments.

This document records the verified current PyLearn/ALP state as of the platform-adaptation grill.

## Native authoring target

For PyLearn, generated content must target the actual **Reel MDX engine and component/runtime contract**, not generic Markdown.

Conceptually:

```text
ALP Learning Unit Specification
        ↓
PyLearn authoring capability
        ↓
content/reel/*.mdx
        ↓
Reel MDX components/runtime
        ↓
PyLearn compiler/player/quality gates
```

Representative native constructs include:

- Scene;
- Mark;
- Code;
- DialectCompare;
- TranslationMap;
- Diagram;
- Trace;
- Steps;
- Quiz;
- Checkpoint;
- RunGo;
- other Reel-native components available in the repository.

The adapter/authoring workflow must discover and use the current PyLearn component contract rather than relying on a stale hard-coded list.

## Existing ALP integration

Verified existing ALP assets include:

- `internal/platform/pylearn/adapter.go`;
- `internal/platform/pylearn/model.go`;
- adapter tests;
- `docs/integrations/PYLEARN.md`;
- `docs/integrations/PYLEARN_EXPORT.md`;
- platform mapping schema/docs;
- Go competency taxonomy and diagnostic pack.

The current adapter is primarily **read/normalize-first**.

It handles/defines normalization semantics for:

- progress;
- exercise attempts;
- concept-mastery rollups;
- quizzes;
- reflections;
- bookmarks;
- project/source evidence references.

Important existing rule: PyLearn's derived concept mastery is not imported as authoritative ALP competency truth.

## Gaps and their resolution

The 2026-10 platform adaptation grill settled every gap below. Spec: adams100111/agentic-learning-partner#66; decisions: ADR-0054 through ADR-0061 and the grill record in `docs/HANDOFF_PLATFORM_ADAPTATION_GRILL.md`.

| Gap | Resolution | Ticket |
|---|---|---|
| Adapter not reachable from the CLI | Generic `alp platform` family (`inspect`, `mapping validate`, `import`, `plan`, `gates record`) with `--adapter`/`--target`, gated by declared capabilities | ALP #68 |
| No PyLearn → competency mapping | Mapping v2 with Mapping Roles, owned by the PyLearn repo, keyed by Declared-Stable Identifiers (ADR-0057, ADR-0058) | ALP #69, PyLearn #45 |
| Write-side authoring missing | Unit specs → Authoring Plan → PyLearn-local Reel authoring skill → Platform Gate Result (ADR-0056, ADR-0060) | ALP #72–#74, PyLearn #47 |
| Hard-coded course registration | One declaration file per course, per-course phases, unknown course is a hard error | PyLearn #43 |
| Activity ingestion not wired | `export:activity` v2 with Synthetic Event Identity and cursor; idempotent `alp platform import` (ADR-0059) | ALP #70, PyLearn #46 |

`go-alp` is the settled identifier for the first ALP-authored target (ADR-0061: shared, learner-free target with per-learner projections).

## Verified PyLearn facts (2026-10)

- ORM is Drizzle on SQLite/libSQL.
- Learner activity tables (`progress`, `quiz_answers`, `concept_mastery`, `attempts`) are upserted latest-state rows with no event IDs; `attempts` has no writer.
- Declared-stable identifiers: lesson front-matter `id`, explicit Scene `id`, quiz/question IDs. Section IDs (slugified headings) and positional scene IDs are unstable.
- The only content → concept link is free-text `concept="..."` on Quiz/SectionQuiz; it doubles as quiz ID and `concept_mastery` key. It is platform-local, not the ALP mapping.
- `lint:lessons` and `gate:reels` print text and exit non-zero on failure; an internal gate report type exists but is not emitted as JSON.
- All content is static MDX shared by every user.

## Existing PyLearn Go course

The current Go curriculum is already substantial and should be treated as:

- reference material;
- a comparison baseline;
- a source of platform authoring constraints;
- historical candidate content.

It must not become ALP learner truth or be copied into ALP core.

## First closed-loop validation goal

The eventual smoke should prove:

1. ALP resolves the learner's current Go state;
2. ALP derives a target-specific adaptation/curriculum projection;
3. a fresh PyLearn Go target is registered;
4. platform-native Reel MDX is authored through the target workflow;
5. PyLearn quality gates pass;
6. the learner uses the generated content;
7. PyLearn exports meaningful activity;
8. ALP converts activity into evidence/assessment/projection;
9. subsequent target adaptation changes only when evidence justifies it.

This is a stronger test than merely verifying Git Store synchronization.
