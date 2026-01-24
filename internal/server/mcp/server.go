package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/a-marczewski/tinymem/internal/app" // Add app import
	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/evidence"
	"github.com/a-marczewski/tinymem/internal/extract"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/recall"
	"github.com/a-marczewski/tinymem/internal/storage"
)

// Server implements the Model Context Protocol server
type Server struct {
	app             *app.App // New: Hold the app instance
	config          *config.Config
	db              *storage.DB
	memoryService   *memory.Service
	evidenceService *evidence.Service
	recallEngine    *recall.Engine
	extractor       *extract.Extractor
	ctx             context.Context
	cancel          context.CancelFunc
}

// MCPRequest represents a request from the MCP client
type MCPRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     *int            `json:"id,omitempty"`
}

// MCPResponse represents a response to the MCP client
type MCPResponse struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  *MCPError       `json:"error,omitempty"`
	ID     *int            `json:"id"`
}

// MCPError represents an error in the MCP protocol
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewServer creates a new MCP server
func NewServer(a *app.App) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	// Create new instances of services using app.App's components
	// These will now receive the logger from the app.App instance automatically
	evidenceService := evidence.NewService(a.DB)
	recallEngine := recall.NewEngine(a.Memory, evidenceService, a.Config)
	extractor := extract.NewExtractor(evidenceService)

	return &Server{
		app:             a, // Store the app instance
		config:          a.Config,
		db:              a.DB,
		memoryService:   a.Memory,
		evidenceService: evidenceService,
		recallEngine:    recallEngine,
		extractor:       extractor,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start starts the MCP server using stdin/stdout
func (s *Server) Start() error {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse the request
		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		// Handle the request based on method
		s.handleRequest(&req)
	}

	return scanner.Err()
}

// handleRequest handles an incoming MCP request
func (s *Server) handleRequest(req *MCPRequest) {
	switch req.Method {
	case "tools/call":
		s.handleToolCall(req)
	case "tools/list":
		s.handleToolsList(req)
	case "prompts/list":
		s.handlePromptsList(req)
	case "initialize":
		s.handleInitialize(req)
	case "ping":
		s.handlePing(req)
	case "notifications/initialized", "initialized":
		// Client notification; no response required.
	case "shutdown":
		s.handleShutdown(req)
	default:
		if req.ID != nil {
			s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *MCPRequest) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "tinyMem",
				"version": "0.1.0",
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]bool{},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

// handlePing handles the ping request from MCP clients
func (s *Server) handlePing(req *MCPRequest) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  map[string]interface{}{},
		"id":      req.ID,
	}

	s.sendResponse(response)
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(req *MCPRequest) {
	tools := []map[string]interface{}{
		{
			"name":        "memory_query",
			"description": "Search memories using full-text or semantic search",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query for finding relevant memories",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of results to return (default: 10)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "memory_recent",
			"description": "Retrieve the most recent memories",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"count": map[string]interface{}{
						"type":        "number",
						"description": "Number of recent memories to retrieve (default: 10)",
					},
				},
			},
		},
		{
			"name":        "memory_write",
			"description": "Create a new memory entry with optional evidence",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Memory type: fact, claim, plan, decision, constraint, observation, note",
						"enum":        []string{"fact", "claim", "plan", "decision", "constraint", "observation", "note"},
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Brief summary of the memory",
					},
					"detail": map[string]interface{}{
						"type":        "string",
						"description": "Detailed description",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Optional unique key for the memory",
					},
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Optional source reference (file path, URL, etc.)",
					},
				},
				"required": []string{"type", "summary"},
			},
		},
		{
			"name":        "memory_stats",
			"description": "Get statistics about stored memories",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "memory_health",
			"description": "Check the health status of the memory system",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "memory_doctor",
			"description": "Run diagnostics on the memory system",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"tools": tools,
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

// handlePromptsList handles the prompts/list request
func (s *Server) handlePromptsList(req *MCPRequest) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"prompts": []map[string]interface{}{},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

// handleToolCall handles tool calls
func (s *Server) handleToolCall(req *MCPRequest) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	argsBytes, _ := json.Marshal(params.Arguments)

	switch params.Name {
	case "memory_query":
		s.handleMemoryQuery(req, argsBytes)
	case "memory_recent":
		s.handleMemoryRecent(req, argsBytes)
	case "memory_write":
		s.handleMemoryWrite(req, argsBytes)
	case "memory_stats":
		s.handleMemoryStats(req)
	case "memory_health":
		s.handleMemoryHealth(req)
	case "memory_doctor":
		s.handleMemoryDoctor(req)
	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Tool not found: %s", params.Name))
	}
}

// handleMemoryQuery handles memory query requests
func (s *Server) handleMemoryQuery(req *MCPRequest, args json.RawMessage) {
	var queryReq struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.Unmarshal(args, &queryReq); err != nil {
		s.sendError(req.ID, -32602, "Invalid arguments for memory_query")
		return
	}

	if queryReq.Limit == 0 {
		queryReq.Limit = 10
	}

	results, err := s.recallEngine.Recall(recall.RecallOptions{
		Query:    queryReq.Query,
		MaxItems: queryReq.Limit,
	})
	if err != nil {
		s.sendError(req.ID, -32603, fmt.Sprintf("Query failed: %v", err))
		return
	}

	// Convert results to text content for MCP
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Found %d memories matching '%s':\n\n", len(results), queryReq.Query))

	for i, result := range results {
		content.WriteString(fmt.Sprintf("%d. [%s] %s (score: %.2f)\n",
			i+1, result.Memory.Type, result.Memory.Summary, result.Score))
		if result.Memory.Detail != "" {
			content.WriteString(fmt.Sprintf("   %s\n", result.Memory.Detail))
		}
		if result.Memory.Source != nil {
			content.WriteString(fmt.Sprintf("   Source: %s\n", *result.Memory.Source))
		}
		content.WriteString(fmt.Sprintf("   Date: %s\n\n", result.Memory.CreatedAt.Format(time.RFC3339)))
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": content.String(),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

// handleMemoryRecent handles recent memory requests
func (s *Server) handleMemoryRecent(req *MCPRequest, args json.RawMessage) {
	var recentReq struct {
		Count int `json:"count"`
	}

	if err := json.Unmarshal(args, &recentReq); err == nil && recentReq.Count > 0 {
		// Use provided count
	} else {
		recentReq.Count = 10
	}

	memories, err := s.memoryService.GetAllMemories("default_project") // In real impl, get from context
	if err != nil {
		s.sendError(req.ID, -32603, fmt.Sprintf("Failed to get recent memories: %v", err))
		return
	}

	// Take only the most recent ones
	limit := recentReq.Count
	if len(memories) < limit {
		limit = len(memories)
	}

	recentMemories := memories[:limit]

	// Convert to text content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Recent %d memories:\n\n", len(recentMemories)))

	for i, mem := range recentMemories {
		content.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, mem.Type, mem.Summary))
		if mem.Detail != "" {
			content.WriteString(fmt.Sprintf("   %s\n", mem.Detail))
		}
		if mem.Source != nil {
			content.WriteString(fmt.Sprintf("   Source: %s\n", *mem.Source))
		}
		content.WriteString(fmt.Sprintf("   Date: %s\n\n", mem.CreatedAt.Format(time.RFC3339)))
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": content.String(),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

// handleMemoryWrite handles memory write requests
func (s *Server) handleMemoryWrite(req *MCPRequest, args json.RawMessage) {
	var writeReq struct {
		Type    string `json:"type"`
		Summary string `json:"summary"`
		Detail  string `json:"detail"`
		Key     string `json:"key"`
		Source  string `json:"source"`
	}
	
	if err := json.Unmarshal(args, &writeReq); err != nil {
		s.sendError(req.ID, -32602, "Invalid arguments for memory.write")
		return
	}

	// Validate memory type
	memType := memory.Type(writeReq.Type)
	if !memType.IsValid() {
		s.sendError(req.ID, -32602, "Invalid memory type")
		return
	}

	// Create new memory
	newMemory := &memory.Memory{
		ProjectID: "default_project", // In real impl, get from context
		Type:      memType,
		Summary:   writeReq.Summary,
		Detail:    writeReq.Detail,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if writeReq.Key != "" {
		newMemory.Key = &writeReq.Key
	}
	if writeReq.Source != "" {
		newMemory.Source = &writeReq.Source
	}

	if err := s.memoryService.CreateMemory(newMemory); err != nil {
		s.sendError(req.ID, -32603, fmt.Sprintf("Failed to create memory: %v", err))
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Memory created successfully with ID: %s\nType: %s\nSummary: %s",
						newMemory.ID, newMemory.Type, newMemory.Summary),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

// handleMemoryStats handles memory statistics requests
func (s *Server) handleMemoryStats(req *MCPRequest) {
	// Get all memories to calculate stats
	memories, err := s.memoryService.GetAllMemories("default_project") // In real impl, get from context
	if err != nil {
		s.app.Logger.Error("Failed to get memory stats for MCP", zap.Error(err))
		s.sendError(req.ID, -32603, "Failed to retrieve memory statistics")
		return
	}

	// Count by type
	typeCounts := make(map[string]int)
	for _, mem := range memories {
		typeCounts[string(mem.Type)]++
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("Memory Statistics\n\n"))
	content.WriteString(fmt.Sprintf("Total memories: %d\n\n", len(memories)))
	content.WriteString("By type:\n")
	for memType, count := range typeCounts {
		content.WriteString(fmt.Sprintf("  %s: %d\n", memType, count))
	}
	if len(memories) > 0 {
		content.WriteString(fmt.Sprintf("\nLast updated: %s\n", memories[0].UpdatedAt.Format(time.RFC3339)))
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": content.String(),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

// handleMemoryHealth handles memory health check requests
func (s *Server) handleMemoryHealth(req *MCPRequest) {
	// Check database connectivity
	if err := s.db.GetConnection().Ping(); err != nil {
		s.app.Logger.Error("Database health check failed for MCP", zap.Error(err))
		s.sendError(req.ID, -32603, "Database health check failed")
		return
	}

	// Check if we can perform a simple query
	if _, err := s.memoryService.GetAllMemories("default_project"); err != nil {
		s.app.Logger.Error("Memory service health check failed for MCP", zap.Error(err))
		s.sendError(req.ID, -32603, "Memory service health check failed")
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "Health Check Status: ✅ HEALTHY\n\n" +
						"✅ Database connectivity: OK\n" +
						"✅ Storage system: OK\n" +
						"✅ Memory service: OK\n",
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}
// handleMemoryDoctor handles memory doctor diagnostic requests
func (s *Server) handleMemoryDoctor(req *MCPRequest) {
	doctorRunner := doctor.NewRunner(s.app.Config, s.app.DB)
	diagnostics := doctorRunner.RunAll()

	var content strings.Builder
	content.WriteString("tinyMem Diagnostics Report\n\n")

	if diagnostics.HasIssues() {
		content.WriteString(fmt.Sprintf("⚠️  Status: %d issue(s) detected\n\n", len(diagnostics.Issues)))
		content.WriteString("Issues:\n")
		for i, issue := range diagnostics.Issues {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, issue.Description))
		}
		if len(diagnostics.Recommendations) > 0 {
			content.WriteString("\nRecommendations:\n")
			for i, rec := range diagnostics.Recommendations {
				content.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
			}
		}
	} else {
		content.WriteString("✅ Status: All systems operational\n\n")
		content.WriteString("No issues detected.\n")
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": content.String(),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}
// handleShutdown handles shutdown requests
func (s *Server) handleShutdown(req *MCPRequest) {
	s.app.Logger.Info("MCP server received shutdown request.")
	s.cancel()

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  map[string]interface{}{},
		"id":      req.ID,
	}

	s.sendResponse(response)
	// Do not os.Exit(0) here. Let the main goroutine handle the exit
	// after all deferred cleanups are done.
}

// sendResponse sends a successful response
func (s *Server) sendResponse(response map[string]interface{}) {
	responseBytes, err := json.Marshal(response)
	if err != nil {
		s.app.Logger.Error("Failed to marshal MCP response", zap.Error(err))
		return
	}
	// MCP protocol communicates over stdout
	fmt.Println(string(responseBytes))
}

// sendError sends an error response
func (s *Server) sendError(id *int, code int, message string) {
	errorResp := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
		"id": id,
	}

	responseBytes, _ := json.Marshal(errorResp)
	fmt.Println(string(responseBytes))
}
