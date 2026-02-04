package tasks

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TaskManager centralizes every interaction with tinyTasks.md so that the server
// can treat the file as the single source of truth without letting other modules
// read or mutate it directly.
type TaskManager struct {
	path string
}

// TaskDefinition describes a task the manager will add to the file.
type TaskDefinition struct {
	Section  string
	Title    string
	Subtasks []SubtaskSpec
}

// SubtaskSpec describes a single subtask line.
type SubtaskSpec struct {
	Title     string
	Completed bool
}

// TaskUpdate describes a mutation to an existing task.
type TaskUpdate struct {
	Title    *string
	Subtasks *[]SubtaskSpec
}

// NewTaskManager returns a manager bound to the provided tinyTasks.md path.
func NewTaskManager(path string) *TaskManager {
	return &TaskManager{path: path}
}

// Path returns the configured file path.
func (tm *TaskManager) Path() string {
	return tm.path
}

// Exists reports whether the task file is present.
func (tm *TaskManager) Exists() (bool, error) {
	if tm.path == "" {
		return false, fmt.Errorf("task path is not configured")
	}
	_, err := os.Stat(tm.path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Hash returns the SHA256 hash of the task file.
func (tm *TaskManager) Hash() (string, error) {
	data, err := os.ReadFile(tm.path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:]), nil
}

// Load parses the task file into a document structure.
func (tm *TaskManager) Load() (*TaskDocument, error) {
	data, err := os.ReadFile(tm.path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	hasTrailing := len(data) > 0 && data[len(data)-1] == '\n'
	lines := strings.Split(content, "\n")
	doc := buildDocument(lines, hasTrailing, tm.path)
	return doc, nil
}

// List returns the tasks in file order.
func (tm *TaskManager) List() ([]*Task, error) {
	doc, err := tm.Load()
	if err != nil {
		return nil, err
	}
	return doc.Tasks(), nil
}

// Complete marks every checkbox within a task as finished.
func (tm *TaskManager) Complete(taskID string) (*Task, error) {
	doc, err := tm.Load()
	if err != nil {
		return nil, err
	}

	tr, ok := doc.TaskRangeByID(taskID)
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	lines := doc.CloneLines()
	for _, idx := range tr.CheckboxIndices {
		lines[idx] = setCheckboxState(lines[idx], true)
	}

	if err := tm.writeLines(lines, doc.HasTrailingNewline); err != nil {
		return nil, err
	}

	updated, err := tm.Load()
	if err != nil {
		return nil, err
	}
	if newRange, ok := updated.TaskRangeByID(taskID); ok {
		return newRange.Task, nil
	}
	return nil, fmt.Errorf("completed task %s could not be reloaded", taskID)
}

// AddTask appends a new task to the file. If the file does not exist, it is created
// with the canonical inert template before the addition.
func (tm *TaskManager) AddTask(def TaskDefinition) (*Task, error) {
	if def.Title == "" {
		return nil, fmt.Errorf("task title is required")
	}

	if err := tm.ensureFile(); err != nil {
		return nil, err
	}

	doc, err := tm.Load()
	if err != nil {
		return nil, err
	}

	section := def.Section
	if section == "" {
		section = "Tasks"
	}

	lines := doc.CloneLines()
	insertIdx := len(lines)
	hasSection := doc.HasSection(section)
	if end, ok := doc.SectionEnds[section]; ok {
		insertIdx = end
	}

	block := buildInsertionBlock(section, def, hasSection)
	lines = insertLines(lines, insertIdx, block)

	if err := tm.writeLines(lines, doc.HasTrailingNewline); err != nil {
		return nil, err
	}

	updated, err := tm.Load()
	if err != nil {
		return nil, err
	}

	// The new task is the last one with the requested section.
	for i := len(updated.TaskOrder) - 1; i >= 0; i-- {
		if updated.TaskOrder[i].Task.Section == section {
			return updated.TaskOrder[i].Task, nil
		}
	}

	return nil, fmt.Errorf("failed to locate newly added task in section %s", section)
}

// UpdateTask mutates the title or subtasks for a given task.
func (tm *TaskManager) UpdateTask(taskID string, update TaskUpdate) (*Task, error) {
	doc, err := tm.Load()
	if err != nil {
		return nil, err
	}

	tr, ok := doc.TaskRangeByID(taskID)
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	lines := doc.CloneLines()

	if update.Title != nil {
		lines[tr.Start] = replaceTitle(lines[tr.Start], *update.Title)
	}

	if update.Subtasks != nil {
		lines = replaceSubtasks(lines, tr, *update.Subtasks)
	}

	if err := tm.writeLines(lines, doc.HasTrailingNewline); err != nil {
		return nil, err
	}

	updated, err := tm.Load()
	if err != nil {
		return nil, err
	}
	if newRange, ok := updated.TaskRangeByID(taskID); ok {
		return newRange.Task, nil
	}
	return nil, fmt.Errorf("updated task %s could not be reloaded", taskID)
}

// CreateInertFile writes the canonical inert tinyTasks.md template.
func (tm *TaskManager) CreateInertFile() error {
	if err := os.MkdirAll(filepath.Dir(tm.path), 0755); err != nil {
		return fmt.Errorf("failed to ensure task directory: %w", err)
	}

	if exists, err := tm.Exists(); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("tinyTasks.md already exists")
	}

	content := defaultTaskTemplate
	return os.WriteFile(tm.path, []byte(content), 0644)
}

func (tm *TaskManager) ensureFile() error {
	if exists, err := tm.Exists(); err != nil {
		return err
	} else if exists {
		return nil
	}
	return tm.CreateInertFile()
}

func (tm *TaskManager) writeLines(lines []string, trailingNewline bool) error {
	if err := os.MkdirAll(filepath.Dir(tm.path), 0755); err != nil {
		return fmt.Errorf("failed to ensure task directory: %w", err)
	}
	content := strings.Join(lines, "\n")
	if trailingNewline && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(tm.path, []byte(content), 0644)
}

func buildInsertionBlock(section string, def TaskDefinition, sectionExists bool) []string {
	var block []string

	if !sectionExists {
		block = append(block, "")
		block = append(block, fmt.Sprintf("## %s", section))
		block = append(block, "")
	} else {
		block = append(block, "")
	}

	block = append(block, fmt.Sprintf("- [ ] %s", def.Title))
	for _, sub := range def.Subtasks {
		block = append(block, fmt.Sprintf("  - [%s] %s", boolToCheckbox(sub.Completed), sub.Title))
	}
	block = append(block, "")

	return block
}

func insertLines(lines []string, index int, block []string) []string {
	if index > len(lines) {
		index = len(lines)
	}
	result := make([]string, 0, len(lines)+len(block))
	result = append(result, lines[:index]...)
	result = append(result, block...)
	result = append(result, lines[index:]...)
	return result
}

func replaceSubtasks(lines []string, tr *TaskRange, specs []SubtaskSpec) []string {
	result := make([]string, 0, len(lines)+len(specs))
	for idx, line := range lines {
		result = append(result, line)
		if idx == tr.Start && len(specs) > 0 {
			result = append(result, buildSubtaskLines(specs)...)
		}
		if idx > tr.Start && idx < tr.End && isSubtaskLine(line) {
			result = result[:len(result)-1]
		}
	}
	return result
}

func buildSubtaskLines(specs []SubtaskSpec) []string {
	var lines []string
	for _, sub := range specs {
		lines = append(lines, fmt.Sprintf("  - [%s] %s", boolToCheckbox(sub.Completed), sub.Title))
	}
	return lines
}

func boolToCheckbox(done bool) string {
	if done {
		return "x"
	}
	return " "
}

func replaceTitle(line, title string) string {
	idx := strings.Index(line, "]")
	if idx == -1 {
		return line
	}
	return line[:idx+1] + " " + title
}

func setCheckboxState(line string, checked bool) string {
	idx := strings.Index(line, "[")
	if idx == -1 || idx+2 >= len(line) {
		return line
	}
	status := " "
	if checked {
		status = "x"
	}
	return line[:idx+1] + status + line[idx+2:]
}

func isSubtaskLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "- [") && !strings.HasPrefix(strings.TrimLeft(line, " \t"), "* [") {
		return false
	}
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}

// TaskDocument represents the parsed structure of tinyTasks.md.
type TaskDocument struct {
	Lines              []string
	HasTrailingNewline bool
	TaskOrder          []*TaskRange
	Ranges             map[string]*TaskRange
	SectionEnds        map[string]int
	SectionHeaders     map[string]int
	SectionCounts      map[string]int
}

// TaskRange describes the lines occupied by a single top-level task.
type TaskRange struct {
	Task            *Task
	Start           int
	End             int
	CheckboxIndices []int
}

func buildDocument(lines []string, trailing bool, path string) *TaskDocument {
	doc := &TaskDocument{
		Lines:              copyLines(lines),
		HasTrailingNewline: trailing,
		TaskOrder:          []*TaskRange{},
		Ranges:             map[string]*TaskRange{},
		SectionEnds:        map[string]int{},
		SectionHeaders:     map[string]int{},
		SectionCounts:      map[string]int{},
	}

	currentSection := ""
	idx := 0

	for idx < len(lines) {
		line := lines[idx]
		if isSectionHeader(line) {
			currentSection = parseSection(line)
			doc.SectionHeaders[currentSection] = idx
			idx++
			continue
		}

		if isTopLevelTask(line) {
			doc.SectionCounts[currentSection]++
			title, topCompleted, total, done, nextIdx := parseTopLevelTask(lines, idx)
			completed := false
			if total > 0 {
				completed = total == done
			} else {
				completed = topCompleted
			}

			task := &Task{
				ID:           generateTaskID(currentSection, doc.SectionCounts[currentSection]),
				Title:        title,
				Section:      currentSection,
				Index:        len(doc.TaskOrder) + 1,
				StepsTotal:   total,
				StepsDone:    done,
				Completed:    completed,
				State:        TaskStateOpen,
				Mode:         TaskModeDormant,
				FilePath:     filepath.Base(path),
				LastUpdated:  time.Now().Format(time.RFC3339),
				LastSeenHash: "",
			}

			tr := &TaskRange{
				Task:            task,
				Start:           idx,
				End:             nextIdx,
				CheckboxIndices: collectCheckboxIndices(lines, idx, nextIdx),
			}

			doc.Ranges[task.ID] = tr
			doc.TaskOrder = append(doc.TaskOrder, tr)
			doc.SectionEnds[currentSection] = nextIdx
			idx = nextIdx
			continue
		}

		idx++
	}

	return doc
}

func collectCheckboxIndices(lines []string, start, end int) []int {
	var indices []int
	for i := start; i < end && i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i], " \t")
		if strings.HasPrefix(trimmed, "- [") || strings.HasPrefix(trimmed, "* [") {
			indices = append(indices, i)
		}
	}
	return indices
}

func copyLines(lines []string) []string {
	c := make([]string, len(lines))
	copy(c, lines)
	return c
}

func (doc *TaskDocument) Tasks() []*Task {
	tasks := make([]*Task, 0, len(doc.TaskOrder))
	for _, tr := range doc.TaskOrder {
		tasks = append(tasks, tr.Task)
	}
	return tasks
}

func (doc *TaskDocument) TaskRangeByID(id string) (*TaskRange, bool) {
	tr, ok := doc.Ranges[id]
	return tr, ok
}

func (doc *TaskDocument) HasSection(name string) bool {
	_, ok := doc.SectionHeaders[name]
	return ok
}

func (doc *TaskDocument) CloneLines() []string {
	return copyLines(doc.Lines)
}

const defaultTaskTemplate = `# Tasks — NOT STARTED
>
> This file was created automatically because a multi-step workflow
> may be required.
>
> No work is authorised until a human edits this file and defines tasks.

## Tasks
<!-- No tasks defined yet -->
`
