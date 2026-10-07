# Closed-loop smoke: ALP × PyLearn `go-alp`

The authoring-plus-closed-loop acceptance bar of spec #66 (Seam 3, Q28, Q38). One local command drives the real `alp` CLI and PyLearn's real scripts and persistence code through the whole loop on the `go-alp` Learning Target. There is no CI; results come from local runs only.

## Run

```bash
devenv shell -- scripts/smoke-closed-loop.sh <path-to-pylearn-checkout>
```

The PyLearn checkout must contain the `go-alp` course (PyLearn `integration/66-alp-reference-integration` or later) and its own `devenv.nix`; `devenv` must be on `PATH`. The script runs `go test -tags closedloop -run '^TestClosedLoopGoALP$' ./internal/e2e/` with `ALP_SMOKE_PYLEARN` set to the checkout. The test is behind the `closedloop` build tag, so `go test ./...` never runs it.

## What it uses

- `alp`, built from this tree into a temporary directory.
- A fresh learner workspace with a **synthetic** learner profile, Global Persona and Go Domain Persona, under an isolated `HOME`. Never real learner data.
- A throwaway PyLearn libSQL database (a SQLite file under the test's temporary directory, passed as `DATABASE_URL`).
- PyLearn's commands, each run as `devenv shell -- …` in the checkout: `bun install --frozen-lockfile`, `export:curriculum`, `validate:platform` (with `ALP_BIN` set to the built `alp`), `db:migrate`, `db:sync-lessons` and `export:activity`.
- `internal/e2e/testdata/pylearn-smoke.ts`, run with PyLearn's `tsx` from `apps/web`, which calls PyLearn's own library code: `newTargetMapping`/`newTargetCurriculum` (what `alp:bootstrap` prints for an undeclared target), and `roster.createUser` plus `persistQuizAnswers` (what `POST /api/quiz` persists).

The checkout is only read. `compile:go` (inside `validate:platform`) rewrites `apps/web/public/wasm/go/manifest.json`; the smoke puts it back, and fails if `git status` of the checkout differs afterwards.

## Steps and assertions

| Step | What runs | Asserted |
|---|---|---|
| 0 | `go build ./cmd/alp`; workspace and personas; `bun install` | — |
| 1 | `export:curriculum --course go-alp`; `alp platform inspect`; `alp platform mapping validate` | all five capabilities, authoring skill `pylearn-alp-authoring`, the realized lesson `go-alp-a3-sync-primitives` listed, mapping valid |
| 2 | starting evidence and accepted assessments (`alp evidence add`, `alp assessment add`); `alp platform plan` on the pre-authoring export: default, then `--intent target-skeleton` | default reports `explicit-intent-required`; the skeleton is planned, every persona document is used, and its `go.concurrency.sync` unit has the spec ID the PyLearn mapping cites (proposed IDs derive from target and competency) |
| 3 | a plan on the authored export (logged), `validate:platform --target go-alp`, `alp platform gates record` with a realization report for the already-realized unit (PyLearn #47); record again; re-plan | publishable; that unit `realized` with its Realization Link rooted at the lesson, the other skeleton units `unrealized` (`realization-not-reported`); second record `already-recorded`; after realization the unit keeps its proposed spec ID and version (`held`), and no plan authors it again |
| 4 | seed four correct quiz answers through `persistQuizAnswers`; `export:activity`; import before linking; `alp platform account link --confirm`; import the same export twice; `export:activity --since` + `import --cursor` | unlinked import refused (`platform-account-not-linked`); first import 4 imported; second import 0 imported, 4 skipped, no evidence file added; the incremental export carries nothing |
| 5 | re-plan; an assessment citing the imported evidence (the smoke plays the assessing agent); re-plan; re-plan again | imported evidence alone leaves the projection revision unchanged; after the assessment the revision changes, the unit's competency is `functional` citing that assessment, `proposedMode` goes `full` → `skim`, and the realized unit gets version 2 under the same ID (`evidenceLinked`, `adaptationMode` material); re-planning unchanged state reproduces the revision |

`go test -v` prints one line per step with the IDs, counts and revisions involved.

## Scope

The smoke plays two agents with fixed synthetic inputs: the authoring target skill (it reuses the unit PyLearn #47 realized rather than authoring a new one) and the assessing agent. **No real Claude or Codex harness runs**; a real-harness smoke is reported separately and never assumed from this one (see [PRODUCTION_V0_RELEASE_SMOKE.md](../PRODUCTION_V0_RELEASE_SMOKE.md) for the production-v0 harness gate).

Spec-ID stability: a plan run on the authored export *before* `gates record` shows the new lesson under an item-derived spec ID, because nothing is realized until the gate result is recorded. Once the Realization Link exists, every later plan uses the proposed unit's ID again (also covered in `internal/cli` by `TestPlatformRealizedUnitKeepsItsProposedIDWhenPlannedBeforeGatesRecord`).
