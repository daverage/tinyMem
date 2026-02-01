#!/bin/bash

# tinyMem Native Linux Builder
# This script must be run ON Linux to build binaries with embedded embeddings.
# It uses precompiled libllama_go.so from kelindar/search.

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
# Setup libllama_go.so from kelindar/search
# ============================================================
echo -e "${BLUE}Setting up libllama_go.so for embedded embeddings...${NC}"

# Detect CPU capabilities
if grep -q avx2 /proc/cpuinfo; then
  VARIANT="avx"
  echo -e "${GREEN}✓${NC} CPU supports AVX2"
elif command -v vulkaninfo >/dev/null 2>&1; then
  VARIANT="vulkan"
  echo -e "${GREEN}✓${NC} Vulkan support detected"
else
  VARIANT="avx"
  echo -e "${YELLOW}⚠${NC} Using AVX variant (default)"
fi

# Architecture
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
  SEARCH_ARCH="x64"
elif [[ "$ARCH" == "aarch64" ]]; then
  SEARCH_ARCH="arm64"
else
  echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
  exit 1
fi

# Check if kelindar/search is in go mod cache
GO_MOD_CACHE=$(go env GOMODCACHE)
SEARCH_PKG_PATH="$GO_MOD_CACHE/github.com/kelindar/search@v0.4.0"

if [[ ! -d "$SEARCH_PKG_PATH" ]]; then
  echo -e "${YELLOW}⚠${NC} kelindar/search not in module cache, downloading..."
  go mod download github.com/kelindar/search
fi

# Find the appropriate precompiled library
SO_SOURCE=""
if [[ "$SEARCH_ARCH" == "x64" ]]; then
  SO_SOURCE="$SEARCH_PKG_PATH/dist/linux-$SEARCH_ARCH-$VARIANT/libllama_go.so"
else
  echo -e "${YELLOW}⚠${NC} No precompiled library for $ARCH, will try x64..."
  SO_SOURCE="$SEARCH_PKG_PATH/dist/linux-x64-$VARIANT/libllama_go.so"
fi

if [[ ! -f "$SO_SOURCE" ]]; then
  echo -e "${RED}❌ Could not find precompiled library at:${NC}"
  echo "   $SO_SOURCE"
  echo
  echo "Available libraries:"
  ls -1 "$SEARCH_PKG_PATH/dist/" 2>/dev/null || echo "  (none found)"
  exit 1
fi

# Copy to project directory
SO_DEST="$PROJECT_ROOT/libllama_go.so"
cp "$SO_SOURCE" "$SO_DEST"
echo -e "${GREEN}✓${NC} Copied: $SO_SOURCE"
echo -e "${GREEN}✓${NC} To: $SO_DEST"
echo

# ============================================================
# Prepare Output Directory
# ============================================================
OUT_DIR="build/releases"
mkdir -p "$OUT_DIR"

echo -e "${BLUE}Building tinyMem for Linux...${NC}"
echo

# ============================================================
# Build Lightweight (No Embeddings)
# ============================================================
echo -e "${BLUE}→ Building lightweight version (HTTP embeddings only)...${NC}"

LIGHTWEIGHT_OUTPUT="$OUT_DIR/tinymem-linux-$ARCH-lite"

CGO_ENABLED=1 go build \
  -tags "fts5" \
  -ldflags "-X github.com/daverage/tinymem/internal/version.Version=${VERSION}" \
  -o "$LIGHTWEIGHT_OUTPUT" \
  ./cmd/tinymem

SIZE_LITE=$(ls -lh "$LIGHTWEIGHT_OUTPUT" | awk '{print $5}')
echo -e "${GREEN}✓${NC} Lightweight: $LIGHTWEIGHT_OUTPUT (${SIZE_LITE})"

# ============================================================
# Build Full (With Embedded Embeddings)
# ============================================================
echo -e "${BLUE}→ Building full version (embedded model)...${NC}"

FULL_OUTPUT="$OUT_DIR/tinymem-linux-$ARCH"

# Set library path for build
export LD_LIBRARY_PATH="$PROJECT_ROOT:${LD_LIBRARY_PATH:-}"

CGO_ENABLED=1 go build \
  -tags "fts5 embeddings" \
  -ldflags "-X github.com/daverage/tinymem/internal/version.Version=${VERSION}" \
  -o "$FULL_OUTPUT" \
  ./cmd/tinymem

SIZE_FULL=$(ls -lh "$FULL_OUTPUT" | awk '{print $5}')
echo -e "${GREEN}✓${NC} Full: $FULL_OUTPUT (${SIZE_FULL})"

# Copy .so alongside binary for distribution
cp "$SO_DEST" "$OUT_DIR/libllama_go.so"
echo -e "${GREEN}✓${NC} Bundled: libllama_go.so ($VARIANT variant)"

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

    # ============================================================
    # Windows Lightweight Build
    # ============================================================
    echo -e "${BLUE}→ Building Windows lightweight version...${NC}"

    WINDOWS_LITE_OUTPUT="$OUT_DIR/tinymem-windows-amd64-lite.exe"

    if [[ "$CC_WINDOWS" == "zig" ]]; then
      CGO_ENABLED=1 \
      GOOS=windows \
      GOARCH=amd64 \
      CC="zig cc -target x86_64-windows-gnu" \
      CXX="zig c++ -target x86_64-windows-gnu" \
      go build \
        -tags "fts5" \
        -ldflags "-X github.com/daverage/tinymem/internal/version.Version=${VERSION}" \
        -o "$WINDOWS_LITE_OUTPUT" \
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
        -o "$WINDOWS_LITE_OUTPUT" \
        ./cmd/tinymem
    fi

    SIZE_WIN_LITE=$(ls -lh "$WINDOWS_LITE_OUTPUT" | awk '{print $5}')
    echo -e "${GREEN}✓${NC} Windows Lightweight: $WINDOWS_LITE_OUTPUT (${SIZE_WIN_LITE})"

    # ============================================================
    # Windows Full Build (with embeddings)
    # ============================================================
    echo -e "${BLUE}→ Building Windows full version (with embedded model)...${NC}"
    echo -e "${YELLOW}⚠ Note: Windows DLL for llama.cpp must be provided separately${NC}"
    echo "   Download from: https://github.com/ggerganov/llama.cpp/releases"
    echo "   Look for: llama-<version>-bin-win-avx2-x64.zip"
    echo "   Extract: llama.dll and rename to llama_go.dll"
    echo

    # Check if Windows DLL exists
    WINDOWS_DLL_PATHS=(
      "$PROJECT_ROOT/llama_go.dll"
      "$OUT_DIR/llama_go.dll"
    )

    WINDOWS_DLL=""
    for dll in "${WINDOWS_DLL_PATHS[@]}"; do
      if [[ -f "$dll" ]]; then
        WINDOWS_DLL="$dll"
        echo -e "${GREEN}✓${NC} Found Windows DLL: $WINDOWS_DLL"
        break
      fi
    done

    if [[ -z "$WINDOWS_DLL" ]]; then
      echo -e "${YELLOW}⚠ llama_go.dll not found - skipping Windows full build${NC}"
      echo "   Place llama_go.dll in project root or build/releases/ and re-run"
      echo
    else
      WINDOWS_FULL_OUTPUT="$OUT_DIR/tinymem-windows-amd64.exe"

      if [[ "$CC_WINDOWS" == "zig" ]]; then
        CGO_ENABLED=1 \
        GOOS=windows \
        GOARCH=amd64 \
        CC="zig cc -target x86_64-windows-gnu" \
        CXX="zig c++ -target x86_64-windows-gnu" \
        go build \
          -tags "fts5 embeddings" \
          -ldflags "-X github.com/daverage/tinymem/internal/version.Version=${VERSION}" \
          -o "$WINDOWS_FULL_OUTPUT" \
          ./cmd/tinymem
      else
        # mingw-w64
        CGO_ENABLED=1 \
        GOOS=windows \
        GOARCH=amd64 \
        CC="x86_64-w64-mingw32-gcc" \
        CXX="x86_64-w64-mingw32-g++" \
        go build \
          -tags "fts5 embeddings" \
          -ldflags "-X github.com/daverage/tinymem/internal/version.Version=${VERSION}" \
          -o "$WINDOWS_FULL_OUTPUT" \
          ./cmd/tinymem
      fi

      SIZE_WIN_FULL=$(ls -lh "$WINDOWS_FULL_OUTPUT" | awk '{print $5}')
      echo -e "${GREEN}✓${NC} Windows Full: $WINDOWS_FULL_OUTPUT (${SIZE_WIN_FULL})"

      # Copy DLL for distribution
      cp "$WINDOWS_DLL" "$OUT_DIR/llama_go.dll"
      echo -e "${GREEN}✓${NC} Bundled: llama_go.dll"
      echo
    fi

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
ls -lh "$OUT_DIR/" | grep -E "tinymem|libllama"
echo

echo -e "${YELLOW}Note: Full build requires libllama_go.so at runtime.${NC}"
echo "      Copy both files together for distribution:"
echo "        - tinymem-linux-$ARCH"
echo "        - libllama_go.so"
echo
echo "      Or install system-wide:"
echo "        sudo cp $OUT_DIR/libllama_go.so /usr/local/lib/"
echo "        sudo ldconfig"
echo

echo -e "${BLUE}Test builds:${NC}"
echo "  $LIGHTWEIGHT_OUTPUT version"
echo "  LD_LIBRARY_PATH=$OUT_DIR $FULL_OUTPUT version"
echo
