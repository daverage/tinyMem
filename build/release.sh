#!/bin/bash

# tinyMem Release Automation Script
# Usage: ./build/release.sh [major|minor|patch] (default: patch)

set -euo pipefail

DIST_DIR="dist"

# ------------------------------------------------------------
# 1. Ensure working directory is clean
# ------------------------------------------------------------
if [[ -n $(git status -s) ]]; then
    echo "❌ Error: Working directory is not clean."
    git status -s
    exit 1
fi

# ------------------------------------------------------------
# 2. Get latest tag
# ------------------------------------------------------------
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "Current version: $LATEST_TAG"

# ------------------------------------------------------------
# 3. Calculate new version
# ------------------------------------------------------------
VERSION=${LATEST_TAG#v}
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

MODE=${1:-patch}

case "$MODE" in
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  *)
    PATCH=$((PATCH + 1))
    ;;
esac

NEW_TAG="v$MAJOR.$MINOR.$PATCH"

# ------------------------------------------------------------
# 4. Confirm
# ------------------------------------------------------------
echo ""
echo "------------------------------------------------"
echo "🚀 Ready to release: $NEW_TAG"
echo "------------------------------------------------"
echo "This will:"
echo "  1. Update internal/version/version.go"
echo "  2. Build binaries"
echo "  3. Commit changes"
echo "  4. Create git tag $NEW_TAG"
echo "  5. Push main + tag"
echo "  6. Create GitHub Release with binaries"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# ------------------------------------------------------------
# 5. Update version.go (portable sed)
# ------------------------------------------------------------
echo "📝 Updating version.go..."
sed -i.bak "s/var Version = \".*\"/var Version = \"$NEW_TAG\"/" internal/version/version.go
rm internal/version/version.go.bak

# ------------------------------------------------------------
# 6. Build
# ------------------------------------------------------------
echo "🔨 Building binaries..."
rm -rf "$DIST_DIR"
./build/build.sh

if [[ ! -d "$DIST_DIR" ]]; then
    echo "❌ Build did not produce $DIST_DIR/"
    exit 1
fi

# ------------------------------------------------------------
# 7. Commit
# ------------------------------------------------------------
echo "💾 Committing release..."
git add .
git commit -m "Release $NEW_TAG"

# ------------------------------------------------------------
# 8. Tag
# ------------------------------------------------------------
echo "🏷️  Tagging $NEW_TAG..."
git tag -a "$NEW_TAG" -m "Release $NEW_TAG"

# ------------------------------------------------------------
# 9. Push
# ------------------------------------------------------------
echo "⬆️  Pushing to origin..."
git push origin main
git push origin "$NEW_TAG"

# ------------------------------------------------------------
# 10. Create GitHub Release + upload binaries
# ------------------------------------------------------------
echo "📦 Creating GitHub Release..."

gh release create "$NEW_TAG" \
  --title "tinyMem $NEW_TAG" \
  --notes "Release $NEW_TAG" \
  "$DIST_DIR"/*

echo ""
echo "✅ Release $NEW_TAG published successfully!"
