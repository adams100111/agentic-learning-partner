#!/usr/bin/env bash
# Make the `alp` CLI match this plugin's version. Runs as the plugin's
# SessionStart hook (hooks/hooks.json); safe to run by hand.
#
# Fast path: the `alp` that resolves (or the one in the install dir) reports
# v<plugin version> -> exit 0, silent, no network. Otherwise the get.sh shipped
# in this plugin tree installs the pinned release into
# ${ALP_INSTALL_DIR:-$HOME/.local/bin}. Never fails the session: problems become
# one stderr line (plus SessionStart hook context on stdout), exit status 0.
#
# Policies:
#   - a local "dev" build is never replaced;
#   - an installed alp NEWER than the plugin is never downgraded (warned once);
#   - an alp earlier on PATH that shadows the installed one is warned about once;
#   - a release that is not published yet (HTTP 404) is a quiet line, retried
#     next session.
#
# env: ALP_SKIP_CLI_INSTALL=1  opt out entirely
#      ALP_INSTALL_DIR         install dir (default: $HOME/.local/bin)
#      ALP_GET_SH              installer override (tests only)
#      ALP_ENSURE_LOCK_WAIT    seconds to wait for a concurrent install (default 30)
set -uo pipefail

ec_log() { printf 'alp plugin: %s\n' "$*" >&2; }

# ec_context MSG — surface MSG to the agent as SessionStart hook context.
# Control characters become spaces; backslash and quote are escaped.
ec_context() {
  local esc
  esc="$(printf '%s' "$1" | tr '\000-\037\177' ' ' | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g')"
  printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$esc"
}

ec_note() { ec_log "$1"; ec_context "$1"; }

ec_fail() { # MSG — report and give up without failing the session
  ec_note "$1"
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
ec_cache="${XDG_CACHE_HOME:-$HOME/.cache}/alp"
ec_run="${XDG_RUNTIME_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}}/alp"

# ec_ver BIN — the version BIN reports without the leading v ("dev" stays).
ec_ver() {
  local json
  json="$("$1" version --json 2>/dev/null)" || return 0
  printf '%s' "$json" | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"v\{0,1\}\([^"]*\)".*/\1/p'
}

# ec_gt A B — true when numeric X.Y.Z A is greater than B (non-numeric: false).
ec_gt() {
  local -a a b
  local x y i
  IFS=. read -r -a a <<<"$1"
  IFS=. read -r -a b <<<"$2"
  for i in 0 1 2; do
    x="${a[$i]:-0}"; y="${b[$i]:-0}"
    x="${x%%[!0-9]*}"; y="${y%%[!0-9]*}"
    [ "${x:-0}" -gt "${y:-0}" ] 2>/dev/null && return 0
    [ "${x:-0}" -lt "${y:-0}" ] 2>/dev/null && return 1
  done
  return 1
}

# ec_once KEY — true the first time KEY is seen (marker under the cache dir).
ec_once() {
  local k
  k="$(printf '%s' "$1" | cksum | cut -d' ' -f1)"
  (umask 077; mkdir -p "$ec_cache") 2>/dev/null
  [ -e "$ec_cache/seen-$k" ] && return 1
  : >"$ec_cache/seen-$k" 2>/dev/null
  return 0
}

# ec_settled — true when nothing needs installing; emits one-time warnings.
# Sets ec_path_bin (alp on PATH, if any).
ec_settled() {
  local rv dv
  ec_path_bin="$(command -v alp 2>/dev/null || true)"
  dv=""
  [ -x "$ec_dest/alp" ] && dv="$(ec_ver "$ec_dest/alp")"
  if [ -n "$ec_path_bin" ]; then rv="$(ec_ver "$ec_path_bin")"; else rv="$dv"; fi

  # The alp that resolves is current (or a local dev build).
  if [ "$rv" = "$ec_version" ] || [ "$rv" = dev ]; then return 0; fi
  # The installed one is current but an older alp earlier on PATH shadows it.
  if [ "$dv" = "$ec_version" ] || [ "$dv" = dev ]; then
    if [ -n "$ec_path_bin" ] && ec_once "shadow|$ec_path_bin|$rv|$ec_dest/alp|$dv"; then
      ec_note "alp ${ec_tag} is installed at ${ec_dest}/alp but is shadowed by ${ec_path_bin} (version ${rv:-unknown}), which comes first on PATH. Put ${ec_dest} first on PATH or remove the older alp."
    fi
    return 0
  fi
  # Never downgrade a newer CLI.
  if { [ -n "$rv" ] && ec_gt "$rv" "$ec_version"; } || { [ -n "$dv" ] && ec_gt "$dv" "$ec_version"; }; then
    if ec_once "newer|$rv|$dv|$ec_version"; then
      ec_note "plugin ${ec_tag} is older than the installed alp (v${rv:-$dv}); leaving the CLI as is. Update the plugin: claude plugin marketplace update agentic-learning-partner"
    fi
    return 0
  fi
  return 1
}

ec_settled && exit 0

ec_get="${ALP_GET_SH:-$ec_root/scripts/get.sh}"
[ -f "$ec_get" ] || ec_fail "alp ${ec_tag} is not installed and the installer is missing (${ec_get}). Install manually: ${ec_manual}"
command -v curl >/dev/null 2>&1 || [ -n "${ALP_GET_SH:-}" ] || ec_fail "alp ${ec_tag} is not installed and curl is missing. Install manually: ${ec_manual}"

# Serialize concurrent sessions with an atomic mkdir lock in a private dir.
ec_lock="$ec_run/ensure-cli.lock"
ec_wait="${ALP_ENSURE_LOCK_WAIT:-30}"
ec_held=0
ec_locking=1
(umask 077; mkdir -p "$ec_run") 2>/dev/null
chmod 700 "$ec_run" 2>/dev/null
[ -d "$ec_run" ] || ec_locking=0 # no writable dir: install unserialized (get.sh installs atomically)
ec_waited=0
while [ "$ec_locking" -eq 1 ]; do
  if mkdir "$ec_lock" 2>/dev/null; then ec_held=1; break; fi
  # Reclaim a lock abandoned by a killed session (older than 10 minutes). The
  # atomic rename lets exactly one reclaimer win; if it turns out to have moved
  # a live lock, put it back.
  if [ -n "$(find "$ec_lock" -maxdepth 0 -mmin +10 2>/dev/null)" ]; then
    ec_stale="$ec_lock.stale.$$"
    if mv "$ec_lock" "$ec_stale" 2>/dev/null; then
      if [ -n "$(find "$ec_stale" -maxdepth 0 -mmin +10 2>/dev/null)" ]; then
        rm -rf "$ec_stale"
      else
        mv "$ec_stale" "$ec_lock" 2>/dev/null || rm -rf "$ec_stale"
      fi
    fi
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
ec_settled && exit 0
if [ "$ec_locking" -eq 1 ] && [ "$ec_held" -ne 1 ]; then
  ec_fail "another session is still installing alp ${ec_tag}; gave up waiting. Install manually: ${ec_manual}"
fi

ec_out="$(mktemp "${TMPDIR:-/tmp}/alp-ensure-cli.XXXXXX")" || ec_out=/dev/null
ec_rc=0
ALP_INSTALL_DIR="$ec_dest" bash "$ec_get" "$ec_tag" >"$ec_out" 2>&1 </dev/null || ec_rc=$?
if [ "$ec_rc" -eq 0 ]; then
  [ "$ec_out" = /dev/null ] || rm -f "$ec_out"
  ec_log "installed alp ${ec_tag} to ${ec_dest}"
  ec_settled || true # warns once when an older alp on PATH shadows the new one
  if [ -z "$ec_path_bin" ] && ec_once "path|$ec_dest"; then
    ec_note "alp ${ec_tag} was installed to ${ec_dest}, which is not on PATH. Add it: export PATH=\"${ec_dest}:\$PATH\""
  fi
  exit 0
fi

ec_detail=""
[ "$ec_out" = /dev/null ] || { ec_detail="$(tail -n 1 "$ec_out" 2>/dev/null)"; rm -f "$ec_out"; }
if [ "$ec_rc" -eq 44 ]; then
  # Version bumped but the release is not published yet: quiet, retry next session.
  ec_log "alp ${ec_tag} is not published yet; will retry next session"
  exit 0
fi
ec_fail "could not install alp ${ec_tag}${ec_detail:+ (${ec_detail})}. Install manually: ${ec_manual}"
