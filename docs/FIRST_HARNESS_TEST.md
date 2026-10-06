# First Real Harness Test

This test verifies that ALP can operate its own learner-state lifecycle from an actual agent harness.

## Preconditions

The reusable ALP plugin/CLI is installed from `agentic-learning-partner`.

The learner workspace contains only bootstrap configuration:

- `workspace.yaml`
- `profile/profile.yaml`
- `personas/global.yaml`
- optional domain persona such as `personas/domains/go.yaml`

There must be no pre-created competency projection, assessment, evidence, review queue, or learning plan.

## Harnesses

Run the test first in one real harness:

1. Claude Code **or**
2. Codex

Then repeat the read/continuation portion in the other harness against the same learner workspace.

## Setup

Configure the harness/project to resolve the learner workspace through one supported mechanism:

1. explicit `--workspace`;
2. project-local `.alp.yaml`;
3. `ALP_WORKSPACE`;
4. user ALP config.

Verify from the harness:

    alp workspace check
    alp validate
    alp domain info go

## First session

Invoke the installed ALP learner/persona workflow, not a hand-written diagnostic prompt.

Suggested user request:

> Use Agentic Learning Partner to begin my Go learning journey. Inspect my existing profile/personas and current workspace first. Do not assume Go competency from prior exposure. Run only the minimum high-information diagnostic needed for the first round.

Expected behavior:

1. ALP inspects existing profile/personas before asking questions.
2. It does not ask for information already present with high confidence.
3. It selects diagnostic activities from the Go domain pack.
4. It distinguishes syntax recall from engineering reasoning.
5. It does not create competency levels before actual evidence exists.
6. After learner answers, it writes schema-valid evidence.
7. It writes targeted competency assessments.
8. It rebuilds deterministic projections.
9. It produces a learner-specific review/learning plan.
10. It can explain the result with:
   - `alp status`
   - `alp competency show <id>`
   - `alp evidence show <id>`

## State assertions after first assessed round

Expected new workspace artifacts include at least:

    evidence/
    assessments/
    state/competencies.yaml

Depending on the workflow, ALP may also produce:

    state/review-queue.yaml
    state/learning-plan.yaml

Every new competency state must trace back to persisted evidence and assessment records.

No harness-specific learner-state file is allowed.

## Cross-harness continuation

Open the second harness against the same workspace.

Ask:

> Continue my Go learning from my current ALP state. First explain what ALP currently believes, why, and what you would test next.

Expected behavior:

1. The second harness reads the same canonical state.
2. It does not restart onboarding.
3. It does not rely on conversation memory from the first harness.
4. It can explain the same competency projection from evidence/assessment provenance.
5. It chooses the next activity from current gaps/review state.

## Failure conditions

The test fails if any of the following occurs:

- the harness invents competency state without evidence;
- it stores learning truth only in chat memory;
- it creates a harness-specific shadow profile/state;
- it repeats already-known bootstrap questions unnecessarily;
- it treats historical prior exposure as demonstrated current competence;
- it bypasses `alp` state mutation semantics;
- the second harness cannot continue from the first harness's persisted state;
- generated projections cannot be rebuilt from canonical evidence/assessments.

## Completion criterion

The first real harness test passes when one harness creates the first genuine learning evidence/state and another harness can continue correctly from that same workspace without shared conversation history.
