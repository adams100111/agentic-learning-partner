# Persona Discovery Wizard

## Goal

Create or refine a high-value learner persona through adaptive conversation, repository/document inspection, and small diagnostic probes.

The wizard should behave like an expert interviewer, not a static form.

## Modes

- initial persona discovery;
- profile refinement;
- domain onboarding;
- profile review;
- document/repository-assisted discovery;
- contradiction resolution.

## Before asking questions

The wizard MUST first inspect available:

- current learner profile;
- personas;
- competency state;
- recent evidence;
- connected project/repository evidence;
- relevant platform profile/evaluation data.

Do not ask for information already known with sufficient confidence.

## Interview strategy

Prioritize **high-information unknowns** and contradictions.

### Round 1 — identity and goals

Discover:

- professional role/level;
- target outcome;
- motivation;
- success definition;
- target domain;
- intended real-world use.

### Round 2 — technical experience

Discover relative strength through concrete shipped-work evidence, not only ratings:

- languages/frameworks;
- architecture/system design;
- databases;
- cloud/infrastructure;
- testing/operations;
- prior target-domain exposure.

### Round 3 — learning behavior

Discover:

- what causes disengagement;
- preferred pace;
- project vs conceptual balance;
- active-recall tolerance;
- syntax-support needs;
- preferred feedback style;
- useful/annoying analogy patterns.

### Round 4 — grill

Challenge claims and contradictions.

Examples:

- "You say you are strong in NestJS. What kind of production systems have you shipped with it?"
- "You say you remember no Go. Can you still reason about this handler?"
- "You prefer real projects. Do short conceptual models help before implementation?"
- "Which analogy made .NET click for you?"

The purpose is not adversarial pressure; it is calibration.

### Round 5 — domain probes

Use compact tasks to separate:

- syntax recall;
- mental model;
- idiomatic/domain knowledge;
- general engineering ability.

Possible task forms:

- code reading;
- bug finding;
- tiny implementation;
- architecture choice;
- debugging;
- production scenario.

### Round 6 — synthesis

Present a concise interpretation:

- durable identity;
- current strengths;
- likely risks;
- recommended teaching behavior;
- domain-specific assumptions;
- unresolved uncertainty.

Then generate structured proposed updates.

## Grill-with-docs behavior

When repositories or documents are available:

1. inspect them before interviewing;
2. extract candidate evidence;
3. identify stale vs current facts;
4. ask only unresolved questions;
5. challenge conflicts between self-report and artifacts;
6. persist evidence separately from persona;
7. update persona only where the evidence supports a durable trait.

## Multi-file transaction

A wizard may update several files in one logical transaction:

```
learner/profile/identity.yaml
learner/profile/experience.yaml
learner/profile/goals.yaml
learner/profile/preferences.yaml
learner/personas/global.yaml
learner/personas/domains/go.yaml
```

Competency evidence remains in the evidence store, not embedded into persona files.

## Avoid overfitting

Do not:

- turn one answer into a permanent trait;
- store irrelevant personal information;
- confuse preference with competency;
- overwrite explicit learner statements using weak inference;
- infer general engineering weakness from syntax failure;
- create a giant prose persona when structured fields suffice.

## Token-efficiency rules

- ask only high-information questions;
- stop a round when uncertainty is sufficiently reduced;
- summarize prior answers rather than replaying the transcript;
- store structured results, not the full conversation;
- retrieve supporting evidence only when needed;
- generate compact effective personas for future sessions.

## Output

The wizard should end with:

- structured persona/profile updates;
- provenance;
- confidence/uncertainty where useful;
- any new competency evidence;
- a concise human-readable summary;
- the effective domain persona for the next learning session.
