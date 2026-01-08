package state

import (
	"sync"
	"time"
)

// ========================================================================
// ETV Cache - Performance Optimization for Consistency Checks
// ========================================================================
// Caches file hash and staleness results to avoid repeated file I/O
// Uses short TTL (default 5 seconds) to balance freshness vs performance
//
// Thread-safe for concurrent access
// ========================================================================

// ETVCache caches ETV (External Truth Verification) check results
// Reduces repeated file reads and hash computations
type ETVCache struct {
	mu      sync.RWMutex
	cache   map[string]*cacheEntry
	ttl     time.Duration
	enabled bool
}

// cacheEntry represents a cached ETV check result
type cacheEntry struct {
	isStale   bool
	diskHash  string
	exists    bool
	timestamp time.Time
}

// NewETVCache creates a new ETV cache with the specified TTL
// ttl: how long to cache results (recommend 5 seconds)
// Pass ttl=0 to disable caching
func NewETVCache(ttl time.Duration) *ETVCache {
	return &ETVCache{
		cache:   make(map[string]*cacheEntry),
		ttl:     ttl,
		enabled: ttl > 0,
	}
}

// Get retrieves a cached result for the given filepath
// Returns nil if not cached or expired
func (c *ETVCache) Get(filepath string) *cacheEntry {
	if !c.enabled {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[filepath]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Since(entry.timestamp) > c.ttl {
		return nil
	}

	return entry
}

// Set stores a cache entry for the given filepath
func (c *ETVCache) Set(filepath string, isStale bool, diskHash string, fileExists bool) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[filepath] = &cacheEntry{
		isStale:   isStale,
		diskHash:  diskHash,
		exists:    fileExists,
		timestamp: time.Now(),
	}
}

// Clear removes all cached entries
func (c *ETVCache) Clear() {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cacheEntry)
}

// CleanExpired removes expired entries from the cache
// Call this periodically to prevent unbounded memory growth
func (c *ETVCache) CleanExpired() int {
	if !c.enabled {
		return 0
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	for filepath, entry := range c.cache {
		if now.Sub(entry.timestamp) > c.ttl {
			delete(c.cache, filepath)
			removed++
		}
	}

	return removed
}

// Size returns the number of cached entries
func (c *ETVCache) Size() int {
	if !c.enabled {
		return 0
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// IsEnabled returns whether caching is enabled
func (c *ETVCache) IsEnabled() bool {
	return c.enabled
}
