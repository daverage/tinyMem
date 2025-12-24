package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andrzejmarczewski/tslp/internal/audit"
	"github.com/andrzejmarczewski/tslp/internal/hydration"
	"github.com/andrzejmarczewski/tslp/internal/llm"
	"github.com/andrzejmarczewski/tslp/internal/logging"
	"github.com/andrzejmarczewski/tslp/internal/runtime"
	"github.com/andrzejmarczewski/tslp/internal/vault"
)

// Server implements the OpenAI-compatible proxy API
// Per spec: local HTTP proxy, streaming responses, async background processing
type Server struct {
	runtime    *runtime.Runtime
	llmClient  *llm.Client
	hydrator   *hydration.Engine
	auditor    *audit.Auditor
	logger     *logging.Logger
	httpServer *http.Server
}

// NewServer creates a new API server
func NewServer(rt *runtime.Runtime, llmClient *llm.Client, hydrator *hydration.Engine, auditor *audit.Auditor, logger *logging.Logger, listenAddr string) *Server {
	s := &Server{
		runtime:   rt,
		llmClient: llmClient,
		hydrator:  hydrator,
		auditor:   auditor,
		logger:    logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // Long timeout for streaming
	}

	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting TSLP proxy server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down proxy server")
	return s.httpServer.Shutdown(ctx)
}

// handleHealth responds to health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleChatCompletions handles OpenAI-compatible chat completion requests
// Per spec section 8: pre-flight hydration, streaming responses
func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	s.logger.ProxyRequest(r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req llm.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Failed to decode request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create episode
	episodeID, err := s.runtime.GetLedger().CreateEpisode(nil, nil, nil)
	if err != nil {
		s.logger.Error("Failed to create episode: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	s.logger.EpisodeCreated(episodeID)

	// Extract user prompt (last user message)
	var userPrompt string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userPrompt = req.Messages[i].Content
			break
		}
	}

	// Store user prompt in vault
	userPromptHash, err := s.runtime.GetVault().Store(userPrompt, vault.ContentTypeCode, nil)
	if err != nil {
		s.logger.Error("Failed to store user prompt: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Pre-flight: JIT Hydration
	// Per spec section 8.1: scan state map, retrieve authoritative artifacts, inject
	hydrationContent, err := s.hydrator.Hydrate()
	if err != nil {
		s.logger.Error("Hydration failed: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Inject hydration content before user prompt
	if hydrationContent != "" {
		s.logger.Debug("Hydrating %d bytes of state", len(hydrationContent))

		// Insert hydration as a system message before the user's last message
		messages := make([]llm.Message, 0, len(req.Messages)+1)
		for i, msg := range req.Messages {
			if i == len(req.Messages)-1 && msg.Role == "user" {
				// Insert hydration before last user message
				messages = append(messages, llm.Message{
					Role:    "system",
					Content: hydrationContent,
				})
			}
			messages = append(messages, msg)
		}
		req.Messages = messages
	}

	// Handle streaming vs non-streaming
	if req.Stream {
		s.handleStreamingCompletion(w, r, &req, episodeID, userPromptHash)
	} else {
		s.handleNonStreamingCompletion(w, r, &req, episodeID, userPromptHash)
	}
}

// handleStreamingCompletion handles streaming chat completions
func (s *Server) handleStreamingCompletion(w http.ResponseWriter, r *http.Request, req *llm.ChatRequest, episodeID, userPromptHash string) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream from LLM
	chunkChan, err := s.llmClient.StreamChat(r.Context(), req.Messages)
	if err != nil {
		s.logger.Error("Failed to start streaming: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Collect full response for post-processing
	var fullResponse strings.Builder

	for chunk := range chunkChan {
		if chunk.Error != nil {
			s.logger.Error("Streaming error: %v", chunk.Error)
			break
		}

		if chunk.Done {
			// Send [DONE] marker
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		// Forward chunk to client
		chunkJSON, err := json.Marshal(chunk.Response)
		if err != nil {
			s.logger.Error("Failed to marshal chunk: %v", err)
			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", chunkJSON)
		flusher.Flush()

		// Collect response content
		if chunk.Response != nil && len(chunk.Response.Choices) > 0 {
			if chunk.Response.Choices[0].Delta != nil {
				fullResponse.WriteString(chunk.Response.Choices[0].Delta.Content)
			}
		}
	}

	// Post-processing: store response and trigger shadow audit
	s.postProcessResponse(episodeID, userPromptHash, fullResponse.String())
}

// handleNonStreamingCompletion handles non-streaming chat completions
func (s *Server) handleNonStreamingCompletion(w http.ResponseWriter, r *http.Request, req *llm.ChatRequest, episodeID, userPromptHash string) {
	response, err := s.llmClient.Chat(r.Context(), req.Messages)
	if err != nil {
		s.logger.Error("LLM call failed: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Extract response content
	var responseContent string
	if len(response.Choices) > 0 {
		responseContent = response.Choices[0].Message.Content
	}

	// Post-processing: store response and trigger shadow audit
	s.postProcessResponse(episodeID, userPromptHash, responseContent)

	// Send response to client
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode response: %v", err)
	}
}

// postProcessResponse handles artifact storage, entity resolution, and shadow audit
// Per spec: async shadow audit, non-blocking
func (s *Server) postProcessResponse(episodeID, userPromptHash, responseContent string) {
	if responseContent == "" {
		return
	}

	// Store response in vault
	responseHash, err := s.runtime.GetVault().Store(responseContent, vault.ContentTypeCode, nil)
	if err != nil {
		s.logger.Error("Failed to store response: %v", err)
		return
	}

	// Update episode with response hash
	// Note: We can't update the episode after creation with current schema
	// This would require an UPDATE statement in ledger package
	// For now, we'll log it
	s.logger.Debug("Response stored: episode=%s hash=%s", episodeID, responseHash)

	// Process artifact (resolve entity, evaluate promotion)
	result, err := s.runtime.ProcessArtifact(responseContent, vault.ContentTypeCode, episodeID, false)
	if err != nil {
		s.logger.Error("Failed to process artifact: %v", err)
		return
	}

	s.logger.PromotionResult(responseHash, "unknown", result.Promoted, result.Reason)

	// Trigger shadow audit (async, non-blocking)
	s.auditor.AuditAsync(episodeID, responseHash)
}

// extractCodeBlocks parses code blocks from markdown-formatted response
// Used to extract actual code artifacts from LLM responses
func extractCodeBlocks(content string) []string {
	var blocks []string
	lines := strings.Split(content, "\n")

	var currentBlock strings.Builder
	inBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inBlock {
				// End of block
				blocks = append(blocks, currentBlock.String())
				currentBlock.Reset()
				inBlock = false
			} else {
				// Start of block
				inBlock = true
			}
		} else if inBlock {
			currentBlock.WriteString(line)
			currentBlock.WriteString("\n")
		}
	}

	return blocks
}
