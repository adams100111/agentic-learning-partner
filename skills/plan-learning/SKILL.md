---
name: plan-learning
description: Create the next learner-specific plan from current competency state, goals, review needs, and available content.
---

# Plan Learning

Use this to decide what the learner should do next.

## Workflow

1. Build compact context with `alp context build --task plan-learning --domain <domain>`.
2. Inspect current state with `alp status`.
3. Use the domain competency graph, current gaps, review needs, project relevance, and learner goals.
4. Prefer the smallest next activity that produces useful evidence.
5. For platform content recommend one consumption mode:
   - skip;
   - skim;
   - challenge-only;
   - full.
6. Do not interpret platform unlock/order constraints as pedagogical truth.
7. Treat time as freshness/review pressure, not automatic competency downgrade.
8. Keep the plan as learner-workspace derived state, not as a platform-owned curriculum rewrite.

Read `docs/LEARNING_PLAN.md` and `docs/REVIEW_AND_DECAY.md` when scheduling or platform sequencing decisions are material.
