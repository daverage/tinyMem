package proxy

import (
	"testing"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/llm"
)

func TestInjectAgentContract(t *testing.T) {
	// Setup test data
	largeContract := "**Start of tinyMem Protocol**\nLARGE CONTRACT"
	smallContract := "**Start of tinyMem Protocol**\nSMALL CONTRACT"

	tests := []struct {
		name           string
		configContract string
		initialMsgs    []llm.Message
		expectedCount  int
		expectedFirst  string
		expectError    bool
	}{
		{
			name:           "Contract Already Present",
			configContract: "large",
			initialMsgs: []llm.Message{
				{Role: "system", Content: "Some system msg"},
				{Role: "user", Content: "Hello **Start of tinyMem Protocol**"},
			},
			expectedCount: 2, // No change
			expectedFirst: "Some system msg",
		},
		{
			name:           "Contract Missing (Large)",
			configContract: "large",
			initialMsgs: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
			expectedCount: 2, // injected + original
			expectedFirst: largeContract,
		},
		{
			name:           "Contract Missing (Small)",
			configContract: "small",
			initialMsgs: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
			expectedCount: 2, // injected + original
			expectedFirst: smallContract,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				config: &config.Config{
					AgentContract: tt.configContract,
				},
				agentContractLarge: largeContract,
				agentContractSmall: smallContract,
			}

			req := llm.ChatCompletionRequest{
				Messages: tt.initialMsgs,
			}

			err := s.injectAgentContract(&req)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(req.Messages) != tt.expectedCount {
				t.Errorf("Expected %d messages, got %d", tt.expectedCount, len(req.Messages))
			}

			if len(req.Messages) > 0 {
				if req.Messages[0].Content != tt.expectedFirst {
					t.Errorf("Expected first message to be %q, got %q", tt.expectedFirst, req.Messages[0].Content)
				}
			}
		})
	}
}
