//go:build embeddings

package embedding

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kelindar/search"
)

//go:embed models/nomic-embed-text-v1.5.Q4_K_M.gguf
var embeddedModel []byte

// LocalEmbedder uses an embedded model for local inference
type LocalEmbedder struct {
	vectorizer *search.Vectorizer
	tempFile   string // Path to temporary model file
}

// NewLocalEmbedder creates a new local embedder with the embedded model
func NewLocalEmbedder() (*LocalEmbedder, error) {
	// Write embedded model to temporary file
	// kelindar/search requires a file path, can't load from bytes directly
	tmpFile, err := writeTempModel()
	if err != nil {
		return nil, fmt.Errorf("failed to write temporary model file: %w", err)
	}

	// Initialize vectorizer with the model file
	// Second parameter is GPU layers (0 = CPU only)
	vectorizer, err := search.NewVectorizer(tmpFile, 0)
	if err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return nil, fmt.Errorf("failed to initialize vectorizer: %w", err)
	}

	return &LocalEmbedder{
		vectorizer: vectorizer,
		tempFile:   tmpFile,
	}, nil
}

// GenerateEmbedding generates an embedding vector for the given text
func (e *LocalEmbedder) GenerateEmbedding(text string) ([]float32, error) {
	if e.vectorizer == nil {
		return nil, fmt.Errorf("vectorizer not initialized")
	}

	embedding, err := e.vectorizer.EmbedText(text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return embedding, nil
}

// Close releases resources and cleans up temporary files
func (e *LocalEmbedder) Close() error {
	if e.vectorizer != nil {
		e.vectorizer.Close()
	}
	if e.tempFile != "" {
		return os.Remove(e.tempFile)
	}
	return nil
}

// writeTempModel writes the embedded model bytes to a temporary file
func writeTempModel() (string, error) {
	// Create temp file with specific name pattern
	tmpFile, err := os.CreateTemp("", "tinymem-model-*.gguf")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write embedded model bytes
	if _, err := tmpFile.Write(embeddedModel); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// IsAvailable returns true if built with embeddings tag
func IsAvailable() bool {
	return true
}

// GetModelPath returns the path where models are stored (for reference)
func GetModelPath() string {
	// This is the embedded path - actual runtime uses temp file
	return filepath.Join("internal", "embedding", "models", "nomic-embed-text-v1.5.Q4_K_M.gguf")
}
