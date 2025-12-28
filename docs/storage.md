# TSLP Storage Layer

## Overview

TSLP uses a **single SQLite database** for all persistent storage. The database is structured into three logical layers as defined in the Gold Specification.

Per requirements:
- **Single SQLite database** - No distributed storage
- **WAL mode enabled** - Write-Ahead Logging for concurrency
- **No ORM** - Direct SQL queries only
- **Versioned migrations** - Explicit schema evolution
- **Idempotent migrations** - Safe to run multiple times
- **Database path from config only** - No hardcoded paths
- **No caching layers** - Database is authoritative
- **DB not exposed outside internal/storage** - Encapsulation

## Database Structure

### Logical Layers

```
┌─────────────────────────────────────────┐
│  VAULT                                  │
│  Content-Addressed Storage              │
│  • Immutable artifacts                  │
│  • SHA-256 addressed                    │
│  • Never modified or deleted            │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  LEDGER                                 │
│  Chronological Evidence                 │
│  • Append-only episodes                 │
│  • State transitions                    │
│  • Shadow audit results                 │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  STATE MAP                              │
│  Single Source of Truth                 │
│  • filepath::symbol → artifact_hash     │
│  • Exactly one authoritative per entity │
│  • Rebuildable from Vault + Ledger      │
└─────────────────────────────────────────┘
```

### Tables

**Vault Layer:**
- `vault_artifacts` - Content-addressed artifact storage

**Ledger Layer:**
- `ledger_episodes` - Episode tracking
- `ledger_state_transitions` - State change log
- `ledger_audit_results` - Shadow audit outcomes

**State Map Layer:**
- `state_map` - Current entity state
- `entity_resolution_cache` - Resolution results cache
- `tombstones` - Deleted entity recovery

**System:**
- `schema_migrations` - Migration tracking (auto-created)

## Schema Details

### vault_artifacts

Stores all artifacts as immutable, content-addressed blobs.

```sql
CREATE TABLE vault_artifacts (
    hash TEXT PRIMARY KEY,           -- SHA-256 of content
    content TEXT NOT NULL,           -- Full artifact content
    content_type TEXT NOT NULL,      -- 'code', 'diff', 'decision', 'user_paste'
    created_at INTEGER NOT NULL,     -- Unix timestamp
    byte_size INTEGER NOT NULL,      -- Size in bytes
    token_count INTEGER              -- Optional token estimate
);
```

**Indexes:**
- `idx_vault_created` on `created_at`
- `idx_vault_type` on `content_type`

### ledger_episodes

Chronological log of all interaction episodes.

```sql
CREATE TABLE ledger_episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id TEXT NOT NULL UNIQUE, -- UUID
    timestamp INTEGER NOT NULL,
    user_prompt_hash TEXT,           -- FK to vault_artifacts
    assistant_response_hash TEXT,    -- FK to vault_artifacts
    metadata TEXT                    -- JSON blob
);
```

**Indexes:**
- `idx_ledger_timestamp` on `timestamp`
- `idx_ledger_episode` on `episode_id`

### ledger_state_transitions

Log of all state machine transitions.

```sql
CREATE TABLE ledger_state_transitions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id TEXT NOT NULL,        -- FK to ledger_episodes
    entity_key TEXT NOT NULL,        -- filepath::symbol
    from_state TEXT,                 -- NULL for new entities
    to_state TEXT NOT NULL,          -- PROPOSED, AUTHORITATIVE, etc.
    artifact_hash TEXT NOT NULL,     -- FK to vault_artifacts
    timestamp INTEGER NOT NULL,
    reason TEXT
);
```

**Indexes:**
- `idx_transitions_entity` on `entity_key`
- `idx_transitions_episode` on `episode_id`

### state_map

The single source of truth for current entity state.

```sql
CREATE TABLE state_map (
    entity_key TEXT PRIMARY KEY,     -- filepath::symbol
    filepath TEXT NOT NULL,
    symbol TEXT NOT NULL,
    artifact_hash TEXT NOT NULL,     -- FK to vault_artifacts
    confidence TEXT NOT NULL,        -- CONFIRMED, INFERRED, UNRESOLVED
    state TEXT NOT NULL,             -- Current state
    last_updated INTEGER NOT NULL,
    metadata TEXT                    -- JSON: AST info, etc.
);
```

**Indexes:**
- `idx_state_filepath` on `filepath`
- `idx_state_symbol` on `symbol`
- `idx_state_state` on `state`
- `idx_state_updated` on `last_updated`

## SQLite Configuration

### WAL Mode

Write-Ahead Logging is **required** and enabled automatically on database open:

```go
PRAGMA journal_mode=WAL
```

**Benefits:**
- Better concurrency (readers don't block writers)
- Faster writes
- Atomic commits

**Files Created:**
- `tslp.db` - Main database
- `tslp.db-wal` - Write-ahead log
- `tslp.db-shm` - Shared memory index

### Foreign Keys

Foreign key constraints are **required** and enabled automatically:

```go
PRAGMA foreign_keys=ON
```

This ensures referential integrity across vault, ledger, and state_map.

### Connection Pool

Single connection pool with max 1 writer (SQLite limitation):

```go
conn.SetMaxOpenConns(1)
```

## Migrations System

### How It Works

1. On startup, storage layer checks for unapplied migrations
2. Migrations are applied in version order (filename sorted)
3. Each migration is tracked in `schema_migrations` table
4. Migrations are **idempotent** - safe to run multiple times
5. Migration failures roll back the entire migration

### Migration Tracking

```sql
CREATE TABLE schema_migrations (
    version TEXT PRIMARY KEY,      -- Migration filename
    applied_at INTEGER NOT NULL    -- Unix timestamp
);
```

### Migration Files

Location: `internal/storage/migrations/`

Naming: `NNN_description.sql` (e.g., `001_initial_schema.sql`)

**Current Migrations:**
- `001_initial_schema.sql` - Creates all tables and indexes

### Adding New Migrations

1. Create new file: `NNN_description.sql` (increment NNN)
2. Use `CREATE TABLE IF NOT EXISTS` for idempotency
3. Use `CREATE INDEX IF NOT EXISTS` for idempotency
4. Test locally before deploying
5. Never modify existing migrations

**Example:**
```sql
-- Migration: 002
-- Description: Add new index for performance
-- Idempotent: Yes

CREATE INDEX IF NOT EXISTS idx_vault_byte_size 
ON vault_artifacts(byte_size);
```

### Idempotency

All migrations must be idempotent. This means:

✓ `CREATE TABLE IF NOT EXISTS`
✓ `CREATE INDEX IF NOT EXISTS`
✓ `INSERT OR IGNORE`
✓ `UPDATE ... WHERE NOT EXISTS`

✗ `CREATE TABLE` (without IF NOT EXISTS)
✗ `DROP TABLE`
✗ `ALTER TABLE` (not idempotent in SQLite)

### Migration Rollback

TSLP does **not** support migration rollback. If a migration fails:

1. The transaction is rolled back
2. The error is logged
3. Startup fails (fail-fast)
4. Manual intervention required

To recover:
1. Fix the migration SQL
2. Restart TSLP
3. Migration will be re-attempted

## Usage

### Opening Database

```go
import "github.com/andrzejmarczewski/tslp/internal/storage"

// Database path comes from config only
db, err := storage.Open(cfg.Database.DatabasePath)
if err != nil {
    // Handle error
}
defer db.Close()
```

### Getting Connection

```go
// Get underlying *sql.DB for queries
conn := db.Conn()

// Direct SQL (no ORM per spec)
rows, err := conn.Query("SELECT hash FROM vault_artifacts WHERE content_type = ?", "code")
```

### Transactions

```go
tx, err := db.Begin()
if err != nil {
    // Handle error
}
defer tx.Rollback() // Safe to call even after Commit

// Do work...
if err := tx.Exec(...); err != nil {
    return err // Rollback via defer
}

// Commit
if err := tx.Commit(); err != nil {
    return err
}
```

### Health Check

```go
if err := db.Ping(); err != nil {
    // Database not healthy
}
```

## Performance Considerations

### Disk Space

- Artifacts are never deleted automatically
- Vault grows unbounded
- Plan for disk capacity accordingly
- WAL files can grow; checkpoint periodically if needed

### Query Performance

- All critical queries have indexes
- Use `EXPLAIN QUERY PLAN` to verify index usage
- Avoid SELECT * in production code
- Use prepared statements for repeated queries

### Concurrency

- SQLite limits: 1 writer, multiple readers
- WAL mode improves read concurrency
- Writers serialize automatically
- No need for application-level locking

## Backup and Recovery

### Manual Backup

```bash
# Stop TSLP first
pkill tslp

# Backup all files
cp runtime/tslp.db runtime/tslp.db.backup
cp runtime/tslp.db-wal runtime/tslp.db-wal.backup
cp runtime/tslp.db-shm runtime/tslp.db-shm.backup

# Or use SQLite backup API
sqlite3 runtime/tslp.db ".backup runtime/tslp.db.backup"
```

### Online Backup

```bash
# WAL checkpoint first
sqlite3 runtime/tslp.db "PRAGMA wal_checkpoint(FULL);"

# Then backup
sqlite3 runtime/tslp.db ".backup runtime/tslp.db.backup"
```

### Recovery

```bash
# Restore from backup
cp runtime/tslp.db.backup runtime/tslp.db
rm -f runtime/tslp.db-wal runtime/tslp.db-shm

# Restart TSLP
./tslp
```

## Troubleshooting

### "database is locked"

**Cause:** Another process has the database open  
**Solution:** Stop all TSLP instances, check for orphaned connections

### "database disk image is malformed"

**Cause:** Corruption (power loss, disk failure)  
**Solution:** Restore from backup, check disk health

### Migration failed

**Cause:** SQL syntax error, constraint violation  
**Solution:** Check logs, fix migration file, restart

### WAL file growing too large

**Cause:** No checkpoints being performed  
**Solution:** Run `PRAGMA wal_checkpoint(TRUNCATE);` manually if needed

## Design Principles

**"Disk is cheap, RAM is bounded."**

1. **No caching** - Database is authoritative, no in-memory caches
2. **No ORM** - Direct SQL for transparency and control
3. **Explicit migrations** - No auto-schema generation
4. **Fail-fast** - Migration errors stop startup immediately
5. **Idempotent** - Migrations safe to re-run
6. **Immutable artifacts** - Write-once, read-many
7. **Append-only ledger** - History is never modified

The storage layer is designed for **correctness** over performance. All operations are durable and consistent.
