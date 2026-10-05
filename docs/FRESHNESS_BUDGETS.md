# Source Freshness Budgets

## Purpose

Different technical claims drift at different rates.

## Default classes

### stable-concept

Revalidate on:

- major semantic contradiction;
- relevant language/spec change.

### version-sensitive-language-runtime

Revalidate when:

- target runtime/language version changes;
- upstream release notes materially affect the claim.

### ecosystem-choice

Default review budget: approximately 90 days, configurable by domain/reference.

### operational-platform

Default review budget: approximately 30 days, configurable.

### security-sensitive

Verify at use/authoring time against current authoritative sources.

## Notes

Budgets are triggers to verify, not proof that content is wrong.

Domain packs may override defaults.
