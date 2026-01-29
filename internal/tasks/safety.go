package tasks

import (
	"strings"
)

// TaskSafetyEnforcer enforces the safety rules for task operations
type TaskSafetyEnforcer struct {
	allowedContinuationPhrases []string
}

// NewTaskSafetyEnforcer creates a new safety enforcer
func NewTaskSafetyEnforcer() *TaskSafetyEnforcer {
	return &TaskSafetyEnforcer{
		allowedContinuationPhrases: []string{
			"continue tasks",
			"resume previous tasks", 
			"finish the remaining tasks",
			"pick up where we left off",
			"continue where we left off",
			"resume tasks",
			"continue the tasks",
			"resume the tasks",
			"complete the remaining tasks",
			"finish remaining tasks",
		},
	}
}

// HasExplicitTaskContinuationIntent checks if the user explicitly requested to continue tasks
func (e *TaskSafetyEnforcer) HasExplicitTaskContinuationIntent(query string) bool {
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	
	for _, phrase := range e.allowedContinuationPhrases {
		if strings.Contains(lowerQuery, phrase) {
			return true
		}
	}
	
	return false
}

// CanActOnTask determines if an action can be performed on a task
func (e *TaskSafetyEnforcer) CanActOnTask(task *Task, query string) bool {
	// If the task is already completed, we can't act on it anyway
	if task.Completed {
		return false
	}
	
	// If explicit intent is given, allow acting on dormant tasks
	if e.HasExplicitTaskContinuationIntent(query) {
		return true
	}
	
	// Otherwise, only allow acting on tasks that are already active
	return task.Mode == TaskModeActive
}

// CanCreateNewTask determines if a new task can be created without affecting existing tasks
func (e *TaskSafetyEnforcer) CanCreateNewTask(query string) bool {
	// Creating a new task is always allowed as long as the query doesn't explicitly
	// request continuing old tasks (which would be a different operation)
	return !e.HasExplicitTaskContinuationIntent(query)
}