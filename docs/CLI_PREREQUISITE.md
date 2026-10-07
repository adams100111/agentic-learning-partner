# CLI prerequisite

Every ALP skill drives the `alp` CLI, and the CLI must be the release matching the plugin.

## Check

1. Read the plugin version: the `version` field of `.claude-plugin/plugin.json` (or `plugin.json`) in the plugin root. The plugin root is two directories above the skill's `SKILL.md` (`<root>/skills/<skill>/SKILL.md`). Call it `X.Y.Z`.
2. Run `alp version --json`. The CLI is current when `version` equals `vX.Y.Z`, or equals `dev` (a local build, left alone).

## Install or upgrade

When `alp` is missing or its version differs, run the installer pinned to that tag:

    bash "<plugin root>/scripts/get.sh" vX.Y.Z

When the plugin root is unknown, run the same pinned installer from the tag, never from `main`:

    curl -fsSL https://raw.githubusercontent.com/adams100111/agentic-learning-partner/vX.Y.Z/scripts/get.sh | bash -s -- vX.Y.Z

The installer verifies the release checksum, installs to `${ALP_INSTALL_DIR:-$HOME/.local/bin}`, needs no sudo, and warns when that directory is not on `PATH`. Add the directory to `PATH` for the current shell, then re-run `alp version --json`.

Claude Code runs this same check at session start (`hooks/hooks.json`); `ALP_SKIP_CLI_INSTALL=1` turns that off, and then this note is the only path. Done when `alp version --json` reports `vX.Y.Z`. If the install fails, tell the user the command above and stop; do not improvise state.
