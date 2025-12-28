package runtime

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/andrzejmarczewski/tslp/internal/entity"
	"github.com/andrzejmarczewski/tslp/internal/fs"
	"github.com/andrzejmarczewski/tslp/internal/hydration"
	"github.com/andrzejmarczewski/tslp/internal/ledger"
	"github.com/andrzejmarczewski/tslp/internal/state"
	"github.com/andrzejmarczewski/tslp/internal/vault"
)

// Runtime manages the core artifact lifecycle and promotion rules
// Per spec sections 5, 6, 7: state machine, promotion gates, structural parity
// Per spec section 15: External Truth Verification (ETV)
type Runtime struct {
	db                 *sql.DB
	vault              *vault.Vault
	ledger             *ledger.Ledger
	state              *state.Manager
	resolver           *entity.Resolver
	hydrationTracker   *hydration.Tracker
	consistencyChecker *state.ConsistencyChecker
}

// New creates a new Runtime instance
func New(db *sql.DB) *Runtime {
	stateManager := state.NewManager(db)
	resolver := entity.NewResolver(db)
	vaultInstance := vault.New(db)

	// Wire up state map adapter for correlation fallback
	// This must be done after both are created to avoid circular dependency
	stateAdapter := state.NewStateMapAdapter(stateManager)
	resolver.SetStateMap(stateAdapter)

	// Create hydration tracker for Gate B (User Verification)
	tracker := hydration.NewTracker(db)

	// Create ETV consistency checker for detecting disk divergence
	// Per spec section 15: External Truth Verification
	fsReader := fs.NewReader()
	consistencyChecker := state.NewConsistencyChecker(fsReader, vaultInstance)

	return &Runtime{
		db:                 db,
		vault:              vaultInstance,
		ledger:             ledger.New(db),
		state:              stateManager,
		resolver:           resolver,
		hydrationTracker:   tracker,
		consistencyChecker: consistencyChecker,
	}
}

// ResetAll truncates all persisted state (vault, ledger, state map).
// Debug-only operation invoked by diagnostics.
func (r *Runtime) ResetAll() error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin reset transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.Exec("PRAGMA foreign_keys = OFF")
	if err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}

	tables := []string{
		"ledger_state_transitions",
		"ledger_audit_results",
		"ledger_episodes",
		"state_map",
		"entity_resolution_cache",
		"tombstones",
		"vault_artifacts",
	}

	for _, table := range tables {
		if _, err = tx.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	_, err = tx.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return fmt.Errorf("failed to re-enable foreign keys: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit reset transaction: %w", err)
	}

	return nil
}

// PromotionResult indicates the outcome of attempting to promote an artifact
type PromotionResult struct {
	Promoted                 bool
	State                    ledger.State
	Reason                   string
	RequiresUserConfirmation bool
	ArtifactHash             string
	EntityKey                *string
	Confidence               string
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
	// Note: filepath is nil here - will be enhanced in later steps
	resolution, err := r.resolver.Resolve(artifactHash, content, nil)
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

	// Populate result with artifact and entity information
	result.ArtifactHash = artifactHash
	result.EntityKey = resolution.EntityKey
	result.Confidence = string(resolution.Confidence)

	return result, nil
}

// evaluatePromotion determines if an artifact should be promoted to AUTHORITATIVE
// Per spec section 6: Gate A (Structural Proof) + Gate B (Authority Grant)
func (r *Runtime) evaluatePromotion(artifactHash string, resolution *entity.Resolution, isUserPaste bool, episodeID string) (*PromotionResult, error) {
	// If entity is UNRESOLVED, artifact stays PROPOSED
	if !resolution.IsResolved() {
		return &PromotionResult{
			Promoted:                 false,
			State:                    ledger.StateProposed,
			Reason:                   "entity resolution failed - no provable entity mapping",
			RequiresUserConfirmation: false,
		}, nil
	}

	// Gate A: Structural Proof
	// Per spec: Entity resolution must be CONFIRMED, full definition present
	if !resolution.IsConfirmed() {
		return &PromotionResult{
			Promoted:                 false,
			State:                    ledger.StateProposed,
			Reason:                   "entity confidence is not CONFIRMED - only INFERRED or UNRESOLVED",
			RequiresUserConfirmation: false,
		}, nil
	}

	// Note: "Full definition present" validation deferred to future enhancement
	// AST parsing confirms structure; completeness checking requires additional analysis

	// Gate B: Authority Grant
	// Per spec section 6: User Write-Head Rule takes precedence
	if isUserPaste {
		// User-pasted code is instantly AUTHORITATIVE
		// Supersede all prior LLM artifacts
		if err := r.supersedePriorArtifacts(*resolution.EntityKey, episodeID); err != nil {
			return nil, err
		}

		return &PromotionResult{
			Promoted:                 true,
			State:                    ledger.StateAuthoritative,
			Reason:                   "user write-head rule - user-pasted code wins",
			RequiresUserConfirmation: false,
		}, nil
	}

	// Check current state
	currentState, err := r.state.Get(*resolution.EntityKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state: %w", err)
	}

	// ETV Gate: External Truth Verification (Spec Section 15)
	// Per spec: Check disk consistency BEFORE allowing any promotion
	// If entity is STALE (disk diverges from State Map), block LLM-originated promotions
	if currentState != nil && r.consistencyChecker != nil {
		isStale, fileExists, checkErr := r.consistencyChecker.IsEntityStale(currentState)

		if checkErr != nil {
			// File read error - treat conservatively
			// Cannot verify consistency, so block promotion
			return &PromotionResult{
				Promoted:                 false,
				State:                    ledger.StateProposed,
				Reason:                   fmt.Sprintf("ETV check failed - cannot verify disk consistency: %v", checkErr),
				RequiresUserConfirmation: true,
			}, nil
		}

		if isStale {
			// CRITICAL SAFETY BLOCK
			// Per spec section 15.4: STALE entities MUST NOT be used as promotion base
			// Per spec section 15.5: Block LLM-originated promotions
			// Per spec section 15.6: Require explicit user action to resolve
			if fileExists {
				return &PromotionResult{
					Promoted:                 false,
					State:                    ledger.StateProposed,
					Reason:                   fmt.Sprintf("STALE - disk content differs from State Map for %s - user confirmation required", currentState.Filepath),
					RequiresUserConfirmation: true,
				}, nil
			} else {
				return &PromotionResult{
					Promoted:                 false,
					State:                    ledger.StateProposed,
					Reason:                   fmt.Sprintf("STALE - file no longer exists on disk: %s - user confirmation required", currentState.Filepath),
					RequiresUserConfirmation: true,
				}, nil
			}
		}
	}

	// If no current state, allow promotion for new entities
	// Structural parity is N/A for new entities (no prior state to compare against)
	if currentState == nil {
		return &PromotionResult{
			Promoted:                 true,
			State:                    ledger.StateAuthoritative,
			Reason:                   "new entity with CONFIRMED resolution",
			RequiresUserConfirmation: false,
		}, nil
	}

	// Gate B: User Verification (Revised, Structural)
	// Per spec: The entity was hydrated in the previous turn and the user initiates
	// a subsequent mutation without providing replacement content
	if r.hydrationTracker != nil && episodeID != "" {
		wasHydrated, err := r.hydrationTracker.WasHydratedInPreviousTurn(episodeID, *resolution.EntityKey)
		if err != nil {
			return nil, fmt.Errorf("failed to check hydration tracking: %w", err)
		}

		if wasHydrated {
			// User saw the entity and initiated a mutation
			// This satisfies Gate B - allow promotion
			if err := r.supersedePriorArtifacts(*resolution.EntityKey, episodeID); err != nil {
				return nil, err
			}

			return &PromotionResult{
				Promoted:                 true,
				State:                    ledger.StateAuthoritative,
				Reason:                   "user verification - entity was hydrated and user initiated mutation",
				RequiresUserConfirmation: false,
			}, nil
		}
	}

	// Check structural parity
	parityResult, err := state.CheckStructuralParity(currentState, resolution, r.vault)
	if err != nil {
		return nil, fmt.Errorf("failed to check structural parity: %w", err)
	}

	if !parityResult.OK {
		// Parity check failed - artifact remains PROPOSED
		reason := fmt.Sprintf("structural parity check failed: %s", parityResult.Reason)
		if len(parityResult.MissingSymbols) > 0 {
			reason = fmt.Sprintf("%s (missing symbols: %v)", reason, parityResult.MissingSymbols)
		}

		return &PromotionResult{
			Promoted:                 false,
			State:                    ledger.StateProposed,
			Reason:                   reason,
			RequiresUserConfirmation: true,
		}, nil
	}

	// Structural parity OK, promote and supersede
	if err := r.supersedePriorArtifacts(*resolution.EntityKey, episodeID); err != nil {
		return nil, err
	}

	return &PromotionResult{
		Promoted:                 true,
		State:                    ledger.StateAuthoritative,
		Reason:                   "structural parity satisfied",
		RequiresUserConfirmation: false,
	}, nil
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
	if err := r.state.Set(entityKey, filepath, symbol, artifactHash, string(resolution.Confidence), newState, metadata); err != nil {
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

// GetHydrationTracker returns the hydration tracker instance
func (r *Runtime) GetHydrationTracker() *hydration.Tracker {
	return r.hydrationTracker
}

// GetConsistencyChecker returns the ETV consistency checker instance
func (r *Runtime) GetConsistencyChecker() *state.ConsistencyChecker {
	return r.consistencyChecker
}
