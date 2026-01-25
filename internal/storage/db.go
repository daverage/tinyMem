package storage

import (
	"database/sql"
	"fmt"
	"log"
	"github.com/a-marczewski/tinymem/internal/config"

	_ "github.com/mattn/go-sqlite3"
)

const (
	SchemaVersion = 1
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection
func NewDB(cfg *config.Config) (*DB, error) {
	db, err := sql.Open("sqlite3", cfg.DBPath+"?_busy_timeout=10000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	database := &DB{conn: db}

	// Run migrations
	if err := database.migrate(); err != nil {
		return nil, err
	}

	return database, nil
}

// migrate applies database migrations
func (db *DB) migrate() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check current schema version
	var version int
	err = tx.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return err
	}

	// Apply migrations incrementally
	for version < SchemaVersion {
		version++
		switch version {
		case 1:
			if err := db.applySchemaV1(tx); err != nil {
				return fmt.Errorf("failed to apply schema v%d: %w", version, err)
			}
		default:
			return fmt.Errorf("unknown schema version: %d", version)
		}
		log.Printf("Applied schema version %d", version)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// applySchemaV1 applies the initial schema
func (db *DB) applySchemaV1(tx *sql.Tx) error {
	// Create memories table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('fact', 'claim', 'plan', 'decision', 'constraint', 'observation', 'note')),
			summary TEXT NOT NULL,
			detail TEXT,
			key TEXT,
			source TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			superseded_by INTEGER REFERENCES memories(id),
			UNIQUE(key, project_id) ON CONFLICT REPLACE
		)
	`)
	if err != nil {
		return err
	}

	// Create evidence table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS evidence (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			memory_id INTEGER NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			verified BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create embeddings table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS embeddings (
			memory_id INTEGER PRIMARY KEY REFERENCES memories(id) ON DELETE CASCADE,
			embedding_data TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Check if FTS5 is available by attempting to create a temporary FTS table
	ftsAvailable := db.isFTS5Available(tx)

	if ftsAvailable {
		// Create FTS5 virtual table for full-text search
		_, err = tx.Exec(`
			CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
				summary, detail, content='memories', content_rowid='id'
			)
		`)
		if err != nil {
			// If FTS5 creation fails despite being available, log and continue
			fmt.Printf("Warning: Could not create FTS5 table: %v\n", err)
		} else {
			// Create the triggers to keep FTS table in sync
			_, err = tx.Exec(`
				CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
					INSERT INTO memories_fts(rowid, summary, detail) VALUES (new.id, new.summary, new.detail);
				END;
			`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`
				CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
					DELETE FROM memories_fts WHERE rowid = old.id;
				END;
			`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`
				CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
					DELETE FROM memories_fts WHERE rowid = old.id;
					INSERT INTO memories_fts(rowid, summary, detail) VALUES (new.id, new.summary, new.detail);
				END;
			`)
			if err != nil {
				return err
			}
		}
	} else {
		// FTS5 not available, continue without it
		fmt.Printf("Warning: FTS5 not available, proceeding without full-text search\n")
	}

	// Update schema version
	_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", SchemaVersion))
	return err
}

// isFTS5Available checks if FTS5 is available in the SQLite instance
func (db *DB) isFTS5Available(tx *sql.Tx) bool {
	// Try to create and drop a temporary FTS5 table to test availability
	_, err := tx.Exec(`CREATE VIRTUAL TABLE test_fts5 USING fts5(content);`)
	if err != nil {
		return false
	}

	// Clean up the test table
	_, err = tx.Exec(`DROP TABLE test_fts5;`)
	return err == nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// GetConnection returns the underlying database connection
func (db *DB) GetConnection() *sql.DB {
	return db.conn
}