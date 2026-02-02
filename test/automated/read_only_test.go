package automated

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/execution"
	"github.com/daverage/tinymem/internal/storage"
)

// TestReadOnlyOperationsWithoutMode verifies that read-only operations
// (PASSIVE level) work without explicit mode declaration
func TestReadOnlyOperationsWithoutMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "read-only-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create controller WITHOUT setting mode explicitly
	ctrl := execution.NewController("", filepath.Join(tmpDir, "tinyTasks.md"))

	// Verify mode is not set
	if ctrl.ModeSet() {
		t.Fatal("Mode should not be set initially")
	}

	// Test 1: ActionMemoryQuery at PASSIVE level should be allowed
	err = ctrl.Enforce(execution.ActionMemoryQuery, execution.ModePassive)
	if err != nil {
		t.Fatalf("memory_query should be allowed without explicit mode: %v", err)
	}

	// Test 2: ActionMemoryQuery with empty minimum (defaults to PASSIVE) should be allowed
	err = ctrl.Enforce(execution.ActionMemoryQuery, "")
	if err != nil {
		t.Fatalf("memory_query with empty minimum should be allowed: %v", err)
	}

	// Test 3: Any read-only action at PASSIVE should be allowed
	readOnlyAction := execution.Action("memory_health")
	err = ctrl.Enforce(readOnlyAction, execution.ModePassive)
	if err != nil {
		t.Fatalf("read-only operation at PASSIVE should be allowed: %v", err)
	}

	// Test 4: Mutating operations (GUARDED level) should be blocked
	err = ctrl.Enforce(execution.ActionMemoryWrite, execution.ModeGuarded)
	if err == nil {
		t.Fatal("memory_write should be blocked without explicit mode")
	}
	expectedMsg := "mutation requires explicit mode; call memory_set_mode"
	if err.Error() != expectedMsg {
		t.Errorf("Wrong error message: got %q, want %q", err.Error(), expectedMsg)
	}

	// Test 5: After setting mode, both read and write should work
	err = ctrl.SetMode(execution.ModeGuarded)
	if err != nil {
		t.Fatalf("Failed to set mode: %v", err)
	}

	err = ctrl.Enforce(execution.ActionMemoryQuery, execution.ModePassive)
	if err != nil {
		t.Fatalf("memory_query should still work after mode set: %v", err)
	}

	err = ctrl.Enforce(execution.ActionMemoryWrite, execution.ModeGuarded)
	if err != nil {
		t.Fatalf("memory_write should work in GUARDED mode: %v", err)
	}
}

// TestReadOnlyWithExistingDB verifies read-only operations work with real database
func TestReadOnlyWithExistingDB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "read-only-db-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		ProjectRoot: tmpDir,
		DBPath:      filepath.Join(tmpDir, "test.db"),
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create controller without explicit mode
	ctrl := execution.NewController("", filepath.Join(tmpDir, "tinyTasks.md"))

	// Verify read operations are allowed
	err = ctrl.Enforce(execution.ActionMemoryQuery, execution.ModePassive)
	if err != nil {
		t.Fatalf("Database read should be allowed without mode: %v", err)
	}
}
