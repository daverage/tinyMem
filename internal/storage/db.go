package storage

import (
	"database/sql"
	"fmt"

	"github.com/a-marczewski/tinymem/internal/config"

	_ "github.com/mattn/go-sqlite3"
)

const (
	SchemaVersion = 4
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
		case 2:
			if err := db.applySchemaV2(tx); err != nil {
				return fmt.Errorf("failed to apply schema v%d: %w", version, err)
			}
		case 3:
			if err := db.applySchemaV3(tx); err != nil {
				return fmt.Errorf("failed to apply schema v%d: %w", version, err)
			}
		case 4:
			if err := db.applySchemaV4(tx); err != nil {
				return fmt.Errorf("failed to apply schema v%d: %w", version, err)
			}
		default:
			return fmt.Errorf("unknown schema version: %d", version)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// applySchemaV2 enforces fact invariants at the database layer.
func (db *DB) applySchemaV2(tx *sql.Tx) error {
	// Fail migration if invalid facts already exist.
	var invalidFacts int
	if err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM memories m
		WHERE m.type = 'fact'
		  AND NOT EXISTS (
			SELECT 1 FROM evidence e WHERE e.memory_id = m.id AND e.verified = 1
		  )
	`).Scan(&invalidFacts); err != nil {
		return err
	}
	if invalidFacts > 0 {
		return fmt.Errorf("found %d fact(s) without verified evidence", invalidFacts)
	}

	// Block direct insertion of facts without evidence.
	_, err := tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_fact_insert_guard
		BEFORE INSERT ON memories
		WHEN NEW.type = 'fact'
		BEGIN
			SELECT RAISE(ABORT, 'fact requires verified evidence');
		END;
	`)
	if err != nil {
		return err
	}

	// Block promotion to fact without verified evidence.
	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_fact_update_guard
		BEFORE UPDATE ON memories
		WHEN NEW.type = 'fact'
		BEGIN
			SELECT CASE
				WHEN (SELECT COUNT(*) FROM evidence e WHERE e.memory_id = NEW.id AND e.verified = 1) = 0
				THEN RAISE(ABORT, 'fact requires verified evidence')
			END;
		END;
	`)
	if err != nil {
		return err
	}

	// Prevent removing the last verified evidence from a fact.
	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS evidence_fact_delete_guard
		BEFORE DELETE ON evidence
		WHEN (SELECT type FROM memories WHERE id = OLD.memory_id) = 'fact'
		  AND (SELECT COUNT(*) FROM evidence e WHERE e.memory_id = OLD.memory_id AND e.verified = 1) <= 1
		BEGIN
			SELECT RAISE(ABORT, 'cannot remove last verified evidence for fact');
		END;
	`)
	if err != nil {
		return err
	}

	// Prevent un-verifying the last verified evidence for a fact.
	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS evidence_fact_unverify_guard
		BEFORE UPDATE OF verified ON evidence
		WHEN (SELECT type FROM memories WHERE id = NEW.memory_id) = 'fact'
		  AND OLD.verified = 1 AND NEW.verified = 0
		  AND (SELECT COUNT(*) FROM evidence e WHERE e.memory_id = NEW.memory_id AND e.verified = 1) <= 1
		BEGIN
			SELECT RAISE(ABORT, 'fact requires at least one verified evidence');
		END;
	`)
	return err
}

// applySchemaV3 adds the recall_tier column to memories table.
func (db *DB) applySchemaV3(tx *sql.Tx) error {
	// Check if recall_tier column already exists
	var exists int
	err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM pragma_table_info('memories')
		WHERE name='recall_tier'
	`).Scan(&exists)
	if err != nil {
		return err
	}

	if exists == 0 {
		// Add the recall_tier column with a default value
		_, err = tx.Exec(`
			ALTER TABLE memories
			ADD COLUMN recall_tier TEXT NOT NULL DEFAULT 'opportunistic'
		`)
		if err != nil {
			return err
		}

		// Update existing records to have appropriate recall_tier values based on their type
		_, err = tx.Exec(`
			UPDATE memories
			SET recall_tier = 'always'
			WHERE type IN ('fact', 'constraint')
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			UPDATE memories
			SET recall_tier = 'contextual'
			WHERE type IN ('decision', 'claim')
		`)
		if err != nil {
			return err
		}

		// For 'observation', 'note', and 'plan', leave as 'opportunistic' (the default)
	}

	// Update FTS5 table to include recall_tier in content sync
	// Drop and recreate the FTS triggers to account for the new column
	_, err = tx.Exec(`DROP TRIGGER IF EXISTS memories_ai`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TRIGGER IF EXISTS memories_ad`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TRIGGER IF EXISTS memories_au`)
	if err != nil {
		return err
	}

	// Recreate the triggers to keep FTS table in sync (only summary and detail are indexed)
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
	return err
}

// applySchemaV4 adds the truth_state column to memories table.
func (db *DB) applySchemaV4(tx *sql.Tx) error {
	// Check if truth_state column already exists
	var exists int
	err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM pragma_table_info('memories')
		WHERE name='truth_state'
	`).Scan(&exists)
	if err != nil {
		return err
	}

	if exists == 0 {
		// Add the truth_state column with a default value
		_, err = tx.Exec(`
			ALTER TABLE memories
			ADD COLUMN truth_state TEXT NOT NULL DEFAULT 'tentative'
		`)
		if err != nil {
			return err
		}

		// Update existing records to have appropriate truth_state values based on their type
		_, err = tx.Exec(`
			UPDATE memories
			SET truth_state = 'verified'
			WHERE type = 'fact'
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			UPDATE memories
			SET truth_state = 'asserted'
			WHERE type IN ('decision', 'constraint')
		`)
		if err != nil {
			return err
		}

		// For other types, leave as 'tentative' (the default)
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
			recall_tier TEXT NOT NULL DEFAULT 'opportunistic',
			truth_state TEXT NOT NULL DEFAULT 'tentative',
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
			// FTS5 creation failure is non-fatal; continue without FTS.
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
