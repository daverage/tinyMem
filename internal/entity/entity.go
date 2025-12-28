package entity

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Confidence represents the level of certainty in entity resolution
// Per spec section 4: only CONFIRMED may advance state
type Confidence string

const (
	ConfidenceConfirmed  Confidence = "CONFIRMED"  // AST or exact structural match
	ConfidenceInferred   Confidence = "INFERRED"   // Regex or similarity match
	ConfidenceUnresolved Confidence = "UNRESOLVED" // No provable mapping
)

// ResolutionMethod indicates how the entity was resolved
type ResolutionMethod string

const (
	MethodAST         ResolutionMethod = "ast"         // Tree-sitter parsing (NOT YET IMPLEMENTED)
	MethodRegex       ResolutionMethod = "regex"       // Deterministic regex from symbols.json
	MethodCorrelation ResolutionMethod = "correlation" // State map correlation
	MethodUnresolved  ResolutionMethod = "unresolved"  // Failed to resolve
)

// Resolution represents the result of entity resolution
// Per spec section 4.1: ordered pipeline with strict confidence levels
type Resolution struct {
	ArtifactHash string
	EntityKey    *string          // nil if UNRESOLVED
	Confidence   Confidence
	Method       ResolutionMethod
	Filepath     *string
	Symbols      []string
	ASTNodeCount *int
	CreatedAt    time.Time
}

// Entity represents a filepath::symbol mapping
// This is the key used in the State Map
type Entity struct {
	Key      string // filepath::symbol
	Filepath string
	Symbol   string
}

// MakeEntityKey constructs the canonical entity key
// Per spec section 3.3: filepath::symbol format
func MakeEntityKey(filepath, symbol string) string {
	return filepath + "::" + symbol
}

// ParseEntityKey splits an entity key into filepath and symbol
func ParseEntityKey(key string) (filepath, symbol string, err error) {
	// Find the :: separator
	for i := 0; i < len(key)-1; i++ {
		if key[i] == ':' && key[i+1] == ':' {
			return key[:i], key[i+2:], nil
		}
	}
	return "", "", fmt.Errorf("invalid entity key format: %s", key)
}

// Resolver handles entity resolution
// Per spec section 4: gatekeeper for state advancement
type Resolver struct {
	db       *sql.DB
	stateMap StateMapProvider
}

// NewResolver creates a new entity resolver
func NewResolver(db *sql.DB) *Resolver {
	return &Resolver{db: db, stateMap: nil}
}

// SetStateMap sets the state map provider for correlation fallback
// Called after state manager is initialized to avoid circular dependency
func (r *Resolver) SetStateMap(stateMap StateMapProvider) {
	r.stateMap = stateMap
}

// Resolve performs entity resolution on an artifact
// Per spec section 4.1: AST → Regex → Correlation → Failure
// Per requirements: No embeddings, no fuzzy search beyond defined similarity rule
func (r *Resolver) Resolve(artifactHash, content string, filepath *string) (*Resolution, error) {
	// Check cache first
	cached, err := r.GetCached(artifactHash)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return cached, nil
	}

	// Resolution Pipeline (ordered per spec):
	// 1. AST Extraction (Primary) - Tree-sitter parsing
	//    Per spec: Returns CONFIRMED confidence when successful
	language := DetectLanguage(content, filepath)
	if language != "unknown" {
		resolution, err := r.resolveViaAST(artifactHash, content, language, filepath)
		if err == nil && resolution.IsResolved() {
			if err := r.Cache(resolution); err != nil {
				return nil, err
			}
			return resolution, nil
		}
		// If AST fails, fall through to regex (don't return error)
	}

	// 2. Language Regex Map (Fallback)
	//    Deterministic patterns from symbols.json
	//    Unique, exact match → CONFIRMED
	//    Partial match → INFERRED
	if language != "unknown" {
		resolution := r.resolveViaRegexPatterns(artifactHash, content, language)
		if resolution.IsResolved() {
			if err := r.Cache(resolution); err != nil {
				return nil, err
			}
			return resolution, nil
		}
	}

	// 3. State Map Correlation (Fallback)
	//    High overlap with single existing entity → INFERRED
	//    Per spec: Can only align with existing entities, never introduces new ones
	if r.stateMap != nil {
		resolution := r.resolveViaCorrelation(artifactHash, content, r.stateMap)
		if resolution.IsResolved() {
			if err := r.Cache(resolution); err != nil {
				return nil, err
			}
			return resolution, nil
		}
	}

	// 4. Failure
	//    No match → UNRESOLVED
	resolution := &Resolution{
		ArtifactHash: artifactHash,
		EntityKey:    nil,
		Confidence:   ConfidenceUnresolved,
		Method:       MethodUnresolved,
		Filepath:     nil,
		Symbols:      []string{},
		ASTNodeCount: nil,
		CreatedAt:    time.Now(),
	}

	// Cache the result
	if err := r.Cache(resolution); err != nil {
		return nil, err
	}

	return resolution, nil
}

// resolveViaAST attempts to resolve entities using Tree-sitter AST parsing
// Per spec section 4.1: Primary resolution method, returns CONFIRMED confidence
func (r *Resolver) resolveViaAST(artifactHash, content, language string, filepath *string) (*Resolution, error) {
	// Parse AST using Tree-sitter
	astResult, err := ParseAST(content, language)
	if err != nil {
		// AST parsing failed - this is a hard failure, do not guess
		return nil, err
	}

	// If no symbols found, this is unresolved
	if len(astResult.Symbols) == 0 {
		return &Resolution{
			ArtifactHash: artifactHash,
			EntityKey:    nil,
			Confidence:   ConfidenceUnresolved,
			Method:       MethodAST,
			Filepath:     filepath,
			Symbols:      []string{},
			ASTNodeCount: &astResult.ASTNodeCount,
			CreatedAt:    time.Now(),
		}, nil
	}

	// Extract symbol names
	symbolNames := make([]string, len(astResult.Symbols))
	for i, sym := range astResult.Symbols {
		symbolNames[i] = sym.Name
	}

	// For single-symbol artifacts, we can create an entity key
	// For multi-symbol artifacts, we need a filepath to disambiguate
	var entityKey *string
	if filepath != nil && len(astResult.Symbols) > 0 {
		// Use the first top-level symbol as the primary entity
		// This follows the convention that a file typically defines one main entity
		key := MakeEntityKey(*filepath, astResult.Symbols[0].Name)
		entityKey = &key
	} else if len(astResult.Symbols) == 1 {
		// Single symbol without filepath - create a synthetic key
		// Note: This is less ideal but allows progression
		key := MakeEntityKey("unknown", astResult.Symbols[0].Name)
		entityKey = &key
	}

	// AST parsing succeeded - return CONFIRMED confidence
	return &Resolution{
		ArtifactHash: artifactHash,
		EntityKey:    entityKey,
		Confidence:   ConfidenceConfirmed,
		Method:       MethodAST,
		Filepath:     filepath,
		Symbols:      symbolNames,
		ASTNodeCount: &astResult.ASTNodeCount,
		CreatedAt:    time.Now(),
	}, nil
}

// Cache stores a resolution result
func (r *Resolver) Cache(resolution *Resolution) error {
	now := time.Now().Unix()

	symbolsJSON, err := json.Marshal(resolution.Symbols)
	if err != nil {
		return fmt.Errorf("failed to marshal symbols: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO entity_resolution_cache
		(artifact_hash, entity_key, confidence, resolution_method, filepath, symbols, ast_node_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, resolution.ArtifactHash, resolution.EntityKey, resolution.Confidence, resolution.Method,
		resolution.Filepath, symbolsJSON, resolution.ASTNodeCount, now)

	if err != nil {
		return fmt.Errorf("failed to cache resolution: %w", err)
	}

	return nil
}

// GetCached retrieves a cached resolution
func (r *Resolver) GetCached(artifactHash string) (*Resolution, error) {
	var res Resolution
	var entityKey, filepath sql.NullString
	var astNodeCount sql.NullInt64
	var symbolsJSON []byte
	var createdAt int64

	err := r.db.QueryRow(`
		SELECT artifact_hash, entity_key, confidence, resolution_method, filepath, symbols, ast_node_count, created_at
		FROM entity_resolution_cache
		WHERE artifact_hash = ?
	`, artifactHash).Scan(&res.ArtifactHash, &entityKey, &res.Confidence, &res.Method,
		&filepath, &symbolsJSON, &astNodeCount, &createdAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached resolution: %w", err)
	}

	if entityKey.Valid {
		res.EntityKey = &entityKey.String
	}
	if filepath.Valid {
		res.Filepath = &filepath.String
	}
	if astNodeCount.Valid {
		count := int(astNodeCount.Int64)
		res.ASTNodeCount = &count
	}

	if err := json.Unmarshal(symbolsJSON, &res.Symbols); err != nil {
		return nil, fmt.Errorf("failed to unmarshal symbols: %w", err)
	}

	res.CreatedAt = time.Unix(createdAt, 0)

	return &res, nil
}

// IsConfirmed checks if a resolution is CONFIRMED
// Per spec: only CONFIRMED may update state map
func (res *Resolution) IsConfirmed() bool {
	return res.Confidence == ConfidenceConfirmed
}

// IsResolved checks if entity was successfully resolved
func (res *Resolution) IsResolved() bool {
	return res.EntityKey != nil && res.Confidence != ConfidenceUnresolved
}
