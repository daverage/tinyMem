package embeddings

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

// Cache provides persistent caching for entity embeddings
// Per HYBRID_RETRIEVAL_DESIGN.md: Cache embeddings with artifact hash
type Cache struct {
	db         *sql.DB
	memory     map[string]*CacheEntry // In-memory cache for performance
	mu         sync.RWMutex
	model      string   // Embedding model name
	embedder   Embedder // Embedding provider
	ttl        time.Duration
	enabled    bool
}

// CacheEntry represents a cached embedding
type CacheEntry struct {
	EntityKey    string
	ArtifactHash string
	Embedding    []float32
	Model        string
	Timestamp    time.Time
}

// NewCache creates a new embedding cache
func NewCache(db *sql.DB, embedder Embedder, model string, ttl time.Duration) *Cache {
	return &Cache{
		db:       db,
		memory:   make(map[string]*CacheEntry),
		model:    model,
		embedder: embedder,
		ttl:      ttl,
		enabled:  ttl > 0 && embedder != nil,
	}
}

// GetOrCompute retrieves embedding from cache or computes it
func (c *Cache) GetOrCompute(entityKey, artifactHash, content string) ([]float32, error) {
	if !c.enabled {
		// Cache disabled - compute directly
		return c.embedder.Embed(content)
	}

	// Check in-memory cache
	c.mu.RLock()
	if entry, exists := c.memory[entityKey]; exists {
		// Verify artifact hash matches (invalidate if changed)
		if entry.ArtifactHash == artifactHash {
			// Check TTL
			if time.Since(entry.Timestamp) < c.ttl {
				c.mu.RUnlock()
				return entry.Embedding, nil
			}
		}
	}
	c.mu.RUnlock()

	// Check database cache
	dbEmbedding, err := c.getFromDB(entityKey, artifactHash)
	if err == nil && dbEmbedding != nil {
		// Store in memory cache
		c.mu.Lock()
		c.memory[entityKey] = &CacheEntry{
			EntityKey:    entityKey,
			ArtifactHash: artifactHash,
			Embedding:    dbEmbedding,
			Model:        c.model,
			Timestamp:    time.Now(),
		}
		c.mu.Unlock()
		return dbEmbedding, nil
	}

	// Cache miss - compute embedding
	embedding, err := c.embedder.Embed(content)
	if err != nil {
		return nil, fmt.Errorf("failed to embed content: %w", err)
	}

	// Store in database
	if err := c.storeInDB(entityKey, artifactHash, embedding); err != nil {
		// Log error but don't fail - cache is best-effort
		// Would need logger here
	}

	// Store in memory cache
	c.mu.Lock()
	c.memory[entityKey] = &CacheEntry{
		EntityKey:    entityKey,
		ArtifactHash: artifactHash,
		Embedding:    embedding,
		Model:        c.model,
		Timestamp:    time.Now(),
	}
	c.mu.Unlock()

	return embedding, nil
}

// getFromDB retrieves embedding from database
func (c *Cache) getFromDB(entityKey, artifactHash string) ([]float32, error) {
	var embeddingBlob []byte
	var storedHash string
	var storedModel string

	err := c.db.QueryRow(`
		SELECT artifact_hash, embedding, embedding_model
		FROM entity_embeddings
		WHERE entity_key = ?
	`, entityKey).Scan(&storedHash, &embeddingBlob, &storedModel)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("embedding not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query embedding: %w", err)
	}

	// Verify hash and model match
	if storedHash != artifactHash {
		return nil, fmt.Errorf("artifact hash mismatch (cache stale)")
	}
	if storedModel != c.model {
		return nil, fmt.Errorf("embedding model mismatch")
	}

	// Deserialize embedding
	embedding, err := deserializeEmbedding(embeddingBlob)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize embedding: %w", err)
	}

	return embedding, nil
}

// storeInDB stores embedding in database
func (c *Cache) storeInDB(entityKey, artifactHash string, embedding []float32) error {
	// Serialize embedding
	embeddingBlob, err := serializeEmbedding(embedding)
	if err != nil {
		return fmt.Errorf("failed to serialize embedding: %w", err)
	}

	// Insert or replace
	_, err = c.db.Exec(`
		INSERT OR REPLACE INTO entity_embeddings (entity_key, artifact_hash, embedding, embedding_model, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, entityKey, artifactHash, embeddingBlob, c.model, time.Now().Unix())

	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	return nil
}

// Clear removes all cached embeddings for an entity
func (c *Cache) Clear(entityKey string) error {
	// Remove from memory
	c.mu.Lock()
	delete(c.memory, entityKey)
	c.mu.Unlock()

	// Remove from database
	_, err := c.db.Exec("DELETE FROM entity_embeddings WHERE entity_key = ?", entityKey)
	return err
}

// ClearAll removes all cached embeddings
func (c *Cache) ClearAll() error {
	// Clear memory
	c.mu.Lock()
	c.memory = make(map[string]*CacheEntry)
	c.mu.Unlock()

	// Clear database
	_, err := c.db.Exec("DELETE FROM entity_embeddings")
	return err
}

// GetStats returns cache statistics
func (c *Cache) GetStats() (int, int, error) {
	c.mu.RLock()
	memoryCount := len(c.memory)
	c.mu.RUnlock()

	var dbCount int
	err := c.db.QueryRow("SELECT COUNT(*) FROM entity_embeddings").Scan(&dbCount)
	if err != nil {
		return memoryCount, 0, err
	}

	return memoryCount, dbCount, nil
}

// serializeEmbedding converts float32 slice to bytes
func serializeEmbedding(embedding []float32) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write dimension count (4 bytes)
	dim := uint32(len(embedding))
	if err := binary.Write(buf, binary.LittleEndian, dim); err != nil {
		return nil, err
	}

	// Write embedding values (4 bytes per float32)
	for _, val := range embedding {
		if err := binary.Write(buf, binary.LittleEndian, val); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// deserializeEmbedding converts bytes back to float32 slice
func deserializeEmbedding(data []byte) ([]float32, error) {
	buf := bytes.NewReader(data)

	// Read dimension count
	var dim uint32
	if err := binary.Read(buf, binary.LittleEndian, &dim); err != nil {
		return nil, err
	}

	// Read embedding values
	embedding := make([]float32, dim)
	for i := uint32(0); i < dim; i++ {
		if err := binary.Read(buf, binary.LittleEndian, &embedding[i]); err != nil {
			return nil, err
		}
	}

	return embedding, nil
}
