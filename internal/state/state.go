package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrzejmarczewski/tinyMem/internal/ledger"
)

// EntityState represents the current state of an entity in the state map
// Per spec section 3.3: exactly one authoritative artifact per entity
type EntityState struct {
	EntityKey    string
	Filepath     string
	Symbol       string
	ArtifactHash string
	Confidence   string // "CONFIRMED", "INFERRED", or "UNRESOLVED"
	State        ledger.State
	LastUpdated  time.Time
	Metadata     map[string]interface{}
}

// Manager manages the State Map
// Per spec: single source of truth, rebuildable from Vault + Ledger
type Manager struct {
	db *sql.DB
}

// NewManager creates a new state map manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// Get retrieves the current state for an entity
func (m *Manager) Get(entityKey string) (*EntityState, error) {
	var es EntityState
	var lastUpdated int64
	var metadataJSON []byte

	err := m.db.QueryRow(`
		SELECT entity_key, filepath, symbol, artifact_hash, confidence, state, last_updated, metadata
		FROM state_map
		WHERE entity_key = ?
	`, entityKey).Scan(&es.EntityKey, &es.Filepath, &es.Symbol, &es.ArtifactHash,
		&es.Confidence, &es.State, &lastUpdated, &metadataJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity state: %w", err)
	}

	es.LastUpdated = time.Unix(lastUpdated, 0)

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &es.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &es, nil
}

// Set creates or updates an entity in the state map
// Per spec: explicit state transitions only
func (m *Manager) Set(entityKey, filepath, symbol, artifactHash string, confidence string, state ledger.State, metadata map[string]interface{}) error {
	now := time.Now().Unix()

	var metadataJSON []byte
	var err error
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = m.db.Exec(`
		INSERT INTO state_map (entity_key, filepath, symbol, artifact_hash, confidence, state, last_updated, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_key) DO UPDATE SET
			filepath = excluded.filepath,
			symbol = excluded.symbol,
			artifact_hash = excluded.artifact_hash,
			confidence = excluded.confidence,
			state = excluded.state,
			last_updated = excluded.last_updated,
			metadata = excluded.metadata
	`, entityKey, filepath, symbol, artifactHash, confidence, state, now, metadataJSON)

	if err != nil {
		return fmt.Errorf("failed to set entity state: %w", err)
	}

	return nil
}

// GetAuthoritative retrieves all entities in AUTHORITATIVE state
// Per spec section 8: used for JIT hydration
func (m *Manager) GetAuthoritative() ([]*EntityState, error) {
	rows, err := m.db.Query(`
		SELECT entity_key, filepath, symbol, artifact_hash, confidence, state, last_updated, metadata
		FROM state_map
		WHERE state = ?
		ORDER BY filepath, symbol
	`, ledger.StateAuthoritative)
	if err != nil {
		return nil, fmt.Errorf("failed to query authoritative entities: %w", err)
	}
	defer rows.Close()

	var entities []*EntityState
	for rows.Next() {
		var es EntityState
		var lastUpdated int64
		var metadataJSON []byte

		err := rows.Scan(&es.EntityKey, &es.Filepath, &es.Symbol, &es.ArtifactHash,
			&es.Confidence, &es.State, &lastUpdated, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		es.LastUpdated = time.Unix(lastUpdated, 0)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &es.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		entities = append(entities, &es)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}

// Count returns the number of entries in the state map
func (m *Manager) Count() (int, error) {
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM state_map").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count state map entries: %w", err)
	}
	return count, nil
}

// GetByFilepath retrieves all entities for a given filepath
func (m *Manager) GetByFilepath(filepath string) ([]*EntityState, error) {
	rows, err := m.db.Query(`
		SELECT entity_key, filepath, symbol, artifact_hash, confidence, state, last_updated, metadata
		FROM state_map
		WHERE filepath = ?
		ORDER BY symbol
	`, filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities by filepath: %w", err)
	}
	defer rows.Close()

	var entities []*EntityState
	for rows.Next() {
		var es EntityState
		var lastUpdated int64
		var metadataJSON []byte

		err := rows.Scan(&es.EntityKey, &es.Filepath, &es.Symbol, &es.ArtifactHash,
			&es.Confidence, &es.State, &lastUpdated, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		es.LastUpdated = time.Unix(lastUpdated, 0)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &es.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		entities = append(entities, &es)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}

// GetBySymbol retrieves all entities with a specific symbol
func (m *Manager) GetBySymbol(symbol string) ([]*EntityState, error) {
	rows, err := m.db.Query(`
		SELECT entity_key, filepath, symbol, artifact_hash, confidence, state, last_updated, metadata
		FROM state_map
		WHERE symbol = ?
		ORDER BY filepath
	`, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities by symbol: %w", err)
	}
	defer rows.Close()

	var entities []*EntityState
	for rows.Next() {
		var es EntityState
		var lastUpdated int64
		var metadataJSON []byte

		err := rows.Scan(&es.EntityKey, &es.Filepath, &es.Symbol, &es.ArtifactHash,
			&es.Confidence, &es.State, &lastUpdated, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		es.LastUpdated = time.Unix(lastUpdated, 0)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &es.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		entities = append(entities, &es)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}

// Delete removes an entity from the state map
// Per spec section 9.2: used for tombstoning
func (m *Manager) Delete(entityKey string) error {
	_, err := m.db.Exec("DELETE FROM state_map WHERE entity_key = ?", entityKey)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}
	return nil
}

// IsAuthoritative checks if an entity is in AUTHORITATIVE state
func (es *EntityState) IsAuthoritative() bool {
	return es.State == ledger.StateAuthoritative
}

// IsConfirmed checks if an entity has CONFIRMED confidence
func (es *EntityState) IsConfirmed() bool {
	return string(es.Confidence) == "CONFIRMED"
}

// GetAllEntities retrieves all entities from the state map
// Used by entity resolution correlation fallback
func (m *Manager) GetAllEntities() ([]EntityState, error) {
	rows, err := m.db.Query(`
		SELECT entity_key, filepath, symbol, artifact_hash, confidence, state, last_updated, metadata
		FROM state_map
		ORDER BY entity_key
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query all entities: %w", err)
	}
	defer rows.Close()

	var entities []EntityState
	for rows.Next() {
		var es EntityState
		var lastUpdated int64
		var metadataJSON []byte

		err := rows.Scan(&es.EntityKey, &es.Filepath, &es.Symbol, &es.ArtifactHash,
			&es.Confidence, &es.State, &lastUpdated, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		es.LastUpdated = time.Unix(lastUpdated, 0)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &es.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		entities = append(entities, es)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}
