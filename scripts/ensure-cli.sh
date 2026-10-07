#!/usr/bin/env bash
# Make the `alp` CLI match this plugin's version. Runs as the plugin's
# SessionStart hook (hooks/hooks.json); safe to run by hand.
#
# Fast path: `alp` resolves and reports v<plugin version> -> exit 0, silent,
# no network. Otherwise the get.sh shipped in this plugin tree installs the
# pinned release into ${ALP_INSTALL_DIR:-$HOME/.local/bin}. It never fails the
# session: any problem becomes one stderr line (plus SessionStart hook context
# on stdout) naming the manual command, and the exit status stays 0.
#
# env: ALP_SKIP_CLI_INSTALL=1  opt out entirely
#      ALP_INSTALL_DIR         install dir (default: $HOME/.local/bin)
#      ALP_GET_SH              installer override (tests only)
#      ALP_ENSURE_LOCK_WAIT    seconds to wait for a concurrent install (default 90)
set -uo pipefail

ec_log() { printf 'alp plugin: %s\n' "$*" >&2; }

# ec_context MSG — surface MSG to the agent as SessionStart hook context.
ec_context() {
  local esc
  esc="$(printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g')"
  printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$esc"
}

ec_fail() { # MSG — report and give up without failing the session
  ec_log "$1"
  ec_context "$1"
  exit 0
}

[ "${ALP_SKIP_CLI_INSTALL:-}" = "1" ] && exit 0

ec_root="${CLAUDE_PLUGIN_ROOT:-${PLUGIN_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}}"

# The plugin version: Claude manifest first, then the portable root manifest.
ec_version=""
for ec_manifest in "$ec_root/.claude-plugin/plugin.json" "$ec_root/plugin.json"; do
  [ -f "$ec_manifest" ] || continue
  ec_version="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$ec_manifest" | head -n 1)"
  [ -n "$ec_version" ] && break
done
case "$ec_version" in
  [0-9]*.[0-9]*.[0-9]*) ;;
  *) ec_log "could not read the plugin version from ${ec_root}; skipping CLI check"; exit 0 ;;
esac
ec_tag="v${ec_version}"
ec_dest="${ALP_INSTALL_DIR:-$HOME/.local/bin}"
ec_manual="curl -fsSL https://raw.githubusercontent.com/adams100111/agentic-learning-partner/${ec_tag}/scripts/get.sh | bash -s -- ${ec_tag}"

# ec_current — true when the alp that resolves is acceptable for this plugin.
# A local "dev" build (go install) is left alone.
ec_current() {
  local bin json have
  bin="$(command -v alp 2>/dev/null || true)"
  [ -n "$bin" ] || { [ -x "$ec_dest/alp" ] && bin="$ec_dest/alp"; }
  [ -n "$bin" ] || return 1
  json="$("$bin" version --json 2>/dev/null)" || return 1
  have="$(printf '%s' "$json" | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  [ "$have" = "$ec_tag" ] || [ "$have" = "dev" ]
}

ec_current && exit 0

ec_get="${ALP_GET_SH:-$ec_root/scripts/get.sh}"
[ -f "$ec_get" ] || ec_fail "alp ${ec_tag} is not installed and the installer is missing (${ec_get}). Install manually: ${ec_manual}"
command -v curl >/dev/null 2>&1 || [ -n "${ALP_GET_SH:-}" ] || ec_fail "alp ${ec_tag} is not installed and curl is missing. Install manually: ${ec_manual}"

# Serialize concurrent sessions with an atomic mkdir lock.
ec_lock="${TMPDIR:-/tmp}/alp-ensure-cli-$(id -u).lock"
ec_wait="${ALP_ENSURE_LOCK_WAIT:-90}"
ec_held=0
ec_waited=0
while :; do
  if mkdir "$ec_lock" 2>/dev/null; then ec_held=1; break; fi
  # Reclaim a lock abandoned by a killed session (older than 10 minutes).
  if [ -n "$(find "$ec_lock" -maxdepth 0 -mmin +10 2>/dev/null)" ]; then
    rmdir "$ec_lock" 2>/dev/null || true
    continue
  fi
  [ "$ec_waited" -ge "$ec_wait" ] && break
  sleep 1
  ec_waited=$((ec_waited + 1))
done
if [ "$ec_held" -eq 1 ]; then
  trap 'rmdir "$ec_lock" 2>/dev/null || true' EXIT
fi
# Another session may have finished the install while we waited.
if ec_current; then exit 0; fi
[ "$ec_held" -eq 1 ] || ec_fail "another session is still installing alp ${ec_tag}; gave up waiting. Install manually: ${ec_manual}"

ec_out="$(mktemp "${TMPDIR:-/tmp}/alp-ensure-cli.XXXXXX")" || ec_out=/dev/null
if ALP_INSTALL_DIR="$ec_dest" bash "$ec_get" "$ec_tag" >"$ec_out" 2>&1 </dev/null; then
  ec_log "installed alp ${ec_tag} to ${ec_dest}"
  case ":${PATH}:" in
    *":${ec_dest}:"*) ;;
    *)
      ec_state="${XDG_STATE_HOME:-$HOME/.local/state}/alp"
      if [ ! -e "$ec_state/path-warned" ]; then
        mkdir -p "$ec_state" 2>/dev/null && : >"$ec_state/path-warned" 2>/dev/null
        ec_msg="alp ${ec_tag} was installed to ${ec_dest}, which is not on PATH. Add it: export PATH=\"${ec_dest}:\$PATH\""
        ec_log "$ec_msg"
        ec_context "$ec_msg"
      fi
      ;;
  esac
  [ "$ec_out" = /dev/null ] || rm -f "$ec_out"
  exit 0
fi

ec_detail=""
[ "$ec_out" = /dev/null ] || { ec_detail="$(tail -n 1 "$ec_out" 2>/dev/null)"; rm -f "$ec_out"; }
ec_fail "could not install alp ${ec_tag}${ec_detail:+ (${ec_detail})}. Install manually: ${ec_manual}"
