# Workspace Export and Restore

## Goal

Make learner state portable and recoverable independently of Store provider.

## Export

Conceptual command:

```
alp workspace export
```

The output is a deterministic `.alp` archive.

Archive contents include canonical learner state only.

A manifest records:

- archive format version;
- learner/workspace identity;
- state schema version;
- creation timestamp;
- canonical paths;
- SHA-256 digest per entry.

Machine-specific configuration, Git metadata, credentials, recovery journals, and rebuildable caches are excluded by default.

## Verify

Conceptual command:

```
alp workspace verify <archive>
```

Verification checks:

- archive structure;
- manifest integrity;
- per-entry digest;
- state schema compatibility;
- canonical document validity.

## Restore

Conceptual command:

```
alp workspace restore <archive>
```

Restore verifies the archive before mutating the target Store.

The target provider may be different from the source provider.

## Encryption

Production v0 does not define ALP-native archive encryption.

The archive must be treated as sensitive learner data and protected by the destination/user environment.

The format reserves room for a future encrypted envelope without changing canonical ALP contents.
