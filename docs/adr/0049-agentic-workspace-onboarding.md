# ADR-0049: Use one agentic setup wizard backed by deterministic workspace commands

**Status:** Accepted

## Context

Portable ALP installation is incomplete if users must manually clone repositories, edit config paths, initialize state, or know provider details.

At the same time, critical workspace mutations must not exist only in prompt logic.

## Decision

Production v0 provides one setup wizard that inspects existing state/environment first and asks only unresolved user decisions.

The wizard stages its flow:

1. environment/installation inspection;
2. existing workspace/config detection;
3. workspace/provider choice;
4. remote/synchronization setup when relevant;
5. existing learner state detection;
6. profile/persona bootstrap only when needed;
7. optional first domain onboarding;
8. readiness summary.

Deterministic CLI/runtime commands perform all mutations.

Production v0 supports first-class operations for:

- initialize workspace;
- clone/connect existing workspace;
- select/switch named workspace;
- inspect workspace/provider status;
- synchronize supported providers;
- export/verify/restore;
- move/convert workspace between providers.

If a connected forge integration can create a private repository, the wizard asks for one explicit approval for that external side effect. The connector creates the private repository and returns a Git remote URL. ALP core remains forge-agnostic.

Selecting session synchronization grants durable consent for ordinary safe fetch/push operations against the configured remote and branch. Changing remotes, converting providers, accepting unverifiable privacy, or performing destructive recovery remains an explicit decision.

## Consequences

- installation becomes agentic without hiding state semantics in prompts;
- existing-state users skip unnecessary persona onboarding;
- GitHub/GitLab integrations improve setup without becoming Store providers;
- safe routine synchronization does not require repetitive approvals.
