package runtime

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/andrzejmarczewski/tslp/internal/entity"
	"github.com/andrzejmarczewski/tslp/internal/ledger"
	"github.com/andrzejmarczewski/tslp/internal/state"
	"github.com/andrzejmarczewski/tslp/internal/vault"
)

// Runtime manages the core artifact lifecycle and promotion rules
// Per spec sections 5, 6, 7: state machine, promotion gates, structural parity
type Runtime struct {
	db      *sql.DB
	vault   *vault.Vault
	ledger  *ledger.Ledger
	state   *state.Manager
	resolver *entity.Resolver
}

// New creates a new Runtime instance
func New(db *sql.DB) *Runtime {
	return &Runtime{
		db:       db,
		vault:    vault.New(db),
		ledger:   ledger.New(db),
		state:    state.NewManager(db),
		resolver: entity.NewResolver(db),
	}
}

// PromotionResult indicates the outcome of attempting to promote an artifact
type PromotionResult struct {
	Promoted       bool
	State          ledger.State
	Reason         string
	RequiresUserConfirmation bool
}

// ProcessArtifact handles the complete lifecycle of a new artifact
// Per spec: Store → Resolve → Evaluate → Promote (if gates pass)
func (r *Runtime) ProcessArtifact(content string, contentType vault.ContentType, episodeID string, isUserPaste bool) (*PromotionResult, error) {
	// Step 1: Store in vault (immutable, content-addressed)
	artifactHash, err := r.vault.Store(content, contentType, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to store artifact: %w", err)
	}

	// Step 2: Resolve entity
	resolution, err := r.resolver.Resolve(artifactHash, content)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve entity: %w", err)
	}

	// Step 3: Evaluate promotion eligibility
	result, err := r.evaluatePromotion(artifactHash, resolution, isUserPaste, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate promotion: %w", err)
	}

	// Step 4: Apply state change if promoted
	if result.Promoted && resolution.EntityKey != nil {
		if err := r.applyStateChange(*resolution.EntityKey, artifactHash, resolution, result.State, episodeID, result.Reason); err != nil {
			return nil, fmt.Errorf("failed to apply state change: %w", err)
		}
	}

	return result, nil
}

// evaluatePromotion determines if an artifact should be promoted to AUTHORITATIVE
// Per spec section 6: Gate A (Structural Proof) + Gate B (Authority Grant)
func (r *Runtime) evaluatePromotion(artifactHash string, resolution *entity.Resolution, isUserPaste bool, episodeID string) (*PromotionResult, error) {
	// If entity is UNRESOLVED, artifact stays PROPOSED
	if !resolution.IsResolved() {
		return &PromotionResult{
			Promoted: false,
			State:    ledger.StateProposed,
			Reason:   "entity resolution failed - no provable entity mapping",
			RequiresUserConfirmation: false,
		}, nil
	}

	// Gate A: Structural Proof
	// Per spec: Entity resolution must be CONFIRMED, full definition present
	if !resolution.IsConfirmed() {
		return &PromotionResult{
			Promoted: false,
			State:    ledger.StateProposed,
			Reason:   "entity confidence is not CONFIRMED - only INFERRED or UNRESOLVED",
			RequiresUserConfirmation: false,
		}, nil
	}

	// TODO: Check "full definition present" - requires AST analysis
	// TODO: Check structural parity (AST node count, token count)

	// Gate B: Authority Grant
	// Per spec section 6: User Write-Head Rule takes precedence
	if isUserPaste {
		// User-pasted code is instantly AUTHORITATIVE
		// Supersede all prior LLM artifacts
		if err := r.supersedePriorArtifacts(*resolution.EntityKey, episodeID); err != nil {
			return nil, err
		}

		return &PromotionResult{
			Promoted: true,
			State:    ledger.StateAuthoritative,
			Reason:   "user write-head rule - user-pasted code wins",
			RequiresUserConfirmation: false,
		}, nil
	}

	// Check current state
	currentState, err := r.state.Get(*resolution.EntityKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state: %w", err)
	}

	// If no current state, check structural parity against empty
	if currentState == nil {
		// TODO: Implement structural parity checks
		// For now, allow promotion for new entities with CONFIRMED resolution
		return &PromotionResult{
			Promoted: true,
			State:    ledger.StateAuthoritative,
			Reason:   "new entity with CONFIRMED resolution",
			RequiresUserConfirmation: false,
		}, nil
	}

	// Check if mutation base is acknowledged
	// Per spec: "User Verification (Revised, Structural)"
	// The entity was hydrated in the previous turn and the user initiates
	// a subsequent mutation without providing replacement content
	// TODO: Implement hydration tracking to detect this scenario

	// For now, require structural parity
	parityOK, err := r.checkStructuralParity(currentState, resolution)
	if err != nil {
		return nil, err
	}

	if !parityOK {
		return &PromotionResult{
			Promoted: false,
			State:    ledger.StateProposed,
			Reason:   "structural parity check failed - potential symbol loss or collapse",
			RequiresUserConfirmation: true,
		}, nil
	}

	// Structural parity OK, promote and supersede
	if err := r.supersedePriorArtifacts(*resolution.EntityKey, episodeID); err != nil {
		return nil, err
	}

	return &PromotionResult{
		Promoted: true,
		State:    ledger.StateAuthoritative,
		Reason:   "structural parity satisfied",
		RequiresUserConfirmation: false,
	}, nil
}

// checkStructuralParity verifies no loss of top-level symbols
// Per spec section 7: mechanical, not semantic
func (r *Runtime) checkStructuralParity(currentState *state.EntityState, newResolution *entity.Resolution) (bool, error) {
	// TODO: Implement actual structural parity checks:
	// - No loss of existing top-level symbols
	// - AST node count may not collapse beyond threshold
	// - Token count collapse triggers downgrade

	// For now, assume parity is OK if we have symbol information
	if len(newResolution.Symbols) == 0 {
		return false, nil
	}

	// Retrieve current artifact to compare
	currentArtifact, err := r.vault.Get(currentState.ArtifactHash)
	if err != nil {
		return false, fmt.Errorf("failed to get current artifact: %w", err)
	}
	if currentArtifact == nil {
		return false, fmt.Errorf("current artifact not found in vault")
	}

	// TODO: Compare symbol sets, AST node counts, token counts
	// This requires full AST parsing implementation

	return true, nil
}

// supersedePriorArtifacts marks previous AUTHORITATIVE artifacts as SUPERSEDED
func (r *Runtime) supersedePriorArtifacts(entityKey, episodeID string) error {
	currentState, err := r.state.Get(entityKey)
	if err != nil {
		return err
	}

	if currentState != nil && currentState.State == ledger.StateAuthoritative {
		// Record state transition to SUPERSEDED
		fromState := ledger.StateAuthoritative
		if err := r.ledger.RecordStateTransition(
			episodeID,
			entityKey,
			&fromState,
			ledger.StateSuperseded,
			currentState.ArtifactHash,
			"superseded by new authoritative artifact",
		); err != nil {
			return fmt.Errorf("failed to record supersession: %w", err)
		}
	}

	return nil
}

// applyStateChange updates the state map and records the transition
func (r *Runtime) applyStateChange(entityKey, artifactHash string, resolution *entity.Resolution, newState ledger.State, episodeID, reason string) error {
	// Get current state for transition tracking
	currentState, err := r.state.Get(entityKey)
	if err != nil {
		return err
	}

	var fromState *ledger.State
	if currentState != nil {
		fromState = &currentState.State
	}

	// Parse entity key
	filepath, symbol, err := entity.ParseEntityKey(entityKey)
	if err != nil {
		return fmt.Errorf("invalid entity key: %w", err)
	}

	// Build metadata
	metadata := make(map[string]interface{})
	if resolution.ASTNodeCount != nil {
		metadata["ast_node_count"] = *resolution.ASTNodeCount
	}
	if len(resolution.Symbols) > 0 {
		metadata["symbols"] = resolution.Symbols
	}
	metadata["resolution_method"] = resolution.Method

	// Update state map
	if err := r.state.Set(entityKey, filepath, symbol, artifactHash, resolution.Confidence, newState, metadata); err != nil {
		return fmt.Errorf("failed to update state map: %w", err)
	}

	// Record transition in ledger
	if err := r.ledger.RecordStateTransition(episodeID, entityKey, fromState, newState, artifactHash, reason); err != nil {
		return fmt.Errorf("failed to record state transition: %w", err)
	}

	return nil
}

// Tombstone marks an entity as TOMBSTONED
// Per spec section 9.2: symbol explicitly removed, retained for N episodes
func (r *Runtime) Tombstone(entityKey, episodeID string) error {
	currentState, err := r.state.Get(entityKey)
	if err != nil {
		return err
	}
	if currentState == nil {
		return fmt.Errorf("entity not found in state map")
	}

	// Record tombstone for recovery
	now := time.Now().Unix()
	_, err = r.db.Exec(`
		INSERT INTO tombstones (entity_key, artifact_hash, tombstoned_at, episode_id, episodes_retained)
		VALUES (?, ?, ?, ?, 0)
	`, entityKey, currentState.ArtifactHash, now, episodeID)
	if err != nil {
		return fmt.Errorf("failed to record tombstone: %w", err)
	}

	// Record state transition
	fromState := currentState.State
	if err := r.ledger.RecordStateTransition(
		episodeID,
		entityKey,
		&fromState,
		ledger.StateTombstoned,
		currentState.ArtifactHash,
		"symbol explicitly removed from authoritative update",
	); err != nil {
		return fmt.Errorf("failed to record tombstone transition: %w", err)
	}

	// Remove from state map
	if err := r.state.Delete(entityKey); err != nil {
		return fmt.Errorf("failed to delete from state map: %w", err)
	}

	return nil
}

// GetVault returns the vault instance
func (r *Runtime) GetVault() *vault.Vault {
	return r.vault
}

// GetLedger returns the ledger instance
func (r *Runtime) GetLedger() *ledger.Ledger {
	return r.ledger
}

// GetState returns the state manager instance
func (r *Runtime) GetState() *state.Manager {
	return r.state
}

// GetResolver returns the entity resolver instance
func (r *Runtime) GetResolver() *entity.Resolver {
	return r.resolver
}
