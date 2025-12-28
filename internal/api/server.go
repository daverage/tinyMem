package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
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
	runtime      *runtime.Runtime
	llmClient    *llm.Client
	hydrator     *hydration.Engine
	auditor      *audit.Auditor
	logger       *logging.Logger
	debugMode    bool
	httpServer   *http.Server
	listenAddr   string
	databasePath string
	llmProvider  string
	llmEndpoint  string
	startTime    time.Time
	workingDir   string
}

const (
	maxLLMResponseLogBytes = 1024
	llmContextTimeout      = 5 * time.Minute
	recentContextPairs     = 4
	recentContextMaxBytes  = 4 * 1024
	toolMaxRounds          = 3
	toolMaxOutputBytes     = 8 * 1024
)

const toolPolicy = "[TOOL POLICY]\n" +
	"Tools run only from explicit fenced blocks.\n" +
	"```tool\n" +
	"{\"type\":\"shell\",\"command\":\"rg --files\"}\n" +
	"```\n" +
	"Fenced \"bash\"/\"sh\" blocks are treated as shell commands.\n" +
	"User-provided tool/bash blocks will be executed before the LLM response.\n" +
	"Only these blocks are executed. Results are returned as [TOOL RESULT] blocks.\n" +
	"Do not fabricate tool output.\n" +
	"[END TOOL POLICY]"

// NewServer creates a new API server
func NewServer(rt *runtime.Runtime, llmClient *llm.Client, hydrator *hydration.Engine, auditor *audit.Auditor, logger *logging.Logger, listenAddr, databasePath, llmProvider, llmEndpoint string, debugMode bool) *Server {
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = ""
	}

	s := &Server{
		runtime:      rt,
		llmClient:    llmClient,
		hydrator:     hydrator,
		auditor:      auditor,
		logger:       logger,
		debugMode:    debugMode,
		listenAddr:   listenAddr,
		databasePath: databasePath,
		llmProvider:  llmProvider,
		llmEndpoint:  llmEndpoint,
		workingDir:   workingDir,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/v1/user/code", s.handleUserCode)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/doctor", s.handleDoctor)
	mux.HandleFunc("/state", s.handleState)
	mux.HandleFunc("/recent", s.handleRecent)

	// Debug-only endpoints
	if debugMode {
		mux.HandleFunc("/debug/last-prompt", s.handleDebugLastPrompt)
		mux.HandleFunc("/debug/reset", s.handleDebugReset)
	}

	s.httpServer = &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: llmContextTimeout,
	}

	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.startTime = time.Now()
	s.logger.Info("Starting TSLP proxy server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down proxy server")
	return s.httpServer.Shutdown(ctx)
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

	// Inject tool usage policy as a system message (first)
	req.Messages = append([]llm.Message{
		{
			Role:    "system",
			Content: toolPolicy,
		},
	}, req.Messages...)

	// Extract user prompt (last user message)
	var userPrompt string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userPrompt = req.Messages[i].Content
			break
		}
	}

	var userPromptHash string
	var userPromptHashPtr *string
	if userPrompt != "" {
		hash, err := s.runtime.GetVault().Store(userPrompt, vault.ContentTypeUserInput, nil)
		if err != nil {
			s.logger.Error("Failed to store user prompt: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		userPromptHash = hash
		userPromptHashPtr = &userPromptHash
	}

	// Create episode
	episodeID, err := s.runtime.GetLedger().CreateEpisode(userPromptHashPtr, nil, nil)
	if err != nil {
		s.logger.Error("Failed to create episode: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	s.logger.EpisodeCreated(episodeID)

	// Pre-flight: JIT Hydration
	// Per spec section 8.1: scan state map, retrieve authoritative artifacts, inject
	hydrationContent, hydratedKeys, err := s.hydrator.HydrateWithTracking(episodeID)
	if err != nil {
		s.logger.Error("Hydration failed: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Inject hydration content before user prompt
	if hydrationContent != "" {
		s.logger.Debug("Hydrating %d bytes of state (%d entities)", len(hydrationContent), len(hydratedKeys))

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

	// Inject recent context before the user's last message
	if recentContext, err := s.buildRecentContext(episodeID); err != nil {
		s.logger.Error("Failed to build recent context: %v", err)
	} else if recentContext != "" {
		messages := make([]llm.Message, 0, len(req.Messages)+1)
		for i, msg := range req.Messages {
			if i == len(req.Messages)-1 && msg.Role == "user" {
				messages = append(messages, llm.Message{
					Role:    "system",
					Content: recentContext,
				})
			}
			messages = append(messages, msg)
		}
		req.Messages = messages
	}

	// Store the exact prompt the LLM will see (including hydration notices)
	if promptPayload, err := json.Marshal(req.Messages); err != nil {
		s.logger.Error("Failed to marshal LLM prompt: %v", err)
	} else {
		promptHash, err := s.runtime.GetVault().Store(string(promptPayload), vault.ContentTypePrompt, nil)
		if err != nil {
			s.logger.Error("Failed to store LLM prompt: %v", err)
		} else if err := s.runtime.GetLedger().UpdateEpisodeMetadata(episodeID, map[string]interface{}{
			"llm_prompt_hash":  promptHash,
			"llm_prompt_bytes": len(promptPayload),
		}); err != nil {
			s.logger.Error("Failed to update episode metadata: %v", err)
		}
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

	ctx, cancel := context.WithTimeout(r.Context(), llmContextTimeout)
	defer cancel()

	response, responseContent, err := s.completeWithTools(ctx, req.Messages, episodeID)
	if err != nil {
		s.logger.Error("LLM call failed: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Post-processing: store response and trigger shadow audit
	s.postProcessResponse(episodeID, userPromptHash, responseContent)

	// Emit a single SSE chunk with final content
	streamChunk := llm.ChatResponse{
		ID:      response.ID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   response.Model,
		Choices: []llm.Choice{
			{
				Index: 0,
				Delta: &llm.Delta{
					Content: responseContent,
				},
			},
		},
	}

	chunkJSON, err := json.Marshal(streamChunk)
	if err != nil {
		s.logger.Error("Failed to marshal chunk: %v", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", chunkJSON)
	flusher.Flush()
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// handleNonStreamingCompletion handles non-streaming chat completions
func (s *Server) handleNonStreamingCompletion(w http.ResponseWriter, r *http.Request, req *llm.ChatRequest, episodeID, userPromptHash string) {
	ctx, cancel := context.WithTimeout(r.Context(), llmContextTimeout)
	defer cancel()

	response, responseContent, err := s.completeWithTools(ctx, req.Messages, episodeID)
	if err != nil {
		s.logger.Error("LLM call failed: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
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

	truncated := truncateForLog(responseContent, maxLLMResponseLogBytes)
	s.logger.Debug("LLM response content (%d bytes): %s", len(responseContent), truncated)

	// Store response in vault
	responseHash, err := s.runtime.GetVault().Store(responseContent, vault.ContentTypeCode, nil)
	if err != nil {
		s.logger.Error("Failed to store response: %v", err)
		return
	}

	// Update episode with response hash (so diagnostics can show the assistant output)
	s.logger.Debug("Response stored: episode=%s hash=%s", episodeID, responseHash)
	if err := s.runtime.GetLedger().UpdateEpisodeAssistantResponse(episodeID, responseHash); err != nil {
		s.logger.Error("Failed to update episode response hash: %v", err)
	}

	codeBlocks := extractCodeBlocks(responseContent)
	if len(codeBlocks) == 0 {
		// Process artifact (resolve entity, evaluate promotion)
		result, err := s.runtime.ProcessArtifact(responseContent, vault.ContentTypeCode, episodeID, false)
		if err != nil {
			s.logger.Error("Failed to process artifact: %v", err)
			return
		}

		s.logger.PromotionEvaluated(responseHash, "unknown", result.Promoted, result.Reason)
	} else {
		for _, block := range codeBlocks {
			block = strings.TrimSpace(block)
			if block == "" {
				continue
			}

			result, err := s.runtime.ProcessArtifact(block, vault.ContentTypeCode, episodeID, false)
			if err != nil {
				s.logger.Error("Failed to process code block: %v", err)
				continue
			}

			s.logger.PromotionEvaluated(result.ArtifactHash, "unknown", result.Promoted, result.Reason)
		}
	}

	// Trigger shadow audit (async, non-blocking)
	s.auditor.AuditAsync(episodeID, responseHash)
}

type toolCall struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type toolResult struct {
	Command   string
	ExitCode  int
	Stdout    string
	Stderr    string
	Duration  time.Duration
	ExecError error
}

func (s *Server) completeWithTools(ctx context.Context, messages []llm.Message, episodeID string) (*llm.ChatResponse, string, error) {
	var toolCallHashes []string
	var toolResultHashes []string
	rounds := 0

	if userContent, ok := lastUserMessageContent(messages); ok {
		calls, err := extractToolCalls(userContent)
		if err != nil {
			s.logger.Error("Failed to parse user tool calls: %v", err)
		} else if len(calls) > 0 {
			rounds++
			for _, call := range calls {
				if call.Type != "shell" || strings.TrimSpace(call.Command) == "" {
					s.logger.Error("Ignoring invalid tool call: type=%s command=%q", call.Type, call.Command)
					continue
				}

				callJSON, _ := json.Marshal(call)
				if callHash, err := s.runtime.GetVault().Store(string(callJSON), vault.ContentTypeToolCall, nil); err == nil {
					toolCallHashes = append(toolCallHashes, callHash)
				} else {
					s.logger.Error("Failed to store tool call: %v", err)
				}

				result := s.executeTool(ctx, call)
				resultText := formatToolResult(result)

				if resultHash, err := s.runtime.GetVault().Store(resultText, vault.ContentTypeToolResult, nil); err == nil {
					toolResultHashes = append(toolResultHashes, resultHash)
				} else {
					s.logger.Error("Failed to store tool result: %v", err)
				}

				messages = append(messages, llm.Message{Role: "tool", Content: resultText})
			}
		}
	}

	response, err := s.llmClient.Chat(ctx, messages)
	if err != nil {
		return nil, "", err
	}

	responseContent := extractResponseContent(response)

	for rounds < toolMaxRounds {
		calls, err := extractToolCalls(responseContent)
		if err != nil {
			s.logger.Error("Failed to parse tool calls: %v", err)
			break
		}
		if len(calls) == 0 {
			break
		}

		rounds++
		messages = append(messages, llm.Message{Role: "assistant", Content: responseContent})

		for _, call := range calls {
			if call.Type != "shell" || strings.TrimSpace(call.Command) == "" {
				s.logger.Error("Ignoring invalid tool call: type=%s command=%q", call.Type, call.Command)
				continue
			}

			callJSON, _ := json.Marshal(call)
			if callHash, err := s.runtime.GetVault().Store(string(callJSON), vault.ContentTypeToolCall, nil); err == nil {
				toolCallHashes = append(toolCallHashes, callHash)
			} else {
				s.logger.Error("Failed to store tool call: %v", err)
			}

			result := s.executeTool(ctx, call)
			resultText := formatToolResult(result)

			if resultHash, err := s.runtime.GetVault().Store(resultText, vault.ContentTypeToolResult, nil); err == nil {
				toolResultHashes = append(toolResultHashes, resultHash)
			} else {
				s.logger.Error("Failed to store tool result: %v", err)
			}

			messages = append(messages, llm.Message{Role: "tool", Content: resultText})
		}

		response, err = s.llmClient.Chat(ctx, messages)
		if err != nil {
			return response, responseContent, err
		}
		responseContent = extractResponseContent(response)
	}

	if rounds > 0 {
		_ = s.runtime.GetLedger().UpdateEpisodeMetadata(episodeID, map[string]interface{}{
			"tool_rounds":        rounds,
			"tool_call_hashes":   toolCallHashes,
			"tool_result_hashes": toolResultHashes,
		})
	}

	return response, responseContent, nil
}

func extractResponseContent(response *llm.ChatResponse) string {
	if response == nil || len(response.Choices) == 0 {
		return ""
	}
	return response.Choices[0].Message.Content
}

func lastUserMessageContent(messages []llm.Message) (string, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content, true
		}
	}
	return "", false
}

func extractToolCalls(content string) ([]toolCall, error) {
	lines := strings.Split(content, "\n")
	var calls []toolCall

	inBlock := false
	var blockLang string
	var block strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			fence := strings.TrimSpace(line)
			if !inBlock {
				if fence == "```tool" || fence == "```bash" || fence == "```sh" || fence == "```zsh" || fence == "```shell" {
					inBlock = true
					blockLang = strings.TrimPrefix(fence, "```")
					block.Reset()
				}
			} else {
				raw := strings.TrimSpace(block.String())
				if raw != "" {
					if blockLang == "tool" {
						parsed, err := parseToolCallJSON(raw)
						if err != nil {
							return nil, err
						}
						calls = append(calls, parsed...)
					} else {
						calls = append(calls, toolCall{
							Type:    "shell",
							Command: raw,
						})
					}
				}
				inBlock = false
				blockLang = ""
			}
			continue
		}

		if inBlock {
			block.WriteString(line)
			block.WriteString("\n")
		}
	}

	return calls, nil
}

func parseToolCallJSON(raw string) ([]toolCall, error) {
	if strings.HasPrefix(strings.TrimSpace(raw), "[") {
		var calls []toolCall
		if err := json.Unmarshal([]byte(raw), &calls); err != nil {
			return nil, fmt.Errorf("failed to parse tool call array: %w", err)
		}
		return calls, nil
	}

	var call toolCall
	if err := json.Unmarshal([]byte(raw), &call); err != nil {
		return nil, fmt.Errorf("failed to parse tool call: %w", err)
	}
	return []toolCall{call}, nil
}

func (s *Server) executeTool(ctx context.Context, call toolCall) toolResult {
	command := exec.CommandContext(ctx, "sh", "-lc", call.Command)
	if s.workingDir != "" {
		command.Dir = s.workingDir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	start := time.Now()
	err := command.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return toolResult{
		Command:   call.Command,
		ExitCode:  exitCode,
		Stdout:    truncateByBytes(stdout.String(), toolMaxOutputBytes),
		Stderr:    truncateByBytes(stderr.String(), toolMaxOutputBytes),
		Duration:  duration,
		ExecError: err,
	}
}

func formatToolResult(result toolResult) string {
	stdout := result.Stdout
	stderr := result.Stderr
	if stdout == "" {
		stdout = "(empty)"
	}
	if stderr == "" {
		stderr = "(empty)"
	}
	if result.ExecError != nil && stderr == "(empty)" {
		stderr = result.ExecError.Error()
	}

	return fmt.Sprintf(`[TOOL RESULT]
Command: %s
ExitCode: %d
Duration: %s
Stdout:
%s
Stderr:
%s
[END TOOL RESULT]`, result.Command, result.ExitCode, result.Duration.String(), stdout, stderr)
}

func (s *Server) buildRecentContext(episodeID string) (string, error) {
	episode, err := s.runtime.GetLedger().GetEpisode(episodeID)
	if err != nil {
		return "", err
	}
	if episode == nil {
		return "", nil
	}

	episodes, err := s.runtime.GetLedger().GetRecentEpisodesBefore(episode.Timestamp.Unix(), recentContextPairs*3)
	if err != nil {
		return "", err
	}

	type pair struct {
		user      string
		assistant string
	}
	var pairs []pair
	for _, e := range episodes {
		if e.UserPromptHash == nil || e.AssistantResponseHash == nil {
			continue
		}

		userArtifact, err := s.runtime.GetVault().Get(*e.UserPromptHash)
		if err != nil || userArtifact == nil {
			continue
		}
		assistantArtifact, err := s.runtime.GetVault().Get(*e.AssistantResponseHash)
		if err != nil || assistantArtifact == nil {
			continue
		}

		pairs = append(pairs, pair{
			user:      truncateByBytes(userArtifact.Content, recentContextMaxBytes),
			assistant: truncateByBytes(assistantArtifact.Content, recentContextMaxBytes),
		})

		if len(pairs) >= recentContextPairs {
			break
		}
	}

	if len(pairs) == 0 {
		return "", nil
	}

	// Reverse to chronological order
	for i, j := 0, len(pairs)-1; i < j; i, j = i+1, j-1 {
		pairs[i], pairs[j] = pairs[j], pairs[i]
	}

	var sb strings.Builder
	sb.WriteString("[RECENT CONTEXT]\n")
	for i, p := range pairs {
		sb.WriteString(fmt.Sprintf("Pair %d:\nUSER:\n%s\nASSISTANT:\n%s\n", i+1, p.user, p.assistant))
	}
	sb.WriteString("[END RECENT CONTEXT]")

	contextBlock := sb.String()
	contextHash, err := s.runtime.GetVault().Store(contextBlock, vault.ContentTypePrompt, nil)
	if err == nil {
		_ = s.runtime.GetLedger().UpdateEpisodeMetadata(episodeID, map[string]interface{}{
			"recent_context_hash":  contextHash,
			"recent_context_bytes": len([]byte(contextBlock)),
		})
	}

	return contextBlock, nil
}

func truncateByBytes(content string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len([]byte(content)) <= maxBytes {
		return content
	}
	return string([]byte(content)[:maxBytes])
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

func truncateForLog(content string, maxBytes int) string {
	if len(content) <= maxBytes {
		return content
	}

	return content[:maxBytes] + "...(truncated)"
}
