package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daverage/tinymem/internal/app" // Add app import
	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/cove"
	"github.com/daverage/tinymem/internal/doctor" // Add doctor import
	"github.com/daverage/tinymem/internal/enforcement"
	"github.com/daverage/tinymem/internal/evidence"
	"github.com/daverage/tinymem/internal/execution"
	"github.com/daverage/tinymem/internal/extract"
	"github.com/daverage/tinymem/internal/intent"
	"github.com/daverage/tinymem/internal/llm"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/recall"
	"github.com/daverage/tinymem/internal/server"
	"github.com/daverage/tinymem/internal/server/taskops"
	"github.com/daverage/tinymem/internal/storage"
	"github.com/daverage/tinymem/internal/tasks"
	"go.uber.org/zap" // Add zap import
)

// Server implements the Model Context Protocol server
type Server struct {
	app             *app.App // New: Hold the app instance
	config          *config.Config
	db              *storage.DB
	memoryService   *memory.Service
	evidenceService *evidence.Service
	recallEngine    recall.Recaller
	extractor       *extract.Extractor
	coveVerifier    *cove.Verifier
	llmProvider     llm.Provider // LLM provider for CoVe
	taskService     *tasks.Service
	taskManager     *tasks.TaskManager
	taskOperator    *taskops.Operator
	intentGate      *server.IntentGate
	modeCtrl        *execution.Controller
	enforcer        *enforcement.Recorder
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

	// Initialize shared recall services
	recallServices := a.InitializeRecallServices()

	// Create MCP-specific services
	taskManager := tasks.NewTaskManager(filepath.Join(a.Project.Path, "tinyTasks.md"))
	taskService := tasks.NewService(a.Core.DB, a.Memory, a.Project.ID, taskManager)
	taskOperator := taskops.New(taskManager, taskService)
	intentGate := server.NewIntentGate(a.Execution, a.Enforcement, a.Core.Logger)

	return &Server{
		app:             a, // Store the app instance
		config:          a.Core.Config,
		db:              a.Core.DB,
		memoryService:   a.Memory,
		evidenceService: recallServices.EvidenceService,
		recallEngine:    recallServices.RecallEngine,
		extractor:       recallServices.Extractor,
		coveVerifier:    recallServices.CoVeVerifier,
		llmProvider:     recallServices.LLMProvider,
		taskOperator:    taskOperator,
		intentGate:      intentGate,
		taskService:     taskService,
		modeCtrl:        a.Execution,
		taskManager:     taskManager,
		enforcer:        a.Enforcement,
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
				"tools": map[string]interface{}{
					"listChanged": false,
				},
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
			"description": "Search memories via lexical CoVe recall (Boundary: ActionMemoryQuery; mode: PASSIVE+; outcomes: ALLOW/BLOCK).",
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
			"name":        "memory_set_mode",
			"description": "Explicitly set the execution mode (Boundary: execution_mode; this call must happen before sensitive actions).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"mode": map[string]interface{}{
						"type":        "string",
						"description": "Target execution mode",
						"enum":        []string{"PASSIVE", "GUARDED", "STRICT"},
					},
				},
				"required": []string{"mode"},
			},
		},
		{
			"name":        "memory_recent",
			"description": "Retrieve the most recent memories (Boundary: ActionMemoryQuery; PASSIVE).",
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
			"name":        "memory_run_metadata",
			"description": "Read the enforcement run metadata (execution_mode, events, proven counts).",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "memory_claim_success",
			"description": "Record a claimed success and flag adversarial claims that lack enforcement.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"success": map[string]interface{}{
						"type":        "boolean",
						"description": "Was a success claimed?",
					},
					"enforced": map[string]interface{}{
						"type":        "boolean",
						"description": "Was enforcement observed for the claimed success?",
					},
					"boundary": map[string]interface{}{
						"type":        "string",
						"description": "Optional boundary label describing the claim context.",
					},
					"details": map[string]interface{}{
						"type":        "string",
						"description": "Optional detail text for auditing.",
					},
				},
				"required": []string{"success", "enforced"},
			},
		},
		{
			"name":        "memory_write",
			"description": "Create a new memory entry (Boundary: memory write/task creation; GUARDED/STRICT).",
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
					"classification": map[string]interface{}{
						"type":        "string",
						"description": "Optional classification for better recall precision (e.g., 'decision', 'constraint', 'glossary', 'invariant')",
					},
					"evidence": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type":        "string",
									"description": "Evidence type: file_exists, grep_hit, cmd_exit0, test_pass",
								},
								"content": map[string]interface{}{
									"type":        "string",
									"description": "Evidence content (path, pattern::file, or command)",
								},
							},
							"required": []string{"type", "content"},
						},
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
		{
			"name":        "memory_eval_stats",
			"description": "Get detailed metrics for evaluation and scoring.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "memory_check_task_authority",
			"description": "Check tinyTasks.md status and return authorization for multi-step work. Returns file existence, unchecked tasks, and authorization status.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "task_list",
			"description": "List tasks tracked in tinyTasks.md.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "task_add",
			"description": "Add a new task to tinyTasks.md via the server-managed TaskManager.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Top-level task title",
					},
					"section": map[string]interface{}{
						"type":        "string",
						"description": "Optional section heading",
					},
					"subtasks": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"title": map[string]interface{}{
									"type":        "string",
									"description": "Subtask title",
								},
								"completed": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether the subtask is already done",
								},
							},
							"required": []string{"title"},
						},
					},
				},
				"required": []string{"title"},
			},
		},
		{
			"name":        "task_update",
			"description": "Update the title or subtasks of an existing task via TaskManager.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Identifier of the task to update",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "New title for the task",
					},
					"subtasks": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"title": map[string]interface{}{
									"type":        "string",
									"description": "Subtask title",
								},
								"completed": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether the subtask is done",
								},
							},
							"required": []string{"title"},
						},
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "task_complete",
			"description": "Mark every checkbox within a task as completed through TaskManager.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Identifier of the task to complete",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "artifact_create",
			"description": "Create or update a project artifact (file) under TinyMem governance. The server validates path and performs the write.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Relative file path within the workspace (e.g. index.html).",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Full file contents to write.",
					},
					"overwrite": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether an existing file may be overwritten (default: true).",
					},
					"link_task_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional task ID this artifact is associated with.",
					},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			"name":        "artifact_read",
			"description": "Read the contents of a workspace artifact. Path is validated and constrained to the workspace. TaskManager-owned files are excluded.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Relative file path within the workspace.",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "artifact_list",
			"description": "List files in the workspace. Internal directories and TaskManager-owned files are always excluded.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Optional glob pattern. Patterns without '/' match the base filename in any subdirectory.",
					},
				},
			},
		},
	}

	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok {
			if def, exists := intent.Lookup(name); exists {
				for k, v := range def.Metadata() {
					tool[k] = v
				}
			}
		}
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
	case "memory_eval_stats":
		s.handleMemoryEvalStats(req)
	case "memory_run_metadata":
		s.handleMemoryRunMetadata(req)
	case "memory_claim_success":
		s.handleMemoryClaimSuccess(req, argsBytes)
	case "memory_set_mode":
		s.handleMemorySetMode(req, argsBytes)
	case "memory_check_task_authority":
		s.handleMemoryCheckTaskAuthority(req)
	case "task_list":
		s.handleTaskList(req)
	case "task_add":
		s.handleTaskAdd(req, argsBytes)
	case "task_update":
		s.handleTaskUpdate(req, argsBytes)
	case "task_complete":
		s.handleTaskComplete(req, argsBytes)
	case "artifact_create":
		s.handleArtifactCreate(req, argsBytes)
	case "artifact_read":
		s.handleArtifactRead(req, argsBytes)
	case "artifact_list":
		s.handleArtifactList(req, argsBytes)
	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Tool not found: %s", params.Name))
	}
}

func (s *Server) handleMemorySetMode(req *MCPRequest, args json.RawMessage) {
	if args == nil {
		s.sendToolError(req.ID, -32602, "Missing arguments for memory_set_mode", "memory_set_mode")
		return
	}

	if _, ok := s.ensureIntent(req, "memory_set_mode", execution.ActionModeDeclaration); !ok {
		return
	}

	var payload struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(args, &payload); err != nil {
		s.sendToolError(req.ID, -32602, "Invalid arguments for memory_set_mode", "memory_set_mode")
		return
	}

	mode, err := execution.ParseMode(payload.Mode)
	if err != nil {
		s.sendToolError(req.ID, -32602, fmt.Sprintf("Unsupported mode: %v", err), "memory_set_mode")
		return
	}

	if err := s.modeCtrl.SetMode(mode); err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to set mode: %v", err), "memory_set_mode")
		return
	}

	if s.enforcer != nil {
		s.enforcer.SetMode(string(mode))
		s.recordEnforcementEvent(enforcement.Event{
			Code:     server.EnforcementCodeModeUpdated,
			Boundary: "execution_mode",
			Mode:     string(mode),
			Outcome:  enforcement.OutcomeAllow,
		})
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Execution mode updated to %s", mode),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

func (s *Server) handleMemoryRunMetadata(req *MCPRequest) {
	if _, ok := s.ensureIntent(req, "memory_run_metadata", execution.ActionMemoryRunMetadata); !ok {
		return
	}

	if s.enforcer == nil {
		s.sendToolError(req.ID, -32603, "Enforcement metadata unavailable", "memory_run_metadata")
		return
	}

	meta := s.enforcer.Metadata()
	payload, err := json.Marshal(meta)
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to marshal metadata: %v", err), "memory_run_metadata")
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": string(payload),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

func (s *Server) handleMemoryCheckTaskAuthority(req *MCPRequest) {
	if _, ok := s.ensureIntent(req, "memory_check_task_authority", execution.ActionTaskAuthority); !ok {
		return
	}

	result, err := server.BuildTaskAuthorityPayload(s.taskManager)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result = map[string]interface{}{
				"exists":          false,
				"unchecked_tasks": []string{},
				"authorization":   "create_allowed",
				"task_count":      0,
			}
		} else {
			s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to describe task authority: %v", err), "memory_check_task_authority")
			return
		}
	}

	payload, err := json.Marshal(result)
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to marshal task authority: %v", err), "memory_check_task_authority")
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": string(payload),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

func buildNextTaskSignalsFromList(parsedTasks []*tasks.Task) map[string]interface{} {
	for _, task := range parsedTasks {
		if task.Completed {
			continue
		}

		return map[string]interface{}{
			"steps_total":      task.StepsTotal,
			"steps_done":       task.StepsDone,
			"has_subtasks":     task.StepsTotal > 0,
			"title_word_count": len(strings.Fields(task.Title)),
			"section_title":    task.Section,
			"is_flat_task":     task.StepsTotal == 0,
		}
	}

	return nil
}

func (s *Server) handleTaskList(req *MCPRequest) {
	if _, ok := s.ensureIntent(req, "task_list", execution.ActionTaskList); !ok {
		return
	}
	if s.taskOperator == nil {
		s.sendToolError(req.ID, -32603, "Task operator unavailable", "task_list")
		return
	}

	tasks, err := s.taskOperator.List()
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to list tasks: %v", err), "task_list")
		return
	}

	s.sendJSONResult(req.ID, tasks, "task_list")
}

func (s *Server) handleTaskAdd(req *MCPRequest, args json.RawMessage) {
	if args == nil {
		s.sendToolError(req.ID, -32602, "Missing arguments for task_add", "task_add")
		return
	}
	if _, ok := s.ensureIntent(req, "task_add", execution.ActionTaskAdd); !ok {
		return
	}
	if s.taskOperator == nil {
		s.sendToolError(req.ID, -32603, "Task operator unavailable", "task_add")
		return
	}

	var payload struct {
		Title    string                    `json:"title"`
		Section  string                    `json:"section"`
		Subtasks []server.TaskSubtaskInput `json:"subtasks"`
	}
	if err := json.Unmarshal(args, &payload); err != nil {
		s.sendToolError(req.ID, -32602, "Invalid arguments for task_add", "task_add")
		return
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		s.sendToolError(req.ID, -32602, "Task title is required", "task_add")
		return
	}

	def := tasks.TaskDefinition{
		Title:    title,
		Section:  strings.TrimSpace(payload.Section),
		Subtasks: server.BuildSubtaskSpecs(payload.Subtasks),
	}

	task, err := s.taskOperator.Add(def)
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to add task: %v", err), "task_add")
		return
	}

	s.sendJSONResult(req.ID, task, "task_add")
}

func (s *Server) handleTaskUpdate(req *MCPRequest, args json.RawMessage) {
	if args == nil {
		s.sendToolError(req.ID, -32602, "Missing arguments for task_update", "task_update")
		return
	}
	if _, ok := s.ensureIntent(req, "task_update", execution.ActionTaskUpdate); !ok {
		return
	}
	if s.taskOperator == nil {
		s.sendToolError(req.ID, -32603, "Task operator unavailable", "task_update")
		return
	}

	var payload struct {
		TaskID   string                     `json:"task_id"`
		Title    *string                    `json:"title"`
		Subtasks *[]server.TaskSubtaskInput `json:"subtasks"`
	}
	if err := json.Unmarshal(args, &payload); err != nil {
		s.sendToolError(req.ID, -32602, "Invalid arguments for task_update", "task_update")
		return
	}

	taskID := strings.TrimSpace(payload.TaskID)
	if taskID == "" {
		s.sendToolError(req.ID, -32602, "task_id is required", "task_update")
		return
	}

	update := tasks.TaskUpdate{}
	if payload.Title != nil {
		trimmed := strings.TrimSpace(*payload.Title)
		update.Title = &trimmed
	}
	if payload.Subtasks != nil {
		specs := server.BuildSubtaskSpecs(*payload.Subtasks)
		update.Subtasks = &specs
	}

	task, err := s.taskOperator.Update(taskID, update)
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to update task: %v", err), "task_update")
		return
	}

	s.sendJSONResult(req.ID, task, "task_update")
}

func (s *Server) handleTaskComplete(req *MCPRequest, args json.RawMessage) {
	if args == nil {
		s.sendToolError(req.ID, -32602, "Missing arguments for task_complete", "task_complete")
		return
	}
	if _, ok := s.ensureIntent(req, "task_complete", execution.ActionTaskComplete); !ok {
		return
	}
	if s.taskOperator == nil {
		s.sendToolError(req.ID, -32603, "Task operator unavailable", "task_complete")
		return
	}

	var payload struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(args, &payload); err != nil {
		s.sendToolError(req.ID, -32602, "Invalid arguments for task_complete", "task_complete")
		return
	}

	taskID := strings.TrimSpace(payload.TaskID)
	if taskID == "" {
		s.sendToolError(req.ID, -32602, "task_id is required", "task_complete")
		return
	}

	task, err := s.taskOperator.Complete(taskID)
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to complete task: %v", err), "task_complete")
		return
	}

	s.sendJSONResult(req.ID, task, "task_complete")
}

func (s *Server) handleArtifactCreate(req *MCPRequest, args json.RawMessage) {
	if args == nil {
		s.sendToolError(req.ID, -32602, "Missing arguments for artifact_create", "artifact_create")
		return
	}
	if _, ok := s.ensureIntent(req, "artifact_create", execution.ActionArtifactCreate); !ok {
		return
	}

	var payload struct {
		Path       string `json:"path"`
		Content    string `json:"content"`
		Overwrite  *bool  `json:"overwrite"`
		LinkTaskID string `json:"link_task_id"`
	}
	if err := json.Unmarshal(args, &payload); err != nil {
		s.sendToolError(req.ID, -32602, "Invalid arguments for artifact_create", "artifact_create")
		return
	}

	if strings.TrimSpace(payload.Path) == "" {
		s.sendToolError(req.ID, -32602, "path is required", "artifact_create")
		return
	}

	overwrite := true
	if payload.Overwrite != nil {
		overwrite = *payload.Overwrite
	}

	result, err := server.WriteArtifact(s.app.Project.Path, payload.Path, payload.Content, overwrite, payload.LinkTaskID)
	if err != nil {
		if strings.Contains(err.Error(), "escapes workspace") || strings.Contains(err.Error(), "must be relative") {
			s.recordEnforcementEvent(enforcement.Event{
				Code:     server.EnforcementCodeArtifactPathEscape,
				Boundary: string(execution.ActionArtifactCreate),
				Mode:     string(s.modeCtrl.Mode()),
				Outcome:  enforcement.OutcomeBlock,
				Details:  err.Error(),
			})
		}
		s.sendToolError(req.ID, -32603, fmt.Sprintf("artifact_create failed: %v", err), "artifact_create")
		return
	}

	s.modeCtrl.RecordFileEdit(payload.Path)
	s.sendJSONResult(req.ID, result, "artifact_create")
}

func (s *Server) handleArtifactRead(req *MCPRequest, args json.RawMessage) {
	if args == nil {
		s.sendToolError(req.ID, -32602, "Missing arguments for artifact_read", "artifact_read")
		return
	}
	if _, ok := s.ensureIntent(req, "artifact_read", execution.ActionArtifactRead); !ok {
		return
	}

	var payload struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &payload); err != nil {
		s.sendToolError(req.ID, -32602, "Invalid arguments for artifact_read", "artifact_read")
		return
	}

	if strings.TrimSpace(payload.Path) == "" {
		s.sendToolError(req.ID, -32602, "path is required", "artifact_read")
		return
	}

	result, err := server.ReadArtifact(s.app.Project.Path, payload.Path)
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("artifact_read failed: %v", err), "artifact_read")
		return
	}

	s.sendJSONResult(req.ID, result, "artifact_read")
}

func (s *Server) handleArtifactList(req *MCPRequest, args json.RawMessage) {
	if _, ok := s.ensureIntent(req, "artifact_list", execution.ActionArtifactList); !ok {
		return
	}

	var payload struct {
		Pattern string `json:"pattern"`
	}
	if args != nil {
		if err := json.Unmarshal(args, &payload); err != nil {
			s.sendToolError(req.ID, -32602, "Invalid arguments for artifact_list", "artifact_list")
			return
		}
	}

	entries, err := server.ListArtifacts(s.app.Project.Path, payload.Pattern)
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("artifact_list failed: %v", err), "artifact_list")
		return
	}

	s.sendJSONResult(req.ID, entries, "artifact_list")
}

func (s *Server) sendJSONResult(id *int, payload interface{}, tool string) {
	data, err := json.Marshal(payload)
	if err != nil {
		s.sendToolError(id, -32603, fmt.Sprintf("Failed to marshal %s result: %v", tool, err), tool)
		return
	}
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": string(data),
				},
			},
		},
		"id": id,
	}
	s.sendResponse(response)
}

// checkAndAutoCreateTaskFile detects multi-step work and creates inert tinyTasks.md if needed
func (s *Server) checkAndAutoCreateTaskFile(context string) error {
	exists, err := s.taskManager.Exists()
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if !tasks.ShouldAutoCreateTaskFile(context) {
		return nil
	}

	return s.taskManager.CreateInertFile()
}

func (s *Server) handleMemoryClaimSuccess(req *MCPRequest, args json.RawMessage) {
	if args == nil {
		s.sendToolError(req.ID, -32602, "Missing arguments for memory_claim_success", "memory_claim_success")
		return
	}

	if _, ok := s.ensureIntent(req, "memory_claim_success", execution.ActionClaimSuccess); !ok {
		return
	}

	var payload struct {
		Success  bool   `json:"success"`
		Enforced bool   `json:"enforced"`
		Boundary string `json:"boundary"`
		Details  string `json:"details"`
	}
	if err := json.Unmarshal(args, &payload); err != nil {
		s.sendToolError(req.ID, -32602, "Invalid arguments for memory_claim_success", "memory_claim_success")
		return
	}

	if payload.Boundary == "" {
		payload.Boundary = "claimed_success"
	}

	if s.enforcer != nil && payload.Success {
		s.enforcer.RecordClaimedSuccess()
		if !payload.Enforced {
			s.recordEnforcementEvent(enforcement.Event{
				Code:     server.EnforcementCodeClaimViolation,
				Boundary: payload.Boundary,
				Mode:     string(s.modeCtrl.Mode()),
				Outcome:  enforcement.OutcomeViolation,
				Details:  payload.Details,
			})
		}
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Claim recorded: success=%t enforced=%t", payload.Success, payload.Enforced),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}

// handleMemoryQuery handles memory query requests
func (s *Server) handleMemoryQuery(req *MCPRequest, args json.RawMessage) {
	var queryReq struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.Unmarshal(args, &queryReq); err != nil {
		s.sendToolError(req.ID, -32602, "Invalid arguments for memory_query", "memory_query")
		return
	}

	if _, ok := s.ensureIntent(req, "memory_query", execution.ActionMemoryQuery); !ok {
		return
	}

	if queryReq.Limit == 0 {
		queryReq.Limit = 10
	}

	results, err := s.recallEngine.Recall(recall.RecallOptions{
		ProjectID: s.app.Project.ID,
		Query:     queryReq.Query,
		MaxItems:  queryReq.Limit,
	})
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("Query failed: %v", err), "memory_query")
		return
	}

	// Mark that memory recall was performed (required before mutations)
	if s.intentGate != nil {
		s.intentGate.RecordRecall()
	}

	// Apply CoVe recall filtering if enabled
	if s.coveVerifier != nil && len(results) > 0 {
		// Convert results to CoVe format
		recallMemories := make([]cove.RecallMemory, 0, len(results))
		for i, result := range results {
			recallMemories = append(recallMemories, cove.RecallMemory{
				ID:      fmt.Sprintf("%d", i),
				Type:    string(result.Memory.Type),
				Summary: result.Memory.Summary,
				Detail:  result.Memory.Detail,
			})
		}

		// Apply CoVe filtering
		filtered, err := s.coveVerifier.FilterRecall(s.ctx, recallMemories, queryReq.Query)
		if err == nil {
			// Build a map of accepted indices
			accepted := make(map[int]bool)
			for _, m := range filtered {
				var idx int
				fmt.Sscanf(m.ID, "%d", &idx)
				accepted[idx] = true
			}

			// Filter results
			filteredResults := make([]recall.RecallResult, 0, len(filtered))
			for i, result := range results {
				if accepted[i] {
					filteredResults = append(filteredResults, result)
				}
			}
			results = filteredResults
		}
	}

	// Apply task safety filtering: filter out tasks that shouldn't be acted upon
	explicitTaskContinuation := s.taskService.HasExplicitTaskContinuationIntent(queryReq.Query)
	safeResults := s.filterRecallResultsForTaskSafety(results, queryReq.Query, explicitTaskContinuation)
	results = safeResults

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

	if _, ok := s.ensureIntent(req, "memory_recent", execution.ActionMemoryQuery); !ok {
		return
	}

	memories, err := s.memoryService.GetAllMemories(s.app.Project.ID) // In real impl, get from context
	if err != nil {
		s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to get recent memories: %v", err), "memory_recent")
		return
	}

	// Mark that memory recall was performed (required before mutations)
	if s.intentGate != nil {
		s.intentGate.RecordRecall()
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
		Type           string `json:"type"`
		Summary        string `json:"summary"`
		Detail         string `json:"detail"`
		Key            string `json:"key"`
		Source         string `json:"source"`
		Classification string `json:"classification"` // Optional classification field
		Evidence       []struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		} `json:"evidence"`
	}

	if _, ok := s.ensureIntent(req, "memory_write", execution.ActionMemoryWrite); !ok {
		return
	}

	if err := json.Unmarshal(args, &writeReq); err != nil {
		s.sendToolError(req.ID, -32602, "Invalid arguments for memory.write", "memory_write")
		return
	}

	// Validate memory type
	memType := memory.Type(writeReq.Type)
	if !memType.IsValid() {
		s.sendToolError(req.ID, -32602, "Invalid memory type", "memory_write")
		return
	}

	if isTinyTasksPath(writeReq.Source) {
		s.sendToolError(req.ID, -32603, "Direct tinyTasks.md mutations are not allowed; use the task_* tools", "memory_write")
		return
	}

	if s.intentGate != nil {
		if memType == memory.Task {
			if err := s.intentGate.EnforceMode(execution.ActionTaskMutation, execution.ModeStrict); err != nil {
				s.sendToolError(req.ID, -32603, err.Error(), "memory_write")
				return
			}
		} else {
			if err := s.intentGate.EnforceMode(execution.ActionMemoryWrite, execution.ModeGuarded); err != nil {
				s.sendToolError(req.ID, -32603, err.Error(), "memory_write")
				return
			}
		}
	}

	// AUTO-CREATE: Check if tinyTasks.md should be created for multi-step work
	// Use the memory summary and detail to detect multi-step patterns
	workContext := fmt.Sprintf("%s %s", writeReq.Summary, writeReq.Detail)
	_ = s.checkAndAutoCreateTaskFile(workContext) // Best-effort, don't fail on error

	// Optional CoVe filtering for non-fact writes (fail-safe on error)
	if s.coveVerifier != nil && memType != memory.Fact {
		candidates := []cove.CandidateMemory{
			{
				ID:      "0",
				Type:    string(memType),
				Summary: writeReq.Summary,
				Detail:  writeReq.Detail,
				Score:   1.0, // Write operations default to pass-through (no semantic score)
			},
		}
		verified, err := s.coveVerifier.VerifyCandidates(candidates)
		if err == nil && len(verified) == 0 {
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"result": map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": "Memory discarded by CoVe (confidence below threshold).",
						},
					},
				},
				"id": req.ID,
			}
			s.sendResponse(response)
			return
		}
	}

	newMemory := &memory.Memory{
		ProjectID: s.app.Project.ID, // Use s.app.Project.ID
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
	if writeReq.Classification != "" {
		newMemory.Classification = &writeReq.Classification
	}

	if memType == memory.Fact {
		if len(writeReq.Evidence) == 0 {
			s.recordEnforcementEvent(enforcement.Event{
				Code:     server.EnforcementCodeFactEvidence,
				Boundary: string(execution.ActionFactPromotion),
				Mode:     string(s.modeCtrl.Mode()),
				Outcome:  enforcement.OutcomeBlock,
				Details:  "evidence required for fact creation",
			})
			s.sendToolError(req.ID, -32603, "Fact creation requires verified evidence", "memory_write")
			return
		}
		var inputs []memory.EvidenceInput
		for _, ev := range writeReq.Evidence {
			inputs = append(inputs, memory.EvidenceInput{
				Type:    ev.Type,
				Content: ev.Content,
			})
		}

		verify := func(evidenceType, content string) (bool, error) {
			return evidence.VerifyEvidence(evidenceType, content, s.config)
		}
		if err := s.memoryService.CreateFactWithEvidence(newMemory, inputs, verify); err != nil {
			s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to create fact: %v", err), "memory_write")
			return
		}
	} else {
		if err := s.memoryService.CreateMemory(newMemory); err != nil {
			s.sendToolError(req.ID, -32603, fmt.Sprintf("Failed to create memory: %v", err), "memory_write")
			return
		}
	}

	if writeReq.Source != "" {
		s.modeCtrl.RecordFileEdit(writeReq.Source)
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Memory created successfully with ID: %d\nType: %s\nSummary: %s",
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
	if _, ok := s.ensureIntent(req, "memory_stats", execution.ActionMemoryStats); !ok {
		return
	}
	// Get all memories to calculate stats
	memories, err := s.memoryService.GetAllMemories(s.app.Project.ID)
	if err != nil {
		s.app.Core.Logger.Error("Failed to get memory stats for MCP", zap.Error(err))
		s.sendToolError(req.ID, -32603, "Failed to retrieve memory statistics", "memory_stats")
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

	// Add CoVe statistics if available
	if coveStats := s.extractor.GetCoVeStats(); coveStats != nil {
		content.WriteString("\n\nCoVe (Chain-of-Verification) Statistics:\n")
		content.WriteString(fmt.Sprintf("  Candidates evaluated: %d\n", coveStats.CandidatesEvaluated))
		content.WriteString(fmt.Sprintf("  Candidates discarded: %d\n", coveStats.CandidatesDiscarded))
		if coveStats.CandidatesEvaluated > 0 {
			content.WriteString(fmt.Sprintf("  Average confidence: %.2f\n", coveStats.AvgConfidence))
			discardRate := float64(coveStats.CandidatesDiscarded) / float64(coveStats.CandidatesEvaluated) * 100
			content.WriteString(fmt.Sprintf("  Discard rate: %.1f%%\n", discardRate))
		}
		content.WriteString(fmt.Sprintf("  Errors: %d\n", coveStats.CoVeErrors))
		content.WriteString(fmt.Sprintf("  Last updated: %s\n", coveStats.LastUpdated.Format(time.RFC3339)))
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
	if _, ok := s.ensureIntent(req, "memory_health", execution.ActionMemoryHealth); !ok {
		return
	}
	// Check database connectivity
	if err := s.db.GetConnection().Ping(); err != nil {
		s.app.Core.Logger.Error("Database health check failed for MCP", zap.Error(err))
		s.sendToolError(req.ID, -32603, "Database health check failed", "memory_health")
		return
	}

	// Check if we can perform a simple query
	if _, err := s.memoryService.GetAllMemories(s.app.Project.ID); err != nil {
		s.app.Core.Logger.Error("Memory service health check failed for MCP", zap.Error(err))
		s.sendToolError(req.ID, -32603, "Memory service health check failed", "memory_health")
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
	if _, ok := s.ensureIntent(req, "memory_doctor", execution.ActionMemoryDoctor); !ok {
		return
	}
	doctorRunner := doctor.NewRunnerWithMode(s.app.Core.Config, s.app.Core.DB, s.app.Project.ID, s.app.Memory, doctor.MCPMode)
	diagnostics := doctorRunner.RunAll()

	var content strings.Builder
	content.WriteString("tinyMem Diagnostics Report\n\n")

	if len(diagnostics.Issues) > 0 { // Corrected: use len(diagnostics.Issues)
		content.WriteString(fmt.Sprintf("⚠️  Status: %d issue(s) detected\n\n", len(diagnostics.Issues)))
		content.WriteString("Issues:\n")
		for i, issue := range diagnostics.Issues { // Corrected: issue is already a string
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, issue))
		}
		content.WriteString("\nRecommendations:\n") // Generic recommendations as Diagnostics has no Recommendations field
		content.WriteString("- Check the .tinyMem directory permissions\n")
		content.WriteString("- Verify database file is not corrupted\n")
		content.WriteString("- Ensure sufficient disk space is available\n")
		content.WriteString("- Review configuration settings\n")
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
	s.app.Core.Logger.Info("MCP server received shutdown request.")
	s.Close()

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  map[string]interface{}{},
		"id":      req.ID,
	}

	s.sendResponse(response)
	// Do not os.Exit(0) here. Let the main goroutine handle the exit
	// after all deferred cleanups are done.
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

// Close gracefully shuts down the MCP server and its resources.
func (s *Server) Close() {
	// Close the recall engine to flush any pending metrics
	if s.recallEngine != nil {
		s.recallEngine.Close()
	}
	s.cancel()
}

// sendResponse sends a successful response
func (s *Server) sendResponse(response map[string]interface{}) {
	responseBytes, err := json.Marshal(response)
	if err != nil {
		s.app.Core.Logger.Error("Failed to marshal MCP response", zap.Error(err))
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

func (s *Server) sendToolError(id *int, code int, message, tool string) {
	if tool != "" && s.modeCtrl != nil {
		s.modeCtrl.RecordFailure(tool)
	}
	s.sendError(id, code, message)
}

func (s *Server) recordEnforcementEvent(evt enforcement.Event) {
	if s.enforcer != nil {
		s.enforcer.RecordEvent(evt)
	}
}

func (s *Server) ensureIntent(req *MCPRequest, tool string, action execution.Action) (intent.Definition, bool) {
	if s.intentGate == nil {
		s.sendToolError(req.ID, -32603, "Intent gate unavailable", tool)
		return intent.Definition{}, false
	}

	def, err := s.intentGate.EnsureIntent(tool, action)
	if err != nil {
		s.sendToolError(req.ID, -32603, err.Error(), tool)
		return def, false
	}

	return def, true
}

func isTinyTasksPath(path string) bool {
	if path == "" {
		return false
	}

	return strings.EqualFold(filepath.Base(path), "tinyTasks.md")
}
