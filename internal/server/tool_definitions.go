package server

import (
	"github.com/daverage/tinymem/internal/intent"
)

// ToolInputSchema describes the input schema for a tool.
type ToolInputSchema map[string]interface{}

// ToolSpec describes a single tool exposed via MCP.
type ToolSpec struct {
	Name        string
	Description string
	InputSchema ToolInputSchema
}

var toolSpecs = []ToolSpec{
	{
		Name:        "memory_query",
		Description: "Search memories via lexical CoVe recall (Boundary: ActionMemoryQuery; mode: PASSIVE+; outcomes: ALLOW/BLOCK).",
		InputSchema: ToolInputSchema{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query string",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results",
				},
			},
		},
	},
	{
		Name:        "memory_recent",
		Description: "Fetch recent memories with no minimum mode requirement; observation only.",
		InputSchema: ToolInputSchema{
			"type": "object",
			"properties": map[string]interface{}{
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Number of recent memories to retrieve",
				},
			},
		},
	},
	{
		Name:        "memory_write",
		Description: "Store a memory entry (facts require evidence and prior recall in GUARDED/STRICT modes).",
		InputSchema: ToolInputSchema{
			"type": "object",
			"properties": map[string]interface{}{
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Memory type (decision, fact, constraint, etc.)",
				},
				"summary": map[string]interface{}{
					"type":        "string",
					"description": "Brief summary of the memory",
				},
				"detail": map[string]interface{}{
					"type":        "string",
					"description": "Detailed context for the memory",
				},
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Optional key to deduplicate memories",
				},
				"source": map[string]interface{}{
					"type":        "string",
					"description": "Optional source path or identifier",
				},
				"classification": map[string]interface{}{
					"type":        "string",
					"description": "Optional classification for recall precision",
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
		Name:        "memory_stats",
		Description: "Get statistics about stored memories",
		InputSchema: ToolInputSchema{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "memory_health",
		Description: "Check the health status of the memory system",
		InputSchema: ToolInputSchema{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "memory_doctor",
		Description: "Run diagnostics on the memory system",
		InputSchema: ToolInputSchema{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "memory_eval_stats",
		Description: "Get detailed metrics for evaluation and scoring.",
		InputSchema: ToolInputSchema{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "memory_run_metadata",
		Description: "Read the enforcement run metadata (execution_mode, events, proven counts).",
		InputSchema: ToolInputSchema{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "memory_claim_success",
		Description: "Report whether a claimed success was observed and enforced; adversarial claims without enforcement get flagged.",
		InputSchema: ToolInputSchema{
			"type": "object",
			"properties": map[string]interface{}{
				"success": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether a success was claimed",
				},
				"enforced": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether enforcement observed the success",
				},
				"boundary": map[string]interface{}{
					"type":        "string",
					"description": "Optional enforcement boundary label",
				},
				"details": map[string]interface{}{
					"type":        "string",
					"description": "Optional textual details",
				},
			},
			"required": []string{"success", "enforced"},
		},
	},
	{
		Name:        "memory_set_mode",
		Description: "Declare PASSIVE, GUARDED, or STRICT before mutations; this is the intent gatekeeper for all memory writes.",
		InputSchema: ToolInputSchema{
			"type": "object",
			"properties": map[string]interface{}{
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "Execution mode to set (PASSIVE, GUARDED, STRICT)",
				},
			},
		},
	},
	{
		Name:        "memory_check_task_authority",
		Description: "Check tinyTasks.md status and return authorization for multi-step work. Returns file existence, unchecked tasks, and authorization status.",
		InputSchema: ToolInputSchema{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "task_list",
		Description: "List the tasks currently tracked in tinyTasks.md.",
		InputSchema: ToolInputSchema{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "task_add",
		Description: "Add a new task via the TaskManager; the server owns the file write.",
		InputSchema: ToolInputSchema{
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
		Name:        "task_update",
		Description: "Update a task title or subtasks through the TaskManager.",
		InputSchema: ToolInputSchema{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "string",
					"description": "Task identifier",
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
		Name:        "task_complete",
		Description: "Mark a task as completed through the TaskManager.",
		InputSchema: ToolInputSchema{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "string",
					"description": "Task identifier",
				},
			},
			"required": []string{"task_id"},
		},
	},
}

// ToolSpecs exposes a copy of the tool specifications for tests.
func ToolSpecs() []ToolSpec {
	return append([]ToolSpec(nil), toolSpecs...)
}

// BuildToolList renders the tool specifications with metadata for JSON responses.
func BuildToolList() []map[string]interface{} {
	specs := ToolSpecs()
	result := make([]map[string]interface{}, len(specs))
	for i, spec := range specs {
		entry := map[string]interface{}{
			"name":        spec.Name,
			"description": spec.Description,
			"inputSchema": spec.InputSchema,
		}
		ToolMetadata(spec.Name, entry)
		result[i] = entry
	}
	return result
}

// ToolMetadata augments a tool map with intent metadata.
func ToolMetadata(name string, base map[string]interface{}) {
	if def, ok := intent.Lookup(name); ok {
		for k, v := range def.Metadata() {
			base[k] = v
		}
	}
}
