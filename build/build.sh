#!/bin/bash

# tinyMem Build Script
# Builds platform binaries into build/releases (never tracked by git)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

OUT_DIR="build/releases"
mkdir -p "$OUT_DIR"

# ------------------------------------------------------------
# Determine version (prefer code, fallback to git)
# ------------------------------------------------------------
VERSION="$(grep -E 'var Version =' internal/version/version.go \
  | sed -E 's/.*"([^"]+)".*/\1/' || true)"

if [[ -z "$VERSION" ]]; then
  VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

echo "Building tinyMem version: $VERSION"

# ------------------------------------------------------------
# Build tags
# ------------------------------------------------------------
BUILD_TAGS=("fts5")

if [[ -n "${TINYMEM_EXTRA_BUILD_TAGS:-}" ]]; then
  read -r -a EXTRA <<< "$TINYMEM_EXTRA_BUILD_TAGS"
  BUILD_TAGS+=("${EXTRA[@]}")
fi

TAGS_FLAG=(-tags "${BUILD_TAGS[*]}")
LDFLAGS="-X github.com/andrzejmarczewski/tinyMem/internal/version.Version=${VERSION}"

# ------------------------------------------------------------
# Build helper
# ------------------------------------------------------------
build_target() {
  local label=$1
  local goos=$2
  local goarch=$3
  local out=$4

  echo "→ $label"
  CGO_ENABLED=1 GOOS="$goos" GOARCH="$goarch" \
    go build "${TAGS_FLAG[@]}" -ldflags "$LDFLAGS" \
    -o "$out" ./cmd/tinymem
}

# ------------------------------------------------------------
# macOS
# ------------------------------------------------------------
build_target "macOS ARM64" darwin arm64 "$OUT_DIR/tinymem-darwin-arm64"
build_target "macOS AMD64" darwin amd64 "$OUT_DIR/tinymem-darwin-amd64"

# ------------------------------------------------------------
# Linux (only when on Linux)
# ------------------------------------------------------------
if [[ "$OSTYPE" == linux* ]]; then
  build_target "Linux AMD64" linux amd64 "$OUT_DIR/tinymem-linux-amd64"
  build_target "Linux ARM64" linux arm64 "$OUT_DIR/tinymem-linux-arm64"
else
  echo "Skipping Linux builds (not on Linux)"
fi

# ------------------------------------------------------------
# Windows (only when on Windows-like env)
# ------------------------------------------------------------
if [[ "$OSTYPE" == msys* || "$OSTYPE" == cygwin* ]]; then
  build_target "Windows AMD64" windows amd64 "$OUT_DIR/tinymem-windows-amd64.exe"
  build_target "Windows ARM64" windows arm64 "$OUT_DIR/tinymem-windows-arm64.exe"
else
  echo "Skipping Windows builds (not on Windows)"
fi

echo
echo "Build complete. Artifacts:"
ls -lh "$OUT_DIR"
