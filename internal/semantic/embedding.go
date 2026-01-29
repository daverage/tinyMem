package semantic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/daverage/tinymem/internal/config"
)

// EmbeddingClient handles communication with embedding services
type EmbeddingClient struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

// EmbeddingRequest represents a request to generate embeddings
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents the response from an embedding service
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// NewEmbeddingClient creates a new embedding client
func NewEmbeddingClient(cfg *config.Config) *EmbeddingClient {
	baseURL := cfg.EmbeddingBaseURL
	if baseURL == "" {
		baseURL = config.DefaultLLMBaseURL
	}
	return &EmbeddingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		model: cfg.EmbeddingModel,
	}
}

// GenerateEmbedding generates embeddings for the given text
func (c *EmbeddingClient) GenerateEmbedding(text string) ([]float32, error) {
	req := EmbeddingRequest{
		Model: c.model,
		Input: []string{text},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(
		embeddingURL(c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, err
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddingResp.Data[0].Embedding, nil
}

func embeddingURL(baseURL string) string {
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/embeddings"
	}
	if strings.HasSuffix(baseURL, "/api") {
		return baseURL + "/embeddings"
	}
	return baseURL + "/v1/embeddings"
}

// CosineSimilarity calculates cosine similarity between two vectors
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
