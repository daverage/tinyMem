package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/daverage/tinymem/internal/app"
	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/cove"
	"github.com/daverage/tinymem/internal/evidence"
	"github.com/daverage/tinymem/internal/extract"
	"github.com/daverage/tinymem/internal/inject"
	"github.com/daverage/tinymem/internal/llm"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/recall"
	"github.com/daverage/tinymem/internal/semantic"
	"go.uber.org/zap"
)

// ResponseCapture holds response data for extraction
type ResponseCapture struct {
	ResponseText string
	Model        string
	Timestamp    time.Time
}

type recallStatus string

const (
	recallStatusNone     recallStatus = "none"
	recallStatusInjected recallStatus = "injected"
	recallStatusFailed   recallStatus = "failed"
)

// MemoryNotification describes the recall/notification state for a proxied request.
type MemoryNotification struct {
	RecallCount  int
	RecallStatus recallStatus
}

// Server implements the HTTP proxy server
type Server struct {
	app             *app.App // New: Hold the app instance
	config          *config.Config
	injector        *inject.MemoryInjector
	llmClient       *llm.Client
	memoryService   *memory.Service
	evidenceService *evidence.Service
	recallEngine    recall.Recaller
	extractor       *extract.Extractor
	responseBuffer  chan ResponseCapture // Channel for capturing responses for extraction
	server          *http.Server
}

// NewServer creates a new proxy server
func NewServer(a *app.App) *Server {
	// Create new instances of services using app.App's components
	evidenceService := evidence.NewService(a.DB, a.Config)
	var recallEngine recall.Recaller
	if a.Config.SemanticEnabled {
		recallEngine = semantic.NewSemanticEngine(a.DB, a.Memory, evidenceService, a.Config, a.Logger)
	} else {
		recallEngine = recall.NewEngine(a.Memory, evidenceService, a.Config, a.Logger, a.DB.GetConnection())
	}
	injector := inject.NewMemoryInjector(recallEngine)
	llmClient := llm.NewClient(a.Config)
	extractor := extract.NewExtractor(evidenceService)

	// Create CoVe verifier if enabled (safely disabled by default)
	if a.Config.CoVeEnabled {
		coveVerifier := cove.NewVerifier(a.Config, llmClient)
		extractor.SetCoVeVerifier(coveVerifier)

		// Also set CoVe verifier for recall filtering if enabled
		if a.Config.CoVeRecallFilterEnabled {
			injector.SetCoVeVerifier(coveVerifier)
			a.Logger.Info("CoVe enabled (extraction + recall filtering)",
				zap.Float64("confidence_threshold", a.Config.CoVeConfidenceThreshold),
				zap.Int("max_candidates", a.Config.CoVeMaxCandidates),
			)
		} else {
			a.Logger.Info("CoVe enabled (extraction only)",
				zap.Float64("confidence_threshold", a.Config.CoVeConfidenceThreshold),
				zap.Int("max_candidates", a.Config.CoVeMaxCandidates),
			)
		}
	}

	server := &Server{
		app:             a, // Store the app instance
		config:          a.Config,
		injector:        injector,
		llmClient:       llmClient,
		memoryService:   a.Memory,
		evidenceService: evidenceService,
		recallEngine:    recallEngine,
		extractor:       extractor,
		responseBuffer:  make(chan ResponseCapture, 10), // Buffered channel to prevent blocking
	}

	// Start a goroutine to process response captures
	go server.processResponseCaptures()

	return server
}

// processResponseCaptures processes captured responses for memory extraction
func (s *Server) processResponseCaptures() {
	for capture := range s.responseBuffer {
		// Extract memories from the response text
		if s.extractor != nil {
			err := s.extractor.ExtractAndQueueForVerification(capture.ResponseText, s.memoryService, s.evidenceService, s.app.ProjectID)
			if err != nil {
				s.app.Logger.Error("Error extracting memories from response", zap.Error(err), zap.String("model", capture.Model))
			}
		}
	}
}

// Start starts the proxy server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Proxy endpoint for chat completions
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.ProxyPort),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 150 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s.server.ListenAndServe()
}

// Stop stops the proxy server
func (s *Server) Stop() error {
	// Close the recall engine to flush any pending metrics
	if s.recallEngine != nil {
		s.recallEngine.Close()
	}

	if s.server != nil {
		// Close the response buffer channel to stop the processing goroutine
		close(s.responseBuffer)
		return s.server.Shutdown(context.Background())
	}
	return nil
}

// handleChatCompletions handles chat completion requests
func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.app.Logger.Error("Unable to read request body", zap.Error(err))
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	var req llm.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.app.Logger.Error("Invalid JSON in request body", zap.Error(err))
		http.Error(w, "Invalid JSON in request", http.StatusBadRequest)
		return
	}

	notification := MemoryNotification{
		RecallStatus: recallStatusNone,
	}

	// Extract the user's message to use for memory recall
	userMessage := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}

	// Inject memories into the prompt
	if userMessage != "" {
		// Perform recall to get relevant memories
		recallResults, err := s.recallEngine.Recall(recall.RecallOptions{
			ProjectID: s.app.ProjectID,
			Query:     userMessage,
			MaxItems:  s.config.RecallMaxItems,
			MaxTokens: s.config.RecallMaxTokens,
		})
		if err != nil {
			s.app.Logger.Warn("Failed to recall memories for prompt injection", zap.Error(err), zap.String("user_message", userMessage))
			notification.RecallStatus = recallStatusFailed
		} else {
			notifyCount := len(recallResults)
			notification.RecallCount = notifyCount
			if notifyCount > 0 {
				notification.RecallStatus = recallStatusInjected
			}
			// Format memories and prepend to the last user message
			var memories []*memory.Memory
			for _, result := range recallResults {
				memories = append(memories, result.Memory)
			}

			memoryText := s.injector.FormatMemoriesForSystemMessage(memories)

			// Add memory to the messages as a system message
			req.Messages = append([]llm.Message{{Role: "system", Content: memoryText}}, req.Messages...)
		}
	}

	// Forward the request to the LLM backend
	ctx := r.Context()

	// Check if streaming is requested
	if req.Stream {
		s.handleStreamingRequest(w, ctx, req, notification)
	} else {
		s.handleNonStreamingRequest(w, ctx, req, notification)
	}
}

// handleStreamingRequest handles streaming chat completion requests
func (s *Server) handleStreamingRequest(w http.ResponseWriter, ctx context.Context, req llm.ChatCompletionRequest, notification MemoryNotification) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get the streaming response from the LLM
	chunkChan, errChan := s.llmClient.StreamChatCompletions(ctx, req)

	// Rolling buffer to collect only the most recent response content for extraction
	// This prevents full response buffering which could cause memory issues
	rollingBuffer := NewRollingBuffer(s.config.ExtractionBufferBytes)
	var responseMutex sync.Mutex

	// Process the stream
	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				// Channel closed, stream is complete
				// Get the final content from the rolling buffer for extraction
				finalContent := rollingBuffer.String()

				// Log response token count if metrics are enabled
				if s.config.MetricsEnabled {
					tokenCount := estimateTokenCount(finalContent)
					s.app.Logger.Info("Response metrics",
						zap.String("model", req.Model),
						zap.Int("response_tokens", tokenCount),
					)
				}

				// Send the final content to the processing channel for extraction
				select {
				case s.responseBuffer <- ResponseCapture{
					ResponseText: finalContent,
					Model:        req.Model,
					Timestamp:    time.Now(),
				}:
				default:
					s.app.Logger.Warn("Response buffer is full, skipping extraction for streaming request", zap.String("model", req.Model))
				}

				s.emitMemoryNotificationEvent(w, notification)
				return
			}

			// Add chunk to rolling buffer
			responseMutex.Lock()
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					_, _ = rollingBuffer.Write([]byte(choice.Delta.Content))

					// Log an error if buffer exceeds max size, do not panic
					if rollingBuffer.Len() > s.config.ExtractionBufferBytes {
						s.app.Logger.Error("Rolling buffer exceeded configured max size",
							zap.Int("current_size", rollingBuffer.Len()),
							zap.Int("max_size", s.config.ExtractionBufferBytes))
						// Optionally, truncate the buffer or handle as a fatal error
						// For now, we'll just log and continue, as the system can still function.
					}
				}
			}
			responseMutex.Unlock()

			// Send chunk to client
			chunkBytes, err := json.Marshal(chunk)
			if err != nil {
				s.app.Logger.Error("Failed to marshal streaming chunk", zap.Error(err))
				continue
			}

			// Terminate the SSE event with a blank line so clients treat it as a complete message.
			fmt.Fprintf(w, "data: %s\n\n", chunkBytes)
			w.(http.Flusher).Flush()
		case err, ok := <-errChan:
			if ok && err != nil {
				s.app.Logger.Error("Error from LLM streaming channel", zap.Error(err))
				// Send error event to client
				errorMsg := fmt.Sprintf("data: {\"error\": \"%v\"}\n\n", err)
				fmt.Fprint(w, errorMsg)
				w.(http.Flusher).Flush()
				return
			}
		case <-ctx.Done():
			s.app.Logger.Info("Streaming request context cancelled")
			// Request context cancelled
			return
		}
	}
}

// handleNonStreamingRequest handles non-streaming chat completion requests
func (s *Server) handleNonStreamingRequest(w http.ResponseWriter, ctx context.Context, req llm.ChatCompletionRequest, notification MemoryNotification) {
	resp, err := s.llmClient.ChatCompletionsRaw(ctx, req)
	if err != nil {
		s.app.Logger.Error("LLM ChatCompletions failed for non-streaming request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// For non-streaming responses, limit the amount of data captured for extraction.
	rollingBuffer := NewRollingBuffer(s.config.ExtractionBufferBytes)

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if notification.RecallStatus != "" {
		w.Header().Set("X-TinyMem-Recall-Status", string(notification.RecallStatus))
	}
	w.Header().Set("X-TinyMem-Recall-Count", strconv.Itoa(notification.RecallCount))
	w.WriteHeader(resp.StatusCode)

	tee := io.TeeReader(resp.Body, rollingBuffer)
	if _, err := io.Copy(w, tee); err != nil {
		s.app.Logger.Error("Failed to proxy non-streaming response", zap.Error(err))
		return
	}

	finalText := rollingBuffer.String()

	// Log response token count if metrics are enabled
	if s.config.MetricsEnabled {
		tokenCount := estimateTokenCount(finalText)
		s.app.Logger.Info("Response metrics",
			zap.String("model", req.Model),
			zap.Int("response_tokens", tokenCount),
		)
	}

	// Send response for extraction via channel
	select {
	case s.responseBuffer <- ResponseCapture{
		ResponseText: finalText,
		Model:        req.Model,
		Timestamp:    time.Now(),
	}:
	default:
		s.app.Logger.Warn("Response buffer is full, skipping extraction for non-streaming request", zap.String("model", req.Model))
	}

	// Response already forwarded to client.
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) emitMemoryNotificationEvent(w http.ResponseWriter, notification MemoryNotification) {
	payload := map[string]interface{}{
		"type":          "tinymem.memory_status",
		"recall_count":  notification.RecallCount,
		"recall_status": notification.RecallStatus,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		s.app.Logger.Warn("Failed to encode memory notification", zap.Error(err))
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", data)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// estimateTokenCount provides a conservative estimate of token count
// This overestimates to ensure we stay within budget
func estimateTokenCount(text string) int {
	if text == "" {
		return 0
	}

	// Conservative approach: count characters and divide by a small number
	// This ensures we never underestimate
	charCount := utf8.RuneCountInString(text)

	// Divide by 3 to overestimate (typical ratio is ~4 chars per token)
	// This is a conservative approach to ensure we stay within budget
	conservativeEstimate := charCount / 3

	// Also count words as a secondary measure and take the higher estimate
	words := strings.Fields(text)
	wordBasedEstimate := len(words) * 2 // Multiply by 2 to be conservative

	// Return the higher of the two estimates to ensure safety
	if conservativeEstimate > wordBasedEstimate {
		return conservativeEstimate
	}
	return wordBasedEstimate
}
