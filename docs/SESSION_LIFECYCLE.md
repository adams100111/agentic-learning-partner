# Session Lifecycle

## Purpose

A session is the logical transaction/checkpoint boundary for normal ALP learning work.

## Begin

A session:

1. resolves the active workspace;
2. requires the Store capabilities needed by the configured policy;
3. acquires the workspace writer lock when mutation is expected;
4. performs provider synchronization according to policy;
5. pins a base Workspace Revision;
6. creates a machine-local recovery journal;
7. opens an isolated transaction staging area.

If Git Store synchronization cannot reach the remote and the provider supports offline operation, the session may continue offline from the current local revision.

## During the session

Learning workflows stage canonical changes through the transaction.

The canonical published workspace must not expose partially validated staged changes.

## Close

Session close:

1. validates staged canonical changes;
2. applies the logical ChangeSet;
3. regenerates derived state;
4. validates the complete resulting workspace;
5. creates a Store checkpoint;
6. writes the compact canonical session record when enabled;
7. synchronizes according to policy;
8. clears the recovery journal after the local checkpoint/sync state is safely represented;
9. releases the writer lock.

Network failure after local checkpoint marks synchronization pending rather than failing the completed learning work.

## Recovery

An interrupted uncheckpointed session is offered for resume/review or explicit discard.

An interrupted session whose checkpoint completed but whose push did not may resume synchronization automatically.

ALP never silently converts unreviewed staged work into canonical learner truth.
