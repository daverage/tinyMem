# tinyMem Release Workflow

This guide documents the coordinated multi-platform build process for tinyMem releases, especially for builds with embedded embeddings.

## Overview

tinyMem supports two build variants:

1. **Lightweight** (~15 MB) - No embedded model, uses HTTP for embeddings
2. **Full** (~96 MB) - Embedded model, works offline

For **full builds with embeddings**, platform-native compilation is **required** because the kelindar/search library uses platform-specific shared libraries.

## Platform-Specific Build Requirements

| Platform | Script | Native Build Required | Precompiled Library |
|----------|--------|----------------------|---------------------|
| **macOS** | `build-macos.sh` | ✅ Yes | ❌ No - must compile libllama_go.dylib |
| **Linux** | `build-linux.sh` | ✅ Yes | ✅ Yes - uses dist/linux-x64-{avx,vulkan}/libllama_go.so |
| **Windows** | `build-windows.bat` | ✅ Yes | ✅ Yes - uses dist/win-x64-{avx,vulkan}/llama_go.dll |

## Release Process

### Option 1: Multi-Platform Coordinated Builds (Recommended)

This approach ensures the best quality builds with proper platform-specific optimizations.

#### Step 1: Prepare on macOS

```bash
# On macOS machine
cd /path/to/tinyMem
git pull origin main

# Run native macOS builder
./build/build-macos.sh

# Verify builds
ls -lh build/releases/
# Should see:
#   tinymem-darwin-arm64-lite
#   tinymem-darwin-arm64
#   libllama_go.dylib

# Collect artifacts
mkdir -p release-artifacts/macos
cp build/releases/tinymem-darwin-* release-artifacts/macos/
cp build/releases/libllama_go.dylib release-artifacts/macos/

# Test before uploading
./build/releases/tinymem-darwin-arm64-lite version
DYLD_LIBRARY_PATH=build/releases ./build/releases/tinymem-darwin-arm64 version
```

#### Step 2: Build on Linux

```bash
# On Linux machine (Ubuntu, Debian, or similar)
cd /path/to/tinyMem
git pull origin main

# Run native Linux builder
./build/build-linux.sh

# Verify builds
ls -lh build/releases/
# Should see:
#   tinymem-linux-x86_64-lite
#   tinymem-linux-x86_64
#   libllama_go.so

# Collect artifacts
mkdir -p release-artifacts/linux
cp build/releases/tinymem-linux-* release-artifacts/linux/
cp build/releases/libllama_go.so release-artifacts/linux/

# Test before uploading
./build/releases/tinymem-linux-x86_64-lite version
LD_LIBRARY_PATH=build/releases ./build/releases/tinymem-linux-x86_64 version
```

#### Step 3: Build on Windows

```powershell
# On Windows machine
cd C:\path\to\tinyMem
git pull origin main

# Run native Windows builder
.\build\build-windows.bat

# Verify builds
dir build\releases\
# Should see:
#   tinymem-windows-x64-lite.exe
#   tinymem-windows-x64.exe
#   llama_go.dll

# Collect artifacts
mkdir release-artifacts\windows
copy build\releases\tinymem-windows-* release-artifacts\windows\
copy build\releases\llama_go.dll release-artifacts\windows\

# Test before uploading
.\build\releases\tinymem-windows-x64-lite.exe version
.\build\releases\tinymem-windows-x64.exe version
```

#### Step 4: Aggregate and Publish

Collect all platform artifacts in one location:

```
release-artifacts/
├── macos/
│   ├── tinymem-darwin-arm64-lite
│   ├── tinymem-darwin-arm64
│   └── libllama_go.dylib
├── linux/
│   ├── tinymem-linux-x86_64-lite
│   ├── tinymem-linux-x86_64
│   └── libllama_go.so
└── windows/
    ├── tinymem-windows-x64-lite.exe
    ├── tinymem-windows-x64.exe
    └── llama_go.dll
```

Create release archives:

```bash
# On the aggregation machine
cd release-artifacts

# macOS
tar -czf tinymem-macos-arm64.tar.gz -C macos tinymem-darwin-arm64 libllama_go.dylib
tar -czf tinymem-macos-arm64-lite.tar.gz -C macos tinymem-darwin-arm64-lite

# Linux
tar -czf tinymem-linux-x86_64.tar.gz -C linux tinymem-linux-x86_64 libllama_go.so
tar -czf tinymem-linux-x86_64-lite.tar.gz -C linux tinymem-linux-x86_64-lite

# Windows
cd windows
7z a ../tinymem-windows-x64.zip tinymem-windows-x64.exe llama_go.dll
7z a ../tinymem-windows-x64-lite.zip tinymem-windows-x64-lite.exe
cd ..
```

#### Step 5: Create GitHub Release

```bash
# Tag the release
git tag -a v0.3.1 -m "Release v0.3.1 - Embedded Embeddings"
git push origin v0.3.1

# Create GitHub release with artifacts
gh release create v0.3.1 \
  --title "v0.3.1 - Embedded Embeddings" \
  --notes-file CHANGELOG.md \
  tinymem-macos-arm64.tar.gz \
  tinymem-macos-arm64-lite.tar.gz \
  tinymem-linux-x86_64.tar.gz \
  tinymem-linux-x86_64-lite.tar.gz \
  tinymem-windows-x64.zip \
  tinymem-windows-x64-lite.zip
```

### Option 2: Single-Platform Builds Only

If you only have access to one platform, build for that platform only:

```bash
# macOS
./build/build-macos.sh

# Linux
./build/build-linux.sh

# Windows
.\build\build-windows.bat
```

Then publish with a note: "macOS/Linux/Windows builds only in this release."

### Option 3: CI/CD Automation (Future)

For automated releases, use GitHub Actions with platform-specific runners:

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build-macos:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build macOS binaries
        run: ./build/build-macos.sh
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: macos-artifacts
          path: build/releases/

  build-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build Linux binaries
        run: ./build/build-linux.sh
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: linux-artifacts
          path: build/releases/

  build-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build Windows binaries
        run: .\build\build-windows.bat
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: windows-artifacts
          path: build/releases/

  release:
    needs: [build-macos, build-linux, build-windows]
    runs-on: ubuntu-latest
    steps:
      - name: Download all artifacts
        uses: actions/download-artifact@v3
      - name: Create release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            macos-artifacts/*
            linux-artifacts/*
            windows-artifacts/*
```

## Pre-Release Checklist

Before creating a release, verify:

- [ ] All tests pass on all platforms
- [ ] Version bumped in `internal/version/version.go`
- [ ] CHANGELOG.md updated
- [ ] Documentation updated (README, EMBEDDINGS.md, etc.)
- [ ] Native builds tested on each platform
- [ ] Full builds tested with semantic search enabled
- [ ] Lightweight builds tested with HTTP fallback
- [ ] Shared libraries included in release archives

## Testing Release Artifacts

### macOS

```bash
# Extract
tar -xzf tinymem-macos-arm64.tar.gz

# Test
./tinymem-darwin-arm64 version
./tinymem-darwin-arm64 health

# Test with embeddings
cat > .tinyMem/config.toml << EOF
[recall]
semantic_enabled = true
EOF

./tinymem-darwin-arm64 write --type note --summary "test" --detail "testing embeddings"
./tinymem-darwin-arm64 query "test"
```

### Linux

```bash
# Extract
tar -xzf tinymem-linux-x86_64.tar.gz

# Test
./tinymem-linux-x86_64 version
./tinymem-linux-x86_64 health

# Test with embeddings
cat > .tinyMem/config.toml << EOF
[recall]
semantic_enabled = true
EOF

./tinymem-linux-x86_64 write --type note --summary "test" --detail "testing embeddings"
./tinymem-linux-x86_64 query "test"
```

### Windows

```powershell
# Extract
Expand-Archive tinymem-windows-x64.zip -DestinationPath .

# Test
.\tinymem-windows-x64.exe version
.\tinymem-windows-x64.exe health

# Test with embeddings
@"
[recall]
semantic_enabled = true
"@ | Out-File -FilePath .tinyMem\config.toml

.\tinymem-windows-x64.exe write --type note --summary "test" --detail "testing embeddings"
.\tinymem-windows-x64.exe query "test"
```

## Troubleshooting

### "Library not found" Errors

**macOS:**
```bash
# Check dylib is alongside binary
ls -l tinymem-darwin-arm64 libllama_go.dylib

# Or set library path
export DYLD_LIBRARY_PATH=.
./tinymem-darwin-arm64 version
```

**Linux:**
```bash
# Check .so is alongside binary
ls -l tinymem-linux-x86_64 libllama_go.so

# Or set library path
export LD_LIBRARY_PATH=.
./tinymem-linux-x86_64 version

# Or install system-wide
sudo cp libllama_go.so /usr/local/lib/
sudo ldconfig
```

**Windows:**
```powershell
# Check DLL is alongside executable
dir tinymem-windows-x64.exe llama_go.dll

# Or add to PATH
$env:PATH = "$(Get-Location);$env:PATH"
.\tinymem-windows-x64.exe version
```

### Build Script Fails

**Check dependencies:**
- Go 1.21+
- Git
- (macOS) Xcode Command Line Tools, CMake (for libllama_go.dylib compilation)
- (Linux) build-essential
- (Windows) Visual Studio Build Tools or MinGW

**Verify module cache:**
```bash
go mod download
go mod verify
```

## Version Numbering

Follow semantic versioning (MAJOR.MINOR.PATCH):

- **MAJOR**: Breaking changes to API or CLI
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

Examples:
- `v0.3.1` - Patch: Bug fixes, embedded embeddings
- `v0.4.0` - Minor: New Ralph features, CoVe improvements
- `v1.0.0` - Major: Stable API, production-ready

## Release Notes Template

```markdown
# tinyMem v0.3.1 - Embedded Embeddings

## New Features
- ✨ Built-in semantic search with embedded vector model (Linux/Windows)
- 🔄 Auto-detection between local and HTTP embeddings
- 📦 Dual build system: full (~96 MB) and lightweight (~15 MB)

## Improvements
- 📚 Comprehensive embedding documentation
- 🛠️ Platform-specific native build scripts
- ⚡ Optimized build process

## Platform Support
- **Linux** ✅ Full builds with embedded model
- **Windows** ✅ Full builds with embedded model
- **macOS** ⚠️ Requires libllama_go.dylib (see docs/EMBEDDINGS.md)

## Breaking Changes
None

## Bug Fixes
- Fixed import cleanup in server modules

## Documentation
- Added docs/EMBEDDINGS.md
- Updated examples/Configuration.md
- Added docs/RELEASE_WORKFLOW.md

## Downloads

### Full Builds (with embedded model)
- macOS: `tinymem-macos-arm64.tar.gz`
- Linux: `tinymem-linux-x86_64.tar.gz`
- Windows: `tinymem-windows-x64.zip`

### Lightweight Builds (HTTP embeddings only)
- macOS: `tinymem-macos-arm64-lite.tar.gz`
- Linux: `tinymem-linux-x86_64-lite.tar.gz`
- Windows: `tinymem-windows-x64-lite.zip`

## Checksums
```
SHA256 checksums will be added here
```
```

## Future Enhancements

- [ ] Automated multi-platform CI/CD pipeline
- [ ] Code signing for macOS and Windows builds
- [ ] Homebrew formula for macOS
- [ ] APT/RPM packages for Linux
- [ ] Chocolatey package for Windows
- [ ] Docker images with both variants
- [ ] Performance benchmarks in release notes
