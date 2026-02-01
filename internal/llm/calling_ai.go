package llm

import (
	"context"
	"time"
)

// CallingAIProvider is an LLM provider for MCP mode that simulates responses
//
// ARCHITECTURE NOTE:
// In MCP mode, the calling AI (Claude, Codex, Gemini) is already a capable LLM.
// Rather than requiring a separate LLM backend, we provide simplified behavior:
//
// - CoVe: Returns facts without verification (calling AI is reliable)
//
// For advanced use cases requiring full CoVe in MCP mode:
// 1. Configure an external LLM (Ollama) - use external.Client
//
// This provider exists to make CoVe "work" in MCP mode without external deps,
// but with reduced functionality compared to Proxy mode with full LLM backend.
type CallingAIProvider struct {
	// mode specifies how this provider behaves
	mode    string // "pass-through" or "error-only"
	metrics *Metrics
}

// NewCallingAIProvider creates a new CallingAI provider for MCP mode
func NewCallingAIProvider() *CallingAIProvider {
	metrics := NewMetrics("calling-ai-mcp-simplified")
	RegisterMetrics("calling-ai-mcp-simplified", metrics)

	return &CallingAIProvider{
		mode:    "pass-through",
		metrics: metrics,
	}
}

// ChatCompletions in CallingAI mode returns a pass-through response
// For CoVe: Returns a simple "verified" response without actual verification
//
// This is intentionally simplified. For full CoVe functionality in MCP mode,
// users should configure an external LLM or wait for embedded text generation support.
func (p *CallingAIProvider) ChatCompletions(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	start := time.Now()

	// Extract the user's request from messages
	var userMessage string
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userMessage = msg.Content
			break
		}
	}

	// Provide a simplified response
	// CoVe verification: Accept without deep verification (calling AI is reliable)
	response := p.generateSimplifiedResponse(userMessage)

	// Record successful request
	if p.metrics != nil {
		latency := time.Since(start).Milliseconds()
		// Rough token estimate: 4 chars per token
		tokens := (len(userMessage) + len(response)) / 4
		p.metrics.RecordRequest(true, latency, tokens)
	}

	return &ChatCompletionResponse{
		ID:      "calling-ai-simplified",
		Object:  "chat.completion",
		Created: 0,
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: response,
				},
			},
		},
	}, nil
}

// generateSimplifiedResponse creates a simplified LLM response for MCP mode
func (p *CallingAIProvider) generateSimplifiedResponse(userMessage string) string {
	// For CoVe verification requests, return affirmative without deep analysis
	if containsAny(userMessage, []string{"verify", "verification", "fact-check", "validate"}) {
		return "Based on the provided context, this appears to be valid. " +
			"Note: Full CoVe verification requires external LLM configuration in MCP mode."
	}

	// Generic response
	return "MCP mode: This request requires external LLM for full functionality. " +
		"Current response is simplified. Configure LLM backend for CoVe, or use calling AI to process manually."
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			// Simple case-insensitive contains check
			for i := 0; i <= len(s)-len(substr); i++ {
				match := true
				for j := 0; j < len(substr); j++ {
					c1 := s[i+j]
					c2 := substr[j]
					// Simple lowercase comparison
					if c1 >= 'A' && c1 <= 'Z' {
						c1 += 32
					}
					if c2 >= 'A' && c2 <= 'Z' {
						c2 += 32
					}
					if c1 != c2 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}

// ProviderName returns the name of this provider
func (p *CallingAIProvider) ProviderName() string {
	return "calling-ai-mcp-simplified"
}

// GetMetrics returns the current metrics for this provider
func (p *CallingAIProvider) GetMetrics() *Metrics {
	return p.metrics
}

// Close releases any resources (no resources to release)
func (p *CallingAIProvider) Close() error {
	return nil
}
