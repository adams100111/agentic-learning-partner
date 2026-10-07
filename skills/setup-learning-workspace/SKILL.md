---
name: setup-learning-workspace
description: Set up or reconnect Agentic Learning Partner learner state, including local or Git-backed multi-device workspaces.
---

# Set Up ALP Learner State

Use this when ALP is newly installed, no usable learner workspace is configured, the user wants to reconnect existing state, or the user wants to add/move a workspace.

## Workflow

1. Inspect before asking:
   - verify `alp` is available and `alp version --json` reports `v<plugin version>` (the `version` in `.claude-plugin/plugin.json` at the plugin root, two directories above this file); when it is missing or differs, run `bash "<plugin root>/scripts/get.sh" v<plugin version>` (details and the tag-pinned `curl` fallback: `docs/CLI_PREREQUISITE.md`);
   - run `alp workspace list` and `alp workspace status` when configuration exists;
   - inspect the current project for `.alp.yaml`;
   - detect whether relevant profile/persona state already exists;
   - determine whether Git and an authenticated forge connector are available when a synced workspace is desired.

2. Classify the setup branch:
   - existing configured workspace → validate/use it;
   - existing remote learner-state repository → clone it;
   - existing local learner workspace → register/use it;
   - new learner → create Local Store or Git Store.

3. Recommend defaults:
   - Git Store for normal multi-device use;
   - Local Store when the user explicitly wants local-only state;
   - `session` synchronization for Git Store;
   - private remote;
   - default local workspace location under the user's ALP data directory.

4. Ask only unresolved decisions. Do not ask for facts already discovered. If a forge connector can create a private repository, obtain one explicit approval before creating it.

5. Execute deterministic ALP commands for state mutations. Typical operations are:
   - `alp workspace init`
   - `alp workspace connect`
   - `alp workspace clone`
   - `alp workspace use`
   - `alp workspace status`
   - `alp workspace sync`
   - `alp workspace move`

6. Never request, display, persist, or copy Git credentials. Git Store uses the host Git/SSH credential infrastructure. A forge connector may return a remote URL after authorized repository creation.

7. Existing-state users keep their existing profile/persona and skip persona discovery unless state is missing or the user asks to refine it.

8. New users continue to learner/profile discovery only after workspace setup succeeds. Do not create competency/evidence state merely from setup answers.

9. Finish only when:
   - the named workspace resolves;
   - `alp workspace status` succeeds;
   - `alp validate` succeeds;
   - Git Store sync succeeds or is explicitly reported as safely pending/offline.

Read `docs/INSTALLATION_UX.md` when provider/remote decisions are unresolved. Read `docs/STORE_PROVIDERS.md` and `docs/STORE_SYNC.md` only when explaining provider or synchronization behavior.
