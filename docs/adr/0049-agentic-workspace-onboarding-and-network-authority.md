# ADR-0049: Use one agentic setup workflow backed by deterministic workspace commands

**Status:** Accepted

## Context

ALP must be installable by new users and portable across devices without requiring manual Git/YAML setup.

Agent prompts alone are not a reliable owner for storage creation, provider configuration, or external side effects.

## Decision

Production v0 provides one agentic setup wizard using progressive disclosure and deterministic CLI/runtime operations.

The wizard inspects the environment and existing configuration before asking user decisions.

The flow may cover:

1. existing/new learner workspace;
2. Store provider selection;
3. local destination;
4. Git remote creation/connection when applicable;
5. sync policy;
6. existing profile/persona discovery;
7. learner/profile bootstrap only when missing;
8. optional domain onboarding.

Core deterministic operations support new/local and existing/remote workflows independently of the wizard.

When a connected forge integration can create a private Git repository, the wizard requires one explicit approval before creating the external resource. ALP receives only the resulting remote URL, never credentials.

Choosing session synchronization is durable consent for ordinary safe fetch/push operations on the configured remote/branch.

Explicit approval remains required for changing remotes, provider conversion, connecting an unverifiable remote, or destructive recovery.

## Consequences

- setup works conversationally and non-interactively;
- agents ask only unresolved decisions;
- external repository creation remains a visible side effect;
- repeated safe synchronization does not create confirmation fatigue.
