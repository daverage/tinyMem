#!/bin/bash

# tinyMem Build & Release Script
# Builds platform binaries and handles full release lifecycle if requested.
#
# Usage:
#   ./build/build.sh                          (Lightweight build ~50-60 MB)
#   ./build/build.sh [major|minor|patch]      (Full release cycle)
#
# Build Options:
#   Default: Lightweight build with FTS5 only
#     - Binary size: ~50-60 MB
#     - Semantic search requires HTTP endpoint (Ollama, etc.)
#     - All features enabled (CoVe, Ralph, semantic)
#
#   Full build with embedded model:
#     TINYMEM_EXTRA_BUILD_TAGS="embeddings" ./build/build.sh
#     - Binary size: ~150-200 MB
#     - Includes embedded nomic-embed-text-v1.5 model
#     - Semantic search works offline, no external service needed
#     - Model file must exist: internal/embedding/models/nomic-embed-text-v1.5.Q4_K_M.gguf
#
#   Minimal build (no LLM features):
#     TINYMEM_EXTRA_BUILD_TAGS="nollm" ./build/build.sh
#     - Binary size: ~14 MB
#     - Disables CoVe and Ralph (which require text generation LLMs)
#     - Semantic search still available via HTTP endpoint
#     - Useful for constrained environments or MCP-only deployments
#
#   Semantic-only build (embedded search, no LLM features):
#     TINYMEM_EXTRA_BUILD_TAGS="embeddings nollm" ./build/build.sh
#     - Binary size: ~95 MB
#     - Embedded semantic search works offline
#     - CoVe and Ralph disabled
#
# Examples:
#   ./build/build.sh                                         # Lightweight build
#   TINYMEM_EXTRA_BUILD_TAGS="embeddings" ./build/build.sh   # Full build with embeddings
#   TINYMEM_EXTRA_BUILD_TAGS="nollm" ./build/build.sh        # Minimal build
#   TINYMEM_EXTRA_BUILD_TAGS="embeddings nollm" ./build/build.sh  # Semantic-only build

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
if [[ -n "${TINYMEM_EXTRA_BUILD_TAGS:-}" ]]; then
  read -r -a EXTRA <<< "$TINYMEM_EXTRA_BUILD_TAGS"
  BUILD_TAGS+=("${EXTRA[@]}")
fi

# Check for embeddings tag and warn about cross-compilation
USING_EMBEDDINGS=false
for tag in "${BUILD_TAGS[@]}"; do
  if [[ "$tag" == "embeddings" ]]; then
    USING_EMBEDDINGS=true
    break
  fi
done

if [ "$USING_EMBEDDINGS" = true ]; then
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "⚠️  WARNING: Building with embedded embeddings"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  echo "For BEST RESULTS, use platform-native build scripts:"
  echo "  • macOS:   ./build/build-macos.sh"
  echo "  • Linux:   ./build/build-linux.sh"
  echo "  • Windows: ./build/build-windows.bat"
  echo
  echo "Cross-compilation with embeddings may fail due to"
  echo "platform-specific shared library requirements."
  echo
  echo "See docs/RELEASE_WORKFLOW.md for multi-platform builds."
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  read -p "Continue anyway? [y/N] " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted. Use platform-native script instead."
    exit 0
  fi
  echo
fi

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

  echo "✅ Release $VERSION processed successfully!"
else
  echo
  echo "Build complete. Artifacts in $OUT_DIR"
fi
