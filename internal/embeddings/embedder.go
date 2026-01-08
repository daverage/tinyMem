package embeddings

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// Embedder interface for embedding providers
type Embedder interface {
	Embed(text string) ([]float32, error)
	CosineSimilarity(a, b []float32) float64
}

// SimpleEmbedder is a deterministic, hash-based embedding provider
// This is a placeholder implementation for testing/development
// In production, use OpenAI embeddings or a local model
type SimpleEmbedder struct {
	dimensions int
}

// NewSimpleEmbedder creates a hash-based embedder
// This generates deterministic embeddings based on text content
// NOT suitable for production - use for testing only
func NewSimpleEmbedder(dimensions int) *SimpleEmbedder {
	if dimensions <= 0 {
		dimensions = 384 // Default dimension
	}
	return &SimpleEmbedder{
		dimensions: dimensions,
	}
}

// Embed generates a deterministic embedding from text
// Uses SHA-256 hashing and character frequency analysis
func (e *SimpleEmbedder) Embed(text string) ([]float32, error) {
	if text == "" {
		return make([]float32, e.dimensions), nil
	}

	embedding := make([]float32, e.dimensions)

	// Component 1: Hash-based features (first half of dimensions)
	hash := sha256.Sum256([]byte(text))
	hashDim := e.dimensions / 2
	for i := 0; i < hashDim; i++ {
		// Use hash bytes to generate pseudo-random but deterministic values
		byteIdx := i % len(hash)
		value := float32(hash[byteIdx]) / 255.0 // Normalize to [0, 1]
		embedding[i] = value
	}

	// Component 2: Character frequency features (second half)
	charFreq := make(map[rune]int)
	totalChars := 0
	for _, char := range text {
		charFreq[char]++
		totalChars++
	}

	// Generate features from character frequencies
	freqDim := e.dimensions - hashDim
	idx := hashDim
	for char, freq := range charFreq {
		if idx >= e.dimensions {
			break
		}
		// Map character frequency to a feature value
		featureValue := float32(freq) / float32(totalChars)
		// Add character code influence
		charInfluence := float32(int(char)%256) / 255.0
		embedding[idx] = featureValue*0.7 + charInfluence*0.3
		idx++
	}

	// Normalize the embedding to unit length
	embedding = normalize(embedding)

	return embedding, nil
}

// CosineSimilarity computes cosine similarity between two embeddings
func (e *SimpleEmbedder) CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	dotProduct := float64(0)
	normA := float64(0)
	normB := float64(0)

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	return similarity
}

// normalize normalizes a vector to unit length
func normalize(vec []float32) []float32 {
	norm := float32(0)
	for _, v := range vec {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm == 0 {
		return vec
	}

	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}
	return result
}

// OpenAIEmbedder uses OpenAI's embedding API
// Requires OPENAI_API_KEY environment variable
type OpenAIEmbedder struct {
	apiKey string
	model  string // e.g., "text-embedding-3-small"
	cache  map[string][]float32 // Simple in-memory cache
}

// NewOpenAIEmbedder creates an OpenAI embedder
func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIEmbedder{
		apiKey: apiKey,
		model:  model,
		cache:  make(map[string][]float32),
	}
}

// Embed calls OpenAI's embedding API
func (e *OpenAIEmbedder) Embed(text string) ([]float32, error) {
	// Check cache
	if cached, ok := e.cache[text]; ok {
		return cached, nil
	}

	// TODO: Implement actual OpenAI API call
	// For now, return error indicating not implemented
	return nil, fmt.Errorf("OpenAI embeddings not yet implemented - use SimpleEmbedder for testing")
}

// CosineSimilarity computes cosine similarity
func (e *OpenAIEmbedder) CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	dotProduct := float64(0)
	normA := float64(0)
	normB := float64(0)

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	return similarity
}

// GetEmbedder creates an embedder based on provider name
func GetEmbedder(provider, model, apiKey string) (Embedder, error) {
	switch strings.ToLower(provider) {
	case "simple", "local", "none", "":
		return NewSimpleEmbedder(384), nil
	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key required")
		}
		return NewOpenAIEmbedder(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unknown embedding provider: %s", provider)
	}
}
