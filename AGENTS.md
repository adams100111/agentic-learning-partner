# ALP agent guide

- **Toolchain and hooks:** devenv pins Go and installs the git hooks. Run commands as `devenv shell -- <cmd>` (for example `devenv shell -- go test ./...`, `devenv shell -- git commit ...`); the hooks need the devenv `go` on PATH.
- **Vocabulary:** `GLOSSARY.md`. **Decisions:** `docs/adr/` (numbers are immutable; the `adr-numbers` hook rejects duplicates).
- **Issues, specs, tickets:** `docs/agents/issue-tracker.md`. **Docs layout:** `docs/agents/domain.md`.
- **Workflow (grill → spec → tickets → implement):** `docs/IMPLEMENTATION_WORKFLOW.md`.
- **Platform adaptation (PyLearn):** `docs/integrations/PYLEARN_REFERENCE_TARGET.md`.
