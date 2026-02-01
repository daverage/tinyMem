#!/bin/bash

# tinyMem Build & Release Script
# Builds platform binaries and handles full release lifecycle if requested.
#
# Usage:
#   ./build/build.sh                          (Standard build)
#   ./build/build.sh [major|minor|patch]      (Full release cycle)
#
# Build Configuration:
#   tinyMem uses a single production configuration:
#     - FTS5 lexical recall (deterministic full-text search)
#     - CoVe filtering (Chain-of-Verification)
#     - Evidence-gated truth promotion
#     - Mode enforcement (PASSIVE, GUARDED, STRICT)
#     - Binary size: ~50-60 MB
#
# Examples:
#   ./build/build.sh                                         # Standard build
#   ./build/build.sh patch                                   # Release build

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

OUT_DIR="build/releases"
mkdir -p "$OUT_DIR"

# ------------------------------------------------------------
# Determine if we are in Release Mode
# ------------------------------------------------------------
MODE="${1:-}"
IS_RELEASE=false
if [[ "$MODE" == "major" || "$MODE" == "minor" || "$MODE" == "patch" ]]; then
  IS_RELEASE=true
fi

# ------------------------------------------------------------
# Safety checks for Release Mode
# ------------------------------------------------------------
if [ "$IS_RELEASE" = true ]; then
  command -v gh >/dev/null || {
    echo "❌ GitHub CLI (gh) not installed. Required for releases."
    exit 1
  }
fi

# ------------------------------------------------------------
# Version calculation
# ------------------------------------------------------------
LATEST_TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)"
IFS='.' read -r MAJOR MINOR PATCH <<< "${LATEST_TAG#v}"

if [ "$IS_RELEASE" = true ]; then
  case "$MODE" in
    major) ((MAJOR++)); MINOR=0; PATCH=0 ;;
    minor) ((MINOR++)); PATCH=0 ;;
    patch) ((PATCH++)) ;;
  esac
  VERSION="v$MAJOR.$MINOR.$PATCH"

  echo "🚀 Preparing Release: $VERSION (Current: $LATEST_TAG)"
else
  # Read current version from code for standard build
  VERSION="$(git describe --tags --dirty --always 2>/dev/null || true)"
  if [[ -z "$VERSION" ]]; then
    VERSION="$(grep -E 'var Version =' internal/version/version.go \
      | sed -E 's/.*"([^"]+)".*/\1/' || true)"
  fi
  if [[ -z "$VERSION" ]]; then
    VERSION="$LATEST_TAG"
  fi
  echo "Building tinyMem version: $VERSION"
fi

# ------------------------------------------------------------
# Build Logic
# ------------------------------------------------------------
BUILD_TAGS=("fts5")

TAGS_FLAG=(-tags "${BUILD_TAGS[*]}")
LDFLAGS="-X github.com/daverage/tinymem/internal/version.Version=${VERSION}"

build_target() {
  local label=$1
  local goos=$2
  local goarch=$3
  local out=$4

  echo "→ $label"
  CC="${CC:-gcc}" CGO_ENABLED=1 GOOS="$goos" GOARCH="$goarch" \
    go build "${TAGS_FLAG[@]}" -ldflags "$LDFLAGS" \
    -o "$out" ./cmd/tinymem
}

# Clear previous releases
rm -rf "$OUT_DIR"/*

# Build binaries
build_target "macOS ARM64" darwin arm64 "$OUT_DIR/tinymem-darwin-arm64"
build_target "macOS AMD64" darwin amd64 "$OUT_DIR/tinymem-darwin-amd64"

# Linux
if [[ "$OSTYPE" == linux* ]]; then
  build_target "Linux AMD64" linux amd64 "$OUT_DIR/tinymem-linux-amd64"
  build_target "Linux ARM64" linux arm64 "$OUT_DIR/tinymem-linux-arm64"
elif command -v x86_64-linux-musl-gcc >/dev/null 2>&1; then
  echo "→ Linux (Cross-compiling via musl-cross)"
  CC=x86_64-linux-musl-gcc build_target "Linux AMD64" linux amd64 "$OUT_DIR/tinymem-linux-amd64"
elif command -v zig >/dev/null 2>&1; then
  echo "→ Linux (Cross-compiling via zig cc)"
  CC="zig cc -target x86_64-linux-musl" build_target "Linux AMD64" linux amd64 "$OUT_DIR/tinymem-linux-amd64"
  CC="zig cc -target aarch64-linux-musl" build_target "Linux ARM64" linux arm64 "$OUT_DIR/tinymem-linux-arm64"
else
  echo "Skipping Linux builds (musl-cross or zig not found). To enable on Mac: brew install FiloSottile/musl-cross/musl-cross or brew install zig"
fi

# Windows
if [[ "$OSTYPE" == msys* || "$OSTYPE" == cygwin* ]]; then
  build_target "Windows AMD64" windows amd64 "$OUT_DIR/tinymem-windows-amd64.exe"
  build_target "Windows ARM64" windows arm64 "$OUT_DIR/tinymem-windows-arm64.exe"
elif command -v zig >/dev/null 2>&1; then
  echo "→ Windows (Cross-compiling via zig cc)"
  CC="zig cc -target x86_64-windows-gnu" build_target "Windows AMD64" windows amd64 "$OUT_DIR/tinymem-windows-amd64.exe"
  CC="zig cc -target aarch64-windows-gnu" build_target "Windows ARM64" windows arm64 "$OUT_DIR/tinymem-windows-arm64.exe"
elif command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
  echo "→ Windows (Cross-compiling via mingw-w64)"
  CC=x86_64-w64-mingw32-gcc build_target "Windows AMD64" windows amd64 "$OUT_DIR/tinymem-windows-amd64.exe"
  echo "Skipping Windows ARM64 build (requires zig or llvm-mingw)."
else
  echo "Skipping Windows builds (mingw-w64 or zig not found). To enable on Mac: brew install mingw-w64 or brew install zig"
fi


# ------------------------------------------------------------
# Finalize Release
# ------------------------------------------------------------
if [ "$IS_RELEASE" = true ]; then
  echo
  read -p "Build successful. Commit message for $VERSION: " COMMIT_MSG
  if [[ -z "$COMMIT_MSG" ]]; then
    echo "❌ Commit message required."
    exit 1
  fi

  # Update version.go
  echo "📝 Updating internal/version/version.go..."
  sed -i.bak "s/var Version = \".*\"/var Version = \"$VERSION\"/" \
    internal/version/version.go
  rm internal/version/version.go.bak

  echo "💾 Committing changes..."
  git add .
  git commit -m "$COMMIT_MSG (Release $VERSION)" || echo "No code changes to commit."

  # Check if tag exists
  if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "⚠️  Tag $VERSION already exists locally. Updating..."
    git tag -d "$VERSION"
  fi

  echo "🏷️  Tagging $VERSION..."
  git tag -a "$VERSION" -m "$COMMIT_MSG"

  echo "⬆️  Pushing to origin..."
  git push origin main
  git push origin "$VERSION" --force

  echo "📦 Creating GitHub Release..."
  gh release create "$VERSION" \
    --title "$VERSION" \
    --notes "$COMMIT_MSG" \
    "$OUT_DIR"/* \
    || echo "⚠️  GitHub release creation failed (may already exist)"

  echo "✅ Release $VERSION processed successfully!"
  echo "   🔗 View at: https://github.com/daverage/tinymem/releases/tag/$VERSION"
else
  echo
  echo "Build complete. Artifacts in $OUT_DIR"
fi
