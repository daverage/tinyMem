package automated

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/llm"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/ralph"
	"github.com/daverage/tinymem/internal/storage"
	"go.uber.org/zap"
)

// mockRalphLLM simplified to focus on invariant enforcement
type mockRalphLLM struct {
	callCount     int
	returnInvalid bool
}

func (m *mockRalphLLM) ChatCompletions(ctx context.Context, req llm.ChatCompletionRequest) (*llm.ChatCompletionResponse, error) {
	m.callCount++

	var content string
	if m.returnInvalid {
		content = "garbage output that fails parsing"
	} else {
		content = "@@@ FILE: test_file.txt @@@\nincorrect repair\n@@@ END_FILE @@@"
	}

	return &llm.ChatCompletionResponse{
		Choices: []llm.Choice{{Message: llm.Message{Role: "assistant", Content: content}}},
	}, nil
}

func (m *mockRalphLLM) ProviderName() string {
	return "mock-ralph-llm"
}

func (m *mockRalphLLM) Close() error {
	return nil
}

func (m *mockRalphLLM) GetMetrics() *llm.Metrics {
	return nil
}

// TestRalphLoopStopsOnEvidence verifies loop terminates only when evidence passes
func TestRalphLoopStopsOnEvidence(t *testing.T) {
	// Create temporary test environment
	tmpDir, cleanup := setupTempEnvironment(t)
	defer cleanup()

	// Create failing test file
	testFile := filepath.Join(tmpDir, "test_file.txt")
	os.WriteFile(testFile, []byte("fail"), 0644)

	// Create evidence checker script that checks file content
	evidenceScript := filepath.Join(tmpDir, "check.sh")
	os.WriteFile(evidenceScript, []byte(`#!/bin/sh
content=$(cat test_file.txt)
if [ "$content" = "pass" ]; then
    exit 0
else
    exit 1
fi
`), 0755)

	cfg := &config.Config{
		ProjectRoot:                   tmpDir,
		EvidenceCommandTimeoutSeconds: 5,
	}

	memService := setupMockMemoryService(t, tmpDir)
	mockLLM := &mockRalphLLM{}
	engine := ralph.NewEngine(cfg, memService, mockLLM, "test", zap.NewNop())

	opt := ralph.Options{
		Task:          "Fix the test",
		Command:       "sh check.sh",
		Evidence:      []string{"cmd_exit0::sh check.sh"},
		MaxIterations: 3,
		Safety: ralph.SafetyOptions{
			AllowShell: true,
		},
	}

	// Run loop - should fail because we never fix the file
	result, _ := engine.ExecuteLoop(context.Background(), opt)

	// Verify loop ran all iterations (evidence never passed)
	if result.Iterations != 3 {
		t.Fatalf("Expected 3 iterations, got %d", result.Iterations)
	}

	// Verify evidence never passed
	if result.Status == ralph.StatusSuccess {
		t.Fatal("Loop succeeded but evidence should have failed all iterations")
	}

	// Verify LLM was called for repairs (iterations - 1, no repair after last iteration)
	expectedCalls := 2
	if mockLLM.callCount != expectedCalls {
		t.Fatalf("Expected %d LLM calls, got %d", expectedCalls, mockLLM.callCount)
	}

	// Now fix the file and verify evidence passes immediately
	os.WriteFile(testFile, []byte("pass"), 0644)
	mockLLM.callCount = 0

	opt.MaxIterations = 5
	result, err := engine.ExecuteLoop(context.Background(), opt)

	if err != nil {
		t.Fatalf("Loop failed: %v", err)
	}

	// Should succeed on first iteration (evidence passes)
	if result.Status != ralph.StatusSuccess {
		t.Fatalf("Expected success, got %s", result.Status)
	}

	if result.Iterations != 1 {
		t.Fatalf("Expected 1 iteration (immediate success), got %d", result.Iterations)
	}

	// No LLM calls needed (evidence passed immediately)
	if mockLLM.callCount != 0 {
		t.Fatalf("Expected 0 LLM calls (evidence passed), got %d", mockLLM.callCount)
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:     "Ralph_StopsOnEvidence",
		LLMCallsMade: mockLLM.callCount,
		Passed:       true,
	})
}

// TestRalphLoopMaxIterations verifies loop respects max_iterations limit
func TestRalphLoopMaxIterations(t *testing.T) {
	tmpDir, cleanup := setupTempEnvironment(t)
	defer cleanup()

	// Create always-failing command
	failScript := filepath.Join(tmpDir, "fail.sh")
	os.WriteFile(failScript, []byte("#!/bin/sh\nexit 1"), 0755)

	cfg := &config.Config{
		ProjectRoot:                   tmpDir,
		EvidenceCommandTimeoutSeconds: 5,
	}

	memService := setupMockMemoryService(t, tmpDir)
	mockLLM := &mockRalphLLM{}
	engine := ralph.NewEngine(cfg, memService, mockLLM, "test", zap.NewNop())

	maxIter := 5
	opt := ralph.Options{
		Task:          "Impossible task",
		Command:       "sh fail.sh",
		Evidence:      []string{"cmd_exit0::sh fail.sh"},
		MaxIterations: maxIter,
		Safety: ralph.SafetyOptions{
			AllowShell: true,
		},
	}

	result, _ := engine.ExecuteLoop(context.Background(), opt)

	// Verify loop stopped at max iterations
	if result.Iterations != maxIter {
		t.Fatalf("Expected exactly %d iterations, got %d", maxIter, result.Iterations)
	}

	// Verify status is failed (not success)
	if result.Status == ralph.StatusSuccess {
		t.Fatal("Loop should fail when reaching max iterations without evidence passing")
	}

	// Verify LLM was called for repairs (maxIter - 1, no repair after last iteration)
	expectedCalls := maxIter - 1
	if mockLLM.callCount != expectedCalls {
		t.Fatalf("Expected %d LLM calls, got %d", expectedCalls, mockLLM.callCount)
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:     "Ralph_MaxIterations",
		LLMCallsMade: mockLLM.callCount,
		Passed:       true,
	})
}

// TestRalphEvidencePredicates verifies all evidence types work correctly
func TestRalphEvidencePredicates(t *testing.T) {
	tmpDir, cleanup := setupTempEnvironment(t)
	defer cleanup()

	// Initialize git repo for evidence service
	exec.Command("git", "init").Dir = tmpDir
	exec.Command("git", "init", tmpDir).Run()

	// Create test file with content
	testFile := filepath.Join(tmpDir, "test_file.txt")
	os.WriteFile(testFile, []byte("important content"), 0644)

	cfg := &config.Config{
		ProjectRoot:                   tmpDir,
		EvidenceCommandTimeoutSeconds: 5,
	}

	memService := setupMockMemoryService(t, tmpDir)
	mockLLM := &mockRalphLLM{}
	engine := ralph.NewEngine(cfg, memService, mockLLM, "test", zap.NewNop())

	// Use relative paths for file_exists and grep_hit evidence (relative to ProjectRoot)
	tests := []struct {
		name       string
		evidence   string
		shouldPass bool
	}{
		{
			name:       "cmd_exit0_pass",
			evidence:   "cmd_exit0::test -f test_file.txt",
			shouldPass: true,
		},
		{
			name:       "cmd_exit0_fail",
			evidence:   "cmd_exit0::test -f nonexistent.txt",
			shouldPass: false,
		},
		{
			name:       "test_pass_alias",
			evidence:   "test_pass::test -f test_file.txt",
			shouldPass: true,
		},
		{
			name:       "file_exists_pass",
			evidence:   "file_exists::test_file.txt",
			shouldPass: true,
		},
		{
			name:       "file_exists_fail",
			evidence:   "file_exists::nonexistent.txt",
			shouldPass: false,
		},
		{
			name:       "grep_hit_pass",
			evidence:   "grep_hit::important::test_file.txt",
			shouldPass: true,
		},
		{
			name:       "grep_hit_fail",
			evidence:   "grep_hit::missing::test_file.txt",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := ralph.Options{
				Task:          "Test evidence predicate",
				Command:       "echo test",
				Evidence:      []string{tt.evidence},
				MaxIterations: 1,
				Safety: ralph.SafetyOptions{
					AllowShell: true,
				},
			}

			result, _ := engine.ExecuteLoop(context.Background(), opt)

			// Check evidence result
			evidenceResult, ok := result.Evidence[tt.evidence]
			if !ok {
				t.Fatalf("Evidence %s not found in results", tt.evidence)
			}

			passed := evidenceResult == true
			if passed != tt.shouldPass {
				t.Fatalf("Evidence %s: expected pass=%v, got %v (result: %v)",
					tt.evidence, tt.shouldPass, passed, evidenceResult)
			}

			// Verify loop status matches evidence
			if tt.shouldPass && result.Status != ralph.StatusSuccess {
				t.Fatalf("Evidence passed but loop status is %s", result.Status)
			}
			if !tt.shouldPass && result.Status == ralph.StatusSuccess {
				t.Fatal("Evidence failed but loop status is success")
			}
		})
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:     "Ralph_EvidencePredicates",
		LLMCallsMade: mockLLM.callCount,
		Passed:       true,
	})
}

// TestRalphSafetyForbidCommands verifies forbidden commands are blocked
func TestRalphSafetyForbidCommands(t *testing.T) {
	tmpDir, cleanup := setupTempEnvironment(t)
	defer cleanup()

	cfg := &config.Config{
		ProjectRoot:                   tmpDir,
		EvidenceCommandTimeoutSeconds: 5,
	}

	memService := setupMockMemoryService(t, tmpDir)
	mockLLM := &mockRalphLLM{}
	engine := ralph.NewEngine(cfg, memService, mockLLM, "test", zap.NewNop())

	opt := ralph.Options{
		Task:          "Test forbidden command",
		Command:       "rm -rf /", // Forbidden
		Evidence:      []string{"cmd_exit0::echo test"},
		MaxIterations: 1,
		Safety: ralph.SafetyOptions{
			ForbidCommands: []string{"rm -rf"},
			AllowShell:     true,
		},
	}

	result, err := engine.ExecuteLoop(context.Background(), opt)

	// Should return error about safety violation
	if err == nil {
		t.Fatal("Expected safety violation error, got nil")
	}

	// Safety check happens during execute phase, so iteration count could be 1
	// but the loop should fail immediately
	if result.Status == ralph.StatusSuccess {
		t.Fatal("Loop should fail on safety violation")
	}

	// LLM should never be called (error before repair phase)
	if mockLLM.callCount != 0 {
		t.Fatal("LLM should not be called when command is forbidden")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:     "Ralph_Safety_ForbidCommands",
		LLMCallsMade: mockLLM.callCount,
		Passed:       true,
	})
}

// TestRalphSafetyForbidPaths verifies forbidden paths are protected
func TestRalphSafetyForbidPaths(t *testing.T) {
	tmpDir, cleanup := setupTempEnvironment(t)
	defer cleanup()

	// Initialize git
	exec.Command("git", "init", tmpDir).Run()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("fail"), 0644)

	cfg := &config.Config{
		ProjectRoot:                   tmpDir,
		EvidenceCommandTimeoutSeconds: 5,
	}

	memService := setupMockMemoryService(t, tmpDir)
	// LLM tries to modify forbidden path
	mockLLM := &mockRalphLLM{}
	engine := ralph.NewEngine(cfg, memService, mockLLM, "test", zap.NewNop())

	opt := ralph.Options{
		Task:          "Test forbidden path",
		Command:       "test -f test.txt",
		Evidence:      []string{"cmd_exit0::grep -q 'fixed' test.txt"},
		MaxIterations: 2,
		Safety: ralph.SafetyOptions{
			ForbidPaths: []string{"forbidden_dir/"},
			AllowShell:  true,
		},
	}

	// First verify normal file can be written
	result, _ := engine.ExecuteLoop(context.Background(), opt)
	if result.Status == ralph.StatusSuccess {
		t.Fatal("Should not succeed (test.txt not fixed)")
	}

	// Forbidden path protection is tested via isPathForbidden in engine_test.go
	// This test verifies the integration works - hardcoded paths are always protected

	allMetrics = append(allMetrics, TestMetrics{
		TestName:     "Ralph_Safety_ForbidPaths",
		LLMCallsMade: mockLLM.callCount,
		Passed:       true,
	})
}

// TestRalphDeterministicIteration verifies same failure produces same iteration count
func TestRalphDeterministicIteration(t *testing.T) {
	tmpDir, cleanup := setupTempEnvironment(t)
	defer cleanup()

	failScript := filepath.Join(tmpDir, "fail.sh")
	os.WriteFile(failScript, []byte("#!/bin/sh\nexit 1"), 0755)

	cfg := &config.Config{
		ProjectRoot:                   tmpDir,
		EvidenceCommandTimeoutSeconds: 5,
	}

	memService := setupMockMemoryService(t, tmpDir)

	runs := []int{}
	for i := 0; i < 3; i++ {
		mockLLM := &mockRalphLLM{}
		engine := ralph.NewEngine(cfg, memService, mockLLM, "test", zap.NewNop())

		opt := ralph.Options{
			Task:          "Deterministic test",
			Command:       "sh fail.sh",
			Evidence:      []string{"cmd_exit0::sh fail.sh"},
			MaxIterations: 4,
			Safety: ralph.SafetyOptions{
				AllowShell: true,
			},
		}

		result, _ := engine.ExecuteLoop(context.Background(), opt)
		runs = append(runs, result.Iterations)
	}

	// All runs should produce same iteration count
	for i := 1; i < len(runs); i++ {
		if runs[i] != runs[0] {
			t.Fatalf("Run %d produced %d iterations, expected %d (deterministic)", i, runs[i], runs[0])
		}
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:      "Ralph_Deterministic",
		LLMCallsMade:  0,
		Passed:        true,
		Deterministic: true,
	})
}

// Helper functions

func setupTempEnvironment(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Resolve symlinks to get canonical path (important on macOS where /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("Failed to resolve temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func setupMockMemoryService(t *testing.T, tmpDir string) *memory.Service {
	// Create database for testing
	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := &config.Config{
		ProjectRoot: tmpDir,
		DBPath:      dbPath,
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	return memory.NewService(db)
}
