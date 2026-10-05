# Source Verification Cache

## Decision

Cache compact verification metadata, not entire external documentation.

Example:

```yaml
source:
  url: https://go.dev/doc/go1.27
  verifiedAt: 2026-10-05
  version: 1.27
  contentHash: ...
  claims:
    - generic-methods-supported
```

## Behavior

Reuse cached verification while within the source's freshness policy.

Fetch authoritative source again when revalidation is due.

Do not mirror large external docs into learner state.
