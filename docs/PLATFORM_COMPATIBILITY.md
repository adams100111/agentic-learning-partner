# Platform / Domain Pack Compatibility

## Contract

A platform content-to-competency mapping declares the domain pack version range it supports.

Example:

```yaml
domain: go
packVersion: ">=1.2 <2.0"
```

## Validation

CI/adapter validation must fail clearly when:

- competency IDs no longer exist;
- pack version is outside supported range;
- mappings require an explicit migration.

## Ownership

The platform owns its mapping file.

The domain pack owns competency IDs/version/migrations.
