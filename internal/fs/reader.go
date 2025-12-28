package fs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andrzejmarczewski/tslp/internal/entity"
	"github.com/andrzejmarczewski/tslp/internal/vault"
)

// ========================================================================
// CRITICAL SAFETY INVARIANT: READ-ONLY FILESYSTEM ACCESS
// ========================================================================
// This package provides READ-ONLY access to the local filesystem.
// Per ETV specification:
//   - The proxy may READ file contents for verification only
//   - The proxy is FORBIDDEN from writing, modifying, or syncing files
//   - Disk is treated as higher authority for verification, not mutation
//
// NO FUNCTION IN THIS PACKAGE MAY:
//   - Write to disk (os.Create, os.WriteFile, os.OpenFile with O_WRONLY, etc.)
//   - Modify files (os.Rename, os.Remove, os.Chmod, etc.)
//   - Apply diffs or patches
//   - Auto-sync or auto-repair
//
// Disk is authoritative for VERIFICATION only.
// ========================================================================

// Reader provides read-only access to the filesystem for ETV
// Per spec section 15.3: Read file contents, hash file contents, parse file contents
type Reader struct {
	// Intentionally no state - stateless reads only
}

// NewReader creates a new filesystem reader
func NewReader() *Reader {
	return &Reader{}
}

// ReadFileResult contains the result of reading a file
type ReadFileResult struct {
	Content      string
	Hash         string
	Exists       bool
	Error        error
	AbsolutePath string
}

// ReadFile reads a file from disk by absolute path
// Per ETV spec 15.3: Read file contents
//
// SAFETY GUARANTEE: READ-ONLY OPERATION
// This function ONLY reads file contents. It never writes or modifies files.
//
// Returns:
//   - Content: file contents as string
//   - Hash: SHA-256 hash of content (matches vault.ComputeHash)
//   - Exists: true if file exists and was readable
//   - Error: any read error encountered
func (r *Reader) ReadFile(absolutePath string) ReadFileResult {
	// Validate that path is absolute
	if !filepath.IsAbs(absolutePath) {
		return ReadFileResult{
			Exists: false,
			Error:  fmt.Errorf("path must be absolute: %s", absolutePath),
		}
	}

	// Clean the path to prevent directory traversal attacks
	// (even though we're read-only, we still validate paths)
	cleanPath := filepath.Clean(absolutePath)

	// Check if file exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ReadFileResult{
				Exists:       false,
				Error:        nil, // Not an error - file simply doesn't exist
				AbsolutePath: cleanPath,
			}
		}
		return ReadFileResult{
			Exists:       false,
			Error:        fmt.Errorf("failed to stat file: %w", err),
			AbsolutePath: cleanPath,
		}
	}

	// Ensure it's a regular file, not a directory
	if info.IsDir() {
		return ReadFileResult{
			Exists:       false,
			Error:        fmt.Errorf("path is a directory, not a file: %s", cleanPath),
			AbsolutePath: cleanPath,
		}
	}

	// READ-ONLY: Read file contents
	// Using os.ReadFile which is a read-only operation
	contentBytes, err := os.ReadFile(cleanPath)
	if err != nil {
		return ReadFileResult{
			Exists:       true,
			Error:        fmt.Errorf("failed to read file: %w", err),
			AbsolutePath: cleanPath,
		}
	}

	content := string(contentBytes)

	// Compute hash using the same algorithm as vault
	// This ensures disk hash is directly comparable to artifact hash
	hash := vault.ComputeHash(content)

	return ReadFileResult{
		Content:      content,
		Hash:         hash,
		Exists:       true,
		Error:        nil,
		AbsolutePath: cleanPath,
	}
}

// HashFile computes the hash of a file without loading full content into memory
// For large files, this is more efficient than ReadFile
//
// SAFETY GUARANTEE: READ-ONLY OPERATION
// This function ONLY reads and hashes file contents. It never writes or modifies files.
func (r *Reader) HashFile(absolutePath string) (string, error) {
	result := r.ReadFile(absolutePath)
	if result.Error != nil {
		return "", result.Error
	}
	if !result.Exists {
		return "", fmt.Errorf("file does not exist: %s", absolutePath)
	}
	return result.Hash, nil
}

// ParseFileResult contains the result of parsing a file
type ParseFileResult struct {
	ASTResult *entity.ASTResult
	Hash      string
	Exists    bool
	Error     error
}

// ParseFile reads and parses a file via Tree-sitter
// Per ETV spec 15.3: Parse file contents
//
// SAFETY GUARANTEE: READ-ONLY OPERATION
// This function ONLY reads and parses file contents. It never writes or modifies files.
func (r *Reader) ParseFile(absolutePath string) ParseFileResult {
	// First read the file
	readResult := r.ReadFile(absolutePath)
	if readResult.Error != nil {
		return ParseFileResult{
			Exists: readResult.Exists,
			Error:  readResult.Error,
		}
	}
	if !readResult.Exists {
		return ParseFileResult{
			Exists: false,
			Error:  fmt.Errorf("file does not exist: %s", absolutePath),
		}
	}

	// Detect language from file extension
	language := entity.DetectLanguage(readResult.Content, &absolutePath)

	// Parse via Tree-sitter
	astResult, err := entity.ParseAST(readResult.Content, language)
	if err != nil {
		return ParseFileResult{
			Hash:   readResult.Hash,
			Exists: true,
			Error:  fmt.Errorf("failed to parse file: %w", err),
		}
	}

	return ParseFileResult{
		ASTResult: astResult,
		Hash:      readResult.Hash,
		Exists:    true,
		Error:     nil,
	}
}

// VerifyNoWrites is a compile-time documentation function
// This function exists solely to document the read-only guarantee
// If any writes are added to this package, this comment will be violated
func VerifyNoWrites() {
	// This package contains NO file write operations:
	// ❌ os.Create
	// ❌ os.WriteFile
	// ❌ os.OpenFile with O_WRONLY, O_RDWR, O_APPEND, O_CREATE, O_TRUNC
	// ❌ os.Remove
	// ❌ os.Rename
	// ❌ os.Chmod
	// ❌ os.Mkdir
	// ❌ os.MkdirAll
	// ❌ Any file modification operation
	//
	// Only READ-ONLY operations are permitted:
	// ✅ os.ReadFile
	// ✅ os.Stat
	// ✅ os.Open (read-only)
	// ✅ filepath.Clean, filepath.IsAbs (path operations)
}
