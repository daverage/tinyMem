package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// DB wraps the SQLite database connection
// Per spec: WAL mode enabled, no ORM, explicit migrations
// Database path comes only from config
type DB struct {
	conn *sql.DB
	path string
}

// Open creates or opens a SQLite database at the given path
// Enables WAL mode and foreign keys as per spec requirements
// Runs versioned, idempotent migrations
func Open(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(1) // SQLite limitation: one writer at a time

	// Enable WAL mode (required by spec)
	var walMode string
	if err := conn.QueryRow("PRAGMA journal_mode=WAL").Scan(&walMode); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	if walMode != "wal" {
		conn.Close()
		return nil, fmt.Errorf("failed to set WAL mode: got %s", walMode)
	}

	// Enable foreign keys (required by spec)
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	if err := conn.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to verify foreign keys: %w", err)
	}
	if fkEnabled != 1 {
		conn.Close()
		return nil, fmt.Errorf("foreign keys not enabled")
	}

	db := &DB{
		conn: conn,
		path: dbPath,
	}

	// Run migrations (versioned and idempotent)
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

// migrate applies versioned migrations in order
// Per spec: migrations are versioned and idempotent
func (db *DB) migrate() error {
	// Create migrations tracking table if it doesn't exist
	if err := db.initMigrationsTable(); err != nil {
		return fmt.Errorf("failed to init migrations table: %w", err)
	}

	// Get list of migration files
	migrations, err := db.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	// Sort migrations by version (filename)
	sort.Strings(migrations)

	// Apply each migration
	for _, migration := range migrations {
		if err := db.applyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration, err)
		}
	}

	return nil
}

// initMigrationsTable creates the schema_migrations table
// This table tracks which migrations have been applied
func (db *DB) initMigrationsTable() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)
	`)
	return err
}

// getMigrationFiles returns list of migration files from embedded FS
func (db *DB) getMigrationFiles() ([]string, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrations = append(migrations, entry.Name())
		}
	}

	return migrations, nil
}

// applyMigration applies a single migration if it hasn't been applied yet
// Per spec: idempotent - safe to run multiple times
func (db *DB) applyMigration(filename string) error {
	// Check if already applied
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", filename).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if count > 0 {
		// Migration already applied, skip
		return nil
	}

	// Read migration file
	content, err := migrationFS.ReadFile("migrations/" + filename)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Begin transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration
	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration as applied
	timestamp := getCurrentTimestamp()
	if _, err := tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", filename, timestamp); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Conn returns the underlying sql.DB connection
// Exposed for direct SQL queries (no ORM per spec)
// Per requirements: DB not exposed outside internal/storage
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Begin starts a new transaction
func (db *DB) Begin() (*sql.Tx, error) {
	return db.conn.Begin()
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}

// Ping verifies database connectivity
func (db *DB) Ping() error {
	return db.conn.Ping()
}

// getCurrentTimestamp returns current Unix timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
