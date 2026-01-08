package state

import (
	"fmt"
	"time"

	"github.com/andrzejmarczewski/tinyMem/internal/fs"
	"github.com/andrzejmarczewski/tinyMem/internal/vault"
)

// ========================================================================
// ETV (External Truth Verification) - Disk vs State Map Consistency
// ========================================================================
// Per spec section 15: External Truth Verification
//
// Authority Model (for verification only):
//   1. User-pasted code (highest)
//   2. Local filesystem - read-only (higher)
//   3. State Map (lower)
//   4. LLM output (lowest)
//
// CRITICAL: Disk is authoritative for VERIFICATION, never for MUTATION.
//           The proxy reads disk to detect divergence, but NEVER writes.
// ========================================================================

// ConsistencyChecker verifies that State Map entities match disk state
// Per spec section 15.4: STALE detection
// Includes optional caching to improve performance
type ConsistencyChecker struct {
	fsReader *fs.Reader
	vault    *vault.Vault
	cache    *ETVCache
}

// NewConsistencyChecker creates a new consistency checker
// Cache is enabled by default with 5-second TTL for performance
func NewConsistencyChecker(fsReader *fs.Reader, vaultInstance *vault.Vault) *ConsistencyChecker {
	return &ConsistencyChecker{
		fsReader: fsReader,
		vault:    vaultInstance,
		cache:    NewETVCache(5 * time.Second), // 5-second cache for performance
	}
}

// NewConsistencyCheckerWithCache creates a checker with custom cache settings
func NewConsistencyCheckerWithCache(fsReader *fs.Reader, vaultInstance *vault.Vault, cache *ETVCache) *ConsistencyChecker {
	return &ConsistencyChecker{
		fsReader: fsReader,
		vault:    vaultInstance,
		cache:    cache,
	}
}

// ConsistencyResult represents the result of a consistency check
type ConsistencyResult struct {
	EntityKey        string
	IsStale          bool
	DiskHash         string
	StateMapHash     string
	FileExists       bool
	FileReadError    error
	ComparisonError  error
	HasFilepath      bool
	SkipReason       string // Why check was skipped (e.g., "no filepath", "not CONFIRMED")
}

// CheckEntity verifies if a single entity is consistent with disk state
// Per spec section 15.4: An entity is STALE when disk hash ≠ authoritative artifact hash
//
// STALE is a DERIVED runtime condition, not a stored artifact state.
// This function NEVER modifies the State Map or any stored state.
//
// Uses cache to avoid repeated file I/O for performance
//
// Returns:
//   - IsStale=true if disk content differs from authoritative artifact
//   - IsStale=false if disk matches, or if entity has no filepath
//   - Errors are returned in the result struct, not as function errors
func (c *ConsistencyChecker) CheckEntity(entity *EntityState) ConsistencyResult {
	result := ConsistencyResult{
		EntityKey:    entity.EntityKey,
		StateMapHash: entity.ArtifactHash,
		IsStale:      false,
	}

	// Only check CONFIRMED entities
	// Per spec section 15: ETV applies to entities with provable file mappings
	if entity.Confidence != "CONFIRMED" {
		result.SkipReason = fmt.Sprintf("entity confidence is %s, not CONFIRMED", entity.Confidence)
		return result
	}

	// Only check entities with a filepath
	// Entities without filepaths cannot be verified against disk
	if entity.Filepath == "" {
		result.SkipReason = "entity has no filepath"
		result.HasFilepath = false
		return result
	}

	result.HasFilepath = true

	// Check cache first for performance
	if c.cache != nil && c.cache.IsEnabled() {
		if cached := c.cache.Get(entity.Filepath); cached != nil {
			result.DiskHash = cached.diskHash
			result.FileExists = cached.exists

			// If file doesn't exist, it's stale
			if !cached.exists {
				result.IsStale = true
				return result
			}

			// Compare cached hash with state map hash
			if cached.diskHash != entity.ArtifactHash {
				result.IsStale = true
			}

			return result
		}
	}

	// Cache miss - READ-ONLY: Read file from disk and compute hash
	// Per spec section 15.3: The proxy may read file contents for verification
	diskResult := c.fsReader.ReadFile(entity.Filepath)

	if diskResult.Error != nil {
		// File read error (permissions, I/O error, etc.)
		result.FileReadError = diskResult.Error
		result.FileExists = diskResult.Exists
		// Cannot determine staleness if we can't read the file
		// Treat as non-stale but record the error
		return result
	}

	result.FileExists = diskResult.Exists

	if !diskResult.Exists {
		// File doesn't exist on disk
		// Per spec: This is a form of divergence
		// The State Map claims to have authoritative state for a file that doesn't exist
		// Mark as STALE to prevent hydration of non-existent files

		// Update cache
		if c.cache != nil && c.cache.IsEnabled() {
			c.cache.Set(entity.Filepath, true, "", false)
		}

		result.IsStale = true
		result.DiskHash = "" // No hash for non-existent file
		return result
	}

	result.DiskHash = diskResult.Hash

	// Compare disk hash with State Map hash
	// Per spec section 15.4: STALE when disk hash ≠ authoritative artifact hash
	isStale := diskResult.Hash != entity.ArtifactHash

	// Update cache for next time
	if c.cache != nil && c.cache.IsEnabled() {
		c.cache.Set(entity.Filepath, isStale, diskResult.Hash, true)
	}

	if isStale {
		// DIVERGENCE DETECTED
		// Disk content differs from what the State Map believes is authoritative
		// This means the user (or another process) has edited the file manually
		result.IsStale = true
		return result
	}

	// Hashes match - entity is consistent with disk
	result.IsStale = false
	return result
}

// CheckAll verifies consistency for all entities in the State Map
// Returns a map of entity keys to their consistency results
//
// This function NEVER modifies the State Map or any stored state.
// STALE is a derived condition computed on-demand.
func (c *ConsistencyChecker) CheckAll(entities []*EntityState) map[string]ConsistencyResult {
	results := make(map[string]ConsistencyResult)

	for _, entity := range entities {
		result := c.CheckEntity(entity)
		results[entity.EntityKey] = result
	}

	return results
}

// IsEntityStale is a convenience function to check if a single entity is STALE
// Returns:
//   - isStale: true if entity diverges from disk
//   - exists: true if file exists on disk
//   - err: any error encountered during check
func (c *ConsistencyChecker) IsEntityStale(entity *EntityState) (isStale bool, exists bool, err error) {
	result := c.CheckEntity(entity)

	if result.FileReadError != nil {
		return false, result.FileExists, result.FileReadError
	}

	return result.IsStale, result.FileExists, nil
}

// GetStaleEntities filters a list of entities to return only STALE ones
// This is useful for diagnostics and hydration filtering
func (c *ConsistencyChecker) GetStaleEntities(entities []*EntityState) []*EntityState {
	var staleEntities []*EntityState

	for _, entity := range entities {
		result := c.CheckEntity(entity)
		if result.IsStale {
			staleEntities = append(staleEntities, entity)
		}
	}

	return staleEntities
}

// CountStale counts how many entities in the State Map are STALE
// Used for diagnostics (GET /doctor)
func (c *ConsistencyChecker) CountStale(entities []*EntityState) int {
	count := 0

	for _, entity := range entities {
		result := c.CheckEntity(entity)
		if result.IsStale {
			count++
		}
	}

	return count
}

// GetFileReadErrors collects all file read errors encountered during consistency checks
// Used for diagnostics (GET /doctor)
func (c *ConsistencyChecker) GetFileReadErrors(entities []*EntityState) []string {
	var errors []string

	for _, entity := range entities {
		result := c.CheckEntity(entity)
		if result.FileReadError != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", entity.EntityKey, result.FileReadError))
		}
	}

	return errors
}

// VerifyNoMutation is a compile-time documentation function
// This function exists solely to document the no-mutation guarantee
func VerifyNoMutation() {
	// This file contains NO State Map mutations:
	// ❌ No Manager.Set() calls
	// ❌ No database writes
	// ❌ No artifact creation or modification
	// ❌ No ledger writes
	// ❌ No promotion logic
	// ❌ No state advancement
	//
	// Only READ operations and DERIVED computations:
	// ✅ Read entities from State Map
	// ✅ Read files from disk (via fs.Reader, which is also read-only)
	// ✅ Compare hashes
	// ✅ Return derived STALE status
	//
	// Per spec section 15.4: STALE is a derived runtime condition, not a stored state.
}
