package hydration

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// Tracker tracks which entities were hydrated in each episode
// Per spec section 6, Gate B: Enables "User Verification" promotion path
type Tracker struct {
	db *sql.DB
}

// NewTracker creates a new hydration tracker
func NewTracker(db *sql.DB) *Tracker {
	return &Tracker{db: db}
}

// RecordHydration records that specific entities were hydrated for an episode
func (t *Tracker) RecordHydration(episodeID string, entityKeys []string) error {
	if len(entityKeys) == 0 {
		return nil // Nothing to record
	}

	entityKeysJSON, err := json.Marshal(entityKeys)
	if err != nil {
		return fmt.Errorf("failed to marshal entity keys: %w", err)
	}

	// Store in episode metadata
	// We'll update the ledger_episodes table to add hydrated_entities column
	_, err = t.db.Exec(`
		UPDATE ledger_episodes
		SET metadata = json_set(COALESCE(metadata, '{}'), '$.hydrated_entities', ?)
		WHERE episode_id = ?
	`, string(entityKeysJSON), episodeID)

	if err != nil {
		return fmt.Errorf("failed to record hydration: %w", err)
	}

	return nil
}

// GetPreviousHydration retrieves the entities that were hydrated in the previous episode
// Returns nil if there is no previous episode or no hydration was recorded
func (t *Tracker) GetPreviousHydration(currentEpisodeID string) ([]string, error) {
	// Get the timestamp of the current episode
	var currentTimestamp int64
	err := t.db.QueryRow(`
		SELECT timestamp FROM ledger_episodes WHERE episode_id = ?
	`, currentEpisodeID).Scan(&currentTimestamp)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Current episode not found
		}
		return nil, fmt.Errorf("failed to get current episode: %w", err)
	}

	// Get the most recent episode before this one
	var metadataJSON []byte
	err = t.db.QueryRow(`
		SELECT metadata FROM ledger_episodes
		WHERE timestamp < ?
		ORDER BY timestamp DESC
		LIMIT 1
	`, currentTimestamp).Scan(&metadataJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No previous episode
		}
		return nil, fmt.Errorf("failed to get previous episode: %w", err)
	}

	// Parse metadata to extract hydrated_entities
	if len(metadataJSON) == 0 {
		return nil, nil
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Extract hydrated_entities
	hydratedData, ok := metadata["hydrated_entities"]
	if !ok {
		return nil, nil // No hydration recorded
	}

	// Handle different JSON representations
	var entityKeys []string
	switch v := hydratedData.(type) {
	case string:
		// Might be JSON-encoded string
		if err := json.Unmarshal([]byte(v), &entityKeys); err != nil {
			return nil, fmt.Errorf("failed to unmarshal hydrated entities: %w", err)
		}
	case []interface{}:
		// Array of interfaces
		for _, item := range v {
			if str, ok := item.(string); ok {
				entityKeys = append(entityKeys, str)
			}
		}
	case []string:
		entityKeys = v
	default:
		return nil, nil // Unknown format
	}

	return entityKeys, nil
}

// WasHydratedInPreviousTurn checks if a specific entity was hydrated in the previous episode
func (t *Tracker) WasHydratedInPreviousTurn(currentEpisodeID, entityKey string) (bool, error) {
	hydratedEntities, err := t.GetPreviousHydration(currentEpisodeID)
	if err != nil {
		return false, err
	}

	if hydratedEntities == nil {
		return false, nil
	}

	for _, key := range hydratedEntities {
		if key == entityKey {
			return true, nil
		}
	}

	return false, nil
}
