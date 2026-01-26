package recall

import (
	"fmt"
	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/evidence"
	"github.com/a-marczewski/tinymem/internal/memory"
	"go.uber.org/zap"
	"sort"
	"strings"
	"unicode/utf8"
)

// Engine handles memory recall operations
type Engine struct {
	memoryService   *memory.Service
	evidenceService *evidence.Service
	config          *config.Config
	logger          *zap.Logger
}

// NewEngine creates a new recall engine
func NewEngine(memoryService *memory.Service, evidenceService *evidence.Service, cfg *config.Config, logger *zap.Logger) *Engine {
	return &Engine{
		memoryService:   memoryService,
		evidenceService: evidenceService,
		config:          cfg,
		logger:          logger,
	}
}

// RecallOptions specifies options for the recall operation
type RecallOptions struct {
	ProjectID         string
	Query             string
	MaxItems          int
	MaxTokens         int
	Types             []memory.Type
	ExcludeSuperseded bool
}

// RecallResult represents a recalled memory with relevance score
type RecallResult struct {
	Memory *memory.Memory
	Score  float64
	Tokens int
}

// Recaller defines the interface for memory recall engines.
type Recaller interface {
	Recall(options RecallOptions) ([]RecallResult, error)
}

// Recall performs memory recall based on the query
func (e *Engine) Recall(options RecallOptions) ([]RecallResult, error) {
	// If no query is provided, return recent memories
	if options.Query == "" {
		memories, err := e.memoryService.GetAllMemories(options.ProjectID)
		if err != nil {
			return nil, err
		}

		results := make([]RecallResult, 0, len(memories))
		for _, mem := range memories {
			tokens := estimateTokenCount(mem.Summary + " " + mem.Detail)

			// Skip if over token limit
			if options.MaxTokens > 0 && tokens > options.MaxTokens {
				continue
			}

			// Check if memory type is in allowed types
			if len(options.Types) > 0 {
				allowed := false
				for _, allowedType := range options.Types {
					if mem.Type == allowedType {
						allowed = true
						break
					}
				}
				if !allowed {
					continue
				}
			}

			results = append(results, RecallResult{
				Memory: mem,
				Score:  1.0, // Default score for recent items
				Tokens: tokens,
			})
		}

		// Sort by date descending with stable ID tiebreaker
		sort.Slice(results, func(i, j int) bool {
			if results[i].Memory.CreatedAt.Equal(results[j].Memory.CreatedAt) {
				return results[i].Memory.ID < results[j].Memory.ID
			}
			return results[i].Memory.CreatedAt.After(results[j].Memory.CreatedAt)
		})

		finalResults := e.applyLimits(results, options)

		// Log recall metrics if enabled
		if e.config.MetricsEnabled {
			totalTokens := 0
			for _, result := range finalResults {
				totalTokens += result.Tokens
			}

			var memoryIDs []string
			for _, result := range finalResults {
				memoryIDs = append(memoryIDs, fmt.Sprintf("%d(%s)", result.Memory.ID, result.Memory.Type))
			}

			e.logger.Info("Recall metrics",
				zap.String("project_id", options.ProjectID),
				zap.String("query", options.Query),
				zap.Int("total_memories", len(finalResults)),
				zap.Int("total_tokens", totalTokens),
				zap.Strings("memory_ids_and_types", memoryIDs),
			)
		}

		return finalResults, nil
	}

	// Perform FTS search
	memories, err := e.memoryService.SearchMemories(options.ProjectID, options.Query, 100) // Get more than needed for filtering
	if err != nil {
		return nil, err
	}

	// Filter by type if specified
	if len(options.Types) > 0 {
		filtered := make([]*memory.Memory, 0)
		for _, mem := range memories {
			for _, allowedType := range options.Types {
				if mem.Type == allowedType {
					filtered = append(filtered, mem)
					break
				}
			}
		}
		memories = filtered
	}

	// Calculate relevance scores and check validation
	results := make([]RecallResult, 0, len(memories))
	for _, mem := range memories {
		// Check if memory is validated (especially for facts)
		isValidated, err := e.evidenceService.IsMemoryValidated(mem)
		if err != nil {
			// Log error but continue
			continue
		}

		// Skip unvalidated facts
		if mem.Type == memory.Fact && !isValidated {
			continue
		}

		// Calculate token count
		tokens := estimateTokenCount(mem.Summary + " " + mem.Detail)

		// Calculate relevance score based on query match
		score := calculateRelevanceScore(mem, options.Query)

		results = append(results, RecallResult{
			Memory: mem,
			Score:  score,
			Tokens: tokens,
		})

	}

	// Sort by score descending with stable ID tiebreaker
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Memory.ID < results[j].Memory.ID
		}
		return results[i].Score > results[j].Score
	})

	finalResults := e.applyLimits(results, options)

	// Log recall metrics if enabled
	if e.config.MetricsEnabled {
		totalTokens := 0
		for _, result := range finalResults {
			totalTokens += result.Tokens
		}

		var memoryIDs []string
		for _, result := range finalResults {
			memoryIDs = append(memoryIDs, fmt.Sprintf("%d(%s)", result.Memory.ID, result.Memory.Type))
		}

		e.logger.Info("Recall metrics",
			zap.String("project_id", options.ProjectID),
			zap.String("query", options.Query),
			zap.Int("total_memories", len(finalResults)),
			zap.Int("total_tokens", totalTokens),
			zap.Strings("memory_ids_and_types", memoryIDs),
		)
	}

	return finalResults, nil
}

func (e *Engine) applyLimits(results []RecallResult, options RecallOptions) []RecallResult {
	if len(results) == 0 {
		return results
	}

	// Separate results by recall tier
	var alwaysResults, contextualResults, opportunisticResults []RecallResult

	for _, result := range results {
		switch result.Memory.RecallTier {
		case memory.Always:
			alwaysResults = append(alwaysResults, result)
		case memory.Contextual:
			contextualResults = append(contextualResults, result)
		case memory.Opportunistic:
			opportunisticResults = append(opportunisticResults, result)
		default:
			// Default to opportunistic for unknown tiers
			opportunisticResults = append(opportunisticResults, result)
		}
	}

	// Log tier counts if metrics are enabled
	if e.config.MetricsEnabled {
		e.logger.Info("Recall tier breakdown",
			zap.String("project_id", options.ProjectID),
			zap.String("query", options.Query),
			zap.Int("always_count", len(alwaysResults)),
			zap.Int("contextual_count", len(contextualResults)),
			zap.Int("opportunistic_count", len(opportunisticResults)),
		)
	}

	// Process results in tier order: Always -> Contextual -> Opportunistic
	limited := make([]RecallResult, 0, len(results))
	totalTokens := 0

	// Add always results first, prioritizing verified > asserted > tentative within each tier
	alwaysResults = sortByTruthState(alwaysResults)
	for _, result := range alwaysResults {
		if options.MaxTokens > 0 && totalTokens+result.Tokens > options.MaxTokens {
			continue
		}
		limited = append(limited, result)
		totalTokens += result.Tokens
		if options.MaxItems > 0 && len(limited) >= options.MaxItems {
			return limited
		}
	}

	// Add contextual results next (until token budget is exhausted), prioritizing verified > asserted > tentative
	contextualResults = sortByTruthState(contextualResults)
	for _, result := range contextualResults {
		if options.MaxTokens > 0 && totalTokens+result.Tokens > options.MaxTokens {
			continue
		}
		limited = append(limited, result)
		totalTokens += result.Tokens
		if options.MaxItems > 0 && len(limited) >= options.MaxItems {
			return limited
		}
	}

	// Only add opportunistic results if there's still space and budget, prioritizing verified > asserted > tentative
	opportunisticResults = sortByTruthState(opportunisticResults)
	for _, result := range opportunisticResults {
		if options.MaxTokens > 0 && totalTokens+result.Tokens > options.MaxTokens {
			continue
		}
		limited = append(limited, result)
		totalTokens += result.Tokens
		if options.MaxItems > 0 && len(limited) >= options.MaxItems {
			break
		}
	}

	// Apply truth-state-based filtering if token or item limits are tight
	// This trims tentative items first when budget is constrained
	if options.MaxTokens > 0 || options.MaxItems > 0 {
		limited = e.trimByTruthState(limited, options)
	}

	return limited
}

// sortByTruthState sorts results by truth state: Verified > Asserted > Tentative
func sortByTruthState(results []RecallResult) []RecallResult {
	// Create a copy to avoid modifying the original slice
	sorted := make([]RecallResult, len(results))
	copy(sorted, results)

	// Sort by truth state priority
	sort.Slice(sorted, func(i, j int) bool {
		// Define priority: Verified (highest) > Asserted > Tentative (lowest)
		priorityI := getTruthStatePriority(sorted[i].Memory.TruthState)
		priorityJ := getTruthStatePriority(sorted[j].Memory.TruthState)

		// Higher priority comes first
		if priorityI != priorityJ {
			return priorityI > priorityJ
		}

		// If priorities are equal, maintain original order (stable sort)
		return sorted[i].Memory.ID < sorted[j].Memory.ID
	})

	return sorted
}

// getTruthStatePriority returns a numeric priority for truth states
func getTruthStatePriority(state memory.TruthState) int {
	switch state {
	case memory.Verified:
		return 3
	case memory.Asserted:
		return 2
	case memory.Tentative:
		return 1
	default:
		return 0 // Lowest priority for unknown states
	}
}

// trimByTruthState removes lower-priority items when limits are tight, preferring verified > asserted > tentative
func (e *Engine) trimByTruthState(results []RecallResult, options RecallOptions) []RecallResult {
	if len(results) == 0 {
		return results
	}

	// If we're already within limits, no need to trim
	totalTokens := 0
	for _, result := range results {
		totalTokens += result.Tokens
	}

	if options.MaxItems > 0 && len(results) <= options.MaxItems && options.MaxTokens > 0 && totalTokens <= options.MaxTokens {
		return results
	}

	// Separate results by truth state
	var verifiedResults, assertedResults, tentativeResults []RecallResult

	for _, result := range results {
		switch result.Memory.TruthState {
		case memory.Verified:
			verifiedResults = append(verifiedResults, result)
		case memory.Asserted:
			assertedResults = append(assertedResults, result)
		case memory.Tentative:
			tentativeResults = append(tentativeResults, result)
		default:
			// Treat unknown states as tentative
			tentativeResults = append(tentativeResults, result)
		}
	}

	// Build final results by prioritizing verified, then asserted, then tentative
	finalResults := make([]RecallResult, 0, len(results))
	totalTokens = 0

	// Add verified results first
	for _, result := range verifiedResults {
		if options.MaxTokens > 0 && totalTokens+result.Tokens > options.MaxTokens {
			continue
		}
		finalResults = append(finalResults, result)
		totalTokens += result.Tokens
		if options.MaxItems > 0 && len(finalResults) >= options.MaxItems {
			return finalResults
		}
	}

	// Add asserted results next
	for _, result := range assertedResults {
		if options.MaxTokens > 0 && totalTokens+result.Tokens > options.MaxTokens {
			continue
		}
		finalResults = append(finalResults, result)
		totalTokens += result.Tokens
		if options.MaxItems > 0 && len(finalResults) >= options.MaxItems {
			return finalResults
		}
	}

	// Add tentative results last (and only if there's still room)
	for _, result := range tentativeResults {
		if options.MaxTokens > 0 && totalTokens+result.Tokens > options.MaxTokens {
			continue
		}
		finalResults = append(finalResults, result)
		totalTokens += result.Tokens
		if options.MaxItems > 0 && len(finalResults) >= options.MaxItems {
			break
		}
	}

	return finalResults
}

// calculateRelevanceScore calculates a relevance score based on how well the memory matches the query
func calculateRelevanceScore(mem *memory.Memory, query string) float64 {
	queryWords := strings.Fields(strings.ToLower(query))

	if len(queryWords) == 0 {
		return 1.0 // Default score if no query
	}

	score := 0.0

	// Score based on matches in summary (higher weight)
	summaryLower := strings.ToLower(mem.Summary)
	for _, word := range queryWords {
		if strings.Contains(summaryLower, word) {
			score += 2.0 // Higher weight for summary matches
		}
	}

	// Score based on matches in detail (lower weight)
	if mem.Detail != "" {
		detailLower := strings.ToLower(mem.Detail)
		for _, word := range queryWords {
			if strings.Contains(detailLower, word) {
				score += 1.0 // Lower weight for detail matches
			}
		}
	}

	// Boost constraints and decisions since they're often important
	if mem.Type == memory.Constraint || mem.Type == memory.Decision {
		score *= 1.5
	}

	// Boost verified memories
	if mem.TruthState == memory.Verified {
		score *= 1.5
	} else if mem.TruthState == memory.Asserted {
		score *= 1.2
	}

	// Normalize score based on query length to prevent longer queries from getting unfairly high scores
	if len(queryWords) > 0 {
		score = score / float64(len(queryWords))
	}

	return score
}

// estimateTokenCount provides a conservative estimate of token count
// This overestimates to ensure we stay within budget
func estimateTokenCount(text string) int {
	if text == "" {
		return 0
	}

	// Conservative approach: count characters and divide by a small number
	// This ensures we never underestimate
	charCount := utf8.RuneCountInString(text)

	// Divide by 3 to overestimate (typical ratio is ~4 chars per token)
	// This is a conservative approach to ensure we stay within budget
	conservativeEstimate := charCount / 3

	// Also count words as a secondary measure and take the higher estimate
	words := strings.Fields(text)
	wordBasedEstimate := len(words) * 2 // Multiply by 2 to be conservative

	// Return the higher of the two estimates to ensure safety
	if conservativeEstimate > wordBasedEstimate {
		return conservativeEstimate
	}
	return wordBasedEstimate
}
