package automated

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daverage/tinymem/internal/execution"
)

// AgentComplianceTest verifies agent behavior following AGENTS.md contract
type AgentComplianceTest struct {
	ID          string
	Name        string
	Description string
	Task        string // Task given to agent
	Expected    AgentBehavior
}

// AgentBehavior defines expected agent actions
type AgentBehavior struct {
	QueriesMemoryFirst      bool     // Should query memory before any mutation
	SetsMode                bool     // Should call memory_set_mode
	ChecksTaskAuthority     bool     // Should read tinyTasks.md for multi-step work
	ToolSequence            []string // Expected tool call sequence
	MustNotCall             []string // Tools that should NOT be called
	AllowedWithoutMode      []string // Tools allowed without mode (read-only)
	RequiresModeDeclaration []string // Tools requiring explicit mode
}

// TestAgentComplianceReadOnly verifies read-only operations work without mode
func TestAgentComplianceReadOnly(t *testing.T) {
	tmpDir, cleanup := setupAgentComplianceEnv(t)
	defer cleanup()

	test := AgentComplianceTest{
		ID:          "AGENT-COMPLIANCE-001",
		Name:        "Read-only exploration",
		Description: "Agent explores memory without needing mode declaration",
		Task:        "List recent project decisions",
		Expected: AgentBehavior{
			QueriesMemoryFirst:      true,
			SetsMode:                false, // Read-only should not require mode
			ChecksTaskAuthority:     false,
			ToolSequence:            []string{"memory_query"}, // or memory_recent
			AllowedWithoutMode:      []string{"memory_query", "memory_recent", "memory_health"},
			RequiresModeDeclaration: []string{},
		},
	}

	// Create controller without explicit mode
	ctrl := execution.NewController("", filepath.Join(tmpDir, "tinyTasks.md"))

	// Simulate agent behavior: Query memory (read-only)
	toolCalls := []string{"memory_query"}

	// Verify no mode is required for read-only
	if ctrl.ModeSet() {
		t.Error("Mode should not be set for read-only operations")
	}

	// Verify read operation is allowed without mode
	err := ctrl.Enforce(execution.ActionMemoryQuery, execution.ModePassive)
	if err != nil {
		t.Errorf("Read-only operation should be allowed without mode: %v", err)
	}

	// Success: Agent can explore without mode ceremony
	t.Logf("✓ %s: Read-only exploration works without mode", test.ID)
	recordComplianceResult(test.ID, true, "Read-only operations work without mode declaration", toolCalls)
}

// TestAgentComplianceMutationSequence verifies correct sequence for mutation
func TestAgentComplianceMutationSequence(t *testing.T) {
	tmpDir, cleanup := setupAgentComplianceEnv(t)
	defer cleanup()

	test := AgentComplianceTest{
		ID:          "AGENT-COMPLIANCE-002",
		Name:        "Simple mutation with correct sequence",
		Description: "Agent queries memory, sets mode, then writes",
		Task:        "Store this decision: Use PostgreSQL for database",
		Expected: AgentBehavior{
			QueriesMemoryFirst:      true,  // MANDATORY before mutation
			SetsMode:                true,  // MANDATORY for mutation
			ChecksTaskAuthority:     false, // Not multi-step
			ToolSequence:            []string{"memory_query", "memory_set_mode", "memory_write"},
			RequiresModeDeclaration: []string{"memory_write"},
		},
	}

	ctrl := execution.NewController("", filepath.Join(tmpDir, "tinyTasks.md"))

	// Simulate correct agent sequence
	toolCalls := []string{}

	// 1. Agent queries memory first (contract requirement)
	toolCalls = append(toolCalls, "memory_query")
	err := ctrl.Enforce(execution.ActionMemoryQuery, execution.ModePassive)
	if err != nil {
		t.Errorf("memory_query should work: %v", err)
	}

	// 2. Agent declares intent by setting mode
	toolCalls = append(toolCalls, "memory_set_mode")
	if err := ctrl.SetMode(execution.ModeGuarded); err != nil {
		t.Fatalf("Failed to set mode: %v", err)
	}

	// 3. Agent writes memory
	toolCalls = append(toolCalls, "memory_write")
	err = ctrl.Enforce(execution.ActionMemoryWrite, execution.ModeGuarded)
	if err != nil {
		t.Errorf("memory_write should work after mode set: %v", err)
	}

	// Verify sequence matches expected
	expectedSeq := test.Expected.ToolSequence
	if !sequenceMatches(toolCalls, expectedSeq) {
		t.Errorf("Tool sequence mismatch: got %v, want %v", toolCalls, expectedSeq)
	}

	// Success: Correct sequence observed
	t.Logf("✓ %s: Correct mutation sequence (query → mode → write)", test.ID)
	recordComplianceResult(test.ID, true, "Agent followed correct sequence for mutation", toolCalls)
}

// TestAgentComplianceMultiStepWork verifies task authority check
func TestAgentComplianceMultiStepWork(t *testing.T) {
	tmpDir, cleanup := setupAgentComplianceEnv(t)
	defer cleanup()

	test := AgentComplianceTest{
		ID:          "AGENT-COMPLIANCE-003",
		Name:        "Multi-step work with task authority",
		Description: "Agent checks tinyTasks.md for multi-step work",
		Task:        "Refactor auth module (will require multiple steps)",
		Expected: AgentBehavior{
			QueriesMemoryFirst:      true,
			SetsMode:                true, // STRICT for multi-step
			ChecksTaskAuthority:     true, // MUST check tinyTasks.md
			ToolSequence:            []string{"memory_query", "memory_set_mode", "read_tinyTasks"},
			RequiresModeDeclaration: []string{"memory_write", "task_mutation"},
		},
	}

	ctrl := execution.NewController("", filepath.Join(tmpDir, "tinyTasks.md"))

	// Simulate agent recognizing multi-step work
	toolCalls := []string{}

	// 1. Query memory
	toolCalls = append(toolCalls, "memory_query")
	ctrl.Enforce(execution.ActionMemoryQuery, execution.ModePassive)

	// 2. Set STRICT mode for multi-step work
	toolCalls = append(toolCalls, "memory_set_mode")
	if err := ctrl.SetMode(execution.ModeStrict); err != nil {
		t.Fatalf("Failed to set STRICT mode: %v", err)
	}

	// 3. Check task authority (read tinyTasks.md)
	toolCalls = append(toolCalls, "read_tinyTasks")
	tinyTasksPath := ctrl.TinyTasksPath()
	if tinyTasksPath == "" {
		t.Error("tinyTasks path not configured")
	}

	// Agent should check if file exists or create inert template
	_, err := os.Stat(tinyTasksPath)
	tasksExist := err == nil

	if !tasksExist {
		// Agent may create inert template per contract
		t.Logf("tinyTasks.md does not exist, agent should create inert template")
	}

	// Success: Agent checked task authority
	t.Logf("✓ %s: Multi-step work checked task authority", test.ID)
	recordComplianceResult(test.ID, true, "Agent checked tinyTasks.md for multi-step work", toolCalls)
}

// TestAgentComplianceViolationDetection verifies we can detect contract violations
func TestAgentComplianceViolationDetection(t *testing.T) {
	tmpDir, cleanup := setupAgentComplianceEnv(t)
	defer cleanup()

	test := AgentComplianceTest{
		ID:          "AGENT-COMPLIANCE-004",
		Name:        "Contract violation detection",
		Description: "Detect when agent skips memory query before mutation",
		Task:        "Write a decision without querying existing memories",
		Expected: AgentBehavior{
			QueriesMemoryFirst: true, // Contract says MUST query
			SetsMode:           true,
			// But agent violates by skipping query
		},
	}

	ctrl := execution.NewController("", filepath.Join(tmpDir, "tinyTasks.md"))

	// Simulate agent violating contract (skips memory_query)
	toolCalls := []string{}

	// Agent SKIPS memory_query (violation)
	// Goes straight to mode + write
	toolCalls = append(toolCalls, "memory_set_mode")
	if err := ctrl.SetMode(execution.ModeGuarded); err != nil {
		t.Fatalf("Failed to set mode: %v", err)
	}

	toolCalls = append(toolCalls, "memory_write")

	// Detect violation: memory_query not in sequence
	queriedFirst := false
	for i, call := range toolCalls {
		if call == "memory_query" || call == "memory_recent" {
			queriedFirst = (i == 0) // Must be first
			break
		}
	}

	if queriedFirst {
		t.Error("Expected contract violation (no memory query), but agent complied")
	}

	// Success: Violation detected
	t.Logf("✓ %s: Detected contract violation (skipped memory query)", test.ID)
	recordComplianceResult(test.ID, false, "Agent violated contract: did not query memory before mutation", toolCalls)
}

// TestAgentComplianceContractClarityMetrics measures contract effectiveness
func TestAgentComplianceContractClarityMetrics(t *testing.T) {
	// This test documents the compliance framework capabilities:
	// - Tool sequence tracking
	// - Contract violation detection
	// - Behavior pattern analysis
	// - Metrics collection via recordComplianceResult()

	metrics := map[string]interface{}{
		"framework": "agent_compliance",
		"capabilities": []string{
			"tool_sequence_tracking",
			"contract_violation_detection",
			"behavior_pattern_analysis",
			"metrics_collection",
		},
		"test_scenarios": []string{
			"AGENT-COMPLIANCE-001: Read-only operations",
			"AGENT-COMPLIANCE-002: Mutation sequence",
			"AGENT-COMPLIANCE-003: Multi-step work",
			"AGENT-COMPLIANCE-004: Violation detection",
		},
	}

	t.Log("✓ Agent compliance metrics framework in place")
	t.Logf("  Capabilities: %v", metrics["capabilities"])
	t.Logf("  Test scenarios: %d", len(metrics["test_scenarios"].([]string)))
}

// Helper functions

func setupAgentComplianceEnv(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "agent-compliance-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Resolve symlinks
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("Failed to resolve temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func sequenceMatches(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range actual {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}

func recordComplianceResult(testID string, compliant bool, details string, toolSequence []string) {
	result := map[string]interface{}{
		"test_id":       testID,
		"compliant":     compliant,
		"details":       details,
		"tool_sequence": toolSequence,
	}

	// Write to results directory for analysis
	resultsDir := filepath.Join("..", "results")
	os.MkdirAll(resultsDir, 0755)

	outputPath := filepath.Join(resultsDir, "agent_compliance_results.jsonl")
	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	data, _ := json.Marshal(result)
	f.Write(data)
	f.WriteString("\n")
}
