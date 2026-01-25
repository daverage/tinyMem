package semantic

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/evidence"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/recall"
	"github.com/a-marczewski/tinymem/internal/storage"
)

// SemanticEngine enhances recall with semantic similarity
type SemanticEngine struct {
	db              *storage.DB
	embeddingClient *EmbeddingClient
	memoryService   *memory.Service
	evidenceService *evidence.Service
	config          *config.Config
}

// NewSemanticEngine creates a new semantic recall engine
func NewSemanticEngine(
	db *storage.DB,
	memoryService *memory.Service,
	evidenceService *evidence.Service,
	cfg *config.Config,
) *SemanticEngine {
	return &SemanticEngine{
		db:              db,
		embeddingClient: NewEmbeddingClient(cfg),
		memoryService:   memoryService,
		evidenceService: evidenceService,
		config:          cfg,
	}
}

// SemanticRecall performs semantic recall combined with lexical recall
func (s *SemanticEngine) SemanticRecall(options recall.RecallOptions) ([]recall.RecallResult, error) {
	// First, try to get embeddings for the query
	var queryEmbedding []float32
	if options.Query != "" {
		var err error
		queryEmbedding, err = s.embeddingClient.GenerateEmbedding(options.Query)
		if err != nil {
			// If embedding fails, fall back to lexical search only
			// This ensures the system remains functional even if semantic is unavailable
			return s.fallbackLexicalRecall(options)
		}
	}

	// Get all memories for comparison (in a real system, you'd want to filter first)
	allMemories, err := s.memoryService.GetAllMemories("default_project") // In real impl, get from context
	if err != nil {
		return nil, err
	}

	// Filter by type if specified
	if len(options.Types) > 0 {
		filtered := make([]*memory.Memory, 0)
		for _, mem := range allMemories {
			for _, allowedType := range options.Types {
				if mem.Type == allowedType {
					filtered = append(filtered, mem)
					break
				}
			}
		}
		allMemories = filtered
	}

	// Calculate semantic similarity scores
	semanticScores := make(map[int64]float64)
	for _, mem := range allMemories {
		// Combine summary and detail for embedding
		text := mem.Summary
		if mem.Detail != "" {
			text += " " + mem.Detail
		}

		embedding, err := s.getOrCreateEmbedding(mem.ID, text)
		if err != nil {
			continue // Skip if we can't get embedding
		}

		if queryEmbedding != nil {
			similarity := CosineSimilarity(queryEmbedding, embedding)
			semanticScores[mem.ID] = similarity
		}
	}

	// Get lexical scores using the standard recall engine
	lexicalResults, err := s.fallbackLexicalRecall(recall.RecallOptions{
		Query:             options.Query,
		MaxItems:          100, // Get more for combination
		MaxTokens:         0,   // Handle token budgeting after combination
		Types:             options.Types,
		ExcludeSuperseded: options.ExcludeSuperseded,
	})
	if err != nil {
		return nil, err
	}

	// Combine lexical and semantic scores
	combinedResults := make([]recall.RecallResult, 0, len(lexicalResults))
	for _, result := range lexicalResults {
		// Normalize scores to 0-1 range if needed
		lexicalScore := result.Score
		semanticScore := semanticScores[result.Memory.ID]

		// Combine scores with configurable weights
		// Using equal weights for simplicity; in practice, these could be tuned
		combinedScore := 0.5*normalizeScore(lexicalScore) + 0.5*semanticScore

		combinedResults = append(combinedResults, recall.RecallResult{
			Memory: result.Memory,
			Score:  combinedScore,
			Tokens: result.Tokens,
		})
	}

	// Sort by combined score
	sort.Slice(combinedResults, func(i, j int) bool {
		return combinedResults[i].Score > combinedResults[j].Score
	})

	// Apply token and item limits
	finalResults := make([]recall.RecallResult, 0)
	totalTokens := 0

	for _, result := range combinedResults {
		// Check if memory is validated (especially for facts)
		isValidated, err := s.evidenceService.IsMemoryValidated(result.Memory)
		if err != nil || (result.Memory.Type == memory.Fact && !isValidated) {
			continue
		}

		// Apply token budgeting
		if options.MaxTokens > 0 {
			if totalTokens+result.Tokens > options.MaxTokens {
				break
			}
			totalTokens += result.Tokens
		}

		finalResults = append(finalResults, result)

		// Apply item limit
		if options.MaxItems > 0 && len(finalResults) >= options.MaxItems {
			break
		}
	}

	return finalResults, nil
}

// getOrCreateEmbedding gets an existing embedding or creates a new one
func (s *SemanticEngine) getOrCreateEmbedding(memoryID int64, text string) ([]float32, error) {
	// Try to get from database first
	embedding, err := s.getEmbeddingFromDB(memoryID)
	if err != nil {
		// If not found in DB, generate new embedding
		embedding, err = s.embeddingClient.GenerateEmbedding(text)
		if err != nil {
			return nil, err
		}

		// Store the new embedding in the database
		err = s.storeEmbeddingInDB(memoryID, embedding)
		if err != nil {
			// Log error but don't fail the operation
			// The embedding can still be used even if not stored
		}
	}

	return embedding, nil
}

// getEmbeddingFromDB retrieves an embedding from the database
func (s *SemanticEngine) getEmbeddingFromDB(memoryID int64) ([]float32, error) {
	query := `SELECT embedding_data FROM embeddings WHERE memory_id = ?`
	
	var embeddingJSON string
	err := s.db.GetConnection().QueryRow(query, memoryID).Scan(&embeddingJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("embedding not found")
		}
		return nil, err
	}

	var embedding []float32
	err = json.Unmarshal([]byte(embeddingJSON), &embedding)
	if err != nil {
		return nil, err
	}

	return embedding, nil
}

// storeEmbeddingInDB stores an embedding in the database
func (s *SemanticEngine) storeEmbeddingInDB(memoryID int64, embedding []float32) error {
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return err
	}

	query := `
		INSERT OR REPLACE INTO embeddings (memory_id, embedding_data, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`

	_, err = s.db.GetConnection().Exec(query, memoryID, string(embeddingJSON))
	return err
}

// fallbackLexicalRecall performs lexical recall as a fallback
func (s *SemanticEngine) fallbackLexicalRecall(options recall.RecallOptions) ([]recall.RecallResult, error) {
	// Create a basic recall engine for fallback
	basicRecall := recall.NewEngine(s.memoryService, s.evidenceService, s.config)
	return basicRecall.Recall(options)
}

// normalizeScore normalizes a score to the 0-1 range
func normalizeScore(score float64) float64 {
	if score <= 0 {
		return 0
	}
	// Cap at a reasonable upper bound to keep scores normalized
	maxExpectedScore := 10.0
	if score > maxExpectedScore {
		return 1.0
	}
	return score / maxExpectedScore
}