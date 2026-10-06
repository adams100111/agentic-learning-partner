---
name: discover-learner-persona
description: Discover a learner's durable profile and teaching persona through adaptive, evidence-aware interviewing.
---

# Discover Learner Persona

Use this when the learner is new to ALP or explicitly asks for a fresh persona discovery.

## Workflow

1. Resolve and validate the learner workspace with `alp workspace check`.
2. Inspect existing profile/persona files and relevant connected repositories/documents before asking questions.
3. Read `docs/PERSONA_WIZARD.md` for the interview rounds and anti-overfitting rules.
4. Build an uncertainty list. Ask only questions whose answers can materially change teaching, planning, or assessment.
5. Prefer concrete shipped-work examples over numeric self-ratings.
6. When useful, use small domain probes to separate language recall from engineering ability.
7. Synthesize:
   - durable learner facts;
   - global teaching preferences;
   - domain-specific overrides;
   - unresolved uncertainty.
8. Show the material proposed profile/persona diff before persisting inferred consequential changes.
9. Explicit learner corrections may be applied directly with provenance.
10. Validate changed canonical files with `alp validate`.

Do not derive competency levels directly from persona answers. Demonstrated probes become evidence through the evidence contract.

Keep the resulting persona structured and compact; do not store the interview transcript.
