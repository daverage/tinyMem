package ralph

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/daverage/tinymem/internal/config"
	"go.uber.org/zap"
)

func TestIsPathForbidden(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo for testing gitignore
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git: %v", err)
	}

	// Create a .gitignore
	err = os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("ignored.txt\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write .gitignore: %v", err)
	}

	cfg := &config.Config{
		ProjectRoot: tmpDir,
	}
	e := &Engine{
		cfg:    cfg,
		logger: zap.NewNop(),
	}

	safety := SafetyOptions{
		ForbidPaths: []string{"forbidden_dir/"},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"safe.txt", false},
		{"forbidden_dir/file.txt", true},
		{".tinyMem/data", true},
		{".tinymem/data", true},
		{".gemini/history", true},
		{"tinyTasks.md", true},
		{"ignored.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := e.isPathForbidden(tt.path, safety)
			if got != tt.expected {
				t.Errorf("isPathForbidden(%s) = %v; want %v", tt.path, got, tt.expected)
			}
		})
	}
}
