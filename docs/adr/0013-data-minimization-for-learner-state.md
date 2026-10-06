# ADR-0013: Persist only learning-relevant personal data

**Status:** Accepted

## Context

Persona discovery, repository analysis, and platform adapters can observe much more personal or organizational information than is required to improve learning.

A private repository reduces exposure but does not justify retaining irrelevant data.

## Decision

Persist learner information only when it materially supports teaching, assessment, planning, continuity, or learner-requested personalization.

Platform adapters use allowlisted learning fields rather than raw-record dumps.

Secrets and credentials are excluded. Other sensitive or unrelated information is excluded by default. Human-readable generated views expose only what is needed for the view and avoid unnecessary raw sensitive provenance.

## Consequences

- persona wizard must justify durable fields;
- adapters need explicit export/mapping contracts;
- repository/document inspection may use transient information without persisting it;
- deletion/correction workflows still need to be defined.
