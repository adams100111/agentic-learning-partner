#!/usr/bin/env bash
# Closed-loop go-alp smoke across ALP and PyLearn (spec #66 Seam 3, ticket #75).
# See docs/integrations/CLOSED_LOOP_SMOKE.md.
#
#   devenv shell -- scripts/smoke-closed-loop.sh <pylearn-checkout>
#
# Builds alp from this tree and runs internal/e2e TestClosedLoopGoALP (build tag
# `closedloop`) against the PyLearn checkout, a fresh synthetic workspace and a
# throwaway PyLearn database. Local only: no CI, and no real Claude/Codex
# harness is run.
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: scripts/smoke-closed-loop.sh <pylearn-checkout>" >&2
  exit 2
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
pylearn="$(cd "$1" && pwd)"

cd "$root"
ALP_SMOKE_PYLEARN="$pylearn" exec go test -tags closedloop -count=1 -v -timeout 20m \
  -run '^TestClosedLoopGoALP$' ./internal/e2e/
