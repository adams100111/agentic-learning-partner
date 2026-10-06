# Workspace Discovery and Configuration

## Resolution order

ALP resolves the learner workspace in this order:

1. explicit CLI/session option such as `--workspace`;
2. project-local `.alp.yaml`;
3. `ALP_WORKSPACE` environment variable;
4. user config such as `~/.config/alp/config.yaml`;
5. fail with a clear setup error.

ALP must never silently create learner state inside the engine/plugin repository.

## Named workspaces

Production v0 supports multiple named workspaces on one device while preserving one learner per workspace.

Machine-local configuration conceptually follows:

```yaml
defaultWorkspace: personal

workspaces:
  personal:
    path: ~/.local/share/alp/workspaces/personal
    provider: git
  research:
    path: ~/.local/share/alp/workspaces/research
    provider: local
```

Most users may configure only one workspace.

Workspace registry, checkout paths, selected remote, selected branch, and other machine-specific provider settings belong to local ALP configuration rather than synchronized learner state.

## Project-local pointer

A project-local `.alp.yaml` may contain only non-secret integration configuration, for example:

```yaml
workspace: personal
platform: pylearn
```

It may point to a named workspace or supported explicit path according to CLI resolution rules.

It must not duplicate learner profile or competency state.

## One workspace = one learner

One learner workspace represents exactly one learner.

Multiple workspaces do not create multi-user state inside a workspace.

Hosted multi-user persistence is a separate architecture.

## Derived state

The workspace may commit generated projections/materialized views for fast startup and human readability.

These files are caches only:

- current competency snapshot;
- current focus;
- review/reinforcement queue;
- generated Markdown summaries.

On conflict, regenerate them from canonical inputs instead of hand-merging.

## Git remote privacy

When an ALP/forge integration creates a Git remote, production setup requires that remote to be private.

When connecting an existing remote:

- if repository visibility can be queried securely, warn/block public remotes unless the user explicitly overrides;
- if visibility cannot be determined (for example a generic SSH server), explain that privacy cannot be verified and require explicit acknowledgement once during connection.
