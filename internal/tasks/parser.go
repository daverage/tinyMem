/*
Package tasks implements a file-authoritative task ledger for TinyMem.

The task system follows these principles:
1. tinyTasks.md is the sole source of truth for task state
2. TinyMem never infers task completion
3. Memory entries are fully rebuildable from the file
4. If file and memory disagree, file wins
5. Memory never stores subtask text or implementation detail

The system supports:
- Top-level checkbox tasks with nested subtasks
- Stable task identification based on section and position
- One-way sync from file to memory
- Drift detection and recovery
*/
package tasks

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// TaskState represents the state of a task
type TaskState string

const (
	TaskStateOpen      TaskState = "open"
	TaskStateCompleted TaskState = "completed"
)

// TaskMode represents how the task should be treated
type TaskMode string

const (
	TaskModeDormant TaskMode = "dormant" // Default - read-only, no execution
	TaskModeActive  TaskMode = "active"  // Only when explicitly requested by user
)

// Task represents a parsed task from tinyTasks.md
type Task struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Section      string            `json:"section"`
	Index        int               `json:"index"`
	StepsTotal   int               `json:"steps_total"`
	StepsDone    int               `json:"steps_done"`
	Completed    bool              `json:"completed"`
	State        TaskState         `json:"state"`
	Mode         TaskMode          `json:"mode"`
	LastSeenHash string            `json:"last_seen_hash"`
	FilePath     string            `json:"file_path"`
	LastUpdated  string            `json:"last_updated"`
	ExtraFields  map[string]string `json:"extra_fields,omitempty"` // For future extensibility
}

// ParseTasks parses the tinyTasks.md file and returns a list of tasks
func ParseTasks(r io.Reader) ([]*Task, error) {
	// Read all lines first to allow proper parsing
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var tasks []*Task

	currentSection := ""
	taskIndex := 0
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Check for section headers (Markdown headings)
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			currentSection = parseSection(line)
			i++
			continue
		}

		// Check for top-level checkbox tasks
		if isTopLevelTask(line) {
			taskIndex++

			// Extract task details and advance the index past subtasks
			title, topCompleted, stepsTotal, stepsDone, nextIdx := parseTopLevelTask(lines, i)
			i = nextIdx // Move index to the next position after subtasks

			// Logic for task completion:
			// 1. If there are subtasks, the top-level is completed only if all subtasks are done
			// 2. If there are no subtasks, the top-level is completed if its checkbox is checked
			completed := false
			if stepsTotal > 0 {
				completed = (stepsTotal == stepsDone)
			} else {
				completed = topCompleted
			}

			task := &Task{
				ID:           generateTaskID(currentSection, taskIndex),
				Title:        title,
				Section:      currentSection,
				Index:        taskIndex,
				StepsTotal:   stepsTotal,
				StepsDone:    stepsDone,
				Completed:    completed,
				State:        TaskStateOpen,  // Default state
				Mode:         TaskModeDormant, // Default mode - read-only
				LastSeenHash: "",             // Will be set by caller
				FilePath:     "tinyTasks.md", // Fixed path
				LastUpdated:  "",             // Will be set by caller
			}

			tasks = append(tasks, task)
		} else {
			i++ // Move to next line
		}
	}

	return tasks, nil
}

// parseSection extracts the section name from a markdown header
func parseSection(headerLine string) string {
	// Remove leading # and whitespace
	re := regexp.MustCompile(`^#{1,6}\s*(.*)`)
	matches := re.FindStringSubmatch(headerLine)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// isTopLevelTask checks if a line is a top-level checkbox task
func isTopLevelTask(line string) bool {
	// Check that the line is NOT indented (for subtasks)
	originalLine := line
	if len(originalLine) > 0 && (originalLine[0] == ' ' || originalLine[0] == '\t') {
		// This line is indented, so it's a subtask, not a top-level task
		return false
	}

	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "- [") || len(trimmed) < 6 {
		return false
	}
	// Check if it has the format "- [x] " or "- [ ] " (space after the bracket)
	return ((trimmed[3] == 'x' || trimmed[3] == ' ') && trimmed[4] == ']' && trimmed[5] == ' ')
}

// parseTopLevelTask parses a top-level task and its subtasks
func parseTopLevelTask(lines []string, startIndex int) (title string, topCompleted bool, totalSubtasks, doneSubtasks, nextIndex int) {
	firstLine := lines[startIndex]

	// Determine if the top-level checkbox is checked
	topCompleted = strings.HasPrefix(strings.TrimLeft(firstLine, " \t"), "- [x]")

	// Extract title from the first line
	// First try to match bold formatting
	reBold := regexp.MustCompile(`- \[.\] \*\*(.+?)\*\*`)
	matches := reBold.FindStringSubmatch(firstLine)
	if len(matches) > 1 {
		title = matches[1]
	} else {
		// Then try to match any text after checkbox
		reSimple := regexp.MustCompile(`- \[[ x]\] (.+)`)
		matchesSimple := reSimple.FindStringSubmatch(firstLine)
		if len(matchesSimple) > 1 {
			title = matchesSimple[1]
		} else {
			// Fallback: just remove checkbox prefix
			title = strings.TrimPrefix(firstLine, "- [ ] ")
			title = strings.TrimPrefix(title, "- [x] ")
		}
	}

	// Count subtasks by looking ahead from the next line
	totalSubtasks = 0
	doneSubtasks = 0
	currentIndex := startIndex + 1

	// Process following lines to find subtasks
	for currentIndex < len(lines) {
		nextLine := lines[currentIndex]

		// Check if this is a subtask (indented checkbox)
		if isSubtask(nextLine) {
			totalSubtasks++
			if isSubtaskCompleted(nextLine) {
				doneSubtasks++
			}
			currentIndex++
		} else if isTopLevelTask(nextLine) || isSectionHeader(nextLine) {
			// This is the start of the next task or section, so break
			break
		} else if strings.TrimSpace(nextLine) == "" {
			// Empty line, continue
			currentIndex++
			continue
		} else if !isIndented(nextLine) {
			// Not indented, so it's not part of this task
			break
		} else {
			// Indented but not a subtask (probably prose), continue
			currentIndex++
		}
	}

	nextIndex = currentIndex
	return title, topCompleted, totalSubtasks, doneSubtasks, nextIndex
}

// isSubtask checks if a line is a subtask (indented checkbox)
func isSubtask(line string) bool {
	// Subtasks are indented and have checkbox format
	trimmed := strings.TrimLeft(line, " \t")
	indentation := len(line) - len(trimmed)

	// Subtasks should be indented (typically 4 spaces or a tab)
	if indentation == 0 {
		return false
	}

	// Check if it has the format "- [x] " or "- [ ] "
	if !strings.HasPrefix(trimmed, "- [") || len(trimmed) < 6 {
		return false
	}
	return ((trimmed[3] == 'x' || trimmed[3] == ' ') && trimmed[4] == ']' && trimmed[5] == ' ')
}

// isSubtaskCompleted checks if a subtask is marked as completed
func isSubtaskCompleted(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "- [x]")
}

// isSectionHeader checks if a line is a markdown section header
func isSectionHeader(line string) bool {
	return strings.HasPrefix(line, "# ")
}

// isIndented checks if a line is indented (for distinguishing subtasks from prose)
func isIndented(line string) bool {
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}

// generateTaskID creates a stable task identifier based on section and index
func generateTaskID(section string, index int) string {
	slug := slugify(section)
	return fmt.Sprintf("task:%s:%d", slug, index)
}

// slugify converts a string to a URL-friendly slug
func slugify(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Replace non-alphanumeric characters with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	text = re.ReplaceAllString(text, "-")

	// Remove leading/trailing hyphens
	text = strings.Trim(text, "-")

	return text
}
