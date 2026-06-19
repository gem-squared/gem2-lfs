#!/usr/bin/env bash
set -euo pipefail

# ── gem2-lfs npm build script ──────────────────────────────────
# Cross-compiles Go binary for 4 platforms and stages into npm dirs.
#
# Usage:
#   ./npm/build.sh [VERSION]
#
# Run from the gem2-lfs repo root:
#   cd gem2-lfs && ./npm/build.sh 0.1.0
# ────────────────────────────────────────────────────────────────

VERSION="${1:-0.1.0}"
ENTRY="./cmd/gem2-lfs/"
NPM_DIR="$(cd "$(dirname "$0")" && pwd)"
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "==> Building gem2-lfs v${VERSION}"
echo "    npm dir: ${NPM_DIR}"

# Platform matrix: GOOS GOARCH npm-suffix
TARGETS=(
  "darwin  arm64  gem2-lfs-darwin-arm64"
  "darwin  amd64  gem2-lfs-darwin-x64"
  "linux   amd64  gem2-lfs-linux-x64"
  "linux   arm64  gem2-lfs-linux-arm64"
)

for target in "${TARGETS[@]}"; do
  read -r goos goarch suffix <<< "$target"
  out="${NPM_DIR}/${suffix}/bin/gem2-lfs"
  echo "    ${goos}/${goarch} → ${out}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags="$LDFLAGS" -o "$out" "$ENTRY"
  chmod +x "$out"
done

# ── Update version in all package.json files ────────────────────

PACKAGES=(
  "gem2-lfs"
  "gem2-lfs-darwin-arm64"
  "gem2-lfs-darwin-x64"
  "gem2-lfs-linux-x64"
  "gem2-lfs-linux-arm64"
)

echo "==> Updating version to ${VERSION} in all package.json files"

for pkg in "${PACKAGES[@]}"; do
  pjson="${NPM_DIR}/${pkg}/package.json"
  if [[ -f "$pjson" ]]; then
    # Update "version" field
    sed -i '' "s/\"version\": \"[^\"]*\"/\"version\": \"${VERSION}\"/" "$pjson"
  fi
done

# Also update optionalDependencies versions in the wrapper package
wrapper="${NPM_DIR}/gem2-lfs/package.json"
sed -i '' "s/\"@gem_squared\/gem2-lfs-\([^\"]*\)\": \"[^\"]*\"/\"@gem_squared\/gem2-lfs-\1\": \"${VERSION}\"/g" "$wrapper"

# ── Verify: no Go source in any npm package ─────────────────────

echo "==> Verification: checking for Go source leaks"
leaked=0
for pkg in "${PACKAGES[@]}"; do
  count=$(find "${NPM_DIR}/${pkg}" -name '*.go' -o -name 'go.mod' -o -name 'go.sum' | wc -l | tr -d ' ')
  if [[ "$count" -gt 0 ]]; then
    echo "    ERROR: ${pkg} contains ${count} Go source file(s)"
    leaked=1
  fi
done

if [[ "$leaked" -eq 1 ]]; then
  echo "    FAIL — Go source files found in npm packages"
  exit 1
fi

echo "==> Build complete. Packages ready in ${NPM_DIR}/"
echo ""
echo "    Publish order (platform packages first):"
for pkg in "${PACKAGES[@]}"; do
  if [[ "$pkg" != "gem2-lfs" ]]; then
    echo "      cd ${NPM_DIR}/${pkg} && npm publish --access public"
  fi
done
echo "      cd ${NPM_DIR}/gem2-lfs && npm publish --access public  # LAST"
