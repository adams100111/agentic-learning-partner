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

## Verified gaps before a closed-loop PyLearn smoke

### 1. Adapter is not exposed through the ALP product surface

The adapter exists as an internal package, but the current CLI does not expose a deterministic user/agent workflow such as:

```text
alp platform pylearn import ...
alp platform pylearn analyze ...
alp platform pylearn ...
```

Exact command shape is not decided yet.

### 2. No concrete PyLearn content → ALP competency mapping exists in the PyLearn repository

ADR-0017 remains correct: the **platform/content repository must own this mapping**.

The mapping must evolve in the same PR as platform content.

The next design round must settle:

- mapping file location;
- schema/version;
- teach/reinforce/assess semantics;
- compatibility with ALP domain-pack versions;
- stable content/activity identity.

### 3. Write-side authoring capability is not implemented

Current ALP docs intentionally describe write-side flow as:

```text
adaptation proposal
→ branch
→ platform content changes
→ platform gates
→ PR/review
```

The next tranche must turn that into an explicit generic authoring contract and PyLearn reference implementation/skill.

### 4. PyLearn course registration is partly hard-coded

Verified current PyLearn code has first-class multi-course schema/routing, but registration/sync still contains hard-coded assumptions:

- `COURSE_CONFIG` explicitly lists `pylearn` and `go`;
- `KNOWN_COURSES` explicitly contains the default course and `go`;
- sync explicitly seeds only those known course rows.

Therefore a fresh target such as `go-alp` is not yet a clean first-class course. Unknown Reel course tags are currently redirected to the default course.

This is a PyLearn implementation gap, not a reason to put course registration in ALP core.

### 5. Closed-loop activity ingestion is not wired end-to-end

The conceptual PyLearn export contract exists, but the full production path:

```text
PyLearn activity
→ export
→ ALP platform capability
→ canonical evidence
→ assessment/projection
→ target adaptation
```

is not yet exposed as a complete agent-facing workflow.

## Recommended first smoke target

Do not overwrite the existing `go` course.

Create a separate experimental target, provisionally:

```text
go-alp
```

Purpose:

```text
go
= manually designed pre-ALP curriculum

go-alp
= curriculum/content generated/adapted from canonical ALP learner state
```

The identifier is provisional until the next grill settles target identity/naming.

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
