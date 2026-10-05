# PyLearn Export and Evidence Mapping Contract

## Goal

Provide ALP a versioned, explicit snapshot of PyLearn learner activity without coupling ALP directly to PyLearn's SQLite/Drizzle implementation.

PyLearn remains free to change its internal schema as long as it can produce the export contract.

## Export shape

Initial conceptual payload:

```json
{
  "schemaVersion": 1,
  "exportedAt": "2026-10-05T00:00:00Z",
  "learner": {},
  "courses": [],
  "progress": [],
  "attempts": [],
  "conceptMastery": [],
  "quizAnswers": [],
  "reflections": [],
  "bookmarks": []
}
```

The adapter normalizes these signals into ALP evidence. The raw export itself is not learner truth.

## Signal mapping

### Lesson progress

PyLearn signal:

- unstarted;
- in progress;
- completed;
- last position/section.

ALP interpretation:

- navigation/completion evidence only;
- weak evidence of competency;
- useful for sequencing and detecting abandoned content;
- never sufficient for promotion.

### Exercise attempts

PyLearn signal:

- code;
- pass/fail/error;
- failure message;
- stdout;
- lesson/variant identity.

ALP interpretation:

- useful behavioral evidence;
- strength depends on task design and independence;
- repeated retries may reveal a learning pattern;
- test pass is not automatically mastery.

The content/variant must map to competency IDs before the evidence affects projection.

### Concept mastery

PyLearn's `concept_mastery` table is itself a rollup.

ALP interpretation:

- imported derived signal;
- never treated as canonical ALP competency;
- useful for prioritizing which raw attempts/evidence to inspect.

### Quiz answers

ALP interpretation:

- recognition/reasoning evidence depending on question type;
- typically weaker than implementation/debugging evidence;
- useful for misconceptions and spaced review.

### Reflections

ALP interpretation:

- learner self-report/metacognition;
- useful for confusion, perceived difficulty, goals, and persona refinement;
- not strong implementation evidence by itself.

### Bookmarks/pins

ALP interpretation:

- interest/friction/context signal;
- not competency evidence by default.

### Project/source code

When PyLearn references a learner project/repository, ALP should inspect the relevant commit/diff and produce separate repository evidence.

This can be stronger than platform exercise evidence because it shows independent integration, design, verification, and maintenance.

## Identity and mapping

Every exported lesson/exercise/quiz concept that should affect competency projection needs a stable content identifier.

PyLearn-to-ALP mapping should live in the adapter/configuration, for example:

```yaml
phase-a-l3.error-wrapping:
  - go.language.errors
```

for Go, or the equivalent Python domain IDs.

Do not encode PyLearn lesson IDs into ALP's competency taxonomy.

## Incremental export

v0 may export a complete snapshot.

Later versions should support a cursor/timestamp for incremental synchronization to avoid repeatedly transferring unchanged history.

## Privacy

The export should contain only learning-relevant data.

Do not export credentials, auth secrets, unrelated user metadata, narration API keys, or infrastructure secrets.

## Write-back boundary

ALP does not write learner competency directly into PyLearn's database.

Write-side outputs are:

- adaptation proposals;
- optional generated reinforcement/content branches;
- optional summarized learner-state views if PyLearn later chooses to display them.

ALP remains the authority for its own derived learner state; PyLearn remains the authority for its own application activity records.
