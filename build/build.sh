#!/bin/bash

# tinyMem Build & Release Script
# Builds platform binaries and handles full release lifecycle if requested.
# Usage: 
#   ./build/build.sh                 (Build only)
#   ./build/build.sh [major|minor|patch] (Full release cycle)

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
  if [[ -n "$(git status -s)" ]]; then
    echo "❌ Working directory is not clean. Commit or stash changes before releasing."
    git status -s
    exit 1
  fi

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
  
  echo
  echo "🚀 Preparing Release: $VERSION (Current: $LATEST_TAG)"
  read -p "Continue? (y/N): " CONFIRM
  [[ "$CONFIRM" =~ ^[Yy]$ ]] || exit 1

  # Update version.go
  echo "📝 Updating internal/version/version.go..."
  sed -i.bak "s/var Version = ".*"/var Version = \"$VERSION\"/" \
    internal/version/version.go
  rm internal/version/version.go.bak

  # Commit version bump
  git add internal/version/version.go
  git commit -m "Bump version to $VERSION"
else
  # Read current version from code for standard build
  VERSION="$(grep -E 'var Version =' internal/version/version.go \
    | sed -E 's/.*\"([^\"]+)\".*/\1/' || true)"
  if [[ -z "$VERSION" ]]; then
    VERSION="$LATEST_TAG"
  fi
  echo "Building tinyMem version: $VERSION"
fi

# ------------------------------------------------------------
# Build Logic
# ------------------------------------------------------------
BUILD_TAGS=("fts5")
if [[ -n "${TINYMEM_EXTRA_BUILD_TAGS:-}" ]]; then
  read -r -a EXTRA <<< "$TINYMEM_EXTRA_BUILD_TAGS"
  BUILD_TAGS+=("${EXTRA[@]}")
fi

TAGS_FLAG=(-tags "${BUILD_TAGS[*]}")
LDFLAGS="-X github.com/daverage/tinymem/internal/version.Version=${VERSION}"

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

# Clear previous releases
rm -rf "$OUT_DIR"/*

# macOS
build_target "macOS ARM64" darwin arm64 "$OUT_DIR/tinymem-darwin-arm64"
build_target "macOS AMD64" darwin amd64 "$OUT_DIR/tinymem-darwin-amd64"

# Linux
if [[ "$OSTYPE" == linux* ]]; then
  build_target "Linux AMD64" linux amd64 "$OUT_DIR/tinymem-linux-amd64"
  build_target "Linux ARM64" linux arm64 "$OUT_DIR/tinymem-linux-arm64"
fi

# Windows
if [[ "$OSTYPE" == msys* || "$OSTYPE" == cygwin* ]]; then
  build_target "Windows AMD64" windows amd64 "$OUT_DIR/tinymem-windows-amd64.exe"
  build_target "Windows ARM64" windows arm64 "$OUT_DIR/tinymem-windows-arm64.exe"
fi

# ------------------------------------------------------------
# Finalize Release
# ------------------------------------------------------------
if [ "$IS_RELEASE" = true ]; then
  echo "🏷️  Tagging $VERSION..."
  git tag -a "$VERSION" -m "Release $VERSION"

  echo "⬆️  Pushing to origin..."
  git push origin main
  git push origin "$VERSION"

  echo "📦 Creating GitHub Release..."
  gh release create "$VERSION" \
    --title "tinyMem $VERSION" \
    --notes "Release $VERSION" \
    "$OUT_DIR"/*

  echo "✅ Release $VERSION published successfully!"
else
  echo
  echo "Build complete. Artifacts:"
  ls -lh "$OUT_DIR"
fi