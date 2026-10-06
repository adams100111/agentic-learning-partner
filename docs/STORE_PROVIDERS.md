# Store Providers

## Purpose

A Store is ALP's persistence boundary for one learner workspace.

The learning engine depends on provider-independent Store semantics. Provider-specific capabilities are requested only by workflows that require them.

## Production v0 providers

### Git Store

The default production provider.

A Git Store:

- uses a local working tree as the runtime workspace;
- uses Git commits as durable revisions/checkpoints;
- supports offline reads/writes;
- synchronizes through an ordinary Git remote;
- supports multi-device reconciliation;
- uses existing host Git/SSH credentials.

Git forge APIs are not required for runtime state access.

### Local Store

A production local-only provider using the same canonical learner-state contracts.

It supports:

- persistence;
- local atomic state operations;
- offline use;
- workspace validation;
- provider-independent export/restore.

It does not claim remote synchronization or Git history capabilities.

## Specified future providers

The provider contract must remain capable of supporting, without implementing in production v0:

### S3-compatible

Examples include AWS S3, Cloudflare R2, MinIO, Backblaze B2 S3 API, and Wasabi.

A future implementation will need ALP-native immutable revision manifests because object storage does not provide a Git commit graph.

### WebDAV

Targets include Nextcloud, ownCloud, Synology, and generic WebDAV servers.

Revision/history guarantees are provider-dependent and must be represented through capabilities rather than assumed.

### Remote ALP API

A future managed/hosted provider accessed through an authenticated ALP state API.

The API provider must preserve the same logical evidence/assessment/projection model as local providers.

### PostgreSQL-backed server provider

A future server-side implementation for hosted or organization deployments.

PostgreSQL is an implementation choice behind Store semantics, not a portable end-user workspace format.

## Capability model

Providers advertise capabilities rather than being special-cased throughout the engine.

Candidate capabilities include:

- persistence;
- optimistic concurrency;
- atomic checkpoint;
- revisions;
- history;
- offline access;
- synchronization;
- multi-device;
- remote management.

A workflow must fail clearly when its required capability is unavailable.

## Forge integrations

GitHub, GitLab, Gitea, and Forgejo are not Store providers.

They may participate in onboarding by creating a private Git repository and returning a Git remote URL. Runtime learner-state access remains the responsibility of Git Store.
