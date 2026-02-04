package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactResult is the confirmation payload returned to the caller after a successful write.
type ArtifactResult struct {
	Path         string `json:"path"`
	BytesWritten int    `json:"bytes_written"`
	Created      bool   `json:"created"`
	SHA256       string `json:"sha256"`
	LinkTaskID   string `json:"link_task_id,omitempty"`
}

// ValidateArtifactPath resolves and constrains a relative path to the workspace root.
// It rejects absolute paths, directory-traversal sequences, and any resolved path that
// falls outside the workspace.  The returned string is the fully-resolved absolute path
// that is safe to write.
func ValidateArtifactPath(workspaceRoot, relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("artifact path must not be empty")
	}

	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("artifact path must be relative: %s", relPath)
	}

	cleaned := filepath.Clean(relPath)

	// Reject the workspace root itself and any leading traversal.
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("artifact path escapes workspace: %s", relPath)
	}

	// Resolve the workspace root (handles symlinks at root level).
	absRoot, err := filepath.EvalSymlinks(workspaceRoot)
	if err != nil {
		absRoot, err = filepath.Abs(workspaceRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve workspace root: %w", err)
		}
	}

	target := filepath.Join(absRoot, cleaned)

	// Defense-in-depth: use filepath.Rel to verify containment after join.
	rel, err := filepath.Rel(absRoot, target)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("artifact path escapes workspace: %s", relPath)
	}

	return target, nil
}

// WriteArtifact performs the server-owned file write after path validation.
// The write is atomic: content is written to a temporary file in the target
// directory and then renamed into place so a crash mid-write cannot leave a
// partial file.  Failure is explicit and fail-closed.
func WriteArtifact(workspaceRoot, relPath, content string, overwrite bool, linkTaskID string) (ArtifactResult, error) {
	absPath, err := ValidateArtifactPath(workspaceRoot, relPath)
	if err != nil {
		return ArtifactResult{}, err
	}

	created := true
	if _, statErr := os.Stat(absPath); statErr == nil {
		// File already exists.
		if !overwrite {
			return ArtifactResult{}, fmt.Errorf("artifact already exists and overwrite is false: %s", relPath)
		}
		created = false
	}

	// Ensure parent directories exist.
	dir := filepath.Dir(absPath)
	if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
		return ArtifactResult{}, fmt.Errorf("failed to create directory %s: %w", dir, mkErr)
	}

	// Atomic write: temp file in the same directory, then rename.
	tmp, mkErr := os.CreateTemp(dir, ".tinymem-artifact-*")
	if mkErr != nil {
		return ArtifactResult{}, fmt.Errorf("failed to create temp file: %w", mkErr)
	}
	tmpName := tmp.Name()

	data := []byte(content)
	if _, writeErr := tmp.Write(data); writeErr != nil {
		tmp.Close()
		os.Remove(tmpName)
		return ArtifactResult{}, fmt.Errorf("failed to write artifact: %w", writeErr)
	}
	if closeErr := tmp.Close(); closeErr != nil {
		os.Remove(tmpName)
		return ArtifactResult{}, fmt.Errorf("failed to close temp file: %w", closeErr)
	}

	if renameErr := os.Rename(tmpName, absPath); renameErr != nil {
		os.Remove(tmpName)
		return ArtifactResult{}, fmt.Errorf("failed to finalize artifact write: %w", renameErr)
	}

	h := sha256.New()
	h.Write(data)

	return ArtifactResult{
		Path:         relPath,
		BytesWritten: len(data),
		Created:      created,
		SHA256:       hex.EncodeToString(h.Sum(nil)),
		LinkTaskID:   linkTaskID,
	}, nil
}

// maxArtifactReadBytes is the ceiling on file size that artifact_read will
// return.  Files above this limit are likely binary or generated and should
// not be streamed into an agent context.
const maxArtifactReadBytes = 1 << 20 // 1 MiB

// excludedDirs are directories pruned from workspace listings.
// .tinyMem holds runtime state; .git is version-control noise.
var excludedDirs = map[string]bool{
	".tinyMem":     true,
	".git":         true,
	"node_modules": true,
}

// isTaskManagerOwned returns true when the cleaned relative path refers to the
// root-level tinyTasks.md — the single file owned exclusively by TaskManager
// under Phase 1 invariants.  Nested files with the same name are not excluded.
func isTaskManagerOwned(cleanedRel string) bool {
	return cleanedRel == "tinyTasks.md"
}

// ArtifactReadResult is the payload returned by a successful artifact read.
type ArtifactReadResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

// ReadArtifact reads a workspace artifact after path validation.  It enforces
// a size ceiling and explicitly blocks TaskManager-owned files.
func ReadArtifact(workspaceRoot, relPath string) (ArtifactReadResult, error) {
	absPath, err := ValidateArtifactPath(workspaceRoot, relPath)
	if err != nil {
		return ArtifactReadResult{}, err
	}

	if isTaskManagerOwned(filepath.Clean(relPath)) {
		return ArtifactReadResult{}, fmt.Errorf("tinyTasks.md is owned by TaskManager; use the task_* tools")
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return ArtifactReadResult{}, fmt.Errorf("artifact not found: %s", relPath)
	}
	if info.IsDir() {
		return ArtifactReadResult{}, fmt.Errorf("%s is a directory; use artifact_list to enumerate", relPath)
	}
	if info.Size() > maxArtifactReadBytes {
		return ArtifactReadResult{}, fmt.Errorf("artifact %s exceeds the read size limit (%d bytes; max %d)", relPath, info.Size(), maxArtifactReadBytes)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return ArtifactReadResult{}, fmt.Errorf("failed to read artifact: %w", err)
	}

	h := sha256.New()
	h.Write(data)

	return ArtifactReadResult{
		Path:    relPath,
		Content: string(data),
		Size:    info.Size(),
		SHA256:  hex.EncodeToString(h.Sum(nil)),
	}, nil
}

// ArtifactEntry describes a single file discovered during a workspace listing.
type ArtifactEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// ListArtifacts walks the workspace and returns visible files, optionally
// filtered by a glob pattern.  Internal directories (.tinyMem, .git,
// node_modules) and the root-level tinyTasks.md are always excluded.
func ListArtifacts(workspaceRoot, pattern string) ([]ArtifactEntry, error) {
	absRoot, err := filepath.EvalSymlinks(workspaceRoot)
	if err != nil {
		absRoot, err = filepath.Abs(workspaceRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve workspace root: %w", err)
		}
	}

	var entries []ArtifactEntry

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, _ := filepath.Rel(absRoot, path)
		if rel == "." {
			return nil
		}

		if info.IsDir() {
			if excludedDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if isTaskManagerOwned(rel) {
			return nil
		}

		// Apply glob filter when one was provided.
		if pattern != "" {
			matched, matchErr := matchArtifactPattern(pattern, rel)
			if matchErr != nil {
				return matchErr
			}
			if !matched {
				return nil
			}
		}

		entries = append(entries, ArtifactEntry{
			Path: filepath.ToSlash(rel),
			Size: info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace: %w", err)
	}

	return entries, nil
}

// matchArtifactPattern applies a glob pattern to a relative path.  When the
// pattern contains no path separator it is matched against the base filename
// only, so "*.html" will match files in any subdirectory.  Patterns that
// contain a separator (e.g. "src/*.go") are matched against the full relative
// path.
func matchArtifactPattern(pattern, relPath string) (bool, error) {
	if strings.ContainsRune(pattern, '/') || strings.ContainsRune(pattern, filepath.Separator) {
		return filepath.Match(pattern, relPath)
	}
	return filepath.Match(pattern, filepath.Base(relPath))
}
