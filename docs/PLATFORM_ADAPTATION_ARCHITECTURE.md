# Platform Adaptation Architecture

## Purpose

This document captures the settled architecture for ALP's next platform-integration tranche.

ALP is not a course repository, renderer, or LMS. It is the learning control plane.

Its responsibilities are:

- learner/profile/persona state;
- evidence and assessment semantics;
- competency projections;
- review/reinforcement state;
- adaptive planning;
- platform-neutral curriculum/unit intent;
- effective context for agents;
- cross-platform learning continuity.

External platforms own their own learner-facing representation, runtime, application telemetry, content repository, and quality gates.

PyLearn is the primary reference integration, not the only platform target.

## Core dependency direction

```text
                    Agentic Learning Partner
                              |
                 platform-agnostic learning core
                              |
          +-------------------+-------------------+
          |                                       |
   Domain learner state                    Adaptation intent
   evidence / assessment                   curriculum decisions
   competencies / review                   content requirements
   persona / goals                         reinforcement / gaps
          |                                       |
          +-------------------+-------------------+
                              |
                    Platform Adapter
                 capability-oriented boundary
                              |
           +------------------+------------------+
           |                  |                  |
        PyLearn           future LMS        custom learning app
   reference target       / course engine        / lab system
```

## Competency identity

Competency truth is not course-scoped.

The authoritative learner model is:

```text
learner
└── domain: go
    ├── evidence
    ├── assessments
    └── competency projections
        └── go.concurrency.channels
```

Evidence provenance may identify:

- platform;
- learning target/course;
- content/unit/activity;
- repository/commit;
- harness/session.

Different learning targets can contribute evidence to the same competency.

## Learning Target

ALP uses **Learning Target** as the platform-neutral destination concept.

Examples:

- PyLearn course;
- LMS track;
- workshop;
- project path;
- lab series;
- cohort curriculum.

ALP core must not assume every target is a course.

## Target Adaptation Projection

A Target Adaptation Projection is derived from learner/domain state for one target.

Typical fields/decisions may include:

- target/platform identity;
- relevant competencies;
- coverage/gaps;
- skip / skim / challenge / full mode;
- sequencing;
- reinforcement;
- misconceptions;
- due review;
- required evidence;
- content gaps;
- authoring requirements;
- freshness/source constraints.

It is rebuildable and non-authoritative. It must never become a second competency state.

## Curriculum and Learning Unit Specifications

ALP should express authoring intent in platform-neutral specifications.

A Curriculum Specification describes:

- target goal;
- competency coverage;
- unit graph;
- dependencies;
- sequencing constraints;
- phase/milestone structure where applicable;
- evidence expectations;
- adaptation modes;
- completion/done bars.

A Learning Unit Specification describes one unit:

- competencies taught/reinforced;
- competencies legitimately assessed;
- objective;
- prerequisite assumptions;
- adaptation mode;
- misconceptions/trip-wires;
- transfer analogies;
- required exercise/evidence shape;
- source/freshness requirements;
- dependency/order context;
- done bar.

These specifications deliberately exclude platform-native render syntax.

## Platform capabilities

The umbrella Platform Adapter is capability-oriented.

The initial conceptual capability families are:

### Activity source

Exports/normalizes learner activity and project evidence into ALP-compatible inputs.

### Curriculum reader

Lets ALP inspect existing target structure/content/metadata so adaptation can reason about current coverage instead of generating blindly.

### Content mapper

Provides stable platform content/activity IDs → ALP competency mappings.

The platform/content repository owns the mapping because it owns content identity and semantics.

### Authoring target

Realizes ALP curriculum/unit/adaptation specifications using platform-native content representation.

### Platform validator

Runs the target platform's own quality gates and returns structured validation results.

These names are conceptual. Exact Go interfaces and capability shapes remain open for the next design round.

## Authoring boundary

ALP core does not directly mutate production platform content.

The normal write-side control loop is:

```text
learner/domain state
        ↓
Target Adaptation Projection
        ↓
Curriculum / Learning Unit Specification
        ↓
Authoring Plan
        ↓
platform branch/worktree
        ↓
platform-native content
        ↓
platform-native quality gates
        ↓
review / PR
```

A capable agent may automate branch/worktree/content/PR operations, but platform repository policies remain authoritative.

## Progressive generation

Settled direction:

- ALP may project the complete curriculum/unit graph;
- executable/polished platform-native content should be authored progressively;
- later content should be allowed to change as learner evidence changes;
- the first PyLearn validation should not eagerly generate an entire 60+ Reel course unless the smoke specifically needs that.

## Closed-loop invariant

A full platform integration is not complete merely because it imports activity or generates content.

The target invariant is:

```text
Learner State
    ↓
Target Adaptation Projection
    ↓
Curriculum / Learning Unit Specifications
    ↓
Platform-native authored content
    ↓
Learner activity
    ↓
Platform activity export
    ↓
ALP Evidence → Assessment → Projection
    ↓
Target adaptation changes when justified
```

## Reference integration rule

PyLearn is the design/acceptance reference.

The abstraction test is:

> If the generic contract cannot express PyLearn's Reel system cleanly, it is too weak. If the ALP core contains Reel/Scene/MDX concepts, it is too PyLearn-specific.
