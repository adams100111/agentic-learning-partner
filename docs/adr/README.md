# Architecture Decision Records

ADRs capture decisions that would otherwise be rediscovered or accidentally reversed.

Format:

- Context
- Decision
- Consequences
- Alternatives considered
- Status

Numbering is chronological and immutable. Superseded ADRs remain in the repository and point to their replacement.

Each number identifies exactly one ADR; `scripts/check-adr-numbers.sh` fails on duplicates. Early duplicate-numbered drafts (0010-0013, 0047-0053) were merged into the canonical record for each number, and the second ADR-0053 (capability failure) was renumbered to ADR-0062.
