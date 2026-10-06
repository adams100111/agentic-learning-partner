# Install and Configure ALP

ALP has two pieces:

1. the reusable plugin/CLI;
2. a separate private learner workspace.

No harness owns learner truth.

## CLI

During development from this repository:

    go install ./cmd/alp

For a tagged public release:

    go install github.com/adams100111/agentic-learning-partner/cmd/alp@<version>

Verify:

    alp domain info go

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

## Claude Code

Claude Code packaging is provided by `.claude-plugin/plugin.json` plus the same root `skills/` directory.

For local development, load the repository as a plugin directory. Claude Code namespaces plugin skills under `agentic-learning-partner`.

Validate with a current Claude Code installation:

    claude plugin validate . --strict

## State consistency

Claude Code and Codex both call the same `alp` CLI and resolve the same learner workspace.

Do not keep a separate harness-specific learner profile, competency cache, or assessment history.

## Credentials

ALP does not store Git credentials. Workspace cloning/pushing uses the user's existing Git/GitHub authentication.


## First real learning session

After installation, do not hand-create evidence, assessments, competency projections, review queues, or learning plans.

Run the first real session from an actual harness and follow `docs/FIRST_HARNESS_TEST.md`. The harness should inspect bootstrap profile/persona state, select the minimum high-information diagnostic, and let the ALP runtime create the first genuine learning-state artifacts from actual evidence.
