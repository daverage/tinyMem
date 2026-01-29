#!/bin/bash

# tinyMem Release Script (macOS / Linux)
# Usage: ./build/release.sh [major|minor|patch]

set -euo pipefail

DIST_DIR="build/releases"

# ------------------------------------------------------------
# Safety checks
# ------------------------------------------------------------
if [[ -n "$(git status -s)" ]]; then
  echo "❌ Working directory is not clean."
  echo "Commit or stash changes before releasing."
  git status -s
  exit 1
fi

if git ls-files --error-unmatch "$DIST_DIR" >/dev/null 2>&1; then
  echo "❌ $DIST_DIR is tracked by git. This is forbidden."
  echo "Release artifacts must not be committed."
  exit 1
fi

command -v gh >/dev/null || {
  echo "❌ GitHub CLI (gh) not installed."
  exit 1
}

# ------------------------------------------------------------
# Version calculation
# ------------------------------------------------------------
LATEST_TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)"
IFS='.' read -r MAJOR MINOR PATCH <<< "${LATEST_TAG#v}"

MODE="${1:-patch}"
case "$MODE" in
  major) ((MAJOR++)); MINOR=0; PATCH=0 ;;
  minor) ((MINOR++)); PATCH=0 ;;
  *)     ((PATCH++)) ;;
esac

NEW_TAG="v$MAJOR.$MINOR.$PATCH"

echo
echo "🚀 Release $NEW_TAG"
read -p "Continue? (y/N): " CONFIRM
[[ "$CONFIRM" =~ ^[Yy]$ ]] || exit 1

# ------------------------------------------------------------
# Update version.go
# ------------------------------------------------------------
sed -i.bak "s/var Version = \".*\"/var Version = \"$NEW_TAG\"/" \
  internal/version/version.go
rm internal/version/version.go.bak

# ------------------------------------------------------------
# Build
# ------------------------------------------------------------
rm -rf "$DIST_DIR"
./build/build.sh

# ------------------------------------------------------------
# Commit + tag
# ------------------------------------------------------------
git add internal/version/version.go
git commit -m "Release $NEW_TAG"
git tag -a "$NEW_TAG" -m "Release $NEW_TAG"

git push origin main
git push origin "$NEW_TAG"

# ------------------------------------------------------------
# GitHub Release
# ------------------------------------------------------------
gh release create "$NEW_TAG" \
  --title "tinyMem $NEW_TAG" \
  --notes "Release $NEW_TAG" \
  "$DIST_DIR"/*

echo "✅ Release $NEW_TAG published"
