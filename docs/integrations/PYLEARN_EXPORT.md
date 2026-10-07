# PyLearn Activity Export and Import Contract

## Goal

Give ALP a versioned, explicit view of PyLearn learner activity without coupling ALP to PyLearn's SQLite/Drizzle schema. PyLearn may change its tables freely as long as `export:activity` keeps producing this contract. ALP never reads PyLearn's database (ADR-0059).

The raw export is not learner truth: `alp platform import` turns it into evidence through PyLearn's mapping (ADR-0058).

## `pylearn-export` v2

Schema: [`schemas/pylearn-export.schema.json`](../../schemas/pylearn-export.schema.json). One export covers one PyLearn user since a cursor. v1 (top-level `progress`/`attempts`/`quizAnswers`/... arrays, an undocumented `courses` field, no event identities) is rejected with `invalid-activity-export`.

```json
{
  "schemaVersion": 2,
  "platform": "pylearn",
  "instance": "pylearn-local",
  "exportedAt": "2026-10-07T09:00:00Z",
  "user": {"id": "usr_7f3a"},
  "cursor": {"since": null, "next": "cursor-0001"},
  "targets": [
    {
      "id": "go-alp",
      "records": [
        {
          "kind": "quiz-answer",
          "item": "go-alp-a1-context#quiz:context-cancellation/q1",
          "event": {
            "id": "quiz_answers:go-alp-a1-context#quiz:context-cancellation/q1",
            "revision": "sha256:90932ecd56d1a359359d74376f272fe8ce0433581f894c2d64acdfa8e57e09ba",
            "synthetic": true
          },
          "observedAt": "2026-10-07T08:12:00Z",
          "picked": 2,
          "correct": true
        }
      ]
    }
  ]
}
```

| Field | Meaning |
|---|---|
| `instance` | Opaque identifier of the PyLearn deployment. Part of evidence identity and of the Platform Account Link. |
| `user.id` | Opaque PyLearn `users.id`. Names, emails, PINs and other user metadata are never exported. |
| `cursor.since` | The cursor this export continues from, or `null` for a full snapshot. |
| `cursor.next` | Opaque cursor to export since next time. ALP returns it from `alp platform import`. |
| `targets[].id` | Learning Target (course) ID, as in the curriculum export. Replaces v1 `courses`. |
| `records[].item` | Declared-stable item ID from the curriculum export: lesson `<lessonId>`, quiz `<lessonId>#quiz:<concept>`, question `<lessonId>#quiz:<concept>/<questionId>`. |
| `records[].observedAt` | When the row last changed. Orders revisions; never part of identity. |

### Record kinds

| `kind` | PyLearn row | `item` | Kind fields | Evidence-relevant fields (hashed) |
|---|---|---|---|---|
| `progress` | `progress` (`userId`, `lessonId`) | `<lessonId>` | `status`: `unstarted`, `in_progress`, `completed` | `status` |
| `quiz-answer` | `quiz_answers` (`userId`, `lessonId`, `quizId`, `questionId`) | `<lessonId>#quiz:<concept>/<questionId>` | `picked` (integer), `correct` (boolean) | `correct`, `picked` |
| `reflection` | `reflections` (`userId`, `lessonId`) | `<lessonId>` | `text` | `text` |
| `attempt` | `attempts` (`userId`, `lessonId`, `variantId`) | `<lessonId>` | `variant`, `status`: `passed`, `failed`, `error`; optional `failure` | `status`, `failure` |

`attempts` has no writer in PyLearn today, so `export:activity` excludes attempts until a real writer exists (spec #66); the kind is defined so that a writer can be added without a schema change. `concept_mastery` and bookmarks are not exported: concept mastery is a platform-local rollup keyed by free-text concept tags, not ALP competency truth, and bookmarks reference heading-derived section IDs, which are not declared-stable.

### Synthetic Event Identity

PyLearn activity tables hold latest-state rows with no event IDs, so every record carries a Synthetic Event Identity (ADR-0059) with `"synthetic": true`:

- **`event.id`** — the row key: `<table>:<item>` for `progress`, `quiz_answers` and `reflections` (for example `progress:go-alp-a1-context`), and `attempts:<lessonId>/<variantId>` for attempts. It stays the same for the row's lifetime.
- **`event.revision`** — `"sha256:" + hex(sha256(canonical JSON))` of the record's evidence-relevant fields from the table above: a JSON object with keys sorted and no whitespace, values as exported. For example `{"correct":true,"picked":2}` hashes to `sha256:90932ecd56d1a359359d74376f272fe8ce0433581f894c2d64acdfa8e57e09ba`.

Timestamps, `userId` and positions (`lastPositionMs`, `lastSectionId`) are never hashed: an unchanged row re-exported keeps its revision and is skipped; a changed row gets a new revision and supersedes the earlier evidence.

### Cursor

The Activity Source contract is "records since cursor". `export:activity --since <cursor>` emits only rows changed after that cursor, with `cursor.since` set to it; without `--since` it emits a full snapshot with `cursor.since: null`. Cursors are opaque to ALP.

`alp platform import --cursor <cursor>` accepts an incremental export only when its `cursor.since` equals the given cursor; otherwise it fails with `cursor-mismatch`, because activity between the two cursors would be missed. A full snapshot is accepted with or without a cursor.

## Platform Account Link

Imports are refused (`platform-account-not-linked`) unless the workspace holds a learner-confirmed link for the export's `{platform, instance, user.id}` (Q36). The learner records it once:

```sh
alp platform account link --adapter pylearn --instance pylearn-local --user usr_7f3a --confirm
```

The link (`schemas/platform-account-link.schema.json`) is an append-only workspace record in `platform-accounts/`. `--confirm` states the learner's explicit confirmation; agents must not link accounts on their own. Learners are never matched by email or name.

The link's ID is derived from `{platform, instance, user.id}`, so linking the same account on two devices creates the same record with different confirmation timestamps. Git Store sync treats such links as equivalent when they name the same learner and keeps the earliest confirmed record on every device; the same account linked to two different learners is a sync conflict that needs the learner's resolution. Linking an account that is already linked reports `already-linked` and writes nothing.

## Import

```sh
alp platform import --adapter pylearn --target go-alp \
  --curriculum curriculum-export.json --mapping go-alp.mapping.yaml \
  --export activity-export.json [--cursor <cursor>]
```

The command requires the Activity Source, Content Mapper and Curriculum Reader capabilities, refuses an invalid mapping (`invalid-mapping`), grades evidence by Mapping Role (see [`../PLATFORM_MAPPING.md`](../PLATFORM_MAPPING.md#evidence-grading-on-import-alp-platform-import)), and writes all new evidence through one Store transaction. An import that adds nothing leaves the workspace revision unchanged.

Evidence identity is a hash of platform, instance, target, workspace learner, platform user, item, event ID and event revision (per competency domain and evidence grade). Timestamps are never part of it. For each record:

| Outcome | When |
|---|---|
| `imported` | New evidence for an event not seen before. |
| `skipped` (`already-imported`) | The event's active evidence already has this revision. |
| `skipped` (`stale-revision`) | A different revision whose `observedAt` is earlier than the event's active evidence (for example an old snapshot re-imported). |
| `skipped` (`revision-conflict`) | A different revision with the same `observedAt` as the active evidence; ALP does not guess which is newer. |
| `superseded` | A different revision whose `observedAt` is later than the event's active evidence: new evidence whose `supersedes` lists the event's previously active evidence. An event that returns to an earlier revision content is recorded again the same way. |
| `unmapped` (`not-mapped` / `not-in-curriculum`) | No mapped item at or above the record's item, or the item is not in the curriculum export. |

**Revision order.** A synthetic `event.revision` is a content hash and has no order of its own, so "later revision" means later `observedAt`: ALP compares a record's `observedAt` with that of the event's active evidence (ADR-0059, Notes 2026-10-07). Later supersedes, earlier is `stale-revision`, equal with a different revision is `revision-conflict`. A platform must therefore set `observedAt` to when the row last changed, never to export time.

Output is deterministic JSON: the target, the linked account, the curriculum export and mapping hashes, the cursor range (`since`, `next`), `counts` (`records`, `imported`, `skipped`, `superseded`, `unmapped`) and one entry per record with its outcome, reason, mapped item, written evidence IDs and superseded IDs. Records are reported sorted by item, event ID and time.

## Signal policy

PyLearn's signal-kind policy (what a record shows before Mapping Roles apply):

| Kind | Evidence type | Result | Base strength | Assessable |
|---|---|---|---|---|
| `progress` | `platform-event` | `neutral` | weak | no |
| `quiz-answer`, correct | `quiz` | `pass` | moderate | yes |
| `quiz-answer`, incorrect | `quiz` | `fail` (`conceptual-miss`) | weak | yes |
| `reflection` | `reflection` | `neutral` | weak | no |
| `attempt`, passed | `exercise` | `pass` | moderate | yes |
| `attempt`, failed | `exercise` | `fail` (`implementation-miss`) | moderate | yes |
| `attempt`, error | `exercise` | `fail` (`implementation-miss`) | weak | yes |

Lesson progress is navigation/completion evidence only and never sufficient for promotion. A passing test is not automatically mastery. Reflections are learner self-report.

### Project/source code

When PyLearn references a learner project/repository, ALP should inspect the relevant commit/diff and produce separate repository evidence. This can be stronger than platform exercise evidence because it shows independent integration, design, verification, and maintenance.

## Identity and mapping

PyLearn-to-ALP mapping is owned by the PyLearn content repository (ADR-0017), keyed by declared-stable identifiers (ADR-0057) with mapping roles (ADR-0058). It uses platform mapping schema v2; see [`../PLATFORM_MAPPING.md`](../PLATFORM_MAPPING.md) for the format and `alp platform mapping validate`.

Do not encode PyLearn lesson IDs into ALP's competency taxonomy.

## Privacy

The export contains only learning-relevant data. Do not export credentials, auth secrets, PIN hashes, names, emails, unrelated user metadata, narration API keys, or infrastructure secrets.

## Write-back boundary

ALP does not write learner competency directly into PyLearn's database.

Write-side outputs are:

- adaptation proposals;
- optional generated reinforcement/content branches;
- optional summarized learner-state views if PyLearn later chooses to display them.

ALP remains the authority for its own derived learner state; PyLearn remains the authority for its own application activity records.
