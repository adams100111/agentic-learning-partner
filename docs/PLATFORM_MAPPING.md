# Platform-to-Competency Mapping

## Ownership

The learning platform owns the mapping from its own content IDs to ALP competency IDs (ADR-0017). The mapping changes in the same PR as the content it describes, because the platform controls content identity, lesson/exercise changes, and what a piece of content actually teaches or tests.

The learner workspace does not own platform content mappings. Platform-local tags (for example PyLearn quiz `concept` strings or `concept_mastery` keys) are not ALP competency IDs and are not the mapping.

## Format: schema v2

`schemas/platform-mapping.schema.json` (ADR-0058). One mapping file covers one Learning Target. YAML or JSON (chosen by file extension).

```yaml
schemaVersion: 2
platform: pylearn
target: go
packs:
  - domain: go
    packVersion: ">=0.1.0 <0.2.0"   # semver range the mapping was written against
entries:
  - item: go-b1-goroutines          # a declared-stable item ID of the target
    competencies:
      - id: go.concurrency.goroutines
        role: teaches
  - item: go-goroutine-lifetimes
    strengthCeiling: moderate        # optional cap on evidence strength
    competencies:
      - id: go.concurrency.goroutines
        role: assesses
      - id: go.concurrency.races-deadlocks-leaks
        role: assesses
```

- **`item`** must be a declared-stable identifier (ADR-0057) that the target's curriculum export lists. For PyLearn: lesson `id`, explicit Scene `id`, quiz and question IDs. Heading-derived section slugs and positional scene IDs are never listed, so they are rejected.
- **`role`** is the Mapping Role: `teaches`, `reinforces`, or `assesses`. Only `assesses` activity can yield assessment-grade evidence; `teaches`/`reinforces` activity yields exposure or practice evidence.
- **`strengthCeiling`** (`weak`, `moderate`, `strong`, `production`) caps evidence strength. The mapping never sets strength itself: strength comes from the adapter's signal-kind policy and the observed result.
- **`packs`** declares each domain pack used, with a semver `packVersion` range. A competency belongs to the declared pack whose domain is its ID prefix (the domain-prefix check is the only place ALP interprets an ID's structure).

### Schema v1 is rejected

v1 (a flat `competencies[]` list per `contentId`) carries no Mapping Roles. Roles cannot be inferred without guessing whether content teaches or assesses, so ALP does not migrate v1: it reports `unsupported-mapping-version` and the file must be rewritten as v2.

## Validation: `alp platform mapping validate`

```sh
alp platform mapping validate --adapter pylearn --target go \
  --curriculum curriculum-export.json --mapping go.mapping.yaml
```

The adapter must declare the Content Mapper and Curriculum Reader capabilities. The curriculum export (`schemas/platform-curriculum-export.schema.json`) is the only source of which items exist and are declared-stable; ALP never reads the platform's repository.

Output is deterministic JSON: the mapping's content hash, the curriculum export's hash, each declared pack with its loaded version and compatibility, every entry with its item identity, kind and resolved competencies (`resolvedId` is the current pack ID to use), and `problems[]`. Every problem has a severity, a specific `code`, a JSON Pointer `path` into the mapping, and a message. Exit code 0 means valid (warnings allowed); 1 means at least one error; 2 is a usage error.

Competencies resolve through the domain pack's migrations:

| Pack migration | Result |
|---|---|
| none (current ID) | valid |
| `rename` | valid, `competency-renamed` warning naming the new ID |
| `split` | `competency-split` error: ALP will not guess which part the item covers |
| `merge` | `competency-merged` error: merged competencies require reassessment |
| `reassess` | `competency-reassess` error |
| `remove` | `competency-removed` error |
| unknown, no migration | `unknown-competency` error |

Other error codes: `unreadable-mapping`, `unsupported-mapping-version`, `schema`, `platform-mismatch`, `target-mismatch`, `duplicate-pack`, `unknown-domain-pack`, `invalid-pack-version-range`, `pack-version-out-of-range`, `competency-not-resolved` (its pack is unavailable or out of range), `unstable-identifier`, `duplicate-item`, `undeclared-domain`, `duplicate-competency`, `invalid-pack-migration`. Schema errors stop validation before semantic checks; all other problems are reported together.
