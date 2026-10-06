#!/usr/bin/env bash
# Fails when two ADR files share a number. Numbers duplicated before this
# check existed are grandfathered; ADR numbering is immutable, so they stay.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)/docs/adr"
grandfathered=" 0010 0011 0012 0013 0047 0048 0049 0050 0051 0052 0053 "
status=0
for n in $(ls [0-9][0-9][0-9][0-9]-*.md | cut -c1-4 | sort | uniq -d); do
  if [[ "$grandfathered" != *" $n "* ]]; then
    echo "duplicate ADR number $n:" $(ls "$n"-*.md) >&2
    status=1
  fi
done
exit $status
