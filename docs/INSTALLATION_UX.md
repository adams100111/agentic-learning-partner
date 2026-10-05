# Installation and Setup UX

## Goal

Make ALP usable across Claude Code, Codex, and terminal workflows without custom credential management.

## Initial setup

Conceptual flow:

```
alp init
```

or equivalent agent workflow:

1. detect/verify ALP plugin/CLI;
2. locate or configure learner workspace;
3. optionally clone a supplied workspace repository;
4. validate workspace compatibility;
5. verify available domain packs;
6. show status.

## Git authentication

Use the user's existing Git/GitHub authentication.

ALP does not store or manage Git credentials.

## Domain packs

Domain packs are modular internally but bundled with ALP initially.

CLI UX may expose:

```
alp domains list
alp domain info go
```

External/installable packs can come later without changing the pack contract.
