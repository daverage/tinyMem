package entity

import (
	"strings"
	"time"
)

// StateMapProvider interface allows correlation to query the state map
// This avoids circular dependencies between entity and state packages
type StateMapProvider interface {
	GetAllEntities() ([]StateEntity, error)
}

// StateEntity represents an entity from the state map for correlation
type StateEntity struct {
	EntityKey string
	Symbols   []string
}

// resolveViaCorrelation attempts entity resolution by correlating with existing State Map entities
// Per spec section 4.1: Only runs if AST + regex both fail
// Per spec section 11: Can only compare against existing State Map entities, never introduces new ones
// Always returns INFERRED confidence - correlation can never CONFIRM
func (r *Resolver) resolveViaCorrelation(artifactHash, content string, stateMap StateMapProvider) *Resolution {
	if stateMap == nil {
		// No state map available
		return &Resolution{
			ArtifactHash: artifactHash,
			EntityKey:    nil,
			Confidence:   ConfidenceUnresolved,
			Method:       MethodUnresolved,
			Filepath:     nil,
			Symbols:      []string{},
			ASTNodeCount: nil,
			CreatedAt:    time.Now(),
		}
	}

	// Get all existing entities from state map
	entities, err := stateMap.GetAllEntities()
	if err != nil || len(entities) == 0 {
		// No entities to correlate against
		return &Resolution{
			ArtifactHash: artifactHash,
			EntityKey:    nil,
			Confidence:   ConfidenceUnresolved,
			Method:       MethodCorrelation,
			Filepath:     nil,
			Symbols:      []string{},
			ASTNodeCount: nil,
			CreatedAt:    time.Now(),
		}
	}

	// Extract simple tokens from content for comparison
	// Per spec: structural correlation, not semantic
	contentTokens := extractTokens(content)

	var bestMatch *StateEntity
	var bestScore float64
	matchCount := 0

	// Compare against each existing entity
	for i := range entities {
		entity := &entities[i]

		// Calculate overlap score based on symbol presence in content
		score := calculateOverlap(contentTokens, entity.Symbols)

		if score > bestScore {
			bestScore = score
			bestMatch = entity
			matchCount = 1
		} else if score == bestScore && score > 0 {
			matchCount++
		}
	}

	// Per spec: Require a single clear match
	// If multiple entities have the same high score, it's ambiguous
	if bestMatch == nil || bestScore < 0.5 || matchCount > 1 {
		// No clear match or ambiguous
		return &Resolution{
			ArtifactHash: artifactHash,
			EntityKey:    nil,
			Confidence:   ConfidenceUnresolved,
			Method:       MethodCorrelation,
			Filepath:     nil,
			Symbols:      []string{},
			ASTNodeCount: nil,
			CreatedAt:    time.Now(),
		}
	}

	// Found a single clear match
	// Per spec section 11: Always return INFERRED (correlation can never CONFIRM)
	return &Resolution{
		ArtifactHash: artifactHash,
		EntityKey:    &bestMatch.EntityKey,
		Confidence:   ConfidenceInferred,
		Method:       MethodCorrelation,
		Filepath:     nil,
		Symbols:      bestMatch.Symbols,
		ASTNodeCount: nil,
		CreatedAt:    time.Now(),
	}
}

// extractTokens extracts simple word tokens from content
// Used for structural correlation
func extractTokens(content string) map[string]bool {
	tokens := make(map[string]bool)

	// Simple tokenization: split on non-alphanumeric characters
	var currentToken strings.Builder

	for _, r := range content {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			currentToken.WriteRune(r)
		} else {
			if currentToken.Len() > 0 {
				token := currentToken.String()
				if len(token) > 2 { // Ignore very short tokens
					tokens[token] = true
				}
				currentToken.Reset()
			}
		}
	}

	// Don't forget the last token
	if currentToken.Len() > 0 {
		token := currentToken.String()
		if len(token) > 2 {
			tokens[token] = true
		}
	}

	return tokens
}

// calculateOverlap calculates the overlap score between content tokens and entity symbols
// Returns a score between 0.0 and 1.0
func calculateOverlap(contentTokens map[string]bool, symbols []string) float64 {
	if len(symbols) == 0 {
		return 0.0
	}

	matchCount := 0
	for _, symbol := range symbols {
		if contentTokens[symbol] {
			matchCount++
		}
	}

	// Score is the ratio of matched symbols
	return float64(matchCount) / float64(len(symbols))
}
