#!/bin/bash

# tinyMem Native macOS Builder
# This script must be run ON macOS to build binaries with embedded embeddings.
# It handles compilation of the required libllama_go.dylib library.

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
echo -e "${BLUE}  tinyMem Native macOS Builder${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# ============================================================
# Platform Check
# ============================================================
if [[ "$OSTYPE" != "darwin"* ]]; then
  echo -e "${RED}❌ Error: This script must be run on macOS.${NC}"
  echo "   For Linux, use build-linux.sh"
  echo "   For Windows, use build-windows.bat"
  exit 1
fi

echo -e "${GREEN}✓${NC} Platform: macOS ($(uname -m))"
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
# Check/Build libllama_go.dylib
# ============================================================
echo -e "${BLUE}Checking libllama_go.dylib for embedded embeddings...${NC}"

DYLIB_SEARCH_PATHS=(
  "/usr/local/lib/libllama_go.dylib"
  "/usr/lib/libllama_go.dylib"
  "$PROJECT_ROOT/libllama_go.dylib"
  "$HOME/.local/lib/libllama_go.dylib"
)

DYLIB_FOUND=""
for path in "${DYLIB_SEARCH_PATHS[@]}"; do
  if [[ -f "$path" ]]; then
    DYLIB_FOUND="$path"
    echo -e "${GREEN}✓${NC} Found: $DYLIB_FOUND"
    break
  fi
done

if [[ -z "$DYLIB_FOUND" ]]; then
  echo -e "${YELLOW}⚠ libllama_go.dylib not found in standard locations.${NC}"
  echo
  echo "Options:"
  echo "  1. Build it automatically (requires CMake, C++ compiler)"
  echo "  2. Skip full builds (only build lightweight version)"
  echo
  read -p "Build libllama_go.dylib now? [y/N] " -n 1 -r
  echo

  if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Check for CMake
    if ! command -v cmake >/dev/null 2>&1; then
      echo -e "${RED}❌ CMake not found. Install with: brew install cmake${NC}"
      exit 1
    fi

    # Check for C++ compiler
    if ! command -v clang++ >/dev/null 2>&1; then
      echo -e "${RED}❌ C++ compiler not found. Install Xcode Command Line Tools.${NC}"
      exit 1
    fi

    echo -e "${BLUE}Building libllama_go.dylib from source...${NC}"

    # Create temporary directory
    BUILD_DIR=$(mktemp -d)
    trap "rm -rf $BUILD_DIR" EXIT

    cd "$BUILD_DIR"
    echo -e "${BLUE}→${NC} Cloning kelindar/search..."
    git clone --quiet --recurse-submodules https://github.com/kelindar/search.git
    cd search

    echo -e "${BLUE}→${NC} Pulling LFS files..."
    git lfs pull 2>/dev/null || echo "  (git-lfs not installed, continuing...)"

    echo -e "${BLUE}→${NC} Configuring build..."
    mkdir build && cd build
    cmake -DBUILD_SHARED_LIBS=ON \
          -DCMAKE_BUILD_TYPE=Release \
          -DCMAKE_OSX_ARCHITECTURES="$(uname -m)" \
          .. >/dev/null

    echo -e "${BLUE}→${NC} Compiling (this may take a few minutes)..."
    cmake --build . --config Release >/dev/null

    if [[ ! -f "libllama_go.dylib" ]]; then
      echo -e "${RED}❌ Build failed - libllama_go.dylib not created${NC}"
      exit 1
    fi

    # Install to project directory
    echo -e "${BLUE}→${NC} Installing to $PROJECT_ROOT..."
    cp libllama_go.dylib "$PROJECT_ROOT/"
    DYLIB_FOUND="$PROJECT_ROOT/libllama_go.dylib"

    cd "$PROJECT_ROOT"
    echo -e "${GREEN}✓${NC} Built and installed: $DYLIB_FOUND"
    echo
  else
    echo -e "${YELLOW}Skipping full builds. Only lightweight version will be built.${NC}"
    SKIP_FULL=true
    echo
  fi
fi

# ============================================================
# Prepare Output Directory
# ============================================================
OUT_DIR="build/releases"
mkdir -p "$OUT_DIR"

echo -e "${BLUE}Building tinyMem for macOS...${NC}"
echo

# ============================================================
# Build Lightweight (No Embeddings)
# ============================================================
echo -e "${BLUE}→ Building lightweight version (HTTP embeddings only)...${NC}"

ARCH=$(uname -m)
LIGHTWEIGHT_OUTPUT="$OUT_DIR/tinymem-darwin-$ARCH-lite"

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
if [[ "${SKIP_FULL:-false}" != "true" ]]; then
  echo -e "${BLUE}→ Building full version (embedded model)...${NC}"

  FULL_OUTPUT="$OUT_DIR/tinymem-darwin-$ARCH"

  # Ensure dylib is in search path for build
  export DYLD_LIBRARY_PATH="${DYLIB_FOUND%/*}:${DYLD_LIBRARY_PATH:-}"

  CGO_ENABLED=1 go build \
    -tags "fts5 embeddings" \
    -ldflags "-X github.com/daverage/tinymem/internal/version.Version=${VERSION}" \
    -o "$FULL_OUTPUT" \
    ./cmd/tinymem

  SIZE_FULL=$(ls -lh "$FULL_OUTPUT" | awk '{print $5}')
  echo -e "${GREEN}✓${NC} Full: $FULL_OUTPUT (${SIZE_FULL})"

  # Copy dylib alongside binary for distribution
  cp "$DYLIB_FOUND" "$OUT_DIR/libllama_go.dylib"
  echo -e "${GREEN}✓${NC} Bundled: libllama_go.dylib"
fi

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
  echo "     brew install zig          (recommended)"
  echo "     brew install mingw-w64    (alternative)"
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
    if [[ "${SKIP_FULL:-false}" != "true" ]]; then
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
ls -lh "$OUT_DIR/" | grep tinymem
echo

if [[ "${SKIP_FULL:-false}" != "true" ]]; then
  echo -e "${YELLOW}Note: Full build requires libllama_go.dylib at runtime.${NC}"
  echo "      Copy both files together for distribution:"
  echo "        - tinymem-darwin-$ARCH"
  echo "        - libllama_go.dylib"
  echo
fi

echo -e "${BLUE}Test builds:${NC}"
echo "  $LIGHTWEIGHT_OUTPUT version"
if [[ "${SKIP_FULL:-false}" != "true" ]]; then
  echo "  DYLD_LIBRARY_PATH=$OUT_DIR $FULL_OUTPUT version"
fi
echo
