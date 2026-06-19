#!/usr/bin/env bash
set -euo pipefail

# ── gem2-lfs npm publish script ────────────────────────────────
# Publishes all 5 packages in the correct order:
# platform binaries first, then the wrapper package last.
#
# Usage:
#   ./npm/publish.sh [--dry-run]
# ────────────────────────────────────────────────────────────────

NPM_DIR="$(cd "$(dirname "$0")" && pwd)"
DRY_RUN=""

if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN="--dry-run"
  echo "==> DRY RUN mode (no actual publish)"
fi

# Platform packages first
PLATFORM_PKGS=(
  "gem2-lfs-darwin-arm64"
  "gem2-lfs-darwin-x64"
  "gem2-lfs-linux-x64"
  "gem2-lfs-linux-arm64"
)

echo "==> Publishing platform packages"
for pkg in "${PLATFORM_PKGS[@]}"; do
  dir="${NPM_DIR}/${pkg}"
  echo "    ${pkg}"

  # Verify binary exists
  if [[ ! -f "${dir}/bin/gem2-lfs" ]]; then
    echo "    ERROR: ${dir}/bin/gem2-lfs not found. Run build.sh first."
    exit 1
  fi

  # Show what would be packed
  echo "    --- npm pack --dry-run ---"
  (cd "$dir" && npm pack --dry-run 2>&1 | grep -E '\.go|go\.(mod|sum)|Tarball|total' || true)
  echo ""

  (cd "$dir" && npm publish --access public $DRY_RUN)
done

echo "==> Publishing wrapper package"
(cd "${NPM_DIR}/gem2-lfs" && npm publish --access public $DRY_RUN)

echo "==> Done."
