# ADR-0046: Use a provider-independent ALP workspace archive for export and restore

**Status:** Accepted

## Context

Git history is useful recovery for Git Store, but learner state must remain portable across Store providers.

Backup/restore cannot depend on a Git repository layout if Local Store and future non-Git providers are first-class.

## Decision

Production v0 provides a provider-independent `.alp` workspace archive.

The initial container format is deterministic ZIP.

The archive contains canonical learner state plus a manifest with:

- archive format version;
- workspace/learner identity;
- ALP state schema version;
- creation timestamp;
- canonical entry list;
- SHA-256 digest for each entry.

By default the archive excludes:

- Git metadata;
- machine-local configuration;
- credentials;
- recovery journals;
- rebuildable derived projections/caches.

Restore verifies:

1. archive structure;
2. entry digests;
3. workspace schema compatibility;
4. canonical state validity;

before installing/restoring the workspace into a target Store provider.

ALP-level archive encryption is not required in production v0, but the archive format must allow a future encrypted envelope without changing canonical contents.

## Consequences

- Git Store is not the only backup/migration path;
- provider migration remains possible;
- integrity verification is deterministic;
- exports are sensitive learner data and require destination protection.
