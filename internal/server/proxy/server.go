package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/a-marczewski/tinymem/internal/app"
	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/evidence"
	"github.com/a-marczewski/tinymem/internal/extract"
	"github.com/a-marczewski/tinymem/internal/inject"
	"github.com/a-marczewski/tinymem/internal/llm"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/recall"
	"go.uber.org/zap"
)

// ResponseCapture holds response data for extraction
type ResponseCapture struct {
	ResponseText string
	Model        string
	Timestamp    time.Time
}

// Server implements the HTTP proxy server
type Server struct {
	app             *app.App // New: Hold the app instance
	config          *config.Config
	injector        *inject.MemoryInjector
	llmClient       *llm.Client
	memoryService   *memory.Service
	evidenceService *evidence.Service
	recallEngine    *recall.Engine
	extractor       *extract.Extractor
	responseBuffer  chan ResponseCapture // Channel for capturing responses for extraction
	server          *http.Server
}

// NewServer creates a new proxy server
func NewServer(a *app.App) *Server {
	// Create new instances of services using app.App's components
	evidenceService := evidence.NewService(a.DB)
	recallEngine := recall.NewEngine(a.Memory, evidenceService, a.Config)
	injector := inject.NewMemoryInjector(recallEngine)
	llmClient := llm.NewClient(a.Config)
	extractor := extract.NewExtractor(evidenceService)

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
			err := s.extractor.ExtractAndQueueForVerification(capture.ResponseText, s.memoryService, s.evidenceService, "default_project")
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
			Query:     userMessage,
			MaxItems:  10,  // Configurable
			MaxTokens: 2000, // Configurable
		})
		if err != nil {
			s.app.Logger.Warn("Failed to recall memories for prompt injection", zap.Error(err), zap.String("user_message", userMessage))
			// Continue without memory injection if recall fails
		} else {
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
		s.handleStreamingRequest(w, ctx, req)
	} else {
		s.handleNonStreamingRequest(w, ctx, req)
	}
}
// handleStreamingRequest handles streaming chat completion requests
func (s *Server) handleStreamingRequest(w http.ResponseWriter, ctx context.Context, req llm.ChatCompletionRequest) {
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

				return
			}

			// Add chunk to rolling buffer
			responseMutex.Lock()
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					rollingBuffer.Write([]byte(choice.Delta.Content))

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

			fmt.Fprintf(w, "data: %s\n", chunkBytes)
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
func (s *Server) handleNonStreamingRequest(w http.ResponseWriter, ctx context.Context, req llm.ChatCompletionRequest) {
	response, err := s.llmClient.ChatCompletions(ctx, req)
	if err != nil {
		s.app.Logger.Error("LLM ChatCompletions failed for non-streaming request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store response for extraction
	responseText := ""
	for _, choice := range response.Choices {
		responseText += choice.Message.Content
	}

	// For non-streaming responses, we still need to limit the amount of text sent for extraction
	// Use a rolling buffer to ensure we don't send too much text for extraction
	rollingBuffer := NewRollingBuffer(s.config.ExtractionBufferBytes)
	rollingBuffer.Write([]byte(responseText))
	finalText := rollingBuffer.String()

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

	// Send response back to client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

