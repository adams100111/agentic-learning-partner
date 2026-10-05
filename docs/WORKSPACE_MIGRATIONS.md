# Workspace Schema Migrations

## Principle

Workspace upgrades must be explicit, inspectable, and recoverable.

## Commands

Conceptual CLI:

```
alp workspace check
alp workspace migrate --dry-run
alp workspace migrate
```

## Breaking migration workflow

1. validate current workspace;
2. create/check Git checkpoint;
3. show migration plan;
4. migrate canonical state;
5. rebuild derived projections;
6. validate all invariants;
7. commit migration separately.

No silent destructive migration.

Lossless compatible changes may be automated when safe.
