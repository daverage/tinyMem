package recall

import (
	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/evidence"
	"github.com/a-marczewski/tinymem/internal/memory"
	"sort"
	"strings"
	"unicode/utf8"
)

// Engine handles memory recall operations
type Engine struct {
	memoryService   *memory.Service
	evidenceService *evidence.Service
	config          *config.Config
}

// NewEngine creates a new recall engine
func NewEngine(memoryService *memory.Service, evidenceService *evidence.Service, cfg *config.Config) *Engine {
	return &Engine{
		memoryService:   memoryService,
		evidenceService: evidenceService,
		config:          cfg,
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

		return applyLimits(results, options), nil
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

	return applyLimits(results, options), nil
}

func applyLimits(results []RecallResult, options RecallOptions) []RecallResult {
	if len(results) == 0 {
		return results
	}

	limited := make([]RecallResult, 0, len(results))
	totalTokens := 0

	for _, result := range results {
		if options.MaxTokens > 0 && totalTokens+result.Tokens > options.MaxTokens {
			continue
		}
		limited = append(limited, result)
		totalTokens += result.Tokens
		if options.MaxItems > 0 && len(limited) >= options.MaxItems {
			break
		}
	}

	return limited
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
