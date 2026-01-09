package hydration

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/andrzejmarczewski/tinyMem/internal/logging"
)

// HybridEngine implements hybrid retrieval: structural anchors + semantic ranking
// Per HYBRID_RETRIEVAL_DESIGN.md: Never pure embeddings-only recall
type HybridEngine struct {
	engine    *Engine
	logger    *logging.Logger
	embedder  Embedder // Optional embedding provider
	enabled   bool     // Whether hybrid retrieval is enabled
	threshold float64  // Semantic similarity threshold
}

// Embedder interface for embedding providers
type Embedder interface {
	Embed(text string) ([]float32, error)
	CosineSimilarity(a, b []float32) float64
}

// StructuralAnchor represents a deterministic retrieval anchor
type StructuralAnchor struct {
	EntityKey string
	Reason    string // "explicit_file_mention", "explicit_symbol_mention", "hydrated_previous_turn"
	Priority  int    // Higher = more important (100 = highest)
}

// SemanticCandidate represents a semantically similar entity
type SemanticCandidate struct {
	EntityKey string
	Score     float64 // Cosine similarity score
}

// HydrationPlan represents the final hydration decision
type HydrationPlan struct {
	Entities []HydrationEntity
	Budget   HydrationBudget
}

// HydrationEntity represents a single entity to hydrate with reason
type HydrationEntity struct {
	EntityKey string
	Reason    string // "anchor:explicit_mention" or "semantic:0.85"
	Priority  int
	TokenCount int
}

// NewHybridEngine creates a new hybrid retrieval engine
func NewHybridEngine(engine *Engine, logger *logging.Logger, embedder Embedder, enabled bool, threshold float64) *HybridEngine {
	return &HybridEngine{
		engine:    engine,
		logger:    logger,
		embedder:  embedder,
		enabled:   enabled,
		threshold: threshold,
	}
}

// ExtractAnchors extracts structural anchors from query and episode history
// Phase 1 of hybrid retrieval: deterministic, never skipped
func (h *HybridEngine) ExtractAnchors(query string, episodeID string) []StructuralAnchor {
	var anchors []StructuralAnchor
	seen := make(map[string]bool) // Deduplicate

	// 1. Explicit file mentions
	// Pattern: file paths with extensions (e.g., "auth.go", "/src/main.py")
	filePattern := regexp.MustCompile(`\b[\w/\-\.]+\.(go|js|py|ts|tsx|jsx|java|cpp|c|h|rs|rb|php)\b`)
	fileMentions := filePattern.FindAllString(query, -1)

	for _, filepath := range fileMentions {
		// Get entities for this filepath
		entities, err := h.engine.state.GetByFilepath(filepath)
		if err != nil || len(entities) == 0 {
			continue
		}

		for _, entity := range entities {
			if seen[entity.EntityKey] {
				continue
			}
			anchors = append(anchors, StructuralAnchor{
				EntityKey: entity.EntityKey,
				Reason:    fmt.Sprintf("explicit_file_mention:%s", filepath),
				Priority:  100, // Highest priority
			})
			seen[entity.EntityKey] = true
		}
	}

	// 2. Explicit symbol mentions
	// Pattern: function/type names (CamelCase or snake_case identifiers)
	symbolPattern := regexp.MustCompile(`\b([A-Z][a-zA-Z0-9]*|[a-z_][a-z0-9_]+)\b`)
	symbolMentions := symbolPattern.FindAllString(query, -1)

	for _, symbol := range symbolMentions {
		// Skip common words
		if isCommonWord(symbol) {
			continue
		}

		// Search for entities with this symbol
		entities, err := h.engine.state.GetBySymbol(symbol)
		if err != nil || len(entities) == 0 {
			continue
		}

		for _, entity := range entities {
			if seen[entity.EntityKey] {
				continue
			}
			anchors = append(anchors, StructuralAnchor{
				EntityKey: entity.EntityKey,
				Reason:    fmt.Sprintf("explicit_symbol_mention:%s", symbol),
				Priority:  90,
			})
			seen[entity.EntityKey] = true
		}
	}

	// 3. Previously hydrated entities (structural invariant!)
	if episodeID != "" && h.engine.tracker != nil {
		hydratedKeys, err := h.engine.tracker.GetPreviousHydration(episodeID)
		if err == nil && hydratedKeys != nil {
			for _, key := range hydratedKeys {
				if seen[key] {
					continue
				}
				anchors = append(anchors, StructuralAnchor{
					EntityKey: key,
					Reason:    "hydrated_previous_turn",
					Priority:  80, // High priority (user saw it)
				})
				seen[key] = true
			}
		}
	}

	if h.logger != nil {
		h.logger.Debug("Extracted %d structural anchors from query", len(anchors))
	}

	return anchors
}

// RankSemantics performs semantic ranking on remaining entities
// Phase 2 of hybrid retrieval: advisory, budget-constrained
func (h *HybridEngine) RankSemantics(query string, anchors []StructuralAnchor) []SemanticCandidate {
	// If embedder not available, return empty
	if h.embedder == nil || !h.enabled {
		return nil
	}

	// Get query embedding
	queryEmbedding, err := h.embedder.Embed(query)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("Failed to embed query for semantic ranking: %v", err)
		}
		return nil
	}

	// Get all authoritative entities
	allEntities, err := h.engine.state.GetAuthoritative()
	if err != nil {
		return nil
	}

	// Build anchor set for filtering
	anchorSet := make(map[string]bool)
	for _, anchor := range anchors {
		anchorSet[anchor.EntityKey] = true
	}

	// Rank remaining entities by similarity
	var candidates []SemanticCandidate
	for _, entity := range allEntities {
		if anchorSet[entity.EntityKey] {
			continue // Skip anchors (already included)
		}

		// Get entity content for embedding
		artifact, err := h.engine.vault.Get(entity.ArtifactHash)
		if err != nil || artifact == nil {
			continue
		}

		// Embed entity content
		entityEmbedding, err := h.embedder.Embed(artifact.Content)
		if err != nil {
			continue
		}

		// Compute similarity
		score := h.embedder.CosineSimilarity(queryEmbedding, entityEmbedding)

		// Filter by threshold
		if score > h.threshold {
			candidates = append(candidates, SemanticCandidate{
				EntityKey: entity.EntityKey,
				Score:     score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if h.logger != nil {
		h.logger.Debug("Ranked %d semantic candidates (threshold: %.2f)", len(candidates), h.threshold)
		if len(candidates) > 0 {
			h.logger.Debug("  Top score: %.2f", candidates[0].Score)
		}
	}

	return candidates
}

// BuildHydrationPlan merges anchors and semantic candidates with budget
// Phase 3 of hybrid retrieval: budget-constrained merging
func (h *HybridEngine) BuildHydrationPlan(query string, episodeID string, budget HydrationBudget) (HydrationPlan, error) {
	plan := HydrationPlan{
		Entities: []HydrationEntity{},
		Budget:   budget,
	}

	// Phase 1: Extract anchors (always included)
	anchors := h.ExtractAnchors(query, episodeID)

	// Add anchors to plan (priority order)
	// Sort anchors by priority descending
	sort.Slice(anchors, func(i, j int) bool {
		return anchors[i].Priority > anchors[j].Priority
	})

	for _, anchor := range anchors {
		entity, err := h.engine.state.Get(anchor.EntityKey)
		if err != nil || entity == nil {
			continue
		}

		artifact, err := h.engine.vault.Get(entity.ArtifactHash)
		if err != nil || artifact == nil {
			continue
		}

		// Estimate tokens
		block := HydrationBlock{
			EntityKey:    entity.EntityKey,
			ArtifactHash: entity.ArtifactHash,
			Content:      artifact.Content,
			Filepath:     entity.Filepath,
			Symbol:       entity.Symbol,
		}
		tokenCount := EstimateBlockTokens(block)

		// Check budget
		if budget.MaxTokens > 0 && plan.Budget.UsedTokens+tokenCount > budget.MaxTokens {
			if h.logger != nil {
				h.logger.Warn("Anchor dropped due to budget: %s (reason: %s)", anchor.EntityKey, anchor.Reason)
			}
			continue
		}

		// Add to plan
		plan.Entities = append(plan.Entities, HydrationEntity{
			EntityKey:  anchor.EntityKey,
			Reason:     fmt.Sprintf("anchor:%s", anchor.Reason),
			Priority:   anchor.Priority,
			TokenCount: tokenCount,
		})
		plan.Budget.UsedTokens += tokenCount
		plan.Budget.UsedEntities++
	}

	// Phase 2: Semantic expansion (budget-constrained)
	candidates := h.RankSemantics(query, anchors)

	for _, candidate := range candidates {
		// Check entity limit
		if budget.MaxEntities > 0 && plan.Budget.UsedEntities >= budget.MaxEntities {
			break
		}

		entity, err := h.engine.state.Get(candidate.EntityKey)
		if err != nil || entity == nil {
			continue
		}

		artifact, err := h.engine.vault.Get(entity.ArtifactHash)
		if err != nil || artifact == nil {
			continue
		}

		// Estimate tokens
		block := HydrationBlock{
			EntityKey:    entity.EntityKey,
			ArtifactHash: entity.ArtifactHash,
			Content:      artifact.Content,
			Filepath:     entity.Filepath,
			Symbol:       entity.Symbol,
		}
		tokenCount := EstimateBlockTokens(block)

		// Check token budget
		if budget.MaxTokens > 0 && plan.Budget.UsedTokens+tokenCount > budget.MaxTokens {
			break // Token budget exhausted
		}

		// Add to plan
		plan.Entities = append(plan.Entities, HydrationEntity{
			EntityKey:  candidate.EntityKey,
			Reason:     fmt.Sprintf("semantic:%.2f", candidate.Score),
			Priority:   int(candidate.Score * 100),
			TokenCount: tokenCount,
		})
		plan.Budget.UsedTokens += tokenCount
		plan.Budget.UsedEntities++
	}

	if h.logger != nil {
		h.logger.Info("Hydration plan for episode %s", episodeID)
		h.logger.Info("  Query: %s", truncate(query, 100))
		h.logger.Info("  Anchors: %d", len(anchors))
		h.logger.Info("  Semantic candidates: %d", len(candidates))
		h.logger.Info("  Final entities: %d (tokens: %d/%d)", len(plan.Entities), plan.Budget.UsedTokens, budget.MaxTokens)
	}

	return plan, nil
}

// ExecutePlan hydrates entities according to the plan
func (h *HybridEngine) ExecutePlan(plan HydrationPlan, episodeID string) (string, []string, error) {
	// Extract entity keys in priority order
	var entityKeys []string
	for _, e := range plan.Entities {
		entityKeys = append(entityKeys, e.EntityKey)
	}

	// Record hydration tracking
	if h.engine.tracker != nil && episodeID != "" {
		if err := h.engine.tracker.RecordHydration(episodeID, entityKeys); err != nil {
			// Log error but don't fail
			if h.logger != nil {
				h.logger.Warn("Failed to record hydration tracking: %v", err)
			}
		}
	}

	// Build hydration blocks
	var blocks []HydrationBlock
	for _, e := range plan.Entities {
		entity, err := h.engine.state.Get(e.EntityKey)
		if err != nil || entity == nil {
			continue
		}

		artifact, err := h.engine.vault.Get(entity.ArtifactHash)
		if err != nil || artifact == nil {
			continue
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
			TokenCount:   e.TokenCount,
		})
	}

	// Format hydration
	content := h.engine.formatHydration(blocks)

	return content, entityKeys, nil
}

// isCommonWord filters out common English words from symbol detection
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "from": true, "by": true, "as": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "should": true, "could": true, "can": true, "may": true,
		"this": true, "that": true, "these": true, "those": true, "it": true, "its": true,
		"we": true, "you": true, "they": true, "them": true, "their": true, "our": true,
		"file": true, "function": true, "code": true, "add": true, "fix": true, "update": true,
	}
	return commonWords[strings.ToLower(word)]
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
