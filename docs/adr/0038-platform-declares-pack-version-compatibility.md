# ADR-0038: Platform mappings declare domain-pack compatibility ranges

**Status:** Accepted

## Decision

Platform content mappings include compatible domain-pack version ranges. CI/adapter validation enforces compatibility.

## Consequences

- breaking taxonomy changes cannot silently corrupt mappings;
- platform/domain upgrades become explicit.
