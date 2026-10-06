# Agentic Onboarding

## Goal

A new or returning learner should be able to install ALP, create or connect state, and start learning without manually editing Git or YAML.

## Wizard behavior

The setup skill follows a frontier-driven wizard:

1. inspect installed CLI/plugin;
2. inspect existing ALP config;
3. discover existing local workspaces;
4. inspect Git/SSH/credential readiness when Git Store is relevant;
5. ask only unresolved user decisions;
6. recommend production defaults;
7. execute deterministic ALP commands;
8. validate the resulting workspace;
9. continue to persona/domain onboarding only when state is genuinely missing.

## Recommended defaults

- provider: Git Store for users who want multi-device sync;
- fallback/local-only provider: Local Store;
- sync mode: session;
- Git branch: main;
- remote visibility: private when ALP/forge integration creates it;
- local workspace root: user-local ALP data directory;
- one default named workspace.

## Existing state

Returning users connect/clone an existing workspace, validate/migrate it, and skip profile/persona discovery when those durable records already exist.

## New state

New users initialize a workspace, optionally create/connect a private Git remote, then bootstrap learner profile/persona through the learner-discovery workflow.

No competency level is inferred merely from onboarding self-report.
