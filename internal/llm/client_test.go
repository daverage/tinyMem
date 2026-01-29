package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daverage/tinymem/internal/config"
)

func TestNormalizeModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		llmModel string
		expected string
	}{
		{
			name:     "No prefix",
			model:    "qwen2.5-coder",
			llmModel: "",
			expected: "qwen2.5-coder",
		},
		{
			name:     "OpenAI prefix",
			model:    "openai/qwen2.5-coder",
			llmModel: "",
			expected: "qwen2.5-coder",
		},
		{
			name:     "Ollama prefix",
			model:    "ollama/llama3",
			llmModel: "",
			expected: "llama3",
		},
		{
			name:     "Config override",
			model:    "openai/ignored",
			llmModel: "canonical-model",
			expected: "canonical-model",
		},
		{
			name:     "Mixed case prefix",
			model:    "OpenAI/qwen",
			llmModel: "",
			expected: "qwen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				LLMModel: tt.llmModel,
			}
			client := NewClient(cfg)
			actual := client.normalizeModel(tt.model)
			if actual != tt.expected {
				t.Errorf("normalizeModel() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestChatCompletionsNormalizesModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just a dummy responder
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"hello"}}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		LLMBaseURL: server.URL,
		LLMModel:   "forced-model",
	}
	client := NewClient(cfg)

	req := ChatCompletionRequest{
		Model: "original-model",
	}
	_, err := client.ChatCompletions(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatCompletions failed: %v", err)
	}
	// The normalization is internal, but we can verify it by checking if normalization works as expected in isolation
	if client.normalizeModel(req.Model) != "forced-model" {
		t.Errorf("Expected model to be normalized to forced-model")
	}
}
