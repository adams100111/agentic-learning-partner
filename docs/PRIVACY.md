# Privacy and Data Minimization

## Principle

ALP stores only information that materially improves teaching, assessment, planning, or continuity.

A private Git repository is not a license to collect everything.

## Allowed categories

Examples:

- technical experience;
- learning goals;
- teaching preferences;
- domain progress;
- relevant project evidence;
- learning reflections;
- source/provenance references.

## Avoid by default

- unrelated personal biography;
- credentials/tokens/secrets;
- private communications unrelated to learning;
- health/family/financial details unless explicitly necessary to a user-chosen learning constraint;
- raw repository/platform dumps when selected fields suffice.

## Adapters

Platform/repository adapters should use allowlists and purpose-limited extraction.

Do not mirror whole databases or repositories into learner state.

Free text from a platform is kept only when it is the learning evidence itself. The PyLearn adapter keeps a learner's reflection text (a learning reflection) but reduces exercise failure output to a bounded, sanitized one-line summary without local paths (`docs/integrations/PYLEARN_EXPORT.md`, "What import keeps").

## Generated views

Human views should show enough provenance to explain state but should not expose unnecessary sensitive raw source content.

## Conversation-derived state

The agent should ask: "Will storing this materially improve future learning behavior?"

If no, use it transiently and do not persist it.

## Deletion/correction

Canonical append-only learning evidence supports correction/supersession for auditability, but personal profile/persona facts must also support explicit removal where the learner requests it. Audit semantics must not become a reason to retain unnecessary personal information forever.

Immutable records must therefore not copy persona or profile content. Learning Unit Specification versions store only persona/profile document references and content hashes plus the learner-free teaching shape; the private persona view (risks, analogy override reasons, experience levels) is re-derived from the live persona and profile on every `alp platform plan` and never stored (`docs/TARGET_ADAPTATION.md`, "Persona privacy"). Versions written before 2026-10-07 may still hold that view in `adaptation.persona`; they stay readable, and later versions omit it.
