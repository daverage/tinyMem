package embedding

// Embedder is the common interface for all embedding providers
type Embedder interface {
	// GenerateEmbedding generates an embedding vector for the given text
	GenerateEmbedding(text string) ([]float32, error)
}
