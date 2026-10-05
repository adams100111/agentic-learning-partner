# Product Definition

## Name

**Agentic Learning Partner (ALP)**

## Problem

Traditional e-learning systems primarily deliver predefined content and record completion. AI agents can generate more content, but one-shot generation still fails to solve the deeper problem: the system does not reliably retain a learner model, distinguish transferable expertise from domain gaps, evaluate real work, or continuously adapt the curriculum with provenance.

ALP makes the learning process agentic and stateful.

## Primary user experience

A learner should be able to enter any supported harness and say, effectively:

- continue my Go learning;
- assess what I actually remember;
- review this project as learning evidence;
- tell me what to learn next;
- improve my course based on where I struggled;
- re-check whether this lesson is still technically current.

The agent should not need the learner to re-explain their background or manually locate old evaluations.

## Core product promises

### Persistent across agents

Claude Code, Codex, and future compatible harnesses share the same durable profile/evidence/state.

### Evidence over confidence

Progress reflects demonstrated work, not only self-report or course completion.

### Transfer-aware

Existing engineering knowledge changes what is taught, but does not falsely imply language-specific competence.

### Adaptive but stable

The system prefers small targeted reinforcement over constantly rewriting the curriculum.

### Production-oriented

For engineering domains, completion means the learner can build, verify, operate, and explain real systems—not merely pass syntax quizzes.

### Auditable

Every important learner-state or content adaptation can be traced to evidence and a rubric.

### Current

Version-sensitive technical teaching must carry provenance/verification rather than silently relying on model memory.

## First learner

The initial learner is an experienced software architect/backend engineer whose strongest stack is PHP/Laravel, then TypeScript/NestJS, C#/.NET, with Python/FastAPI as the most recently learned backend stack. The initial Go state is rusty: prior API work exists, but retained recall and production Go competence should be re-diagnosed rather than assumed.

This profile seeds development; it must not be hard-coded into reusable core behavior.

## First domain

Go.

## Second expected domain

Rust.

## First platform integration

PyLearn.

## Non-goals

ALP is not:

- a replacement for PyLearn or another learning UI;
- a content marketplace;
- an LMS enrollment/admin system;
- a chat-memory wrapper;
- a tool that blindly regenerates full courses after each learner event;
- a universal autonomous grader without human-review boundaries.
