-- Migration: Add entity embeddings table for semantic ranking
-- Per HYBRID_RETRIEVAL_DESIGN.md: Cache embeddings with artifact hash for invalidation

CREATE TABLE IF NOT EXISTS entity_embeddings (
    entity_key TEXT PRIMARY KEY,
    artifact_hash TEXT NOT NULL,
    embedding BLOB NOT NULL,              -- Serialized float32 vector
    embedding_model TEXT NOT NULL,        -- e.g., "text-embedding-3-small", "simple-384"
    created_at INTEGER NOT NULL,
    FOREIGN KEY(entity_key) REFERENCES state_map(entity_key) ON DELETE CASCADE,
    FOREIGN KEY(artifact_hash) REFERENCES vault_artifacts(hash)
);

-- Index for quick lookup by artifact hash (for cache invalidation)
CREATE INDEX IF NOT EXISTS idx_embeddings_hash ON entity_embeddings(artifact_hash);

-- Index for filtering by model (in case model changes)
CREATE INDEX IF NOT EXISTS idx_embeddings_model ON entity_embeddings(embedding_model);
