# tinyMem Release Quick Reference

## 🚀 One-Command Release

```bash
./build/build.sh [major|minor|patch]
```

**What it does:**
1. ✅ Cross-compiles for all platforms (6 binaries)
2. ✅ Bumps version automatically  
3. ✅ Updates `internal/version/version.go`
4. ✅ Commits changes with your message
5. ✅ Creates git tag
6. ✅ Pushes to GitHub
7. ✅ Creates GitHub Release with binaries

---

## 📦 Release Types

| Command | Use When | Example |
|---------|----------|---------|
| `./build/build.sh patch` | Bug fixes, minor updates | v0.3.0 → v0.3.1 |
| `./build/build.sh minor` | New features, non-breaking | v0.3.0 → v0.4.0 |
| `./build/build.sh major` | Breaking changes | v0.3.0 → v1.0.0 |

---

## 🛠️ Development Build

```bash
./build/build.sh
# Outputs to build/releases/
# No git operations
```

---

## ✅ Pre-Release Checklist

1. All tests passing: `go test -tags fts5 ./...`
2. Documentation updated
3. CHANGELOG.md updated (if exists)
4. Clean git status: `git status`

---

## 📋 Example Release Session

```bash
# Check current state
git status
go test -tags fts5 ./...

# Run release
./build/build.sh patch

# Prompts:
# 🚀 Preparing Release: v0.4.0 (Current: v0.3.0)
# [builds all binaries...]
# Build successful. Commit message for v0.4.0: _

# Enter your commit message:
De-scope tinyMem to memory governance only

# Script completes:
# 📝 Updating internal/version/version.go...
# 💾 Committing changes...
# 🏷️  Tagging v0.4.0...
# ⬆️  Pushing to origin...
# 📦 Creating GitHub Release...
# ✅ Release v0.4.0 processed successfully!
#    🔗 View at: https://github.com/daverage/tinymem/releases/tag/v0.4.0
```

---

## 🎯 Current Build Configuration

**Single Mode (Simplified):**
- FTS5 lexical recall
- CoVe filtering  
- Evidence-gated truth
- Mode enforcement
- Build tag: `fts5`

**Platforms:**
- macOS (ARM64, AMD64)
- Linux (AMD64, ARM64)
- Windows (AMD64, ARM64)

---

## 🔧 Troubleshooting

**"gh: command not found"**
```bash
brew install gh
gh auth login
```

**Cross-compilation fails (Linux/Windows)**
```bash
# Install zig (recommended)
brew install zig

# Or mingw-w64 (Windows only)
brew install mingw-w64
```

**Version mismatch**
```bash
# Manual version update
nano internal/version/version.go
# Change: var Version = "vX.Y.Z"
```

---

## 📍 Where Everything Goes

| What | Where |
|------|-------|
| Binaries | `build/releases/` |
| Version file | `internal/version/version.go` |
| Build script | `build/build.sh` |
| GitHub releases | `github.com/daverage/tinymem/releases` |

---

**Pro Tip:** The build script handles EVERYTHING. Just run it and provide a commit message. ✨
