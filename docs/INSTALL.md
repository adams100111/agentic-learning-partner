# Install and Configure ALP

ALP has two pieces:

1. the reusable plugin/CLI;
2. a separate private learner workspace.

No harness owns learner truth.

## Quick start (Claude Code)

    claude plugin marketplace add adams100111/agentic-learning-partner
    claude plugin install agentic-learning-partner@agentic-learning-partner

That is all. On the next session start the plugin's `SessionStart` hook installs the `alp` CLI at exactly the plugin's version (macOS and Linux, arm64 and amd64; no sudo, no `go` toolchain). When `alp` already reports that version the hook exits silently without touching the network. After `claude plugin marketplace update`, the next session upgrades the CLI to match.

The hook runs the plugin's own `scripts/get.sh` pinned to the plugin's tag, so the CLI and the skills always come from the same release. It never blocks a session: on failure it prints the manual command below and carries on.

- Opt out: set `ALP_SKIP_CLI_INSTALL=1`. The `setup-learning-workspace` skill then reports a missing or mismatched CLI instead.
- Install location: `~/.local/bin`, or `ALP_INSTALL_DIR`. The hook warns once if it is not on your `PATH`.
- A locally built `alp` (`go install`, version `dev`) is never replaced.

Troubleshooting:

- `alp: command not found` after the first session: add `~/.local/bin` to `PATH` (the hook says so once), or restart the shell.
- The hook failed (offline, no `curl`, rate limit): run the manual command below, or set `GITHUB_TOKEN` and restart the session.
- Check what ran: `alp version --json` should report `v` plus the version in `.claude-plugin/plugin.json`.

## CLI (manual, or without Claude Code)

Install the latest release (macOS and Linux, arm64 and amd64; no sudo, no CI, no `go` toolchain needed):

    curl -fsSL https://raw.githubusercontent.com/adams100111/agentic-learning-partner/main/scripts/get.sh | bash

Prefer pinning to a tag, which is what the plugin does (script and binary from the same release):

    curl -fsSL https://raw.githubusercontent.com/adams100111/agentic-learning-partner/v0.1.1/scripts/get.sh | bash -s -- v0.1.1

The installer downloads the release archive for your OS/architecture, verifies it against the release's `SHA256SUMS` (refusing to install on a mismatch), and installs `alp` to `~/.local/bin` (override with `ALP_INSTALL_DIR`). It warns if that directory is not on your `PATH`. Set `GITHUB_TOKEN` or `GH_TOKEN` only if you hit GitHub API rate limits while resolving the latest version.

Upgrade by re-running the same command; it is idempotent and replaces the existing binary.

Verify:

    alp version
    alp version --json
    alp domain info go

Uninstall:

    rm ~/.local/bin/alp

Your learner workspace is separate and is not touched.

### Development

To build from a checkout instead of a release (version reports `dev`):

    go install ./cmd/alp

### Releasing (maintainers)

Releases are cut locally with `scripts/release.sh <vX.Y.Z>`; there is no CI. Bump the version in `plugin.json`, `.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json`, merge to `main`, then run it from a clean, up-to-date `main`. It vets and tests, cross-compiles the four targets into `dist/`, uploads to a draft GitHub Release, verifies the downloaded assets against `SHA256SUMS`, then tags and publishes. Use `--dry-run` to build and checksum without tagging or publishing.

## Learner workspace

The preferred setup path is the portable `setup-learning-workspace` skill. It inspects existing configuration/state first and delegates mutations to deterministic CLI commands.

Core lifecycle commands include:

    alp workspace init <name> ...
    alp workspace connect <name> --path ...
    alp workspace clone <name> <remote> ...
    alp workspace list
    alp workspace use <name>
    alp workspace status [name]
    alp workspace sync [name]
    alp workspace move <name> ...

Workspace discovery continues to support these resolution layers:

1. `--workspace /path/to/workspace`;
2. project-local `.alp.yaml`;
3. `ALP_WORKSPACE`;
4. `~/.config/alp/config.yaml`.

Example project pointer:

    workspace: ~/dev/agentic-learning-state

Then:

    alp workspace status
    alp workspace check
    alp validate

For Git Store, ordinary synchronization uses the host's existing Git/SSH credentials. ALP does not ask for or persist those credentials.

## OpenAI / Codex

The repository root is a portable Agent Plugins package with `plugin.json` and `skills/` at the root. The root manifest is canonical. No MCP server is required for v0.

Use the normal current Codex/OpenAI plugin installation or local marketplace workflow for a local plugin directory.

Codex reads the same `hooks/hooks.json` from a plugin root and sets `CLAUDE_PLUGIN_ROOT` for plugin hooks, so the SessionStart installer is expected to run there too, subject to Codex's own hook trust/enablement (not verified end to end here). Where it does not run, the skills follow `docs/CLI_PREREQUISITE.md` and the manual command above covers it.

## Claude Code

Claude Code packaging is provided by `.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json` plus the same root `skills/` directory (all skills are auto-discovered from `skills/`).

Install from the GitHub marketplace (the repository is public; no authentication is needed; this is the quick start above):

    claude plugin marketplace add adams100111/agentic-learning-partner
    claude plugin install agentic-learning-partner@agentic-learning-partner

Verify that all skills are listed:

    claude plugin details agentic-learning-partner@agentic-learning-partner

Update later with `claude plugin marketplace update agentic-learning-partner`.

For local development, load the repository as a plugin directory. Claude Code namespaces plugin skills under `agentic-learning-partner`.

Validate with a current Claude Code installation (a warning about the root `CLAUDE.md` not being loaded as plugin context is expected):

    claude plugin validate .

## State consistency

Claude Code and Codex both call the same `alp` CLI and resolve the same learner workspace.

Do not keep a separate harness-specific learner profile, competency cache, or assessment history.

## Credentials

ALP does not store Git credentials. Workspace cloning/pushing uses the user's existing Git/GitHub authentication.


## First real learning session

After installation, do not hand-create evidence, assessments, competency projections, review queues, or learning plans.

Run the first real session from an actual harness and follow `docs/FIRST_HARNESS_TEST.md`. The harness should inspect bootstrap profile/persona state, select the minimum high-information diagnostic, and let the ALP runtime create the first genuine learning-state artifacts from actual evidence.
