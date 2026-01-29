package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/daverage/tinymem/internal/config"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client handles communication with LLM services
type Client struct {
	baseURL    string
	apiKey     string
	config     *config.Config
	httpClient *http.Client
}

// ChatCompletionRequest represents a request to the chat completion API
type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	// Add other fields as needed
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse represents a response from the chat completion API
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Choice represents a choice in the chat completion response
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason *string `json:"finish_reason,omitempty"`
}

// StreamChunk represents a chunk in a streaming response
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice represents a choice in a streaming response
type StreamChoice struct {
	Index        int     `json:"index"`
	Delta        Message `json:"delta"`
	FinishReason *string `json:"finish_reason,omitempty"`
}

// NewClient creates a new LLM client
func NewClient(cfg *config.Config) *Client {
	baseURL := cfg.LLMBaseURL
	if baseURL == "" {
		baseURL = config.DefaultLLMBaseURL
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  cfg.LLMAPIKey,
		config:  cfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// NewClientWithConfig creates a new LLM client with proper configuration
func NewClientWithConfig(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ChatCompletions sends a chat completion request
func (c *Client) ChatCompletions(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	req.Model = c.normalizeModel(req.Model)
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", chatCompletionURL(c.baseURL), strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// ChatCompletionsRaw sends a chat completion request and returns the raw HTTP response.
func (c *Client) ChatCompletionsRaw(ctx context.Context, req ChatCompletionRequest) (*http.Response, error) {
	req.Model = c.normalizeModel(req.Model)
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", chatCompletionURL(c.baseURL), strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// StreamChatCompletions sends a streaming chat completion request
func (c *Client) StreamChatCompletions(ctx context.Context, req ChatCompletionRequest) (<-chan StreamChunk, <-chan error) {
	req.Stream = true
	req.Model = c.normalizeModel(req.Model)

	jsonData, err := json.Marshal(req)
	if err != nil {
		errChan := make(chan error, 1)
		errChan <- err
		close(errChan)
		emptyChan := make(chan StreamChunk)
		close(emptyChan)
		return emptyChan, errChan
	}

	chunkChan := make(chan StreamChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		request, err := http.NewRequestWithContext(ctx, "POST", chatCompletionURL(c.baseURL), strings.NewReader(string(jsonData)))
		if err != nil {
			errChan <- err
			return
		}

		request.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			request.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
		request.Header.Set("Accept", "text/event-stream")
		request.Header.Set("Cache-Control", "no-cache")
		request.Header.Set("Connection", "keep-alive")

		resp, err := c.httpClient.Do(request)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("stream request failed with status %d: %s", resp.StatusCode, string(body))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				// Skip [DONE] marker
				if data == "[DONE]" {
					break
				}

				var chunk StreamChunk
				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					// Skip invalid JSON lines
					continue
				}

				select {
				case chunkChan <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	return chunkChan, errChan
}

// ConvertStreamToText converts a stream of chunks to text
func (c *Client) ConvertStreamToText(ctx context.Context, chunks <-chan StreamChunk, errors <-chan error) (<-chan string, <-chan error) {
	textChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(textChan)
		defer close(errChan)

		for {
			select {
			case chunk, ok := <-chunks:
				if !ok {
					// Channel closed, stream is complete
					return
				}

				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						select {
						case textChan <- choice.Delta.Content:
						case <-ctx.Done():
							return
						}
					}
				}
			case err, ok := <-errors:
				if ok && err != nil {
					errChan <- err
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return textChan, errChan
}

// normalizeModel ensures the model name is correct for the backend.
// It applies overrides from config and strips common provider prefixes (e.g., "openai/")
// which are often added by clients like LiteLLM/Aider but not recognized by local servers.
func (c *Client) normalizeModel(model string) string {
	// If a global model override is set in config, use it.
	if c.config != nil && c.config.LLMModel != "" {
		return c.config.LLMModel
	}

	// Strip common provider prefixes
	prefixes := []string{"openai/", "ollama/", "huggingface/", "anthropic/", "google/"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(model), prefix) {
			return model[len(prefix):]
		}
	}

	return model
}

func chatCompletionURL(baseURL string) string {
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/chat/completions"
	}
	if strings.HasSuffix(baseURL, "/api") {
		return baseURL + "/chat"
	}
	return baseURL + "/v1/chat/completions"
}
