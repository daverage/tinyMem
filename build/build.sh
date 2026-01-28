#!/bin/bash

# tinyMem Build Script
# Builds all platform binaries and places them in the releases folder

set -e  # Exit immediately if a command exits with a non-zero status

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to project root to ensure consistent path resolution
cd "$PROJECT_ROOT"

# Create releases directory if it doesn't exist
mkdir -p build/releases

# Determine default build tags (FTS5 required)
build_tags=("fts5")
if [[ -n "$TINYMEM_EXTRA_BUILD_TAGS" ]]; then
    read -r -a extra_tags <<< "$TINYMEM_EXTRA_BUILD_TAGS"
    for tag in "${extra_tags[@]}"; do
        if [[ -n "$tag" ]]; then
            build_tags+=("$tag")
        fi
    done
fi

tags_flag=(-tags "${build_tags[*]}")
tag_summary="${build_tags[*]}"

echo "Building tinyMem binaries (tags: ${tag_summary})..."

build_target() {
    local platform_label=$1
    local goos=$2
    local goarch=$3
    local output=$4

    local env_vars=(CGO_ENABLED=1)
    if [[ -n "$goos" ]]; then
        env_vars+=("GOOS=${goos}")
    fi
    if [[ -n "$goarch" ]]; then
        env_vars+=("GOARCH=${goarch}")
    fi

    echo "Building ${platform_label} (including icons for Windows)..."
    env "${env_vars[@]}" go build "${tags_flag[@]}" -o "${output}" ./cmd/tinymem
    echo "✓ Built ${output}"
}

# Build macOS binaries
build_target "macOS ARM64" darwin arm64 build/releases/tinymem-darwin-arm64
build_target "macOS AMD64" darwin amd64 build/releases/tinymem-darwin-amd64

# Build Windows binaries only if we're on a Windows-compatible system
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    build_target "Windows AMD64" windows amd64 build/releases/tinymem-windows-amd64.exe
    build_target "Windows ARM64" windows arm64 build/releases/tinymem-windows-arm64.exe
else
    echo "Skipping Windows builds (not on Windows system)"
    echo "  To build for Windows, run this script from a Windows system with appropriate toolchain"
fi

# Build Linux binaries only if we're on a Linux-compatible system
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    build_target "Linux AMD64" linux amd64 build/releases/tinymem-linux-amd64
    build_target "Linux ARM64" linux arm64 build/releases/tinymem-linux-arm64
else
    echo "Skipping Linux builds (not on Linux system)"
    echo "  To build for Linux, run this script from a Linux system with appropriate toolchain"
fi

echo ""
echo "Build completed successfully!"
echo ""
echo "Binaries created in build/releases/:"
ls -la build/releases/
echo ""
echo "Build script completed."
