# Issue Tracker

Implementation work is tracked in GitHub Issues for `adams100111/agentic-learning-partner`.

## Workflow

- Specs are published as GitHub issues.
- Implementation tickets are published as separate GitHub issues.
- Tickets declare blocking relationships in their issue body.
- A parent spec issue remains open until its implementation graph is complete.
- Pull requests reference the spec and relevant tickets.
- Issues produced from an approved spec are agent-ready by construction; they do not require a separate triage pass.

## Tooling

Use the connected GitHub integration/API for issue creation, updates, and relationships where supported.

Do not maintain a duplicate local issue tracker under `.scratch/`.
