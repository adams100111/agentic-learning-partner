# Agentic Setup Wizard

## Purpose

The setup wizard turns ALP installation into a usable learner workspace without requiring users to edit YAML or operate Git manually.

The wizard is an agent workflow. Workspace mutations remain deterministic CLI/runtime operations.

## Inspection first

Before asking the learner anything, inspect:

- whether `alp` is installed and callable;
- configured named workspaces;
- current/default workspace status;
- project-local `.alp.yaml`;
- existing profile/persona state;
- Git availability;
- whether a relevant forge connector is available and authenticated.

The wizard asks only decisions that remain unresolved after inspection.

## Setup branches

### Existing configured state

If a valid workspace already exists:

1. show the resolved workspace/provider;
2. validate it;
3. select it when necessary;
4. do not repeat profile/persona onboarding.

### Existing Git state

For a user with an existing learner-state remote:

1. obtain/confirm the remote URL;
2. if privacy cannot be verified, obtain the one required acknowledgement;
3. clone with `alp workspace clone`;
4. allow the CLI to validate/migrate the workspace;
5. set/use the named workspace;
6. verify status/sync.

### Existing local state

Register/use the existing workspace through supported lifecycle/configuration commands rather than copying learner state into the plugin repository.

### New local-only state

Use Local Store when the learner wants no remote synchronization.

Create the workspace with `alp workspace init --provider local`.

### New synchronized state

Recommend Git Store with session synchronization.

If a forge connector can create repositories:

1. recommend a private learner-state repository;
2. ask once for explicit approval to create the external repository;
3. use the connector to create it privately;
4. pass only the resulting Git remote URL to deterministic ALP setup;
5. initialize/connect Git Store.

If no forge connector is available, initialize local Git state and explain how to connect an existing remote without collecting credentials.

## Recommended defaults

- provider: Git Store for multi-device use;
- Git branch: `main`;
- synchronization: `session`;
- remote visibility: private;
- one learner per workspace;
- one default named workspace for ordinary users.

Local Store is the recommended alternative when remote synchronization is deliberately unwanted.

## Authorization

Selecting session synchronization is durable authorization for ordinary safe fetch/push operations against the configured branch/remote.

Ask explicitly before:

- creating a remote repository;
- changing the configured remote;
- moving to a different Store provider;
- accepting an existing remote whose privacy cannot be verified;
- destructive recovery behavior.

Do not repeatedly ask for ordinary session synchronization.

## Credentials

ALP never asks the learner to paste a PAT, SSH private key, password, or credential-helper secret into chat.

Git Store delegates authentication to the machine's Git/SSH credential infrastructure.

Forge integrations authorize their own external actions and return a remote URL, not credential material.

## Learner bootstrap

Workspace setup and learner discovery are distinct stages.

If profile/persona state already exists, use it.

For a new workspace with missing learner context, continue into the persona discovery workflow after the workspace is valid.

Historical exposure/self-report may be stored as durable learner context. It must not create competency levels without evidence.

## Completion

Setup is complete only when:

1. the named workspace is configured;
2. workspace status identifies the expected provider/workspace ID;
3. schemas validate;
4. Git Store is synchronized, or a network failure is explicitly represented as safe pending synchronization;
5. no learner state was created inside the reusable plugin repository.
