#!/usr/bin/env bash
# Install the ALP CLI from a GitHub Release.
#
#   curl -fsSL https://raw.githubusercontent.com/adams100111/agentic-learning-partner/main/scripts/get.sh | bash
#   ... | bash -s -- v0.1.0        # pin a version
#
# env: ALP_INSTALL_DIR     install dir (default: $HOME/.local/bin)
#      GITHUB_TOKEN/GH_TOKEN  optional; only sent to api.github.com to avoid
#                             anonymous API rate limits when resolving "latest".
# No sudo. Idempotent. Refuses on checksum mismatch.
set -euo pipefail

GS_REPO="adams100111/agentic-learning-partner"
GS_TMP=""

gs_err() { echo "get.sh: $*" >&2; }
gs_cleanup() { if [ -n "$GS_TMP" ]; then rm -rf "$GS_TMP"; GS_TMP=""; fi; }

gs_fetch() { # URL OUT
  curl -fsSL --proto '=https' --proto-redir '=https' "$1" -o "$2"
}

gs_sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{ print $1 }'
  elif command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{ print $1 }'
  else gs_err "need sha256sum or shasum to verify the download"; return 1; fi
}

# gs_latest_tag — the newest published release tag (drafts are never "latest").
gs_latest_tag() {
  local token="${GH_TOKEN:-${GITHUB_TOKEN:-}}" json
  if [ -n "$token" ]; then
    json="$(curl -fsSL --proto '=https' --proto-redir '=https' \
      -H @<(printf 'Authorization: Bearer %s\n' "$token") \
      "https://api.github.com/repos/${GS_REPO}/releases/latest")"
  else
    json="$(curl -fsSL --proto '=https' --proto-redir '=https' \
      "https://api.github.com/repos/${GS_REPO}/releases/latest")"
  fi
  printf '%s\n' "$json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

gs_main() {
  local version="${1:-}" os arch asset base expected actual dest

  case "$(uname -s)" in
    Darwin) os=darwin ;;
    Linux) os=linux ;;
    *) gs_err "unsupported OS: $(uname -s)"; return 1 ;;
  esac
  case "$(uname -m)" in
    arm64|aarch64) arch=arm64 ;;
    x86_64|amd64) arch=amd64 ;;
    *) gs_err "unsupported architecture: $(uname -m)"; return 1 ;;
  esac

  if [ -z "$version" ]; then
    version="$(gs_latest_tag)" || version=""
    [ -n "$version" ] || { gs_err "no published release found (or network/rate-limit error; set GITHUB_TOKEN to lift the limit)"; return 1; }
  fi
  case "$version" in
    v[0-9]*.[0-9]*.[0-9]*) ;;
    *) gs_err "version must look like vX.Y.Z, got '${version}'"; return 1 ;;
  esac

  asset="alp_${version}_${os}_${arch}.tar.gz"
  base="https://github.com/${GS_REPO}/releases/download/${version}"
  trap 'gs_cleanup' EXIT
  GS_TMP="$(mktemp -d)"
  chmod 700 "$GS_TMP"

  gs_err "downloading ${asset}..."
  gs_fetch "${base}/${asset}" "$GS_TMP/$asset" || { gs_err "download failed: ${base}/${asset}"; return 1; }
  gs_fetch "${base}/SHA256SUMS" "$GS_TMP/SHA256SUMS" || { gs_err "download failed: ${base}/SHA256SUMS"; return 1; }

  expected="$(awk -v f="$asset" '$2 == f || $2 == "*" f { print $1; exit }' "$GS_TMP/SHA256SUMS")"
  [ -n "$expected" ] || { gs_err "${asset} is not listed in SHA256SUMS"; return 1; }
  actual="$(gs_sha256 "$GS_TMP/$asset")"
  if [ "$expected" != "$actual" ]; then
    gs_err "checksum mismatch for ${asset} (expected ${expected}, got ${actual}); refusing to install"
    return 1
  fi

  mkdir -p "$GS_TMP/x"
  tar -xzf "$GS_TMP/$asset" -C "$GS_TMP/x" alp
  dest="${ALP_INSTALL_DIR:-$HOME/.local/bin}"
  mkdir -p "$dest"
  install -m 755 "$GS_TMP/x/alp" "$dest/alp.new.$$"
  mv -f "$dest/alp.new.$$" "$dest/alp"

  gs_err "installed ${dest}/alp"
  case ":${PATH}:" in
    *":${dest}:"*) ;;
    *) gs_err "warning: ${dest} is not on your PATH; add: export PATH=\"${dest}:\$PATH\"" ;;
  esac
  "$dest/alp" version
}

# Run only when executed (incl. via `curl | bash`), not when sourced.
if ! (return 0 2>/dev/null); then
  gs_main "$@"
fi
