#!/usr/bin/env bash
# Cut an ALP release locally (no CI): verify, cross-compile, checksum, then
# publish to GitHub Releases with `gh` using a draft-first flow:
#   draft release -> upload assets -> download back + verify checksums ->
#   annotated tag + push -> publish and mark latest.
# A draft is invisible to releases/latest, so get.sh never sees a half-built release.
#
# usage: scripts/release.sh <vX.Y.Z> [--dry-run]
#
# --dry-run builds dist/ and SHA256SUMS but creates no tag, push, or release.
set -euo pipefail

REPO_SLUG="adams100111/agentic-learning-partner"
MODULE="github.com/adams100111/agentic-learning-partner"
PLATFORMS="darwin/arm64 darwin/amd64 linux/amd64 linux/arm64"

die() { echo "release: $*" >&2; exit 1; }

DRY_RUN=0
VERSION=""
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=1 ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
    -*) die "unknown flag: $arg" ;;
    *) [ -z "$VERSION" ] || die "multiple versions given"; VERSION="$arg" ;;
  esac
done
[ -n "$VERSION" ] || die "usage: scripts/release.sh <vX.Y.Z> [--dry-run]"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# Use the devenv-pinned Go toolchain.
if [ -z "${DEVENV_ROOT:-}" ] && command -v devenv >/dev/null 2>&1; then
  exec devenv shell -- "$ROOT/scripts/release.sh" "$@"
fi
command -v go >/dev/null 2>&1 || die "go not found (run inside devenv)"
command -v git >/dev/null 2>&1 || die "git not found"
if [ "$DRY_RUN" -eq 0 ]; then
  command -v gh >/dev/null 2>&1 || die "gh not found"
fi

# --- guard: never race a CI release workflow ---------------------------------
shopt -s nullglob
for wf in .github/workflows/*release*; do
  wf_state="unknown"
  if command -v gh >/dev/null 2>&1; then
    wf_state="$(gh workflow list --all --json path,state -R "$REPO_SLUG" \
      --jq ".[] | select(.path == \"$wf\") | .state" 2>/dev/null || true)"
  fi
  case "$wf_state" in
    disabled_*) echo "release: $wf is $wf_state; continuing" >&2 ;;
    *) die "$wf exists and is enabled (or its state is unreadable); CI release workflows are not allowed here" ;;
  esac
done
shopt -u nullglob

# --- preconditions -----------------------------------------------------------
[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "version must be semver vX.Y.Z, got '$VERSION'"
PLAIN="${VERSION#v}"

if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
  die "tag $VERSION already exists locally"
fi

state_problems=()
[ "$(git rev-parse --abbrev-ref HEAD)" = "main" ] || state_problems+=("not on main")
[ -z "$(git status --porcelain)" ] || state_problems+=("working tree is not clean")
git fetch --quiet origin main --tags || die "git fetch origin failed"
if git ls-remote --exit-code --tags origin "refs/tags/$VERSION" >/dev/null 2>&1; then
  die "tag $VERSION already exists on origin"
fi
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] || state_problems+=("HEAD is not equal to origin/main")
if [ "${#state_problems[@]}" -gt 0 ]; then
  msg="repository state: $(IFS='; '; echo "${state_problems[*]}")"
  if [ "$DRY_RUN" -eq 1 ]; then
    echo "release: warning (dry-run): $msg" >&2
  else
    die "$msg"
  fi
fi

manifest_version() { sed -n 's/^[[:space:]]*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$1" | head -n 1; }
mismatch=0
for f in plugin.json .claude-plugin/plugin.json .claude-plugin/marketplace.json; do
  got="$(manifest_version "$f")"
  if [ "$got" != "$PLAIN" ]; then
    echo "release: $f has version '$got', expected '$PLAIN'" >&2
    mismatch=1
  fi
done
[ "$mismatch" -eq 0 ] || die "bump \"version\" to \"$PLAIN\" in plugin.json, .claude-plugin/plugin.json and .claude-plugin/marketplace.json, commit and merge to main, then re-run"

# --- verify ------------------------------------------------------------------
echo "==> go vet"
go vet ./...
echo "==> go test"
go test -count=1 ./...

# --- build -------------------------------------------------------------------
COMMIT="$(git rev-parse HEAD)"
EPOCH="$(git show -s --format=%ct HEAD)"
# Build date = commit date, so rebuilding the same commit is byte-identical.
BUILD_DATE="$(date -u -r "$EPOCH" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "@$EPOCH" +%Y-%m-%dT%H:%M:%SZ)"
TOUCH_STAMP="$(date -u -r "$EPOCH" +%Y%m%d%H%M.%S 2>/dev/null || date -u -d "@$EPOCH" +%Y%m%d%H%M.%S)"
LDFLAGS="-s -w -X $MODULE/internal/buildinfo.Version=$VERSION -X $MODULE/internal/buildinfo.Commit=$COMMIT -X $MODULE/internal/buildinfo.Date=$BUILD_DATE"

rm -rf dist
mkdir -p dist
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

for plat in $PLATFORMS; do
  goos="${plat%/*}"; goarch="${plat#*/}"
  name="alp_${VERSION}_${goos}_${goarch}"
  dir="$STAGE/$name"
  mkdir -p "$dir"
  echo "==> build $goos/$goarch"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -buildvcs=false -ldflags "$LDFLAGS" -o "$dir/alp" ./cmd/alp
  cp README.md "$dir/README.md"
  files=(alp)
  if [ -f LICENSE ]; then cp LICENSE "$dir/LICENSE"; files+=(LICENSE); fi
  files+=(README.md)
  chmod 755 "$dir/alp"
  (cd "$dir" && touch -t "$TOUCH_STAMP" "${files[@]}" \
    && tar -cf - "${files[@]}" | gzip -n >"$ROOT/dist/$name.tar.gz")
done

cp scripts/get.sh dist/get.sh

(
  cd dist
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -- *.tar.gz get.sh
  else
    shasum -a 256 -- *.tar.gz get.sh
  fi >SHA256SUMS
)

# --- notes -------------------------------------------------------------------
NOTES="$STAGE/notes.md"
PREV="$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true)"
{
  echo "## $VERSION"
  echo
  if [ -n "$PREV" ]; then
    echo "Changes since $PREV:"
    echo
    git log --no-merges --pretty='- %s (%h)' "$PREV..HEAD"
  else
    echo "Initial release."
    echo
    git log --no-merges --pretty='- %s (%h)'
  fi
  echo
  echo "Install: \`curl -fsSL https://raw.githubusercontent.com/$REPO_SLUG/main/scripts/get.sh | bash -s -- $VERSION\`"
} >"$NOTES"

echo
echo "==> artifacts in dist/"
ls -l dist
echo
cat dist/SHA256SUMS

if [ "$DRY_RUN" -eq 1 ]; then
  echo
  echo "dry-run: no tag, push, or release created. Release notes preview:"
  cat "$NOTES"
  exit 0
fi

# --- publish: draft -> upload -> verify round-trip -> tag -> publish --------------
echo "==> draft release $VERSION"
gh release create "$VERSION" -R "$REPO_SLUG" --draft --target "$COMMIT" \
  --title "$VERSION" --notes-file "$NOTES"
gh release upload "$VERSION" -R "$REPO_SLUG" dist/* --clobber

echo "==> verify uploaded assets"
VERIFY="$STAGE/verify"
mkdir -p "$VERIFY"
gh release download "$VERSION" -R "$REPO_SLUG" -D "$VERIFY" --clobber
for f in dist/*; do
  [ -f "$VERIFY/$(basename "$f")" ] || die "$(basename "$f") did not round-trip; release left as a DRAFT"
done
(
  cd "$VERIFY"
  if command -v sha256sum >/dev/null 2>&1; then sha256sum -c SHA256SUMS; else shasum -a 256 -c SHA256SUMS; fi
) >/dev/null || die "downloaded assets do not match SHA256SUMS; release left as a DRAFT"
echo "verified $(wc -l <"$VERIFY/SHA256SUMS" | tr -d ' ') asset(s) against SHA256SUMS"

echo "==> tag + publish $VERSION"
git tag -a "$VERSION" -m "$VERSION"
git push origin "refs/tags/$VERSION"
gh release edit "$VERSION" -R "$REPO_SLUG" --draft=false --latest
echo "released $VERSION"
