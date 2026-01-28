#!/bin/bash

# tinyMem Release Automation Script
# Usage: ./build/release.sh [major|minor|patch] (default: patch)

set -e

# 1. Ensure working directory is clean
if [[ -n $(git status -s) ]]; then
    echo "❌ Error: Working directory is not clean. Please commit or stash changes first."
    git status -s
    exit 1
fi

# 2. Get latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "Current version: $LATEST_TAG"

# 3. Calculate new version
# Remove 'v' prefix
VERSION=${LATEST_TAG#v}
IFS='.' read -r -a PARTS <<< "$VERSION"
MAJOR=${PARTS[0]}
MINOR=${PARTS[1]}
PATCH=${PARTS[2]}

MODE=${1:-patch}

if [[ "$MODE" == "major" ]]; then
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
elif [[ "$MODE" == "minor" ]]; then
    MINOR=$((MINOR + 1))
    PATCH=0
else
    PATCH=$((PATCH + 1))
fi

NEW_TAG="v$MAJOR.$MINOR.$PATCH"

# 4. Confirm with user
echo ""
echo "------------------------------------------------"
echo "🚀 Ready to release: $NEW_TAG"
echo "------------------------------------------------"
echo "This will:"
echo "  1. Create git tag $NEW_TAG"
echo "  2. Run ./build/build.sh to generate binaries"
echo "  3. Push 'main' and tags to origin"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# 5. Create Tag
echo "🏷️  Tagging $NEW_TAG..."
git tag -a "$NEW_TAG" -m "Release $NEW_TAG"

# 6. Build
echo "🔨 Building binaries..."
./build/build.sh

# 7. Push
echo "⬆️  Pushing to origin..."
git push origin main
git push origin "$NEW_TAG"

echo ""
echo "✅ Release $NEW_TAG completed successfully!"
