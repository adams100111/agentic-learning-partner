#!/usr/bin/env bash
# Fails when two ADR files share a number. Every ADR number must be unique;
# there is no allowlist (earlier duplicates were consolidated).
set -euo pipefail
cd "$(git rev-parse --show-toplevel)/docs/adr"
status=0
for n in $(ls [0-9][0-9][0-9][0-9]-*.md | cut -c1-4 | sort | uniq -d); do
  echo "duplicate ADR number $n:" $(ls "$n"-*.md) >&2
  status=1
done
exit $status
