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
	"github.com/daverage/tinymem/internal/tasks"
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
	taskService     *tasks.Service
	responseBuffer  chan ResponseCapture // Channel for capturing responses for extraction
	shutdownCh      chan struct{}
	shutdownOnce    sync.Once
	server          *http.Server
}

// NewServer creates a new proxy server
func NewServer(a *app.App) *Server {
	// Create new instances of services using app.App's components
	evidenceService := evidence.NewService(a.Core.DB, a.Core.Config)
	var recallEngine recall.Recaller
	if a.Core.Config.SemanticEnabled {
		recallEngine = semantic.NewSemanticEngine(a.Core.DB, a.Memory, evidenceService, a.Core.Config, a.Core.Logger)
	} else {
		recallEngine = recall.NewEngine(a.Memory, evidenceService, a.Core.Config, a.Core.Logger, a.Core.DB.GetConnection())
	}
	injector := inject.NewMemoryInjector(recallEngine)
	llmClient := llm.NewClient(a.Core.Config)
	extractor := extract.NewExtractor(evidenceService)
	taskService := tasks.NewService(a.Core.DB, a.Memory, a.Project.ID)

	// Create CoVe verifier if enabled (safely enabled by default)
	if a.Core.Config.CoVeEnabled {
		coveVerifier := cove.NewVerifier(a.Core.Config, llmClient)
		coveVerifier.SetStatsStore(cove.NewSQLiteStatsStore(a.Core.DB.GetConnection()), a.Project.ID)
		extractor.SetCoVeVerifier(coveVerifier)

		// Also set CoVe verifier for recall filtering
		injector.SetCoVeVerifier(coveVerifier)
		a.Core.Logger.Info("CoVe enabled (extraction + recall filtering)",
			zap.Float64("confidence_threshold", a.Core.Config.CoVeConfidenceThreshold),
			zap.Int("max_candidates", a.Core.Config.CoVeMaxCandidates),
		)
	}

	server := &Server{
		app:             a, // Store the app instance
		config:          a.Core.Config,
		injector:        injector,
		llmClient:       llmClient,
		memoryService:   a.Memory,
		evidenceService: evidenceService,
		recallEngine:    recallEngine,
		extractor:       extractor,
		taskService:     taskService,
		responseBuffer:  make(chan ResponseCapture, 10), // Buffered channel to prevent blocking
		shutdownCh:      make(chan struct{}),
	}

	// Start a goroutine to process response captures
	go server.processResponseCaptures()

	return server
}

// processResponseCaptures processes captured responses for memory extraction
func (s *Server) processResponseCaptures() {
	for {
		select {
		case capture := <-s.responseBuffer:
			// Extract memories from the response text
			if s.extractor != nil {
				err := s.extractor.ExtractAndQueueForVerification(capture.ResponseText, s.memoryService, s.evidenceService, s.app.Project.ID)
				if err != nil {
					s.app.Core.Logger.Error("Error extracting memories from response", zap.Error(err), zap.String("model", capture.Model))
				}
			}
		case <-s.shutdownCh:
			return
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
		s.shutdownOnce.Do(func() { close(s.shutdownCh) })
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

	// Parse the request body with a size limit to prevent memory exhaustion
	const maxBodyBytes = 1 << 20 // 1 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req llm.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.app.Core.Logger.Error("Invalid JSON in request body", zap.Error(err))
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

	// Check if the user explicitly requested to continue tasks
	explicitTaskContinuation := s.taskService.HasExplicitTaskContinuationIntent(userMessage)

	// Inject memories into the prompt
	if userMessage != "" {
		// Perform recall to get relevant memories
		recallResults, err := s.recallEngine.Recall(recall.RecallOptions{
			ProjectID: s.app.Project.ID,
			Query:     userMessage,
			MaxItems:  s.config.RecallMaxItems,
			MaxTokens: s.config.RecallMaxTokens,
		})
		if err != nil {
			s.app.Core.Logger.Warn("Failed to recall memories for prompt injection", zap.Error(err), zap.String("user_message", userMessage))
			notification.RecallStatus = recallStatusFailed
		} else {
			recallResults = s.injector.FilterRecallResults(r.Context(), recallResults, userMessage)

			// Apply task safety filtering: filter out tasks that shouldn't be acted upon
			safeRecallResults := s.filterRecallResultsForTaskSafety(recallResults, userMessage, explicitTaskContinuation)

			notifyCount := len(safeRecallResults)
			notification.RecallCount = notifyCount
			if notifyCount > 0 {
				notification.RecallStatus = recallStatusInjected
			}
			// Format memories and prepend to the last user message
			var memories []*memory.Memory
			for _, result := range safeRecallResults {
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

	// Emit recall status as a notification event for streaming clients.
	if notification.RecallStatus != "" {
		s.emitMemoryNotificationEvent(w, notification)
	}

	// Process the stream
	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				finalText := rollingBuffer.String()

				// Log response token count if metrics are enabled
				if s.config.MetricsEnabled {
					tokenCount := estimateTokenCount(finalText)
					s.app.Core.Logger.Info("Response metrics",
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
				case <-s.shutdownCh:
				default:
					s.app.Core.Logger.Warn("Response buffer is full, skipping extraction for streaming request", zap.String("model", req.Model))
				}

				return
			}

			// Forward the chunk to the client
			chunkData, err := json.Marshal(chunk)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", chunkData)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			// Capture response text for extraction
			responseMutex.Lock()
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					_, _ = rollingBuffer.Write([]byte(choice.Delta.Content))
				}
			}
			responseMutex.Unlock()
		case err, ok := <-errChan:
			if ok && err != nil {
				s.app.Core.Logger.Error("Error from LLM streaming channel", zap.Error(err))

				// Provide a more helpful error message
				errMsg := err.Error()
				if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "i/o timeout") {
					errMsg = fmt.Sprintf("tinyMem Proxy: Unable to reach LLM backend at %s. Ensure your local LLM server is running. Error: %v", s.config.LLMBaseURL, err)
				}

				// Send error event to client
				errorMsg := fmt.Sprintf("data: {\"error\": \"%s\"}\n\n", strings.ReplaceAll(errMsg, "\"", "\\\""))
				fmt.Fprint(w, errorMsg)
				w.(http.Flusher).Flush()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleNonStreamingRequest handles non-streaming chat completion requests
func (s *Server) handleNonStreamingRequest(w http.ResponseWriter, ctx context.Context, req llm.ChatCompletionRequest, notification MemoryNotification) {
	resp, err := s.llmClient.ChatCompletionsRaw(ctx, req)
	if err != nil {
		s.app.Core.Logger.Error("LLM ChatCompletions failed for non-streaming request", zap.Error(err))

		// Provide a more helpful error message if it's a connection error
		errMsg := err.Error()
		if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "i/o timeout") {
			errMsg = fmt.Sprintf("tinyMem Proxy: Unable to reach LLM backend at %s. Ensure your local LLM server (LM Studio/Ollama) is running and its 'Local Server' is started. Error: %v", s.config.LLMBaseURL, err)
		}

		http.Error(w, errMsg, http.StatusInternalServerError)
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
		s.app.Core.Logger.Error("Failed to proxy non-streaming response", zap.Error(err))
		return
	}

	finalText := rollingBuffer.String()

	// Log response token count if metrics are enabled
	if s.config.MetricsEnabled {
		tokenCount := estimateTokenCount(finalText)
		s.app.Core.Logger.Info("Response metrics",
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
	case <-s.shutdownCh:
	default:
		s.app.Core.Logger.Warn("Response buffer is full, skipping extraction for non-streaming request", zap.String("model", req.Model))
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
		s.app.Core.Logger.Warn("Failed to encode memory notification", zap.Error(err))
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

// filterRecallResultsForTaskSafety filters out tasks that shouldn't be acted upon based on safety rules
func (s *Server) filterRecallResultsForTaskSafety(results []recall.RecallResult, query string, explicitTaskContinuation bool) []recall.RecallResult {
	if explicitTaskContinuation {
		// If user explicitly requested task continuation, allow all tasks
		return results
	}

	// Otherwise, filter out unfinished tasks that are in dormant mode
	var safeResults []recall.RecallResult

	for _, result := range results {
		if result.Memory.Type == memory.Task {
			// Check if this is an unfinished task that should be filtered out
			if s.isUnfinishedDormantTask(result.Memory) {
				// Skip this task - it's an unfinished dormant task and user didn't explicitly request continuation
				continue
			}
		}
		// Include non-task memories or tasks that are completed
		safeResults = append(safeResults, result)
	}

	return safeResults
}

// isUnfinishedDormantTask checks if a memory is an unfinished task in dormant mode
func (s *Server) isUnfinishedDormantTask(mem *memory.Memory) bool {
	if mem.Type != memory.Task {
		return false
	}

	// Check if task is completed
	if strings.Contains(mem.Detail, "Completed: true") {
		return false // Completed tasks are fine to include
	}

	// Check if task is in dormant mode
	if strings.Contains(mem.Detail, "Mode: dormant") {
		return true // Unfinished dormant task - should be filtered
	}

	return false
}
