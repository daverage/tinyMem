//go:build !embeddings

package embedding

import "errors"

// LocalEmbedder is a stub for builds without embedded model support
type LocalEmbedder struct{}

// NewLocalEmbedder returns an error when built without embeddings tag
func NewLocalEmbedder() (*LocalEmbedder, error) {
	return nil, errors.New("local embeddings not available (build with -tags embeddings)")
}

// GenerateEmbedding is not available in lightweight builds
func (e *LocalEmbedder) GenerateEmbedding(text string) ([]float32, error) {
	return nil, errors.New("not available")
}

// Close is a no-op for the stub
func (e *LocalEmbedder) Close() error {
	return nil
}

// IsAvailable returns false if built without embeddings tag
func IsAvailable() bool {
	return false
}
