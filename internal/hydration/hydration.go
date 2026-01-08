package hydration

import (
	"fmt"
	"strings"

	"github.com/andrzejmarczewski/tinyMem/internal/state"
	"github.com/andrzejmarczewski/tinyMem/internal/vault"
)

// Engine handles JIT (Just-In-Time) hydration of state
// Per spec section 8: small models cannot dereference pointers, so we materialize truth
// Per spec section 15: ETV - STALE entities must not be hydrated
type Engine struct {
	vault              *vault.Vault
	state              *state.Manager
	tracker            *Tracker
	consistencyChecker *state.ConsistencyChecker
}

// New creates a new hydration engine
func New(v *vault.Vault, s *state.Manager, tracker *Tracker, consistencyChecker *state.ConsistencyChecker) *Engine {
	return &Engine{
		vault:              v,
		state:              s,
		tracker:            tracker,
		consistencyChecker: consistencyChecker,
	}
}

// HydrationBlock represents a single injected state block
type HydrationBlock struct {
	EntityKey    string
	ArtifactHash string
	Content      string
	Filepath     string
	Symbol       string
	Method       string
	TokenCount   int // Estimated token count for this block
}

// HydrationBudget defines limits for hydration
type HydrationBudget struct {
	MaxTokens   int // Maximum tokens to hydrate (0 = unlimited)
	MaxEntities int // Maximum entities to hydrate (0 = unlimited)
	UsedTokens  int // Tokens used so far
	UsedEntities int // Entities included so far
}

// EstimateTokens estimates the number of tokens in a text string
// Uses a simple heuristic: ~1.3 tokens per word, plus overhead for code
// This is a rough approximation - real tokenization varies by model
func EstimateTokens(content string) int {
	if content == "" {
		return 0
	}

	// Count characters and words
	charCount := len(content)
	wordCount := len(strings.Fields(content))

	// Heuristic: code has more tokens per word than natural language
	// Use character count / 4 as baseline (roughly 4 chars per token)
	// Add word count * 0.3 to account for word boundaries
	estimate := (charCount / 4) + int(float64(wordCount)*0.3)

	// Minimum of 1 token for non-empty content
	if estimate < 1 {
		estimate = 1
	}

	return estimate
}

// EstimateBlockTokens estimates tokens for a complete hydration block
// Includes the template overhead (entity key, artifact hash, etc.)
func EstimateBlockTokens(block HydrationBlock) int {
	templateOverhead := 50 // Estimated tokens for [CURRENT STATE: AUTHORITATIVE] template
	contentTokens := EstimateTokens(block.Content)
	entityKeyTokens := len(block.EntityKey) / 4
	return templateOverhead + contentTokens + entityKeyTokens
}

// HydrateWithBudget retrieves AUTHORITATIVE entities with budget constraints
// Returns hydration content, hydrated entity keys, and final budget state
func (h *Engine) HydrateWithBudget(episodeID string, budget HydrationBudget) (string, []string, HydrationBudget, error) {
	// Get all authoritative entities
	entities, err := h.state.GetAuthoritative()
	if err != nil {
		return "", nil, budget, fmt.Errorf("failed to get authoritative entities: %w", err)
	}

	if len(entities) == 0 {
		return "", nil, budget, nil // No hydration needed
	}

	// ETV: Filter out STALE entities
	var staleEntities []*state.EntityState
	var freshEntities []*state.EntityState

	if h.consistencyChecker != nil {
		for _, entity := range entities {
			isStale, _, checkErr := h.consistencyChecker.IsEntityStale(entity)
			if checkErr != nil {
				staleEntities = append(staleEntities, entity)
			} else if isStale {
				staleEntities = append(staleEntities, entity)
			} else {
				freshEntities = append(freshEntities, entity)
			}
		}
	} else {
		freshEntities = entities
	}

	// Build hydration blocks with budget enforcement
	var blocks []HydrationBlock
	var entityKeys []string

	for _, entity := range freshEntities {
		// Check entity limit
		if budget.MaxEntities > 0 && budget.UsedEntities >= budget.MaxEntities {
			break // Entity limit reached
		}

		artifact, err := h.vault.Get(entity.ArtifactHash)
		if err != nil || artifact == nil {
			continue
		}

		// Extract resolution method
		method := "unknown"
		if entity.Metadata != nil {
			if m, ok := entity.Metadata["resolution_method"].(string); ok {
				method = m
			}
		}

		// Create block and estimate tokens
		block := HydrationBlock{
			EntityKey:    entity.EntityKey,
			ArtifactHash: entity.ArtifactHash,
			Content:      artifact.Content,
			Filepath:     entity.Filepath,
			Symbol:       entity.Symbol,
			Method:       method,
		}
		block.TokenCount = EstimateBlockTokens(block)

		// Check token budget
		if budget.MaxTokens > 0 && budget.UsedTokens+block.TokenCount > budget.MaxTokens {
			break // Token budget exhausted
		}

		// Add block and update budget
		blocks = append(blocks, block)
		entityKeys = append(entityKeys, block.EntityKey)
		budget.UsedTokens += block.TokenCount
		budget.UsedEntities++
	}

	// Record hydration tracking
	if h.tracker != nil && episodeID != "" {
		if err := h.tracker.RecordHydration(episodeID, entityKeys); err != nil {
			// Log error but don't fail - tracking is non-critical
		}
	}

	// Build final hydration content
	var sb strings.Builder

	// Emit STATE NOTICE for STALE entities
	if len(staleEntities) > 0 {
		staleNotice := GenerateStaleNotice(staleEntities)
		sb.WriteString(staleNotice)
		sb.WriteString("\n")
		budget.UsedTokens += EstimateTokens(staleNotice)
	}

	// Format fresh entities
	if len(blocks) > 0 {
		sb.WriteString(h.formatHydration(blocks))
	}

	return sb.String(), entityKeys, budget, nil
}

// Hydrate retrieves all AUTHORITATIVE entities and formats them for injection
// Per spec section 8.1: scan state map, retrieve artifacts, inject using strict template
// Per spec section 15.6: STALE entities are excluded from hydration
// Returns the hydration content and the list of hydrated entity keys
func (h *Engine) HydrateWithTracking(episodeID string) (string, []string, error) {
	// Get all authoritative entities
	entities, err := h.state.GetAuthoritative()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get authoritative entities: %w", err)
	}

	if len(entities) == 0 {
		return "", nil, nil // No hydration needed
	}

	// ETV: Filter out STALE entities
	// Per spec section 15.6: STALE entities must NOT be hydrated
	// Per spec section 15.6: Never feed stale code to the LLM
	var staleEntities []*state.EntityState
	var freshEntities []*state.EntityState

	if h.consistencyChecker != nil {
		for _, entity := range entities {
			isStale, _, checkErr := h.consistencyChecker.IsEntityStale(entity)
			if checkErr != nil {
				// If we can't verify, treat conservatively - exclude from hydration
				staleEntities = append(staleEntities, entity)
			} else if isStale {
				staleEntities = append(staleEntities, entity)
			} else {
				freshEntities = append(freshEntities, entity)
			}
		}
	} else {
		// No consistency checker - hydrate all (backward compatibility)
		freshEntities = entities
	}

	// Retrieve artifacts and build hydration blocks for fresh entities only
	var blocks []HydrationBlock
	for _, entity := range freshEntities {
		artifact, err := h.vault.Get(entity.ArtifactHash)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get artifact %s: %w", entity.ArtifactHash, err)
		}
		if artifact == nil {
			return "", nil, fmt.Errorf("artifact %s not found in vault", entity.ArtifactHash)
		}

		// Extract resolution method from metadata if available
		method := "unknown"
		if entity.Metadata != nil {
			if m, ok := entity.Metadata["resolution_method"].(string); ok {
				method = m
			}
		}

		blocks = append(blocks, HydrationBlock{
			EntityKey:    entity.EntityKey,
			ArtifactHash: entity.ArtifactHash,
			Content:      artifact.Content,
			Filepath:     entity.Filepath,
			Symbol:       entity.Symbol,
			Method:       method,
		})
	}

	// Collect entity keys for tracking (only fresh entities were hydrated)
	entityKeys := make([]string, len(blocks))
	for i, block := range blocks {
		entityKeys[i] = block.EntityKey
	}

	// Record hydration tracking
	if h.tracker != nil && episodeID != "" {
		if err := h.tracker.RecordHydration(episodeID, entityKeys); err != nil {
			// Log error but don't fail - tracking is non-critical
			// Would need logger here, for now just continue
		}
	}

	// Build final hydration content
	var sb strings.Builder

	// Emit STATE NOTICE for STALE entities
	// Per spec section 15.6: Inform LLM about divergence
	if len(staleEntities) > 0 {
		sb.WriteString(GenerateStaleNotice(staleEntities))
		sb.WriteString("\n")
	}

	// Format fresh entities using strict template
	if len(blocks) > 0 {
		sb.WriteString(h.formatHydration(blocks))
	}

	return sb.String(), entityKeys, nil
}

// Hydrate is a backward-compatible wrapper that doesn't require episodeID
func (h *Engine) Hydrate() (string, error) {
	content, _, err := h.HydrateWithTracking("")
	return content, err
}

// formatHydration applies the strict injection template
// Per spec section 8.2: fixed template format
// Optimized to pre-allocate capacity and avoid fmt.Sprintf where possible
func (h *Engine) formatHydration(blocks []HydrationBlock) string {
	if len(blocks) == 0 {
		return ""
	}

	// Pre-calculate approximate capacity to reduce allocations
	// Rough estimate: 150 bytes of template + average content size
	estimatedSize := len(blocks) * 150
	for _, block := range blocks {
		estimatedSize += len(block.Content)
	}

	var sb strings.Builder
	sb.Grow(estimatedSize)

	for _, block := range blocks {
		sb.WriteString("[CURRENT STATE: AUTHORITATIVE]\nEntity: ")
		sb.WriteString(block.EntityKey)
		sb.WriteString("\nArtifact: ")
		sb.WriteString(block.ArtifactHash)
		sb.WriteString("\nSource: ")

		// Determine source description based on method
		switch block.Method {
		case "regex":
			sb.WriteString("Confirmed via regex")
		case "correlation":
			sb.WriteString("Inferred via correlation")
		default:
			sb.WriteString("Confirmed via AST")
		}

		sb.WriteString("\n\n")
		sb.WriteString(block.Content)
		sb.WriteString("\n[END CURRENT STATE]\n\n")
	}

	return sb.String()
}

// GenerateSafetyNotice creates a STATE NOTICE for non-CONFIRMED artifacts
// Per spec section 8.3: mandatory when previous artifact was INFERRED or UNRESOLVED
func GenerateSafetyNotice() string {
	return `[STATE NOTICE]
The previous output could not be structurally linked to a known entity.
It has NOT modified the State Map.
[END NOTICE]

`
}

// GenerateStaleNotice creates a STATE NOTICE for STALE entities
// Per spec section 15.6: Inform LLM about disk divergence
func GenerateStaleNotice(staleEntities []*state.EntityState) string {
	var sb strings.Builder

	sb.WriteString("[STATE NOTICE: DISK DIVERGENCE DETECTED]\n")
	sb.WriteString("The following entities in the State Map have diverged from disk:\n\n")

	for _, entity := range staleEntities {
		sb.WriteString(fmt.Sprintf("  Entity: %s\n", entity.EntityKey))
		sb.WriteString(fmt.Sprintf("  File:   %s\n", entity.Filepath))
		sb.WriteString("  Reason: File has been modified on disk since last State Map update\n")
		sb.WriteString("\n")
	}

	sb.WriteString("These entities have been EXCLUDED from hydration to prevent stale code injection.\n")
	sb.WriteString("The LLM must NOT assume knowledge of their current content.\n")
	sb.WriteString("\n")
	sb.WriteString("To resolve:\n")
	sb.WriteString("  - User must paste updated file content via /v1/user/code endpoint, or\n")
	sb.WriteString("  - User must explicitly acknowledge overwrite\n")
	sb.WriteString("[END NOTICE]\n\n")

	return sb.String()
}

// HydrateForEntities retrieves specific entities and formats them for injection
// Used when only specific entities are relevant to the current prompt
func (h *Engine) HydrateForEntities(entityKeys []string) (string, error) {
	if len(entityKeys) == 0 {
		return "", nil
	}

	var blocks []HydrationBlock
	for _, key := range entityKeys {
		entity, err := h.state.Get(key)
		if err != nil {
			return "", fmt.Errorf("failed to get entity %s: %w", key, err)
		}
		if entity == nil {
			continue // Entity not found, skip
		}

		// Only hydrate AUTHORITATIVE entities
		if !entity.IsAuthoritative() {
			continue
		}

		artifact, err := h.vault.Get(entity.ArtifactHash)
		if err != nil {
			return "", fmt.Errorf("failed to get artifact %s: %w", entity.ArtifactHash, err)
		}
		if artifact == nil {
			return "", fmt.Errorf("artifact %s not found in vault", entity.ArtifactHash)
		}

		method := "unknown"
		if entity.Metadata != nil {
			if m, ok := entity.Metadata["resolution_method"].(string); ok {
				method = m
			}
		}

		blocks = append(blocks, HydrationBlock{
			EntityKey:    entity.EntityKey,
			ArtifactHash: entity.ArtifactHash,
			Content:      artifact.Content,
			Filepath:     entity.Filepath,
			Symbol:       entity.Symbol,
			Method:       method,
		})
	}

	if len(blocks) == 0 {
		return "", nil
	}

	return h.formatHydration(blocks), nil
}

// HydrateByFilepath retrieves all entities for specific filepaths
// Useful when user mentions specific files in their prompt
func (h *Engine) HydrateByFilepath(filepaths []string) (string, error) {
	if len(filepaths) == 0 {
		return "", nil
	}

	var blocks []HydrationBlock
	for _, filepath := range filepaths {
		entities, err := h.state.GetByFilepath(filepath)
		if err != nil {
			return "", fmt.Errorf("failed to get entities for filepath %s: %w", filepath, err)
		}

		for _, entity := range entities {
			// Only hydrate AUTHORITATIVE entities
			if !entity.IsAuthoritative() {
				continue
			}

			artifact, err := h.vault.Get(entity.ArtifactHash)
			if err != nil {
				return "", fmt.Errorf("failed to get artifact %s: %w", entity.ArtifactHash, err)
			}
			if artifact == nil {
				return "", fmt.Errorf("artifact %s not found in vault", entity.ArtifactHash)
			}

			method := "unknown"
			if entity.Metadata != nil {
				if m, ok := entity.Metadata["resolution_method"].(string); ok {
					method = m
				}
			}

			blocks = append(blocks, HydrationBlock{
				EntityKey:    entity.EntityKey,
				ArtifactHash: entity.ArtifactHash,
				Content:      artifact.Content,
				Filepath:     entity.Filepath,
				Symbol:       entity.Symbol,
				Method:       method,
			})
		}
	}

	if len(blocks) == 0 {
		return "", nil
	}

	return h.formatHydration(blocks), nil
}
