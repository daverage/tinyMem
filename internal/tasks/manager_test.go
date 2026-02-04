package tasks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskManagerAddAndComplete(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewTaskManager(filepath.Join(tempDir, "tinyTasks.md"))

	task, err := manager.AddTask(TaskDefinition{
		Section: "Phase 1",
		Title:   "Implement TaskManager",
		Subtasks: []SubtaskSpec{
			{Title: "Write parsing logic", Completed: true},
			{Title: "Add coverage", Completed: false},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "Implement TaskManager", task.Title)
	require.False(t, task.Completed)

	tasks, err := manager.List()
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	completed, err := manager.Complete(task.ID)
	require.NoError(t, err)
	require.True(t, completed.Completed)

	final, err := manager.List()
	require.NoError(t, err)
	require.Len(t, final, 1)
	require.True(t, final[0].Completed)
}

func TestTaskManagerUpdateTask(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewTaskManager(filepath.Join(tempDir, "tinyTasks.md"))

	err := os.WriteFile(manager.path, []byte("# Initial\n\n- [ ] Original Task\n  - [ ] Subtask A\n"), 0644)
	require.NoError(t, err)

	tasks, err := manager.List()
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	newTitle := "Updated Task"
	newSubtasks := []SubtaskSpec{{Title: "Fresh subtask", Completed: true}}

	updated, err := manager.UpdateTask(tasks[0].ID, TaskUpdate{
		Title:    &newTitle,
		Subtasks: &newSubtasks,
	})
	require.NoError(t, err)
	require.Equal(t, newTitle, updated.Title)
	require.Equal(t, 1, updated.StepsTotal)
	require.Equal(t, 1, updated.StepsDone)
}
