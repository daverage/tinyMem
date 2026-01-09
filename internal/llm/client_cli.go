package llm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CLIClient wraps CLI-based LLM tools (Claude, Gemini, etc.)
// Provides the same interface as HTTP client but executes shell commands
type CLIClient struct {
	command     string   // e.g., "claude", "gemini", "sgpt"
	args        []string // base arguments (e.g., ["--model", "claude-3-opus"])
	model       string
	contextMode string // "stdin" or "args"
}

// CLIProviderConfig defines how to invoke a specific CLI tool
type CLIProviderConfig struct {
	Command     string   // Base command (e.g., "claude")
	BaseArgs    []string // Arguments before message (e.g., ["--no-cache"])
	Model       string   // Model name
	ContextMode string   // How to pass context: "stdin" or "args"
}

// Predefined CLI provider configurations
var CLIProviders = map[string]CLIProviderConfig{
	"claude": {
		Command:     "claude",
		BaseArgs:    []string{},
		Model:       "claude-3-5-sonnet-20241022",
		ContextMode: "stdin",
	},
	"gemini": {
		Command:     "gemini",
		BaseArgs:    []string{"chat"},
		Model:       "gemini-pro",
		ContextMode: "args",
	},
	"sgpt": {
		Command:     "sgpt",
		BaseArgs:    []string{"--no-cache"},
		Model:       "gpt-4",
		ContextMode: "args",
	},
	"aichat": {
		Command:     "aichat",
		BaseArgs:    []string{},
		Model:       "gpt-4",
		ContextMode: "stdin",
	},
}

// NewCLIClient creates a new CLI-based LLM client
// provider: "claude", "gemini", "sgpt", or custom command
func NewCLIClient(provider, model string) (*CLIClient, error) {
	// Check if it's a known provider
	if config, exists := CLIProviders[provider]; exists {
		// Use predefined config, override model if provided
		if model != "" {
			config.Model = model
		}
		return &CLIClient{
			command:     config.Command,
			args:        config.BaseArgs,
			model:       config.Model,
			contextMode: config.ContextMode,
		}, nil
	}

	// Treat as custom command
	return &CLIClient{
		command:     provider,
		args:        []string{},
		model:       model,
		contextMode: "stdin", // default to stdin for custom commands
	}, nil
}

// Chat sends a chat completion request via CLI
// Implements the same interface as HTTP client
func (c *CLIClient) Chat(ctx context.Context, messages []Message) (*ChatResponse, error) {
	// Format messages into a single prompt
	prompt := c.formatMessages(messages)

	// Build command
	cmdArgs := append([]string{}, c.args...)

	var cmd *exec.Cmd
	if c.contextMode == "stdin" {
		// Pass prompt via stdin
		cmd = exec.CommandContext(ctx, c.command, cmdArgs...)
		cmd.Stdin = strings.NewReader(prompt)
	} else {
		// Pass prompt as argument
		cmdArgs = append(cmdArgs, prompt)
		cmd = exec.CommandContext(ctx, c.command, cmdArgs...)
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("CLI command failed: %w (output: %s)", err, string(output))
	}

	// Parse output into ChatResponse structure
	response := &ChatResponse{
		ID:      generateID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   c.model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: Content{Value: strings.TrimSpace(string(output))},
				},
			},
		},
	}

	return response, nil
}

// StreamChat is not supported for CLI clients (returns non-streaming response)
func (c *CLIClient) StreamChat(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	// CLI tools don't typically support streaming
	// Fall back to non-streaming and emit a single chunk
	chunkChan := make(chan StreamChunk, 1)

	go func() {
		defer close(chunkChan)

		response, err := c.Chat(ctx, messages)
		if err != nil {
			chunkChan <- StreamChunk{Error: err}
			return
		}

		chunkChan <- StreamChunk{Response: response}
		chunkChan <- StreamChunk{Done: true}
	}()

	return chunkChan, nil
}

// GetModel returns the configured model
func (c *CLIClient) GetModel() string {
	return c.model
}

// formatMessages converts message array into a single prompt string
// Optimized for CLI tools that expect simple text input
func (c *CLIClient) formatMessages(messages []Message) string {
	var sb strings.Builder

	for i, msg := range messages {
		// Skip system messages or combine them
		if msg.Role == "system" {
			// Include system messages as context
			sb.WriteString("## Context\n")
			sb.WriteString(msg.Content.GetString())
			sb.WriteString("\n\n")
			continue
		}

		// For user/assistant messages, use simple format
		if msg.Role == "user" {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(msg.Content.GetString())
		} else if msg.Role == "assistant" {
			sb.WriteString("\n\nAssistant: ")
			sb.WriteString(msg.Content.GetString())
		}
	}

	return sb.String()
}

// generateID generates a random ID for the response
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "chatcmpl-" + hex.EncodeToString(b)
}

// IsCLIProvider checks if a provider string is a CLI provider
func IsCLIProvider(provider string) bool {
	// Check known CLI providers
	if _, exists := CLIProviders[provider]; exists {
		return true
	}

	// If it starts with "cli:", treat it as CLI
	return strings.HasPrefix(provider, "cli:")
}

// ParseCLIProvider parses a provider string for CLI mode
// Format: "cli:command" or just "claude", "gemini", etc.
func ParseCLIProvider(provider string) string {
	if strings.HasPrefix(provider, "cli:") {
		return strings.TrimPrefix(provider, "cli:")
	}
	return provider
}
