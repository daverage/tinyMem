-- TSLP v5.3 Initial Schema
-- Migration: 001
-- Description: Create vault, ledger, and state_map tables
-- Idempotent: Yes (uses IF NOT EXISTS)

-- ============================================================================
-- VAULT: Immutable Content Store (Content-Addressed Storage)
-- ============================================================================

CREATE TABLE IF NOT EXISTS vault_artifacts (
    hash TEXT PRIMARY KEY,           -- SHA-256 of content
    content TEXT NOT NULL,            -- Full artifact content (code, diff, etc)
    content_type TEXT NOT NULL,       -- 'code', 'diff', 'decision', 'user_paste'
    created_at INTEGER NOT NULL,      -- Unix timestamp
    byte_size INTEGER NOT NULL,       -- Size in bytes
    token_count INTEGER               -- Optional: token count estimate
);

CREATE INDEX IF NOT EXISTS idx_vault_created ON vault_artifacts(created_at);
CREATE INDEX IF NOT EXISTS idx_vault_type ON vault_artifacts(content_type);

-- ============================================================================
-- LEDGER: Chronological Evidence (Append-Only)
-- ============================================================================

CREATE TABLE IF NOT EXISTS ledger_episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id TEXT NOT NULL UNIQUE,  -- UUID for this episode
    timestamp INTEGER NOT NULL,        -- Unix timestamp
    user_prompt_hash TEXT,             -- Reference to vault_artifacts
    assistant_response_hash TEXT,      -- Reference to vault_artifacts
    metadata TEXT,                     -- JSON blob for extensions
    FOREIGN KEY(user_prompt_hash) REFERENCES vault_artifacts(hash),
    FOREIGN KEY(assistant_response_hash) REFERENCES vault_artifacts(hash)
);

CREATE INDEX IF NOT EXISTS idx_ledger_timestamp ON ledger_episodes(timestamp);
CREATE INDEX IF NOT EXISTS idx_ledger_episode ON ledger_episodes(episode_id);

-- State transition records
CREATE TABLE IF NOT EXISTS ledger_state_transitions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id TEXT NOT NULL,
    entity_key TEXT NOT NULL,          -- filepath::symbol
    from_state TEXT,                   -- NULL for new entities
    to_state TEXT NOT NULL,            -- PROPOSED, AUTHORITATIVE, SUPERSEDED, TOMBSTONED
    artifact_hash TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    reason TEXT,                       -- Why this transition occurred
    FOREIGN KEY(episode_id) REFERENCES ledger_episodes(episode_id),
    FOREIGN KEY(artifact_hash) REFERENCES vault_artifacts(hash)
);

CREATE INDEX IF NOT EXISTS idx_transitions_entity ON ledger_state_transitions(entity_key);
CREATE INDEX IF NOT EXISTS idx_transitions_episode ON ledger_state_transitions(episode_id);

-- Shadow audit results
CREATE TABLE IF NOT EXISTS ledger_audit_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id TEXT NOT NULL,
    artifact_hash TEXT NOT NULL,
    entity_key TEXT,
    status TEXT NOT NULL,              -- 'completed', 'partial', 'discussion'
    audit_response TEXT,               -- Full JSON response from audit
    timestamp INTEGER NOT NULL,
    FOREIGN KEY(episode_id) REFERENCES ledger_episodes(episode_id),
    FOREIGN KEY(artifact_hash) REFERENCES vault_artifacts(hash)
);

CREATE INDEX IF NOT EXISTS idx_audit_episode ON ledger_audit_results(episode_id);

-- ============================================================================
-- STATE MAP: Single Source of Truth
-- ============================================================================

CREATE TABLE IF NOT EXISTS state_map (
    entity_key TEXT PRIMARY KEY,       -- filepath::symbol (unique entity identifier)
    filepath TEXT NOT NULL,            -- File path component
    symbol TEXT NOT NULL,              -- Symbol name component
    artifact_hash TEXT NOT NULL,       -- Current authoritative artifact
    confidence TEXT NOT NULL,          -- CONFIRMED, INFERRED, UNRESOLVED
    state TEXT NOT NULL,               -- PROPOSED, AUTHORITATIVE, SUPERSEDED, TOMBSTONED
    last_updated INTEGER NOT NULL,     -- Unix timestamp
    metadata TEXT,                     -- JSON: AST node count, token count, etc.
    FOREIGN KEY(artifact_hash) REFERENCES vault_artifacts(hash)
);

CREATE INDEX IF NOT EXISTS idx_state_filepath ON state_map(filepath);
CREATE INDEX IF NOT EXISTS idx_state_symbol ON state_map(symbol);
CREATE INDEX IF NOT EXISTS idx_state_state ON state_map(state);
CREATE INDEX IF NOT EXISTS idx_state_updated ON state_map(last_updated);

-- ============================================================================
-- ENTITY RESOLUTION TRACKING
-- ============================================================================

CREATE TABLE IF NOT EXISTS entity_resolution_cache (
    artifact_hash TEXT PRIMARY KEY,
    entity_key TEXT,                   -- NULL if UNRESOLVED
    confidence TEXT NOT NULL,          -- CONFIRMED, INFERRED, UNRESOLVED
    resolution_method TEXT NOT NULL,   -- 'ast', 'regex', 'correlation', 'unresolved'
    filepath TEXT,
    symbols TEXT,                      -- JSON array of detected symbols
    ast_node_count INTEGER,
    created_at INTEGER NOT NULL,
    FOREIGN KEY(artifact_hash) REFERENCES vault_artifacts(hash)
);

CREATE INDEX IF NOT EXISTS idx_resolution_entity ON entity_resolution_cache(entity_key);
CREATE INDEX IF NOT EXISTS idx_resolution_method ON entity_resolution_cache(resolution_method);

-- ============================================================================
-- TOMBSTONE TRACKING (for recovery)
-- ============================================================================

CREATE TABLE IF NOT EXISTS tombstones (
    entity_key TEXT NOT NULL,
    artifact_hash TEXT NOT NULL,       -- Last known good artifact
    tombstoned_at INTEGER NOT NULL,    -- When it was tombstoned
    episode_id TEXT NOT NULL,          -- Episode that caused tombstoning
    episodes_retained INTEGER DEFAULT 0, -- Counter for cleanup
    PRIMARY KEY(entity_key, tombstoned_at),
    FOREIGN KEY(artifact_hash) REFERENCES vault_artifacts(hash),
    FOREIGN KEY(episode_id) REFERENCES ledger_episodes(episode_id)
);

CREATE INDEX IF NOT EXISTS idx_tombstones_entity ON tombstones(entity_key);
CREATE INDEX IF NOT EXISTS idx_tombstones_episode ON tombstones(episode_id);
