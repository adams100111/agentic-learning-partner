# Workspace Discovery and Configuration

## Resolution order

ALP resolves the learner workspace in this order:

1. explicit CLI/session option such as `--workspace`;
2. project-local `.alp.yaml`;
3. `ALP_WORKSPACE` environment variable;
4. user config such as `~/.config/alp/config.yaml`;
5. fail with a clear setup error.

ALP must never silently create learner state inside the engine/plugin repository.

## Project-local pointer

A project-local `.alp.yaml` may contain only non-secret integration configuration, for example:

```yaml
workspace: ~/dev/agentic-learning-state
platform: pylearn
```

It must not duplicate learner profile or competency state.

## One workspace = one learner

For v1, one learner workspace represents exactly one learner.

This keeps identity, privacy, Git history, and context selection simple.

Multi-user hosted persistence is a separate future architecture.

## Derived state

The workspace may commit generated projections/materialized views for fast startup and human readability.

These files are caches only:

- current competency snapshot;
- current focus;
- review/reinforcement queue;
- generated Markdown summaries.

On conflict, regenerate them from canonical inputs instead of hand-merging.
