package intent

import "github.com/daverage/tinymem/internal/execution"

type Category string

const (
	CategoryRecall      Category = "recall"
	CategoryMemory      Category = "memory_write"
	CategoryTask        Category = "task"
	CategoryDiagnostics Category = "diagnostics"
	CategoryMode        Category = "mode_declaration"
)

// Definition describes the declarative intent metadata attached to a tool.
type Definition struct {
	Name           string
	Category       Category
	RequiredMode   execution.Mode
	RequiresRecall bool
	Description    string
	Scope          string
	SideEffects    []string
}

// Lookup returns the metadata registered for a tool name.
func Lookup(name string) (Definition, bool) {
	def, ok := ToolDefinitions[name]
	return def, ok
}

// Metadata produces a generic representation of the definition for serialization.
func (d Definition) Metadata() map[string]interface{} {
	m := map[string]interface{}{
		"intent_category":        string(d.Category),
		"intent_required_mode":   string(d.RequiredMode),
		"intent_requires_recall": d.RequiresRecall,
	}
	if d.Description != "" {
		m["intent_description"] = d.Description
	}
	if d.Scope != "" {
		m["intent_scope"] = d.Scope
	}
	if len(d.SideEffects) > 0 {
		m["intent_side_effects"] = d.SideEffects
	}
	return m
}

var ToolDefinitions = map[string]Definition{
	"memory_query": {
		Name:         "memory_query",
		Category:     CategoryRecall,
		RequiredMode: execution.ModePassive,
		Description:  "Lexical recall search (observation only).",
		Scope:        "memory",
		SideEffects:  []string{"marks recall performed"},
	},
	"memory_recent": {
		Name:         "memory_recent",
		Category:     CategoryRecall,
		RequiredMode: execution.ModePassive,
		Description:  "Fetch recent memories (observation only).",
		Scope:        "memory",
		SideEffects:  []string{"marks recall performed"},
	},
	"memory_write": {
		Name:           "memory_write",
		Category:       CategoryMemory,
		RequiredMode:   execution.ModeGuarded,
		RequiresRecall: true,
		Description:    "Create or update memories when the agent has a valid guarded/strict intent.",
		Scope:          "memory",
		SideEffects:    []string{"mutates memory ledger", "may auto-create tinyTasks.md"},
	},
	"memory_stats": {
		Name:         "memory_stats",
		Category:     CategoryDiagnostics,
		RequiredMode: execution.ModePassive,
		Description:  "Read-only statistics about the memory store.",
		Scope:        "memory",
	},
	"memory_health": {
		Name:         "memory_health",
		Category:     CategoryDiagnostics,
		RequiredMode: execution.ModePassive,
		Description:  "Health probe covering database connectivity and integrity.",
		Scope:        "memory",
	},
	"memory_doctor": {
		Name:         "memory_doctor",
		Category:     CategoryDiagnostics,
		RequiredMode: execution.ModePassive,
		Description:  "Run maintenance diagnostics and produce guided output.",
	},
	"memory_eval_stats": {
		Name:         "memory_eval_stats",
		Category:     CategoryDiagnostics,
		RequiredMode: execution.ModePassive,
		Description:  "Return enforcement and evaluation metrics for the current session.",
	},
	"memory_run_metadata": {
		Name:         "memory_run_metadata",
		Category:     CategoryDiagnostics,
		RequiredMode: execution.ModePassive,
		Description:  "Expose enforcement metadata (mode, events, counts).",
	},
	"memory_claim_success": {
		Name:         "memory_claim_success",
		Category:     CategoryDiagnostics,
		RequiredMode: execution.ModePassive,
		Description:  "Record whether claimed successes were enforced.",
	},
	"memory_set_mode": {
		Name:         "memory_set_mode",
		Category:     CategoryMode,
		RequiredMode: execution.ModePassive,
		Description:  "Declare the desired execution mode before mutations.",
	},
	"memory_check_task_authority": {
		Name:         "memory_check_task_authority",
		Category:     CategoryTask,
		RequiredMode: execution.ModePassive,
		Description:  "Return tinyTasks.md authorization signals without mutating state.",
	},
	"task_list": {
		Name:         "task_list",
		Category:     CategoryTask,
		RequiredMode: execution.ModePassive,
		Description:  "List tasks parsed from tinyTasks.md (server-managed).",
		Scope:        "tasks",
		SideEffects:  []string{"reads tinyTasks.md"},
	},
	"task_add": {
		Name:           "task_add",
		Category:       CategoryTask,
		RequiredMode:   execution.ModeStrict,
		Description:    "Append a new task to tinyTasks.md via the TaskManager.",
		Scope:          "tasks",
		SideEffects:    []string{"mutates tinyTasks.md"},
		RequiresRecall: true,
	},
	"task_update": {
		Name:           "task_update",
		Category:       CategoryTask,
		RequiredMode:   execution.ModeStrict,
		Description:    "Modify an existing task via the TaskManager.",
		Scope:          "tasks",
		SideEffects:    []string{"mutates tinyTasks.md"},
		RequiresRecall: true,
	},
	"task_complete": {
		Name:           "task_complete",
		Category:       CategoryTask,
		RequiredMode:   execution.ModeStrict,
		Description:    "Mark a task as completed via the TaskManager.",
		Scope:          "tasks",
		SideEffects:    []string{"mutates tinyTasks.md"},
		RequiresRecall: true,
	},
}
