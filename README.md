# Agentic Learning Partner

Agentic Learning Partner (ALP) is a portable, stateful learning system for AI coding agents.

It combines:

- a durable learner profile and global/domain personas;
- append-only evidence and auditable assessments;
- deterministic competency projections;
- compact task-specific agent context;
- adaptive review and learning plans;
- domain packs (Go first; Rust later);
- project and platform telemetry;
- reusable Agent Skills;
- a deterministic Go CLI;
- and shared packaging for OpenAI/Codex and Claude Code.

## Production-v0 candidate

ALP now contains the production-v0 candidate implementation. The core learning engine is joined by production Store providers, multi-device Git synchronization, staged session transactions, recovery, workspace lifecycle/onboarding, and provider-independent backup/restore.

The release candidate still requires the real Claude Code ↔ Codex private-GitHub smoke in `docs/PRODUCTION_V0_RELEASE_SMOKE.md` before a production-v0 release tag is claimed.

Core commands include:

    alp validate
    alp workspace init <name> ...
    alp workspace connect <name> --path ...
    alp workspace clone <name> <remote> ...
    alp workspace list
    alp workspace use <name>
    alp workspace status [name]
    alp workspace sync [name]
    alp workspace export [name] --out workspace.alp
    alp workspace verify workspace.alp
    alp workspace restore <name> workspace.alp --mode recover|clone|merge
    alp workspace move <name> --provider local|git --path ...
    alp workspace migrate --dry-run

    alp session begin --harness <harness>
    alp session status
    alp session put --path <ALP_PATH> --file <FILE>
    alp session close --summary "..."
    alp session abort
    alp session recover-sync

    alp domain list
    alp domain info go

    alp context build --task teach --domain go
    alp context inspect --task teach --domain go

    alp persona show --domain go
    alp status
    alp competency show go.runtime.context
    alp evidence show <id>

    alp evidence add --file evidence.yaml
    alp assessment add --file assessment.yaml
    alp state rebuild

    alp diagnostic --domain go
    alp plan build --domain go

The Go domain pack ships with a competency taxonomy and adaptive diagnostic catalog. PyLearn is the first read/analyze platform adapter.

## Learner state is separate

This repository is the reusable engine/plugin.

Personal learner state belongs in a separate private ALP workspace, such as:

    agentic-learning-state/
    ├── workspace.yaml
    ├── profile/
    ├── personas/
    ├── evidence/
    ├── assessments/
    ├── platform-accounts/
    ├── adaptation-decisions/
    ├── specifications/
    ├── authoring-plans/
    └── state/

ALP never treats harness chat memory as canonical learner state.

Workspace discovery order:

1. explicit `--workspace`;
2. project-local `.alp.yaml`;
3. `ALP_WORKSPACE`;
4. `~/.config/alp/config.yaml`.

See `docs/WORKSPACE.md` and `docs/INSTALL.md`.

## Agent packaging

Portable OpenAI/Codex package:

    plugin.json
    skills/

Claude Code package identity:

    .claude-plugin/plugin.json

Both harnesses use the same root skill tree, CLI, schemas, domain packs, and learner workspace. There is no custom MCP server in v0.

## First integration

The first platform target is [PyLearn](https://github.com/adams100111/pylearn).

PyLearn provides learner activity and candidate course material. It is not ALP's technical source of truth. Version-sensitive claims are verified against authoritative upstream sources under `docs/SOURCE_POLICY.md`.

## Verification

The repository contains focused package tests, production Store/session acceptance coverage, and an end-to-end learning test covering:

    PyLearn activity
      -> ALP evidence
      -> assessment
      -> deterministic projection
      -> compact context
      -> Go learning plan

GitHub Actions is intentionally disabled while the repository owner's Actions quota is exhausted. Do not re-enable workflows merely to satisfy CI conventions; use local verification and explicit review until quota is restored.

## Architecture and decisions

Start with:

- `GLOSSARY.md`
- `docs/ARCHITECTURE.md`
- `docs/STATE_MODEL.md`
- `docs/PERSONAS.md`
- `docs/IMPLEMENTATION_WORKFLOW.md`
- `docs/adr/`

Implementation is tracked from GitHub spec issue #2 and its ticket graph.
