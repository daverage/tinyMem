package tasks

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
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
}

// NewService creates a new task service
func NewService(db *storage.DB, memoryService *memory.Service, projectID string) *Service {
	return &Service{
		db:        db,
		memory:    memoryService,
		projectID: projectID,
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
		Detail: fmt.Sprintf("Section: %s\nIndex: %d\nSteps Total: %d\nSteps Done: %d\nCompleted: %t\nFile Path: %s\nLast Updated: %s",
			task.Section, task.Index, task.StepsTotal, task.StepsDone, task.Completed, task.FilePath, task.LastUpdated),
		Key:    &task.ID,
		Source: &task.FilePath,
	}

	// Add extra fields as needed
	extraDetails := fmt.Sprintf("Section: %s\nIndex: %d\nSteps Total: %d\nSteps Done: %d\nCompleted: %t\nFile Path: %s\nLast Updated: %s",
		task.Section, task.Index, task.StepsTotal, task.StepsDone, task.Completed, task.FilePath, task.LastUpdated)

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
	// Read and parse the file
	fileBytes, err := readFileBytes(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Compute the file hash for comparison
	fileHash := computeHash(fileBytes)

	// Parse the tasks from the file
	reader := &ByteReader{Data: fileBytes}
	tasks, err := ParseTasks(reader)
	if err != nil {
		return fmt.Errorf("failed to parse tasks: %w", err)
	}

	// Update each task's metadata
	currentTime := time.Now().Format(time.RFC3339)
	for _, task := range tasks {
		task.LastUpdated = currentTime
		task.LastSeenHash = fileHash
	}

	// Get current tasks in memory
	currentTasks, err := s.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get current tasks: %w", err)
	}

	// Create a map of current tasks by key for quick lookup
	currentTaskKeys := make(map[string]*memory.Memory)
	for _, task := range currentTasks {
		if task.Key != nil {
			currentTaskKeys[*task.Key] = task
		}
	}

	// Process each parsed task
	for _, task := range tasks {
		// Create or update the task in memory
		err := s.CreateOrUpdateTask(task)
		if err != nil {
			return fmt.Errorf("failed to create/update task %s: %w", task.ID, err)
		}

		// Remove from the map since it's processed
		delete(currentTaskKeys, task.ID)
	}

	// Any remaining tasks in the map should be removed (they're no longer in the file)
	for key := range currentTaskKeys {
		err := s.DeleteTask(key)
		if err != nil {
			// Log the error but continue processing other tasks
			fmt.Printf("Warning: failed to delete obsolete task %s: %v\n", key, err)
		}
	}

	// Check for drift between file and memory
	return s.DetectAndRecoverDrift(filePath)
}

// DetectAndRecoverDrift checks for differences between file and memory and recovers if needed
func (s *Service) DetectAndRecoverDrift(filePath string) error {
	// Read and parse the file
	fileBytes, err := readFileBytes(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the tasks from the file
	reader := &ByteReader{Data: fileBytes}
	parsedTasks, err := ParseTasks(reader)
	if err != nil {
		return fmt.Errorf("failed to parse tasks: %w", err)
	}

	// Get current tasks in memory
	memoryTasks, err := s.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks from memory: %w", err)
	}

	// Create maps for comparison
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

	// Check for tasks that exist in memory but not in file
	for key := range memoryTaskMap {
		if _, existsInFile := fileTaskMap[key]; !existsInFile {
			// Task exists in memory but not in file - remove it
			err := s.DeleteTask(key)
			if err != nil {
				fmt.Printf("Warning: failed to delete obsolete task %s: %v\n", key, err)
			}
		}
	}

	// Check for differences in completion status and hash
	fileHash := computeHash(fileBytes)
	for _, fileTask := range fileTaskMap {
		if memTask, existsInMemory := memoryTaskMap[fileTask.ID]; existsInMemory {
			// Check if completion status differs or hash mismatch
			completionMismatch := false
			hashMismatch := false

			// Extract completion status from memory detail
			if memTask.Detail != "" {
				if containsString(memTask.Detail, fmt.Sprintf("Completed: %t", !fileTask.Completed)) {
					// The completion status in memory is opposite to file
					completionMismatch = true
				}

				// Check if there's a hash stored in memory and compare
				if fileTask.LastSeenHash != "" && !containsString(memTask.Detail, "Hash: "+fileTask.LastSeenHash) {
					hashMismatch = true
				}
			}

			if completionMismatch || hashMismatch {
				// Discard and rebuild from file
				err := s.DeleteTask(fileTask.ID)
				if err != nil {
					fmt.Printf("Warning: failed to delete drifted task %s: %v\n", fileTask.ID, err)
				}

				// Re-create from file data
				fileTask.LastSeenHash = fileHash
				fileTask.LastUpdated = time.Now().Format(time.RFC3339)
				err = s.CreateOrUpdateTask(fileTask)
				if err != nil {
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

	hasTaskFile, err := HasTaskFile(projectPath)
	if err != nil {
		// If there's an error checking for the file, we'll just set it to false
		hasTaskFile = false
	}

	status := map[string]interface{}{
		"total_tasks":      totalTasks,
		"completed_tasks":  completedTasks,
		"incomplete_tasks": incompleteTasks,
		"has_task_file":    hasTaskFile,
	}

	return status, nil
}

// HasTaskFile checks if tinyTasks.md exists in the given directory
func HasTaskFile(dirPath string) (bool, error) {
	filePath := dirPath + "/tinyTasks.md"
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetTaskFileHash returns the hash of the tinyTasks.md file
func GetTaskFileHash(dirPath string) (string, error) {
	filePath := dirPath + "/tinyTasks.md"
	fileBytes, err := readFileBytes(filePath)
	if err != nil {
		return "", err
	}
	return computeHash(fileBytes), nil
}

// ByteReader is a simple wrapper to implement io.Reader interface
type ByteReader struct {
	Data []byte
	Pos  int
}

func (br *ByteReader) Read(p []byte) (n int, err error) {
	if br.Pos >= len(br.Data) {
		return 0, io.EOF
	}
	n = copy(p, br.Data[br.Pos:])
	br.Pos += n
	return n, nil
}

// Helper function to read file bytes
func readFileBytes(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// Helper function to compute hash
func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:])
}
