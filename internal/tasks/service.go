package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/storage"
)

// Task mode enables deterministic task tracking using a file-authoritative approach.
// The tinyTasks.md file serves as the authoritative source of task state, while
// TinyMem memory entries serve only as rebuildable summaries. This ensures that:
// 1. Task state is preserved even if memory is lost
// 2. Humans can directly edit task state by modifying the file
// 3. No task inference occurs - completion is explicitly defined in the file
// 4. Task identity remains stable regardless of text changes

// Service handles task operations
type Service struct {
	db        *storage.DB
	memory    *memory.Service
	projectID string
	enforcer  *TaskSafetyEnforcer
	manager   *TaskManager
}

// NewService creates a new task service
func NewService(db *storage.DB, memoryService *memory.Service, projectID string, manager *TaskManager) *Service {
	return &Service{
		db:        db,
		memory:    memoryService,
		projectID: projectID,
		enforcer:  NewTaskSafetyEnforcer(),
		manager:   manager,
	}
}

// CreateOrUpdateTask creates or updates a task in memory
func (s *Service) CreateOrUpdateTask(task *Task) error {
	// Check if a task with this ID already exists
	existingTask, err := s.GetTaskByKey(task.ID)
	if err != nil {
		// If the error is not "not found", return it
		if err.Error() != fmt.Sprintf("memory with key %s not found", task.ID) {
			return err
		}
		// If not found, continue to create new task
		existingTask = nil
	}

	// Prepare memory object
	memObj := &memory.Memory{
		ProjectID: s.projectID,
		Type:      memory.Task,
		Summary:   task.Title,
		Detail: fmt.Sprintf("Section: %s\nIndex: %d\nSteps Total: %d\nSteps Done: %d\nCompleted: %t\nState: %s\nMode: %s\nFile Path: %s\nLast Updated: %s",
			task.Section, task.Index, task.StepsTotal, task.StepsDone, task.Completed, task.State, task.Mode, task.FilePath, task.LastUpdated),
		Key:    &task.ID,
		Source: &task.FilePath,
	}

	// Add extra fields as needed
	extraDetails := fmt.Sprintf("Section: %s\nIndex: %d\nSteps Total: %d\nSteps Done: %d\nCompleted: %t\nState: %s\nMode: %s\nFile Path: %s\nLast Updated: %s",
		task.Section, task.Index, task.StepsTotal, task.StepsDone, task.Completed, task.State, task.Mode, task.FilePath, task.LastUpdated)

	if task.LastSeenHash != "" {
		extraDetails += fmt.Sprintf("\nHash: %s", task.LastSeenHash)
	}

	memObj.Detail = extraDetails

	if existingTask != nil {
		// Update existing task
		memObj.ID = existingTask.ID
		return s.memory.UpdateMemory(memObj)
	} else {
		// Create new task
		return s.memory.CreateMemory(memObj)
	}
}

// CanActOnTask checks if an action can be performed on a task based on safety rules
func (s *Service) CanActOnTask(taskID string, query string) (bool, error) {
	task, err := s.GetTaskByKey(taskID)
	if err != nil {
		return false, err
	}

	// Parse the task details to extract mode
	mode := TaskModeDormant // Default
	if task.Detail != "" {
		lines := strings.Split(task.Detail, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Mode: ") {
				modeStr := strings.TrimPrefix(line, "Mode: ")
				mode = TaskMode(modeStr)
				break
			}
		}
	}

	// Create a temporary task object for the safety check
	tempTask := &Task{
		ID:        taskID,
		Completed: strings.Contains(task.Detail, "Completed: true"),
		State:     TaskStateOpen, // Default
		Mode:      mode,
	}

	return s.enforcer.CanActOnTask(tempTask, query), nil
}

// CanCreateNewTask checks if a new task can be created based on the query
func (s *Service) CanCreateNewTask(query string) bool {
	return s.enforcer.CanCreateNewTask(query)
}

// HasExplicitTaskContinuationIntent checks if the query explicitly requests task continuation
func (s *Service) HasExplicitTaskContinuationIntent(query string) bool {
	return s.enforcer.HasExplicitTaskContinuationIntent(query)
}

// GetTaskByKey retrieves a task by its key
func (s *Service) GetTaskByKey(key string) (*memory.Memory, error) {
	// Search for memories with the specific key
	allMemories, err := s.memory.GetAllMemories(s.projectID)
	if err != nil {
		return nil, err
	}

	for _, mem := range allMemories {
		if mem.Key != nil && *mem.Key == key && mem.Type == memory.Task {
			return mem, nil
		}
	}

	return nil, fmt.Errorf("memory with key %s not found", key)
}

// GetAllTasks retrieves all task memories for the project
func (s *Service) GetAllTasks() ([]*memory.Memory, error) {
	allMemories, err := s.memory.GetAllMemories(s.projectID)
	if err != nil {
		return nil, err
	}

	var tasks []*memory.Memory
	for _, mem := range allMemories {
		if mem.Type == memory.Task {
			tasks = append(tasks, mem)
		}
	}

	return tasks, nil
}

// DeleteTask removes a task from memory
func (s *Service) DeleteTask(key string) error {
	tasks, err := s.GetAllTasks()
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.Key != nil && *task.Key == key {
			// Since we don't have a direct delete method, we'll mark as superseded
			// by creating a new version that indicates deletion
			newSummary := "[DELETED] " + task.Summary
			newDetail := "[MARKED AS DELETED] " + task.Detail
			task.Summary = newSummary
			task.Detail = newDetail
			return s.memory.UpdateMemory(task)
		}
	}

	return fmt.Errorf("task with key %s not found", key)
}

// SyncTasksFromFile synchronizes tasks from the tinyTasks.md file to memory
func (s *Service) SyncTasksFromFile(filePath string) error {
	tasks, err := s.manager.List()
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	fileHash, err := s.manager.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash tasks: %w", err)
	}

	currentTime := time.Now().Format(time.RFC3339)
	for _, task := range tasks {
		task.LastUpdated = currentTime
		task.LastSeenHash = fileHash
	}

	currentTasks, err := s.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get current tasks: %w", err)
	}

	currentTaskKeys := make(map[string]*memory.Memory)
	for _, task := range currentTasks {
		if task.Key != nil {
			currentTaskKeys[*task.Key] = task
		}
	}

	for _, task := range tasks {
		if err := s.CreateOrUpdateTask(task); err != nil {
			return fmt.Errorf("failed to create/update task %s: %w", task.ID, err)
		}
		delete(currentTaskKeys, task.ID)
	}

	for key := range currentTaskKeys {
		if err := s.DeleteTask(key); err != nil {
			fmt.Printf("Warning: failed to delete obsolete task %s: %v\n", key, err)
		}
	}

	return s.DetectAndRecoverDrift(filePath)
}

// DetectAndRecoverDrift checks for differences between file and memory and recovers if needed
func (s *Service) DetectAndRecoverDrift(filePath string) error {
	parsedTasks, err := s.manager.List()
	if err != nil {
		return fmt.Errorf("failed to list tasks for drift detection: %w", err)
	}

	memoryTasks, err := s.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks from memory: %w", err)
	}

	fileTaskMap := make(map[string]*Task)
	for _, task := range parsedTasks {
		fileTaskMap[task.ID] = task
	}

	memoryTaskMap := make(map[string]*memory.Memory)
	for _, task := range memoryTasks {
		if task.Key != nil {
			memoryTaskMap[*task.Key] = task
		}
	}

	for key := range memoryTaskMap {
		if _, exists := fileTaskMap[key]; !exists {
			if err := s.DeleteTask(key); err != nil {
				fmt.Printf("Warning: failed to delete obsolete task %s: %v\n", key, err)
			}
		}
	}

	fileHash, err := s.manager.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash tasks for drift detection: %w", err)
	}

	for _, fileTask := range fileTaskMap {
		if memTask, exists := memoryTaskMap[fileTask.ID]; exists {
			completionMismatch := false
			hashMismatch := false

			if memTask.Detail != "" {
				if containsString(memTask.Detail, fmt.Sprintf("Completed: %t", !fileTask.Completed)) {
					completionMismatch = true
				}
				if fileTask.LastSeenHash != "" && !containsString(memTask.Detail, "Hash: "+fileTask.LastSeenHash) {
					hashMismatch = true
				}
			}

			if completionMismatch || hashMismatch {
				if err := s.DeleteTask(fileTask.ID); err != nil {
					fmt.Printf("Warning: failed to delete drifted task %s: %v\n", fileTask.ID, err)
				}

				fileTask.LastSeenHash = fileHash
				fileTask.LastUpdated = time.Now().Format(time.RFC3339)
				if err := s.CreateOrUpdateTask(fileTask); err != nil {
					return fmt.Errorf("failed to recreate task %s: %w", fileTask.ID, err)
				}
			}
		}
	}

	return nil
}

// containsString checks if a string contains a substring
func containsString(str, substr string) bool {
	return strings.Contains(str, substr)
}

// SyncTasks performs synchronization of tasks from file to memory
func (s *Service) SyncTasks(projectPath string) error {
	return s.SyncTasksFromFile(projectPath + "/tinyTasks.md")
}

// GetTaskStatus returns a summary of task status
func (s *Service) GetTaskStatus(projectPath string) (map[string]interface{}, error) {
	tasks, err := s.GetAllTasks()
	if err != nil {
		return nil, err
	}

	totalTasks := len(tasks)
	completedTasks := 0
	incompleteTasks := 0

	for _, task := range tasks {
		if task.Detail != "" && strings.Contains(task.Detail, "Completed: true") {
			completedTasks++
		} else {
			incompleteTasks++
		}
	}

	hasTaskFile, err := s.manager.Exists()
	if err != nil {
		return nil, err
	}

	fileTasks, err := s.manager.List()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	metrics := collectTaskShapeMetrics(fileTasks)

	status := map[string]interface{}{
		"total_tasks":      totalTasks,
		"completed_tasks":  completedTasks,
		"incomplete_tasks": incompleteTasks,
		"has_task_file":    hasTaskFile,
	}

	status["tasks_with_subtasks"] = metrics.TasksWithSubtasks
	status["flat_tasks"] = metrics.FlatTasks
	status["average_steps_per_task"] = metrics.AverageStepsPerTask

	return status, nil
}

type taskShapeMetrics struct {
	TasksWithSubtasks   int
	FlatTasks           int
	AverageStepsPerTask float64
}

func collectTaskShapeMetrics(tasks []*Task) taskShapeMetrics {
	var metrics taskShapeMetrics
	if len(tasks) == 0 {
		return metrics
	}

	totalSteps := 0
	for _, task := range tasks {
		if task.StepsTotal > 0 {
			metrics.TasksWithSubtasks++
		} else {
			metrics.FlatTasks++
		}
		totalSteps += task.StepsTotal
	}

	metrics.AverageStepsPerTask = float64(totalSteps) / float64(len(tasks))
	return metrics
}

// CreateInertTaskFile creates the canonical inert tinyTasks.md template
// This file is intentionally non-authorizing - humans must edit it to add tasks
func CreateInertTaskFile(projectPath string) error {
	manager := NewTaskManager(filepath.Join(projectPath, "tinyTasks.md"))
	return manager.CreateInertFile()
}

// ShouldAutoCreateTaskFile detects if a request implies multi-step work
func ShouldAutoCreateTaskFile(input string) bool {
	input = strings.ToLower(input)

	// Multi-step indicators
	multiStepPatterns := []string{
		"implement",
		"create",
		"build",
		"add",
		"refactor",
		"update",
		"migrate",
		"setup",
		"configure",
	}

	// Count how many action verbs are present
	actionCount := 0
	for _, pattern := range multiStepPatterns {
		if strings.Contains(input, pattern) {
			actionCount++
		}
	}

	// Also check for explicit multi-step indicators
	explicitMultiStep := []string{
		"step",
		"steps",
		"tasks",
		"task list",
		"workflow",
		"series",
		"sequence",
		"multiple",
		"several",
	}

	for _, indicator := range explicitMultiStep {
		if strings.Contains(input, indicator) {
			return true
		}
	}

	// If we see 2+ action verbs, likely multi-step
	if actionCount >= 2 {
		return true
	}

	// Check for conjunction patterns (A and B, A then B, etc.)
	conjunctions := []string{
		" and ",
		" then ",
		" after ",
		" before ",
		" also ",
		" additionally ",
	}

	conjunctionCount := 0
	for _, conj := range conjunctions {
		if strings.Contains(input, conj) {
			conjunctionCount++
		}
	}

	// Multiple conjunctions suggest multi-step work
	return conjunctionCount >= 1 && actionCount >= 1
}
