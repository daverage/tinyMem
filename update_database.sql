-- TinyMem Database Update Script (Version-aware)
-- This script will update an existing TinyMem database to the latest schema version (v6)
-- It handles both fresh installations and incremental updates from any previous version

-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- Create memories table if it doesn't exist (for fresh installs)
CREATE TABLE IF NOT EXISTS memories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('fact', 'claim', 'plan', 'decision', 'constraint', 'observation', 'note')),
    summary TEXT NOT NULL,
    detail TEXT,
    key TEXT,
    source TEXT,
    recall_tier TEXT NOT NULL DEFAULT 'opportunistic',
    truth_state TEXT NOT NULL DEFAULT 'tentative',
    classification TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    superseded_by INTEGER REFERENCES memories(id),
    UNIQUE(key, project_id) ON CONFLICT REPLACE
);

-- Create evidence table if it doesn't exist (for fresh installs)
CREATE TABLE IF NOT EXISTS evidence (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    memory_id INTEGER NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    content TEXT NOT NULL,
    verified BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create embeddings table if it doesn't exist (for fresh installs)
CREATE TABLE IF NOT EXISTS embeddings (
    memory_id INTEGER PRIMARY KEY REFERENCES memories(id) ON DELETE CASCADE,
    embedding_data TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create recall_metrics table if it doesn't exist (for fresh installs)
CREATE TABLE IF NOT EXISTS recall_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    query TEXT NOT NULL DEFAULT '',
    query_type TEXT NOT NULL CHECK(query_type IN ('empty', 'search')),
    memory_ids TEXT,
    memory_count INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    tier_breakdown TEXT,
    duration_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Migration v2: Add fact validation triggers
-- These are idempotent and will only be created if they don't exist
CREATE TRIGGER IF NOT EXISTS memories_fact_insert_guard
BEFORE INSERT ON memories
WHEN NEW.type = 'fact'
BEGIN
    SELECT RAISE(ABORT, 'fact requires verified evidence');
END;

CREATE TRIGGER IF NOT EXISTS memories_fact_update_guard
BEFORE UPDATE ON memories
WHEN NEW.type = 'fact'
BEGIN
    SELECT CASE
        WHEN (SELECT COUNT(*) FROM evidence e WHERE e.memory_id = NEW.id AND e.verified = 1) = 0
        THEN RAISE(ABORT, 'fact requires verified evidence')
    END;
END;

CREATE TRIGGER IF NOT EXISTS evidence_fact_delete_guard
BEFORE DELETE ON evidence
WHEN (SELECT type FROM memories WHERE id = OLD.memory_id) = 'fact'
  AND (SELECT COUNT(*) FROM evidence e WHERE e.memory_id = OLD.memory_id AND e.verified = 1) <= 1
BEGIN
    SELECT RAISE(ABORT, 'cannot remove last verified evidence for fact');
END;

CREATE TRIGGER IF NOT EXISTS evidence_fact_unverify_guard
BEFORE UPDATE OF verified ON evidence
WHEN (SELECT type FROM memories WHERE id = NEW.memory_id) = 'fact'
  AND OLD.verified = 1 AND NEW.verified = 0
  AND (SELECT COUNT(*) FROM evidence e WHERE e.memory_id = NEW.memory_id AND e.verified = 1) <= 1
BEGIN
    SELECT RAISE(ABORT, 'fact requires at least one verified evidence');
END;

-- Migration v3: Add recall_tier column with appropriate defaults
-- Check if recall_tier column exists
CREATE TEMPORARY TABLE temp_has_recall_tier AS
SELECT COUNT(*) AS has_col
FROM pragma_table_info('memories')
WHERE name = 'recall_tier';

-- Create a new table with recall_tier column if it doesn't exist
CREATE TABLE IF NOT EXISTS memories_with_recall_tier (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('fact', 'claim', 'plan', 'decision', 'constraint', 'observation', 'note')),
    summary TEXT NOT NULL,
    detail TEXT,
    key TEXT,
    source TEXT,
    recall_tier TEXT NOT NULL DEFAULT 'opportunistic',
    truth_state TEXT NOT NULL DEFAULT 'tentative',
    classification TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    superseded_by INTEGER REFERENCES memories(id),
    UNIQUE(key, project_id) ON CONFLICT REPLACE
);

-- Populate the new table with recall_tier if the original doesn't have it
INSERT INTO memories_with_recall_tier (id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by)
SELECT
    m.id,
    m.project_id,
    m.type,
    m.summary,
    m.detail,
    m.key,
    m.source,
    CASE
        WHEN m.type IN ('fact', 'constraint') THEN 'always'
        WHEN m.type IN ('decision', 'claim') THEN 'contextual'
        ELSE 'opportunistic'
    END,
    CASE
        WHEN (SELECT COUNT(*) FROM pragma_table_info('memories') WHERE name = 'truth_state') = 0
        THEN
            CASE
                WHEN m.type = 'fact' THEN 'verified'
                WHEN m.type IN ('decision', 'constraint') THEN 'asserted'
                ELSE 'tentative'
            END
        ELSE m.truth_state
    END,
    CASE
        WHEN (SELECT COUNT(*) FROM pragma_table_info('memories') WHERE name = 'classification') = 0
        THEN NULL
        ELSE m.classification
    END,
    m.created_at,
    m.updated_at,
    m.superseded_by
FROM memories m
WHERE (SELECT has_col FROM temp_has_recall_tier) = 0;

-- Insert records that already had recall_tier
INSERT OR IGNORE INTO memories_with_recall_tier (id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by)
SELECT
    m.id,
    m.project_id,
    m.type,
    m.summary,
    m.detail,
    m.key,
    m.source,
    m.recall_tier,
    CASE
        WHEN (SELECT COUNT(*) FROM pragma_table_info('memories') WHERE name = 'truth_state') = 0
        THEN
            CASE
                WHEN m.type = 'fact' THEN 'verified'
                WHEN m.type IN ('decision', 'constraint') THEN 'asserted'
                ELSE 'tentative'
            END
        ELSE m.truth_state
    END,
    CASE
        WHEN (SELECT COUNT(*) FROM pragma_table_info('memories') WHERE name = 'classification') = 0
        THEN NULL
        ELSE m.classification
    END,
    m.created_at,
    m.updated_at,
    m.superseded_by
FROM memories m;

-- Replace the old table with the new one if we added recall_tier
DELETE FROM memories WHERE (SELECT has_col FROM temp_has_recall_tier) = 0;
INSERT OR REPLACE INTO memories (id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by)
SELECT id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by
FROM memories_with_recall_tier;

DROP TABLE memories_with_recall_tier;

-- Migration v4: Add truth_state column with appropriate defaults
-- Check if truth_state column exists
CREATE TEMPORARY TABLE temp_has_truth_state AS
SELECT COUNT(*) AS has_col
FROM pragma_table_info('memories')
WHERE name = 'truth_state';

-- Create a new table with truth_state column if it doesn't exist
CREATE TABLE IF NOT EXISTS memories_with_truth_state (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('fact', 'claim', 'plan', 'decision', 'constraint', 'observation', 'note')),
    summary TEXT NOT NULL,
    detail TEXT,
    key TEXT,
    source TEXT,
    recall_tier TEXT NOT NULL DEFAULT 'opportunistic',
    truth_state TEXT NOT NULL DEFAULT 'tentative',
    classification TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    superseded_by INTEGER REFERENCES memories(id),
    UNIQUE(key, project_id) ON CONFLICT REPLACE
);

-- Populate the new table with truth_state if the original doesn't have it
INSERT INTO memories_with_truth_state (id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by)
SELECT
    m.id,
    m.project_id,
    m.type,
    m.summary,
    m.detail,
    m.key,
    m.source,
    m.recall_tier,
    CASE
        WHEN m.type = 'fact' THEN 'verified'
        WHEN m.type IN ('decision', 'constraint') THEN 'asserted'
        ELSE 'tentative'
    END,
    CASE
        WHEN (SELECT COUNT(*) FROM pragma_table_info('memories') WHERE name = 'classification') = 0
        THEN NULL
        ELSE m.classification
    END,
    m.created_at,
    m.updated_at,
    m.superseded_by
FROM memories m
WHERE (SELECT has_col FROM temp_has_truth_state) = 0;

-- Insert records that already had truth_state
INSERT OR IGNORE INTO memories_with_truth_state (id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by)
SELECT
    m.id,
    m.project_id,
    m.type,
    m.summary,
    m.detail,
    m.key,
    m.source,
    m.recall_tier,
    m.truth_state,
    CASE
        WHEN (SELECT COUNT(*) FROM pragma_table_info('memories') WHERE name = 'classification') = 0
        THEN NULL
        ELSE m.classification
    END,
    m.created_at,
    m.updated_at,
    m.superseded_by
FROM memories m;

-- Replace the old table with the new one if we added truth_state
DELETE FROM memories WHERE (SELECT has_col FROM temp_has_truth_state) = 0;
INSERT OR REPLACE INTO memories (id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by)
SELECT id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by
FROM memories_with_truth_state;

DROP TABLE memories_with_truth_state;

-- Migration v5: Create recall_metrics table (if not exists)
-- This was handled by CREATE TABLE IF NOT EXISTS at the beginning

-- Migration v6: Add classification column
-- Check if classification column exists
CREATE TEMPORARY TABLE temp_has_classification AS
SELECT COUNT(*) AS has_col
FROM pragma_table_info('memories')
WHERE name = 'classification';

-- Create a new table with classification column if it doesn't exist
CREATE TABLE IF NOT EXISTS memories_with_classification (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('fact', 'claim', 'plan', 'decision', 'constraint', 'observation', 'note')),
    summary TEXT NOT NULL,
    detail TEXT,
    key TEXT,
    source TEXT,
    recall_tier TEXT NOT NULL DEFAULT 'opportunistic',
    truth_state TEXT NOT NULL DEFAULT 'tentative',
    classification TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    superseded_by INTEGER REFERENCES memories(id),
    UNIQUE(key, project_id) ON CONFLICT REPLACE
);

-- Populate the new table with classification if the original doesn't have it
INSERT INTO memories_with_classification (id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by)
SELECT
    m.id,
    m.project_id,
    m.type,
    m.summary,
    m.detail,
    m.key,
    m.source,
    m.recall_tier,
    m.truth_state,
    NULL,  -- Default to NULL for classification
    m.created_at,
    m.updated_at,
    m.superseded_by
FROM memories m
WHERE (SELECT has_col FROM temp_has_classification) = 0;

-- Insert records that already had classification
INSERT OR IGNORE INTO memories_with_classification (id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by)
SELECT
    m.id,
    m.project_id,
    m.type,
    m.summary,
    m.detail,
    m.key,
    m.source,
    m.recall_tier,
    m.truth_state,
    m.classification,
    m.created_at,
    m.updated_at,
    m.superseded_by
FROM memories m;

-- Replace the old table with the new one if we added classification
DELETE FROM memories WHERE (SELECT has_col FROM temp_has_classification) = 0;
INSERT OR REPLACE INTO memories (id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by)
SELECT id, project_id, type, summary, detail, key, source, recall_tier, truth_state, classification, created_at, updated_at, superseded_by
FROM memories_with_classification;

DROP TABLE memories_with_classification;

-- Create indexes for efficient querying if they don't exist
CREATE INDEX IF NOT EXISTS idx_recall_metrics_project_id ON recall_metrics(project_id);
CREATE INDEX IF NOT EXISTS idx_recall_metrics_created_at ON recall_metrics(created_at);

-- Create FTS5 virtual table for full-text search if it doesn't exist
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    summary, detail, content='memories', content_rowid='id'
);

-- Create triggers to keep FTS table in sync if they don't exist
CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, summary, detail) VALUES (new.id, new.summary, new.detail);
END;

CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
    DELETE FROM memories_fts WHERE rowid = old.id;
END;

CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
    DELETE FROM memories_fts WHERE rowid = old.id;
    INSERT INTO memories_fts(rowid, summary, detail) VALUES (new.id, new.summary, new.detail);
END;

-- Update schema version to 6
PRAGMA user_version = 6;

-- Success message
SELECT 'Database updated successfully to schema version 6. Classification field added to memories table.';