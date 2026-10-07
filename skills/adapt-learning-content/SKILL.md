---
name: adapt-learning-content
description: Propose evidence-linked changes to learning content while separating learner adaptation, technical freshness, and curriculum design.
---

# Adapt Learning Content

Use this when existing content is mismatched to the learner, technically stale, or structurally weak.

Prerequisite: the `alp` CLI must match the plugin version. When `alp` is missing or errors, follow `docs/CLI_PREREQUISITE.md` in the plugin root first.

## Workflow

1. Classify the cause:
   - learner adaptation;
   - technical freshness;
   - curriculum design.
2. For learner adaptation, cite the relevant learner evidence/assessment.
3. For technical freshness, verify the claim under `docs/SOURCE_POLICY.md` and the domain freshness rules.
4. For curriculum design, map the issue to domain competencies/prerequisites.
5. Choose the smallest justified intervention:
   - review only;
   - analogy change;
   - trip-wire;
   - micro-exercise;
   - worked example;
   - section revision;
   - lesson restructure;
   - new lesson.
6. Session-local hints/next exercises may adapt immediately.
7. Durable platform content changes should become an adaptation proposal and normal branch/PR change, not an uncontrolled direct rewrite.
8. Respect platform ownership of content-to-competency mappings.

Read `docs/CONTENT_ADAPTATION.md`, `docs/SOURCE_POLICY.md`, and the platform integration contract only as needed.
