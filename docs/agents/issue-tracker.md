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

Use the `gh` CLI (the GitHub MCP integration is not reliably connected).

- Create: `gh issue create -R <owner/repo> --title ... --label ready-for-agent --body-file ...`.
- Sub-issue of a spec: `gh api -X POST repos/<owner/repo>/issues/<spec>/sub_issues -F sub_issue_id=<id>`.
- Native blocking edge: `gh api -X POST repos/<owner/repo>/issues/<n>/dependencies/blocked_by -F issue_id=<id>`.
- Both endpoints take the issue's numeric `id` (`gh api repos/<owner/repo>/issues/<n> -q .id`), not its number; they work across repos of the same owner (PyLearn tickets hang off ALP specs).

Do not maintain a duplicate local issue tracker under `.scratch/`.
