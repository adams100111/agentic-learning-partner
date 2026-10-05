# ADR-0039: Use a spec and ticket-graph workflow for implementation

**Status:** Accepted

## Context

ALP is large enough to span multiple implementation sessions and contains several independently buildable vertical slices with real dependencies.

## Decision

Use a workflow equivalent to:

```
grill-with-docs -> to-spec -> to-tickets -> implement-spec -> code-review -> retro
```

Tickets are vertical slices with explicit blocking edges. Parallel implementation uses isolated branches/worktrees and converges on one integration branch.

TDD is used at pre-agreed public seams.

## Consequences

- implementation context stays smaller per agent;
- independent work can proceed concurrently;
- the spec remains the requirements anchor;
- parallelism is avoided where merge/contention risk outweighs benefit;
- review and retrospective are explicit closing stages.
