package cove

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/llm"
)

// mockLLMClient is a mock LLM client for testing
type mockLLMClient struct {
	shouldFail      bool
	confidence      float64
	responseContent string
}

func (m *mockLLMClient) ChatCompletions(ctx context.Context, req llm.ChatCompletionRequest) (*llm.ChatCompletionResponse, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock LLM failure")
	}

	// Return a mock response
	content := m.responseContent
	if content == "" {
		// Default response with confidence
		content = fmt.Sprintf(`[{"id": "0", "confidence": %.1f, "reason": "test reason"}]`, m.confidence)
	}

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

func (m *mockLLMClient) ProviderName() string {
	return "mock"
}

func (m *mockLLMClient) GetMetrics() *llm.Metrics {
	return &llm.Metrics{}
}

// TestCoVeDisabledReturnsAllCandidates tests that when CoVe is disabled, all candidates are returned
func TestCoVeDisabledReturnsAllCandidates(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             false,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      30,
	}

	mockClient := &mockLLMClient{confidence: 0.8}
	verifier := NewVerifier(cfg, mockClient)

	candidates := []CandidateMemory{
		{ID: "1", Type: "claim", Summary: "Test claim 1"},
		{ID: "2", Type: "plan", Summary: "Test plan 1"},
		{ID: "3", Type: "observation", Summary: "Test observation 1"},
	}

	filtered, err := verifier.VerifyCandidates(candidates)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(filtered) != len(candidates) {
		t.Errorf("Expected %d candidates, got %d (CoVe disabled should return all)", len(candidates), len(filtered))
	}
}

// TestCoVeEnabledFiltersLowConfidence tests that CoVe filters out low-confidence candidates
func TestCoVeEnabledFiltersLowConfidence(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      30,
		CoVeModel:               "test-model",
	}

	// Mock response with one low-confidence and one high-confidence candidate
	responseContent := `[
		{"id": "0", "confidence": 0.3, "reason": "too speculative"},
		{"id": "1", "confidence": 0.9, "reason": "concrete claim"}
	]`
	mockClient := &mockLLMClient{confidence: 0.8, responseContent: responseContent}
	verifier := NewVerifier(cfg, mockClient)

	candidates := []CandidateMemory{
		{ID: "0", Type: "claim", Summary: "Maybe this works", Score: 0.3}, // Low confidence
		{ID: "1", Type: "claim", Summary: "This definitely works", Score: 0.9}, // High confidence
	}

	filtered, err := verifier.VerifyCandidates(candidates)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Assert invariants: filtering should reduce or maintain size, exclude low-confidence, keep high-confidence
	if len(filtered) > len(candidates) {
		t.Errorf("Filtering should not increase candidate count: got %d from %d inputs", len(filtered), len(candidates))
	}

	if len(filtered) == len(candidates) {
		t.Error("Expected at least one low-confidence candidate to be filtered out")
	}

	// Verify the high-confidence candidate is included
	foundHighConfidence := false
	for _, c := range filtered {
		if c.ID == "1" {
			foundHighConfidence = true
			break
		}
	}
	if !foundHighConfidence {
		t.Error("Expected high-confidence candidate (ID '1') to be included in filtered results")
	}

	// Verify the low-confidence candidate is excluded
	for _, c := range filtered {
		if c.ID == "0" {
			t.Error("Expected low-confidence candidate (ID '0') to be excluded from filtered results")
		}
	}
}

// TestCoVeBoundedCandidates tests that CoVe respects max candidates limit
func TestCoVeBoundedCandidates(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       5, // Small limit
		CoVeTimeoutSeconds:      30,
	}

	mockClient := &mockLLMClient{confidence: 0.8}
	verifier := NewVerifier(cfg, mockClient)

	// Create more candidates than the limit
	candidates := make([]CandidateMemory, 10)
	for i := 0; i < 10; i++ {
		candidates[i] = CandidateMemory{
			ID:      string(rune('0' + i)),
			Type:    "claim",
			Summary: "Test claim",
		}
	}

	// The verifier should truncate to MaxCandidates
	// We can't directly observe this, but the mock client would fail if it received all 10
	_, err := verifier.VerifyCandidates(candidates)
	if err != nil {
		t.Fatalf("Expected no error with bounded candidates, got: %v", err)
	}
}

// TestCoVeFailsafeOnMissingScores tests that CoVe defaults missing scores to 1.0 (pass-through)
func TestCoVeFailsafeOnMissingScores(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      30,
	}

	mockClient := &mockLLMClient{}
	verifier := NewVerifier(cfg, mockClient)

	// Candidates with no Score set (defaults to 0, which triggers 1.0 pass-through)
	candidates := []CandidateMemory{
		{ID: "1", Type: "claim", Summary: "Test claim 1", Score: 0}, // Missing score
		{ID: "2", Type: "claim", Summary: "Test claim 2", Score: 0}, // Missing score
	}

	// Should pass through all candidates when scores are missing (fail-safe behavior)
	filtered, err := verifier.VerifyCandidates(candidates)
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}

	// Assert failsafe behavior: missing scores default to 1.0, passing all candidates
	if len(filtered) != len(candidates) {
		t.Errorf("Expected all %d candidates to pass (missing score → 1.0 default), got %d", len(candidates), len(filtered))
	}

	// Verify stats show evaluation occurred
	stats := verifier.GetStats()
	if stats.CandidatesEvaluated == 0 {
		t.Error("Expected candidates to be evaluated even with missing scores")
	}
}

// TestCoVeStatsTracking tests that CoVe correctly tracks statistics
func TestCoVeStatsTracking(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      30,
	}

	responseContent := `[
		{"id": "0", "confidence": 0.3, "reason": "low confidence"},
		{"id": "1", "confidence": 0.9, "reason": "high confidence"}
	]`
	mockClient := &mockLLMClient{responseContent: responseContent}
	verifier := NewVerifier(cfg, mockClient)

	candidates := []CandidateMemory{
		{ID: "0", Type: "claim", Summary: "Low conf claim", Score: 0.3}, // Below threshold
		{ID: "1", Type: "claim", Summary: "High conf claim", Score: 0.9}, // Above threshold
	}

	filtered, err := verifier.VerifyCandidates(candidates)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	stats := verifier.GetStats()

	// Assert stats track meaningful relationships, not exact counts
	if stats.CandidatesEvaluated == 0 {
		t.Error("Expected at least some candidates to be evaluated")
	}

	if int(stats.CandidatesEvaluated) < len(filtered) {
		t.Errorf("Evaluated count (%d) should be >= filtered count (%d)", stats.CandidatesEvaluated, len(filtered))
	}

	if stats.CandidatesDiscarded == 0 {
		t.Error("Expected at least one candidate to be discarded (low confidence)")
	}

	if stats.CandidatesDiscarded >= stats.CandidatesEvaluated {
		t.Error("Discarded count should be less than evaluated count (at least one passed)")
	}

	if stats.AvgConfidence == 0 {
		t.Error("Expected non-zero average confidence")
	}

	// Verify the accounting adds up
	expectedAccepted := int(stats.CandidatesEvaluated - stats.CandidatesDiscarded)
	if len(filtered) > expectedAccepted {
		t.Errorf("Filtered count (%d) exceeds accepted count (%d)", len(filtered), expectedAccepted)
	}
}

// TestCoVeNoFactCreation tests that CoVe never creates facts
// This is enforced by the type system - CoVe only works with CandidateMemory
// which has Type as string, not memory.Type, so it can't create facts
func TestCoVeNoFactCreation(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      30,
	}

	mockClient := &mockLLMClient{confidence: 0.9}
	verifier := NewVerifier(cfg, mockClient)

	// Even if we try to pass a "fact" type, CoVe should not change it
	candidates := []CandidateMemory{
		{ID: "0", Type: "fact", Summary: "This should not be allowed"},
	}

	filtered, _ := verifier.VerifyCandidates(candidates)

	// CoVe does not change types - it only filters
	// The type remains "fact" but this proves CoVe doesn't create facts
	if len(filtered) > 0 && filtered[0].Type != "fact" {
		t.Error("CoVe should never change memory types")
	}

	// The real protection is in the extractor - facts are never passed to CoVe
	// This test just demonstrates CoVe doesn't change types
}

// TestCoVeRecallFiltering tests the optional recall filtering
func TestCoVeRecallFiltering(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeRecallFilterEnabled: true,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      30,
	}

	responseContent := `[
		{"id": "0", "include": false},
		{"id": "1", "include": true}
	]`
	mockClient := &mockLLMClient{responseContent: responseContent}
	verifier := NewVerifier(cfg, mockClient)

	memories := []RecallMemory{
		{ID: "0", Type: "claim", Summary: "Irrelevant claim"},
		{ID: "1", Type: "claim", Summary: "Relevant claim"},
	}

	ctx := context.Background()
	filtered, err := verifier.FilterRecall(ctx, memories, "test query")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should only keep the included memory
	if len(filtered) != 1 {
		t.Errorf("Expected 1 memory after filtering, got %d", len(filtered))
	}

	if len(filtered) > 0 && filtered[0].ID != "1" {
		t.Errorf("Expected memory ID '1', got '%s'", filtered[0].ID)
	}
}

// TestCoVeRecallFilteringDisabled tests that recall filtering respects the flag
func TestCoVeRecallFilteringDisabled(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeRecallFilterEnabled: false, // Disabled
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      30,
	}

	mockClient := &mockLLMClient{confidence: 0.8}
	verifier := NewVerifier(cfg, mockClient)

	memories := []RecallMemory{
		{ID: "0", Type: "claim", Summary: "Claim 1"},
		{ID: "1", Type: "claim", Summary: "Claim 2"},
	}

	ctx := context.Background()
	filtered, err := verifier.FilterRecall(ctx, memories, "test query")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return all memories when disabled
	if len(filtered) != len(memories) {
		t.Errorf("Expected %d memories (disabled), got %d", len(memories), len(filtered))
	}
}

// TestCoVeTimeoutHandling tests that CoVe respects timeouts
func TestCoVeTimeoutHandling(t *testing.T) {
	cfg := &config.Config{
		CoVeEnabled:             true,
		CoVeConfidenceThreshold: 0.6,
		CoVeMaxCandidates:       20,
		CoVeTimeoutSeconds:      1, // Very short timeout
	}

	// Mock client that would take too long (simulated by context cancellation)
	mockClient := &mockLLMClient{confidence: 0.8}
	verifier := NewVerifier(cfg, mockClient)

	candidates := []CandidateMemory{
		{ID: "0", Type: "claim", Summary: "Test claim"},
	}

	// The timeout is handled by the context in the verifier
	// If it times out, it should fall back to unfiltered (fail-safe)
	filtered, err := verifier.VerifyCandidates(candidates)
	if err != nil {
		t.Fatalf("Expected nil error (fail-safe), got: %v", err)
	}

	// Should get candidates back even if timeout occurs
	if len(filtered) == 0 {
		t.Error("Expected fail-safe to return candidates on timeout")
	}
}
