package state

import (
	"encoding/json"

	"github.com/andrzejmarczewski/tslp/internal/entity"
)

// StateMapAdapter adapts the state.Manager to implement entity.StateMapProvider
// This avoids circular dependencies between packages
type StateMapAdapter struct {
	manager *Manager
}

// NewStateMapAdapter creates a new adapter
func NewStateMapAdapter(manager *Manager) *StateMapAdapter {
	return &StateMapAdapter{manager: manager}
}

// GetAllEntities implements entity.StateMapProvider
func (a *StateMapAdapter) GetAllEntities() ([]entity.StateEntity, error) {
	// Get all entities from state map
	stateEntities, err := a.manager.GetAllEntities()
	if err != nil {
		return nil, err
	}

	// Convert to entity.StateEntity format
	result := make([]entity.StateEntity, len(stateEntities))
	for i, se := range stateEntities {
		// Extract symbols from metadata
		var symbols []string
		if se.Metadata != nil {
			if symbolsData, ok := se.Metadata["symbols"]; ok {
				// Symbols might be stored as []interface{} or []string
				switch v := symbolsData.(type) {
				case []interface{}:
					for _, sym := range v {
						if str, ok := sym.(string); ok {
							symbols = append(symbols, str)
						}
					}
				case []string:
					symbols = v
				case string:
					// Might be JSON-encoded
					json.Unmarshal([]byte(v), &symbols)
				}
			}
		}

		// If no symbols in metadata, use the single symbol from entity
		if len(symbols) == 0 {
			symbols = []string{se.Symbol}
		}

		result[i] = entity.StateEntity{
			EntityKey: se.EntityKey,
			Symbols:   symbols,
		}
	}

	return result, nil
}
