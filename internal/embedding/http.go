package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/daverage/tinymem/internal/config"
)

// HTTPEmbedder handles communication with remote embedding services via HTTP
type HTTPEmbedder struct {
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

// NewHTTPEmbedder creates a new HTTP-based embedding client
func NewHTTPEmbedder(cfg *config.Config) *HTTPEmbedder {
	baseURL := cfg.EmbeddingBaseURL
	if baseURL == "" {
		baseURL = config.DefaultLLMBaseURL
	}
	return &HTTPEmbedder{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		model: cfg.EmbeddingModel,
	}
}

// GenerateEmbedding generates embeddings for the given text
func (c *HTTPEmbedder) GenerateEmbedding(text string) ([]float32, error) {
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
