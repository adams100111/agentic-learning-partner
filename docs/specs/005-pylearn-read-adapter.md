# Spec 005 — PyLearn Read Adapter

**Status:** Draft-ready for implementation planning

## Objective

Use PyLearn as ALP's first real platform adapter without making ALP dependent on PyLearn internals or treating PyLearn content as technical truth.

## Input classes

1. versioned PyLearn activity export;
2. repository-authored learner documents;
3. course constitution/curriculum/content metadata;
4. learner project/source-code evidence.

## Behavior

### Normalize activity

Convert platform signals into ALP evidence candidates with:

- platform source reference;
- content ID;
- mapped competency IDs;
- appropriate evidence type/strength;
- timestamp.

### Detect stale learner documents

Compare dated profile/evaluation docs with current ALP persona/evidence and flag conflicts.

Do not silently overwrite either side.

### Content freshness

Course material can be audited under ALP's source policy.

Age triggers verification; it does not prove incorrectness.

### Recommendations

Read-only adapter output may include:

- learner-state proposals;
- reinforcement proposals;
- content freshness flags;
- curriculum gap flags;
- persona/profile conflict flags.

## Acceptance tests

- PyLearn `completed` lesson does not auto-promote a competency;
- a failed exercise can become targeted reinforcement evidence;
- concept mastery is recognized as a derived PyLearn signal;
- old profile statements can be flagged against newer explicit learner state;
- technical claims from PyLearn are not accepted without required source verification;
- ALP competency IDs remain independent of PyLearn course IDs.
