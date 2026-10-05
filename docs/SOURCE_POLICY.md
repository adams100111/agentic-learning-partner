# Source Authority and Freshness Policy

## Principle

No learning platform, prior course, AI-generated document, model memory, or learner-state file is an authoritative source for technical truth merely because it already exists.

PyLearn is ALP's first integration and validation environment. It is NOT a technical source of truth.

## Authority hierarchy

For version-sensitive technical claims, prefer evidence in this order:

1. official language/runtime/framework documentation and release notes;
2. official standards/specifications/proposals where applicable;
3. official package/library documentation and maintained upstream repositories;
4. high-quality primary maintainer material;
5. reputable secondary technical sources;
6. existing ALP/PyLearn course material;
7. model memory only as a lead to verify, not as final authority.

For learner-specific truth:

1. recent demonstrated evidence;
2. current explicit learner statements;
3. recent project/source-code evidence;
4. platform telemetry;
5. historical evaluations/profile snapshots.

Older learner documents remain evidence but may be stale.

## Freshness classes

Every significant version-sensitive claim should be classified implicitly or explicitly as one of:

- **stable concept** — unlikely to change materially (e.g. Go's basic interface satisfaction model);
- **version-sensitive language/runtime** — verify against current release docs;
- **ecosystem choice** — libraries/frameworks/tooling; re-evaluate more aggressively;
- **operational/platform** — cloud/deployment/service behavior; verify at use time;
- **security-sensitive** — authentication, crypto, security practices; verify at use time.

## Required metadata

Domain reference material should support:

- `lastVerified`;
- source URLs or source identifiers;
- version/range verified;
- whether the claim is normative, recommended, experimental, or legacy;
- optional revalidation trigger.

## Revalidation triggers

Revalidate when:

- authoring or materially revising a lesson whose stack choice is version-sensitive;
- the target language/runtime has released a new major/minor version;
- a framework/library has released a significant version;
- an existing recommendation is older than the domain pack's freshness budget;
- the learner/project reports incompatibility;
- upstream marks an API experimental/deprecated;
- ALP detects contradiction between current primary sources and existing content.

## Existing course content

Existing PyLearn or ALP content is treated as a **candidate/reference**, not truth.

Before reusing it:

1. identify claims that are stable vs version-sensitive;
2. verify version-sensitive claims;
3. keep correct material;
4. update stale recommendations;
5. record what changed and why.

Do not rewrite content merely because it is old. Age is a reason to verify, not proof of incorrectness.

## Best-practice claims

"Best practice" is especially non-authoritative.

ALP should distinguish:

- language/runtime guarantees;
- official guidance;
- ecosystem convention;
- project-specific trade-off;
- personal/team preference.

When multiple current approaches are legitimate, teaching material should explain the selection criteria rather than falsely present one as universally correct.
