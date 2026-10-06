# ADR-0050: Use one agentic setup wizard with durable safe-sync consent

**Status:** Accepted

## Context

ALP must support both terminal-first and agent-first onboarding without duplicating setup semantics in prompts.

Existing users should not repeat learner onboarding when connecting an existing workspace, while new users need provider/workspace setup plus profile/persona bootstrap.

Safe session synchronization should not require repeated network approvals once the user has chosen that policy.

## Decision

Production v0 provides one staged setup workflow shared by CLI and agent skill.

The wizard:

1. inspects installed ALP, existing config, workspaces, Git, and available forge integrations;
2. asks only unresolved decisions;
3. recommends defaults;
4. creates or connects a Store;
5. configures remote/sync when relevant;
6. reuses existing learner state when present;
7. bootstraps learner profile/persona only when needed;
8. optionally begins domain onboarding.

A securely connected forge integration may create a private Git repository only after one explicit user approval for that external side effect.

Choosing `sync.mode: session` is durable consent for ordinary safe fetch/reconcile/push operations against the configured remote/branch.

Further approval is required for:

- changing remotes;
- provider conversion;
- first connection to unverifiable/public remotes;
- destructive recovery actions.

Deterministic CLI/runtime operations perform mutations. Prompt logic never owns setup state transitions.

## Consequences

- CLI and agent onboarding share one state machine;
- existing users skip unnecessary persona questioning;
- safe synchronization remains low-friction;
- forge credentials stay outside ALP core.
