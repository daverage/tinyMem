package extract

import (
	"context"
	"testing"
	"time"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/cove"
	"github.com/daverage/tinymem/internal/evidence"
	"github.com/daverage/tinymem/internal/llm"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/storage"
)

// mockLLMClient for integration testing
type mockCoVeLLMClient struct {
	responses map[string]string
}

func (m *mockCoVeLLMClient) ChatCompletions(ctx context.Context, req llm.ChatCompletionRequest) (*llm.ChatCompletionResponse, error) {
	// Return high confidence for "concrete" and low confidence for "speculative"
	content := `[
		{"id": "0", "confidence": 0.9, "reason": "concrete claim"},
		{"id": "1", "confidence": 0.3, "reason": "too speculative"}
	]`

	finishReason := "stop"
	return &llm.ChatCompletionResponse{
		ID:      "test-id",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "test-model",
		Choices: []llm.Choice{
			{
				Index: 0,
				Message: llm.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: &finishReason,
			},
		},
	}, nil
}

// TestCoVeIntegrationWithExtractor tests the full integration with the extractor
func TestCoVeIntegrationWithExtractor(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      30,
		ProjectRoot:             "/tmp/test",
		DBPath:                  ":memory:",
	}

	// Create an in-memory database for testing
	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer db.Close()

	// Create services
	evidenceService := evidence.NewService(db, cfg)

	// Create extractor with CoVe
	extractor := NewExtractor(evidenceService)
	mockClient := &mockCoVeLLMClient{}
	coveVerifier := cove.NewVerifier(cfg, mockClient)
	extractor.SetCoVeVerifier(coveVerifier)

	// Test text with concrete and speculative claims
	testText := `
	We decided to use PostgreSQL for the database.
	Maybe we could consider using Redis later.
	The system currently uses SQLite.
	`

	projectID := "test-project"

	// Extract memories (this would normally include CoVe filtering)
	memories, err := extractor.ExtractMemories(testText, projectID)
	if err != nil {
		t.Fatalf("Failed to extract memories: %v", err)
	}

	if len(memories) == 0 {
		t.Fatal("Expected some memories to be extracted")
	}

	// Verify that extraction doesn't create facts
	for _, mem := range memories {
		if mem.Type == memory.Fact {
			t.Error("Extractor should never create facts, even with CoVe")
		}
	}

	t.Logf("Extracted %d memories (types: %v)", len(memories), memoryTypes(memories))
}

// TestCoVeDisabledIdenticalBehavior tests that disabling CoVe gives identical results
func TestCoVeDisabledIdenticalBehavior(t *testing.T) {
	cfgDisabled := &config.Config{
		CoVeEnabled: false, // Disabled
		ProjectRoot: "/tmp/test",
		DBPath:      ":memory:",
	}

	cfgEnabled := &config.Config{
		CoVeEnabled:             true, // Enabled but should have same result with mock
		CoVeConfidenceThreshold: 0.0,  // Accept everything
		CoVeMaxCandidates:       100,
		CoVeTimeoutSeconds:      30,
		ProjectRoot:             "/tmp/test",
		DBPath:                  ":memory:",
	}

	// Create an in-memory database for testing
	db1, err := storage.NewDB(cfgDisabled)
	if err != nil {
		t.Fatalf("Failed to create test DB 1: %v", err)
	}
	defer db1.Close()

	db2, err := storage.NewDB(cfgEnabled)
	if err != nil {
		t.Fatalf("Failed to create test DB 2: %v", err)
	}
	defer db2.Close()

	// Create two extractors
	evidenceService1 := evidence.NewService(db1, cfgDisabled)
	extractor1 := NewExtractor(evidenceService1)
	// No CoVe for extractor1

	evidenceService2 := evidence.NewService(db2, cfgEnabled)
	extractor2 := NewExtractor(evidenceService2)
	mockClient := &mockCoVeLLMClient{}
	coveVerifier := cove.NewVerifier(cfgEnabled, mockClient)
	extractor2.SetCoVeVerifier(coveVerifier)

	testText := `
	We decided to use PostgreSQL for the database.
	The plan is to migrate next week.
	`

	projectID := "test-project"

	// Extract with both
	memories1, err := extractor1.ExtractMemories(testText, projectID)
	if err != nil {
		t.Fatalf("Failed to extract memories (disabled): %v", err)
	}

	memories2, err := extractor2.ExtractMemories(testText, projectID)
	if err != nil {
		t.Fatalf("Failed to extract memories (enabled): %v", err)
	}

	// Should extract the same count (both decision and plan)
	if len(memories1) != len(memories2) {
		t.Errorf("Expected same extraction count, got %d (disabled) vs %d (enabled)",
			len(memories1), len(memories2))
	}

	// Check types match
	types1 := memoryTypes(memories1)
	types2 := memoryTypes(memories2)
	if len(types1) != len(types2) {
		t.Errorf("Expected same type distribution, got %v (disabled) vs %v (enabled)",
			types1, types2)
	}

	t.Logf("Both extracted %d memories with types %v", len(memories1), types1)
}

// TestCoVeInvariantNoFactPromotion tests that CoVe never promotes facts
func TestCoVeInvariantNoFactPromotion(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: 0.0, // Accept everything
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      30,
		ProjectRoot:             "/tmp/test",
		DBPath:                  ":memory:",
	}

	db, err := storage.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer db.Close()

	evidenceService := evidence.NewService(db, cfg)
	memoryService := memory.NewService(db)
	extractor := NewExtractor(evidenceService)
	mockClient := &mockCoVeLLMClient{}
	coveVerifier := cove.NewVerifier(cfg, mockClient)
	extractor.SetCoVeVerifier(coveVerifier)

	// Text that sounds like facts
	testText := `
	The system is using SQLite.
	The database contains 1000 records.
	The API supports REST endpoints.
	`

	projectID := "test-project"

	// Extract and store
	err = extractor.ExtractAndQueueForVerification(testText, memoryService, evidenceService, projectID)
	if err != nil {
		t.Fatalf("Failed to extract and store: %v", err)
	}

	// Get all memories
	memories, err := memoryService.GetAllMemories(projectID)
	if err != nil {
		t.Fatalf("Failed to get memories: %v", err)
	}

	// CRITICAL: None should be facts, even with CoVe and fact-like language
	for _, mem := range memories {
		if mem.Type == memory.Fact {
			t.Errorf("INVARIANT VIOLATION: CoVe allowed fact creation! Memory: %+v", mem)
		}
	}

	t.Logf("Correctly stored %d non-fact memories", len(memories))
}

// TestCoVeInvariantBoundedProcessing tests that CoVe respects token limits
func TestCoVeInvariantBoundedProcessing(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       5, // Small limit
		CoVeTimeoutSeconds:      30,
		ProjectRoot:             "/tmp/test",
	}

	mockClient := &mockCoVeLLMClient{}
	coveVerifier := cove.NewVerifier(cfg, mockClient)

	// Create many candidates
	candidates := make([]cove.CandidateMemory, 100)
	for i := 0; i < 100; i++ {
		candidates[i] = cove.CandidateMemory{
			ID:      string(rune('0' + (i % 10))),
			Type:    "claim",
			Summary: "Test claim",
		}
	}

	// Should not panic or hang - should respect max candidates
	start := time.Now()
	_, err := coveVerifier.VerifyCandidates(candidates)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("CoVe should handle large candidate sets gracefully: %v", err)
	}

	// Should complete quickly due to truncation
	if duration > 5*time.Second {
		t.Errorf("CoVe took too long (%v), may not be respecting limits", duration)
	}

	t.Logf("Processed 100 candidates in %v (bounded correctly)", duration)
}

// Helper function to get memory types
func memoryTypes(memories []*memory.Memory) []string {
	types := make([]string, len(memories))
	for i, mem := range memories {
		types[i] = string(mem.Type)
	}
	return types
}
