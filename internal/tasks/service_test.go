package tasks

import (
	"os"
	"testing"
	"time"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTasks(t *testing.T) {
	// Test parsing a sample tinyTasks.md content
	sampleContent := `# Tasks - Split Hiss / Rumble

- [x] **Refactor src/dsp/hiss_rumble.rs**
    - [x] Rename/modify HissRumble struct to support independent rumble_hpf and hiss_shelf.
    - [x] Implement process(input_l, input_r, rumble_amt, hiss_amt, sidechain).
    - [x] Rumble: HPF 20 Hz -> 120 Hz.
    - [x] Hiss: HF Shelf 8 kHz, 0 -> -24 dB, gated by speech confidence.
    - [ ] Update debug getters.

- [ ] **Update documentation**
    - [x] Add usage examples
    - [ ] Update API reference`

	reader := &ByteReader{Data: []byte(sampleContent)}
	tasks, err := ParseTasks(reader)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Check first task
	assert.Equal(t, "task:tasks-split-hiss-rumble:1", tasks[0].ID)
	assert.Equal(t, "Refactor src/dsp/hiss_rumble.rs", tasks[0].Title)
	assert.Equal(t, "Tasks - Split Hiss / Rumble", tasks[0].Section)
	assert.Equal(t, 5, tasks[0].StepsTotal)
	assert.Equal(t, 4, tasks[0].StepsDone)
	assert.False(t, tasks[0].Completed) // Not all steps done

	// Check second task
	assert.Equal(t, "task:tasks-split-hiss-rumble:2", tasks[1].ID)
	assert.Equal(t, "Update documentation", tasks[1].Title)
	assert.Equal(t, "Tasks - Split Hiss / Rumble", tasks[1].Section)
	assert.Equal(t, 2, tasks[1].StepsTotal)
	assert.Equal(t, 1, tasks[1].StepsDone)
	assert.False(t, tasks[1].Completed) // Not all steps done
}

func TestGenerateTaskID(t *testing.T) {
	// Test stable ID generation
	id1 := generateTaskID("Test Section", 1)
	id2 := generateTaskID("Test Section", 1)
	id3 := generateTaskID("Test Section", 2)
	id4 := generateTaskID("Another Section", 1)

	assert.Equal(t, id1, id2)    // Same section and index should produce same ID
	assert.NotEqual(t, id1, id3) // Different index should produce different ID
	assert.NotEqual(t, id1, id4) // Different section should produce different ID
}

func TestTaskService(t *testing.T) {
	// Create a temporary database for testing
	tmpFile, err := os.CreateTemp("", "test_db_*.sqlite3")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Initialize database
	cfg := &config.Config{
		DBPath:      tmpFile.Name(),
		TinyMemDir:  os.TempDir(),
		ProjectRoot: t.TempDir(),
	}

	db, err := storage.NewDB(cfg)
	require.NoError(t, err)
	defer db.Close()

	// Initialize memory service
	memoryService := memory.NewService(db)

	// Initialize task service
	service := NewService(db, memoryService, "test-project")

	// Test creating and retrieving a task
	task := &Task{
		ID:          "task:test-section:1",
		Title:       "Test Task",
		Section:     "Test Section",
		Index:       1,
		StepsTotal:  2,
		StepsDone:   1,
		Completed:   false,
		FilePath:    "tinyTasks.md",
		LastUpdated: time.Now().Format(time.RFC3339),
	}

	err = service.CreateOrUpdateTask(task)
	assert.NoError(t, err)

	// Verify task was created
	tasks, err := service.GetAllTasks()
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Test Task", tasks[0].Summary)
}

func TestTaskSync(t *testing.T) {
	// Create a temporary database for testing
	tmpFile, err := os.CreateTemp("", "test_sync_db_*.sqlite3")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Initialize database
	cfg := &config.Config{
		DBPath:      tmpFile.Name(),
		TinyMemDir:  os.TempDir(),
		ProjectRoot: t.TempDir(),
	}

	db, err := storage.NewDB(cfg)
	require.NoError(t, err)
	defer db.Close()

	// Initialize memory service
	memoryService := memory.NewService(db)

	// Initialize task service
	service := NewService(db, memoryService, "test-project")

	// Create a temporary tinyTasks.md file
	tempDir := t.TempDir()
	taskFilePath := tempDir + "/tinyTasks.md"

	sampleContent := `# Test Tasks

- [x] **Complete first task**
    - [x] Subtask 1
    - [x] Subtask 2

- [ ] **Incomplete task**
    - [x] Done subtask
    - [ ] Pending subtask`

	err = os.WriteFile(taskFilePath, []byte(sampleContent), 0644)
	require.NoError(t, err)

	// Perform sync
	err = service.SyncTasks(tempDir)
	assert.NoError(t, err)

	// Verify tasks were synced
	tasks, err := service.GetAllTasks()
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Check that the tasks match what we expect
	status, err := service.GetTaskStatus(tempDir)
	assert.NoError(t, err)

	// Convert to integers for comparison since JSON unmarshaling can return different numeric types
	totalTasks := status["total_tasks"].(int)
	completedTasks := status["completed_tasks"].(int)
	incompleteTasks := status["incomplete_tasks"].(int)

	assert.Equal(t, 2, totalTasks)
	assert.Equal(t, 1, completedTasks)
	assert.Equal(t, 1, incompleteTasks)
}

func TestHasTaskFile(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Test when file doesn't exist
	exists, err := HasTaskFile(tempDir)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Create the task file
	taskFilePath := tempDir + "/tinyTasks.md"
	err = os.WriteFile(taskFilePath, []byte("# Test Tasks\n\n- [ ] Test task"), 0644)
	require.NoError(t, err)

	// Test when file exists
	exists, err = HasTaskFile(tempDir)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestComputeHash(t *testing.T) {
	// Test that the same content produces the same hash
	content1 := []byte("test content")
	content2 := []byte("test content")
	content3 := []byte("different content")

	hash1 := computeHash(content1)
	hash2 := computeHash(content2)
	hash3 := computeHash(content3)

	assert.Equal(t, hash1, hash2)
	assert.NotEqual(t, hash1, hash3)
}
