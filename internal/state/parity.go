package state

import (
	"encoding/json"
	"fmt"

	"github.com/andrzejmarczewski/tslp/internal/entity"
	"github.com/andrzejmarczewski/tslp/internal/vault"
)

// ParityResult represents the result of a structural parity check
type ParityResult struct {
	OK                bool
	Reason            string
	MissingSymbols    []string
	CollapsedStructure bool
}

// CheckStructuralParity verifies that a new artifact doesn't lose information
// Per spec section 7: Enforces parity only for CONFIRMED entities
// Mechanical checks: no loss of top-level symbols, no structure collapse
func CheckStructuralParity(currentState *EntityState, newResolution *entity.Resolution, vaultInstance *vault.Vault) (*ParityResult, error) {
	// Only enforce parity for CONFIRMED entities
	if newResolution.Confidence != "CONFIRMED" {
		// INFERRED and UNRESOLVED don't need parity checks
		return &ParityResult{OK: true, Reason: "parity not required for non-CONFIRMED entities"}, nil
	}

	// If no current state, parity is automatically satisfied (new entity)
	if currentState == nil {
		return &ParityResult{OK: true, Reason: "new entity"}, nil
	}

	// Retrieve current artifact from vault
	currentArtifact, err := vaultInstance.Get(currentState.ArtifactHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get current artifact: %w", err)
	}
	if currentArtifact == nil {
		return nil, fmt.Errorf("current artifact not found in vault: %s", currentState.ArtifactHash)
	}

	// Extract current symbols from metadata
	var currentSymbols []string
	if currentState.Metadata != nil {
		if symbolsData, ok := currentState.Metadata["symbols"]; ok {
			switch v := symbolsData.(type) {
			case []interface{}:
				for _, sym := range v {
					if str, ok := sym.(string); ok {
						currentSymbols = append(currentSymbols, str)
					}
				}
			case []string:
				currentSymbols = v
			case string:
				// Might be JSON-encoded
				json.Unmarshal([]byte(v), &currentSymbols)
			}
		}
	}

	// If no symbols in metadata, use the single symbol from entity
	if len(currentSymbols) == 0 {
		currentSymbols = []string{currentState.Symbol}
	}

	// New artifact must define same symbol set or strict superset
	newSymbols := newResolution.Symbols
	if len(newSymbols) == 0 {
		// No symbols detected in new artifact - this is a collapse
		return &ParityResult{
			OK:                false,
			Reason:            "new artifact has no symbols (structure collapsed)",
			MissingSymbols:    currentSymbols,
			CollapsedStructure: true,
		}, nil
	}

	// Check for missing symbols
	newSymbolSet := make(map[string]bool)
	for _, sym := range newSymbols {
		newSymbolSet[sym] = true
	}

	var missingSymbols []string
	for _, currentSym := range currentSymbols {
		if !newSymbolSet[currentSym] {
			missingSymbols = append(missingSymbols, currentSym)
		}
	}

	if len(missingSymbols) > 0 {
		return &ParityResult{
			OK:                false,
			Reason:            "new artifact is missing existing symbols",
			MissingSymbols:    missingSymbols,
			CollapsedStructure: false,
		}, nil
	}

	// Check AST node count collapse (if available)
	if newResolution.ASTNodeCount != nil {
		// Extract current AST node count from metadata
		var currentNodeCount *int
		if currentState.Metadata != nil {
			if nodeCountData, ok := currentState.Metadata["ast_node_count"]; ok {
				switch v := nodeCountData.(type) {
				case float64:
					count := int(v)
					currentNodeCount = &count
				case int:
					currentNodeCount = &v
				}
			}
		}

		if currentNodeCount != nil {
			// Define collapse threshold: new count should not be less than 50% of original
			// This is a mechanical check to detect significant structure loss
			threshold := float64(*currentNodeCount) * 0.5
			if float64(*newResolution.ASTNodeCount) < threshold {
				return &ParityResult{
					OK:                false,
					Reason:            fmt.Sprintf("AST node count collapsed: %d → %d (threshold: %.0f)", *currentNodeCount, *newResolution.ASTNodeCount, threshold),
					MissingSymbols:    nil,
					CollapsedStructure: true,
				}, nil
			}
		}
	}

	// Token count collapse check (if available)
	// Extract token counts from vault artifacts
	var currentTokenCount, newTokenCount *int
	if currentArtifact.TokenCount != nil {
		currentTokenCount = currentArtifact.TokenCount
	}

	// We don't have newTokenCount yet - this would need to be passed in
	// For now, skip token count check

	if currentTokenCount != nil && newTokenCount != nil {
		// Token count should not collapse beyond threshold
		threshold := float64(*currentTokenCount) * 0.5
		if float64(*newTokenCount) < threshold {
			return &ParityResult{
				OK:                false,
				Reason:            fmt.Sprintf("token count collapsed: %d → %d (threshold: %.0f)", *currentTokenCount, *newTokenCount, threshold),
				MissingSymbols:    nil,
				CollapsedStructure: true,
			}, nil
		}
	}

	// All parity checks passed
	return &ParityResult{
		OK:                true,
		Reason:            "structural parity satisfied",
		MissingSymbols:    nil,
		CollapsedStructure: false,
	}, nil
}
