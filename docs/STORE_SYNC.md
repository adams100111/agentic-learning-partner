# Store Synchronization

## Runtime model

Git Store uses a local checkout for all runtime learner-state access.

Remote synchronization is mediated through ALP rather than raw agent-issued Git commands.

## Default sync mode

Production default:

```yaml
sync:
  mode: session
```

Session behavior:

### Start

1. inspect local workspace;
2. fetch remote;
3. semantically reconcile if remote moved;
4. validate;
5. pin the session base Workspace Revision.

If the network is unavailable and the provider supports offline operation:

- mark the session offline;
- continue from the current local revision;
- preserve all learner work locally.

### Close

1. validate staged canonical changes;
2. apply one logical ChangeSet;
3. regenerate derived state;
4. validate final workspace;
5. checkpoint locally;
6. push when network is available.

If push is unavailable, mark synchronization pending rather than losing the session.

## Sync modes

### session

Default. Pull/reconcile at session start; checkpoint/push at session close.

### manual

No implicit network synchronization. The user or agent explicitly invokes sync.

### eager

Meaningful accepted state mutations may checkpoint/synchronize before session close.

## Semantic reconciliation

ALP classifies state before reconciliation:

### Append-only canonical records

Examples:

- evidence;
- assessments.

Reconcile by stable collision-resistant identity.

Platform Account Links have a deterministic identity (`{platform, instance, platform user ID}`), so the same account linked on two devices yields one identity with different confirmation timestamps. Sync keeps the earliest confirmed link when both name the same learner, and stops for learner resolution when they name different learners.

### Human-maintained canonical documents

Examples:

- profile;
- personas.

Non-overlapping changes merge.

When both sides modify the same durable fact, provenance determines whether ALP can resolve it automatically.

Recommended precedence:

1. explicit learner correction;
2. accepted explicit learner statement;
3. demonstrated/observed durable inference;
4. agent inference/proposal.

Two incompatible explicit learner decisions require learner resolution.

### Derived state

Examples:

- competency projections;
- review queues;
- generated views.

Discard conflicting derived copies and regenerate from canonical inputs.

## Race handling

Git Store performs bounded optimistic retry.

If the remote moves between fetch/reconciliation and push:

1. fetch the new remote tip;
2. reconcile again;
3. rebuild/validate;
4. retry.

Maximum automatic attempts in production v0: 3.

After the bound is reached, return an explicit concurrency error.

## Branch model

One canonical configured branch, default `main`.

Devices do not use permanent per-device branches.

Temporary internal refs may be used during safe reconciliation but are not part of the user-facing state model.

## Credential boundary

ALP never stores Git credentials.

Git operations use existing host mechanisms such as:

- SSH agent;
- OS keychain;
- Git Credential Manager;
- configured Git credential helpers.

## Agent Git authority

Normal agent-facing ALP operations do not expose:

- force push;
- destructive reset;
- history rewriting;
- remote branch deletion;
- arbitrary ref mutation.

Humans may still use ordinary Git outside ALP for advanced recovery.
