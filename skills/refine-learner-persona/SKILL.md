---
name: refine-learner-persona
description: Refine an existing learner profile or persona from explicit corrections, new evidence, or contradictions.
---

# Refine Learner Persona

Use this when the learner corrects ALP's understanding, asks why a teaching preference exists, or new durable information materially changes future teaching.

## Workflow

0. Use an active ALP session for durable persona/profile edits; do not edit synchronized canonical state outside the Store transaction path.

1. Inspect current persona with `alp persona show --format markdown`.
2. Read only the evidence/source material relevant to the disputed or unknown field.
3. Distinguish:
   - explicit learner correction;
   - demonstrated durable pattern;
   - agent inference;
   - temporary session condition.
4. Apply explicit learner corrections directly with provenance and show the resulting change.
5. For materially consequential inferred changes, propose the diff and obtain learner confirmation.
6. Keep global facts in the global profile/persona and domain-only behavior in the domain persona.
7. Stage accepted profile/persona updates with `alp session put`, then close the session and run `alp validate` after publication.

Never use a one-off failure to rewrite a durable persona. Never use persona data to silently promote competency.
