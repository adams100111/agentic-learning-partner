# ADR-0049: Use one agentic setup wizard backed by deterministic workspace commands

**Status:** Accepted

## Context

Portable ALP installation is incomplete if users must manually clone repositories, edit config paths, initialize state, or know provider details.

At the same time, critical workspace mutations must not exist only in prompt logic. Agent prompts alone are not a reliable owner for storage creation, provider configuration, or external side effects.

Existing users should not repeat learner onboarding when connecting an existing workspace, while new users need provider/workspace setup plus profile/persona bootstrap.

## Decision

Production v0 provides one setup wizard that inspects existing state/environment first, uses progressive disclosure, recommends defaults, and asks only unresolved user decisions.

The wizard stages its flow:

1. environment/installation inspection, including Git and available forge integrations;
2. existing workspace/config detection;
3. workspace/provider and local destination choice;
4. remote/synchronization setup when relevant;
5. existing learner state detection;
6. profile/persona bootstrap only when needed;
7. optional first domain onboarding;
8. readiness summary.

Deterministic CLI/runtime commands perform all mutations; prompt logic never owns setup state transitions. These operations support new/local and existing/remote workflows independently of the wizard.

Production v0 supports first-class operations for:

- initialize workspace;
- clone/connect existing workspace;
- select/switch named workspace;
- inspect workspace/provider status;
- synchronize supported providers;
- export/verify/restore;
- move/convert workspace between providers.

If a connected forge integration can create a private repository, the wizard asks for one explicit approval for that external side effect. The connector creates the private repository and returns a Git remote URL; ALP receives only that URL, never credentials. ALP core remains forge-agnostic.

Selecting session synchronization grants durable consent for ordinary safe fetch/reconcile/push operations against the configured remote and branch. Changing remotes, converting providers, accepting unverifiable privacy, or performing destructive recovery remains an explicit decision.

## Consequences

- installation becomes agentic without hiding state semantics in prompts;
- existing-state users skip unnecessary persona onboarding;
- GitHub/GitLab integrations improve setup without becoming Store providers;
- external repository creation remains a visible side effect, and forge credentials stay outside ALP core;
- setup can run conversationally or non-interactively through the same deterministic commands;
- safe routine synchronization does not require repetitive approvals.
