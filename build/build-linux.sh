#!/bin/bash

# tinyMem Native Linux Builder
# This script builds tinyMem binaries on Linux.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  tinyMem Native Linux Builder${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# ============================================================
# Platform Check
# ============================================================
if [[ "$OSTYPE" != "linux"* ]]; then
  echo -e "${RED}❌ Error: This script must be run on Linux.${NC}"
  echo "   For macOS, use build-macos.sh"
  echo "   For Windows, use build-windows.bat"
  exit 1
fi

echo -e "${GREEN}✓${NC} Platform: Linux ($(uname -m))"
echo

# ============================================================
# Version Detection
# ============================================================
VERSION="$(git describe --tags --dirty --always 2>/dev/null || true)"
if [[ -z "$VERSION" ]]; then
  VERSION="$(grep -E 'var Version =' internal/version/version.go \
    | sed -E 's/.*"([^"]+)".*/\1/' || echo "dev")"
fi
echo -e "${GREEN}✓${NC} Version: $VERSION"
echo

# ============================================================
# Check Build Dependencies
# ============================================================
echo -e "${BLUE}Checking build dependencies...${NC}"

# Check for Go
if ! command -v go >/dev/null 2>&1; then
  echo -e "${RED}❌ Go not found. Please install Go 1.21 or later.${NC}"
  exit 1
fi
echo -e "${GREEN}✓${NC} Go $(go version | awk '{print $3}')"

# Check for git
if ! command -v git >/dev/null 2>&1; then
  echo -e "${RED}❌ Git not found. Please install Git.${NC}"
  exit 1
fi
echo -e "${GREEN}✓${NC} Git $(git --version | awk '{print $3}')"

echo

# ============================================================
# Prepare Output Directory
# ============================================================
OUT_DIR="build/releases"
mkdir -p "$OUT_DIR"

echo -e "${BLUE}Building tinyMem for Linux...${NC}"
echo

# ============================================================
# Build with FTS5 Lexical Recall
# ============================================================
echo -e "${BLUE}→ Building tinyMem (FTS5 + CoVe + Evidence + Mode Enforcement)...${NC}"

ARCH=$(uname -m)
OUTPUT="$OUT_DIR/tinymem-linux-$ARCH"

CGO_ENABLED=1 go build \
  -tags "fts5" \
  -ldflags "-X github.com/daverage/tinymem/internal/version.Version=${VERSION}" \
  -o "$OUTPUT" \
  ./cmd/tinymem

SIZE=$(ls -lh "$OUTPUT" | awk '{print $5}')
echo -e "${GREEN}✓${NC} Built: $OUTPUT (${SIZE})"

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# ============================================================
# Cross-Compile for Windows (Optional)
# ============================================================
echo -e "${BLUE}Cross-compilation for Windows...${NC}"
echo

# Check for cross-compilation tools
CROSS_COMPILE_WINDOWS=false
CC_WINDOWS=""

if command -v zig >/dev/null 2>&1; then
  echo -e "${GREEN}✓${NC} Found zig (recommended for cross-compilation)"
  CC_WINDOWS="zig"
  CROSS_COMPILE_WINDOWS=true
elif command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
  echo -e "${GREEN}✓${NC} Found mingw-w64"
  CC_WINDOWS="mingw-w64"
  CROSS_COMPILE_WINDOWS=true
else
  echo -e "${YELLOW}⚠${NC} No Windows cross-compiler found (zig or mingw-w64)"
  echo "   To enable Windows builds:"
  echo "     apt-get install zig          (recommended)"
  echo "     apt-get install mingw-w64    (alternative)"
fi

if [[ "$CROSS_COMPILE_WINDOWS" == "true" ]]; then
  echo
  read -p "Build Windows binaries? [y/N] " -n 1 -r
  echo

  if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}Building for Windows (amd64)...${NC}"
    echo

    WINDOWS_OUTPUT="$OUT_DIR/tinymem-windows-amd64.exe"

    if [[ "$CC_WINDOWS" == "zig" ]]; then
      CGO_ENABLED=1 \
      GOOS=windows \
      GOARCH=amd64 \
      CC="zig cc -target x86_64-windows-gnu" \
      CXX="zig c++ -target x86_64-windows-gnu" \
      go build \
        -tags "fts5" \
        -ldflags "-X github.com/daverage/tinymem/internal/version.Version=${VERSION}" \
        -o "$WINDOWS_OUTPUT" \
        ./cmd/tinymem
    else
      # mingw-w64
      CGO_ENABLED=1 \
      GOOS=windows \
      GOARCH=amd64 \
      CC="x86_64-w64-mingw32-gcc" \
      CXX="x86_64-w64-mingw32-g++" \
      go build \
        -tags "fts5" \
        -ldflags "-X github.com/daverage/tinymem/internal/version.Version=${VERSION}" \
        -o "$WINDOWS_OUTPUT" \
        ./cmd/tinymem
    fi

    SIZE_WIN=$(ls -lh "$WINDOWS_OUTPUT" | awk '{print $5}')
    echo -e "${GREEN}✓${NC} Windows: $WINDOWS_OUTPUT (${SIZE_WIN})"

    echo -e "${GREEN}✅ Windows cross-compilation complete!${NC}"
    echo
  else
    echo -e "${YELLOW}Skipping Windows cross-compilation${NC}"
    echo
  fi
else
  echo
fi

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Build complete!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
echo "Artifacts in: $OUT_DIR/"
ls -lh "$OUT_DIR/" | grep tinymem
echo

echo -e "${BLUE}Test build:${NC}"
echo "  $OUTPUT version"
echo
