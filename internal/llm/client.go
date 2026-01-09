package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ContentPart represents a single content part (for multimodal support)
type ContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// Content represents message content that can be either a string or array of parts
type Content struct {
	Value interface{} // Can be string or []ContentPart
}

// MarshalJSON implements custom JSON marshaling for Content
func (c Content) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Value)
}

// UnmarshalJSON implements custom JSON unmarshaling for Content
func (c *Content) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		c.Value = str
		return nil
	}

	// Try to unmarshal as array of ContentPart
	var parts []ContentPart
	if err := json.Unmarshal(data, &parts); err == nil {
		c.Value = parts
		return nil
	}

	// If both fail, return error
	return fmt.Errorf("content must be string or array of content parts")
}

// GetString returns the content as a string if it's a string, otherwise returns empty string
func (c Content) GetString() string {
	if str, ok := c.Value.(string); ok {
		return str
	}
	return ""
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content Content `json:"content"`
}

// ChatRequest represents an OpenAI-compatible chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse represents an OpenAI-compatible chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage,omitempty"`
}

// Choice represents a single completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message,omitempty"`
	Delta        *Delta  `json:"delta,omitempty"`
	FinishReason *string `json:"finish_reason"`
}

// Delta represents a streaming message delta
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content Content `json:"content,omitempty"`
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Client wraps HTTP client for LLM API calls
// Per spec: streaming responses, OpenAI-compatible
type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
	model      string
}

// NewClient creates a new LLM client
func NewClient(endpoint, apiKey, model string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for streaming
		},
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
	}
}

// Chat sends a non-streaming chat completion request
func (c *Client) Chat(ctx context.Context, messages []Message) (*ChatResponse, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// StreamChat sends a streaming chat completion request
// Returns a channel that receives response chunks
func (c *Client) StreamChat(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	chunkChan := make(chan StreamChunk, 10)

	go c.readStream(ctx, resp.Body, chunkChan)

	return chunkChan, nil
}

// StreamChunk represents a single chunk from a streaming response
type StreamChunk struct {
	Response *ChatResponse
	Error    error
	Done     bool
}

// readStream reads SSE stream and sends chunks to channel
func (c *Client) readStream(ctx context.Context, body io.ReadCloser, chunkChan chan<- StreamChunk) {
	defer close(chunkChan)
	defer body.Close()

	for {
		select {
		case <-ctx.Done():
			chunkChan <- StreamChunk{Error: ctx.Err()}
			return
		default:
		}

		// Read SSE format: "data: {json}\n\n"
		var line []byte
		for {
			b := make([]byte, 1)
			_, err := body.Read(b)
			if err == io.EOF {
				chunkChan <- StreamChunk{Done: true}
				return
			}
			if err != nil {
				chunkChan <- StreamChunk{Error: err}
				return
			}

			line = append(line, b[0])
			if len(line) >= 2 && line[len(line)-2] == '\n' && line[len(line)-1] == '\n' {
				break
			}
		}

		// Parse SSE line
		lineStr := string(line)
		if len(lineStr) < 6 || lineStr[:6] != "data: " {
			continue
		}

		data := lineStr[6:]
		data = data[:len(data)-2] // Remove trailing \n\n

		if data == "[DONE]" {
			chunkChan <- StreamChunk{Done: true}
			return
		}

		var chunk ChatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			chunkChan <- StreamChunk{Error: fmt.Errorf("failed to parse chunk: %w", err)}
			continue
		}

		chunkChan <- StreamChunk{Response: &chunk}
	}
}

// setHeaders sets common request headers
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// GetModel returns the configured model
func (c *Client) GetModel() string {
	return c.model
}
