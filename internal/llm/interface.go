package llm

import "context"

// Provider defines the interface for LLM providers
// Implementations include External (HTTP), CallingAI (MCP mode), and Local (embedded)
type Provider interface {
	// ChatCompletions sends a chat completion request and returns a response
	ChatCompletions(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)

	// ProviderName returns the name of the provider (for logging/debugging)
	ProviderName() string

	// GetMetrics returns the current metrics for this provider
	GetMetrics() *Metrics

	// Close releases any resources held by the provider
	Close() error
}

// Ensure Client implements Provider
var _ Provider = (*Client)(nil)
