package automated

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/cove"
	"github.com/daverage/tinymem/internal/llm"
)

// TestMetrics tracks verification metrics
type TestMetrics struct {
	TestName          string  `json:"test_name"`
	LLMCallsMade      int     `json:"llm_calls_made"`
	Passed            bool    `json:"passed"`
	InputCount        int     `json:"input_count"`
	OutputCount       int     `json:"output_count"`
	Threshold         float64 `json:"threshold,omitempty"`
	Deterministic     bool    `json:"deterministic,omitempty"`
	FailSafeTriggered bool    `json:"failsafe_triggered,omitempty"`
}

var allMetrics []TestMetrics

// mockLLMClient tracks calls to prove zero LLM usage
type mockLLMClient struct {
	callCount int
}

func (m *mockLLMClient) ChatCompletions(ctx context.Context, req llm.ChatCompletionRequest) (*llm.ChatCompletionResponse, error) {
	m.callCount++
	// Should never be called for CoVe filtering in this test (uses pre-computed scores)
	panic("LLM called - CoVe should use pre-computed scores only in this test!")
}

// TestCoVeFilteringWithThresholds verifies threshold-based filtering with zero LLM calls
func TestCoVeFilteringWithThresholds(t *testing.T) {
	thresholds := []float64{0.3, 0.5, 0.7, 0.9}

	// Fixed test data with known scores
	candidates := []cove.CandidateMemory{
		{ID: "1", Type: "note", Summary: "High confidence", Score: 0.95},
		{ID: "2", Type: "note", Summary: "Medium-high confidence", Score: 0.75},
		{ID: "3", Type: "note", Summary: "Medium confidence", Score: 0.55},
		{ID: "4", Type: "note", Summary: "Low confidence", Score: 0.35},
		{ID: "5", Type: "note", Summary: "Very low confidence", Score: 0.15},
	}

	expectedCounts := map[float64]int{
		0.3:  4, // Scores >= 0.3: 0.95, 0.75, 0.55, 0.35
		0.5:  3, // Scores >= 0.5: 0.95, 0.75, 0.55
		0.7:  2, // Scores >= 0.7: 0.95, 0.75
		0.9:  1, // Scores >= 0.9: 0.95
	}

	for _, threshold := range thresholds {
		t.Run("Threshold_"+floatToString(threshold), func(t *testing.T) {
			mockClient := &mockLLMClient{}
			cfg := &config.Config{
				CoVeEnabled:            true,
				CoVeConfidenceThreshold: threshold,
				CoVeMaxCandidates:      20,
			}

			verifier := cove.NewVerifier(cfg, mockClient)

			filtered, err := verifier.VerifyCandidates(candidates)
			if err != nil {
				t.Fatalf("CoVe verification failed: %v", err)
			}

			// Verify LLM was never called
			if mockClient.callCount > 0 {
				t.Fatalf("LLM was called %d times - CoVe should use pre-computed scores only in this test!", mockClient.callCount)
			}

			// Verify exact count matches expected
			expected := expectedCounts[threshold]
			if len(filtered) != expected {
				t.Fatalf("Threshold %.2f: expected %d results, got %d", threshold, expected, len(filtered))
			}

			// Verify all retained candidates meet threshold
			for _, cand := range filtered {
				if cand.Score < threshold {
					t.Fatalf("Candidate %s with score %.2f passed threshold %.2f", cand.ID, cand.Score, threshold)
				}
			}

			// Record metrics
			allMetrics = append(allMetrics, TestMetrics{
				TestName:     "CoVe_Threshold_" + floatToString(threshold),
				LLMCallsMade: mockClient.callCount,
				Passed:       true,
				InputCount:   len(candidates),
				OutputCount:  len(filtered),
				Threshold:    threshold,
			})
		})
	}
}

// TestCoVeZeroRecall verifies explicit zero-recall with high threshold
func TestCoVeZeroRecall(t *testing.T) {
	candidates := []cove.CandidateMemory{
		{ID: "1", Type: "note", Summary: "Low score", Score: 0.3},
		{ID: "2", Type: "note", Summary: "Low score", Score: 0.4},
		{ID: "3", Type: "note", Summary: "Low score", Score: 0.5},
	}

	mockClient := &mockLLMClient{}
	cfg := &config.Config{
		CoVeEnabled:            true,
		CoVeConfidenceThreshold: 0.99, // Impossibly high
		CoVeMaxCandidates:      20,
	}

	verifier := cove.NewVerifier(cfg, mockClient)
	filtered, err := verifier.VerifyCandidates(candidates)

	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	// Verify zero-recall state
	if len(filtered) != 0 {
		t.Fatalf("Expected zero-recall (empty result), got %d results", len(filtered))
	}

	// Verify no LLM calls
	if mockClient.callCount > 0 {
		t.Fatalf("LLM called during zero-recall test")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:     "CoVe_ZeroRecall",
		LLMCallsMade: mockClient.callCount,
		Passed:       true,
		InputCount:   len(candidates),
		OutputCount:  len(filtered),
		Threshold:    0.99,
	})
}

// TestCoVeDeterminism verifies same input produces same output
func TestCoVeDeterminism(t *testing.T) {
	candidates := []cove.CandidateMemory{
		{ID: "1", Type: "note", Summary: "Test 1", Score: 0.8},
		{ID: "2", Type: "note", Summary: "Test 2", Score: 0.6},
		{ID: "3", Type: "note", Summary: "Test 3", Score: 0.4},
	}

	cfg := &config.Config{
		CoVeEnabled:            true,
		CoVeConfidenceThreshold: 0.5,
		CoVeMaxCandidates:      20,
	}

	var results [][]string

	// Run 5 times
	for i := 0; i < 5; i++ {
		mockClient := &mockLLMClient{}
		verifier := cove.NewVerifier(cfg, mockClient)
		filtered, err := verifier.VerifyCandidates(candidates)

		if err != nil {
			t.Fatalf("Run %d failed: %v", i+1, err)
		}

		// Extract IDs in order
		ids := make([]string, len(filtered))
		for j, cand := range filtered {
			ids[j] = cand.ID
		}
		results = append(results, ids)
	}

	// Verify all runs produced identical results
	firstRun := results[0]
	for i, run := range results[1:] {
		if len(run) != len(firstRun) {
			t.Fatalf("Run %d produced different count: %d vs %d", i+2, len(run), len(firstRun))
		}
		for j, id := range run {
			if id != firstRun[j] {
				t.Fatalf("Run %d produced different order at position %d: %s vs %s", i+2, j, id, firstRun[j])
			}
		}
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:      "CoVe_Determinism",
		LLMCallsMade:  0,
		Passed:        true,
		InputCount:    len(candidates),
		OutputCount:   len(firstRun),
		Deterministic: true,
	})
}

// TestCoVeExtremeZeroRecall verifies behavior when threshold exceeds all scores
func TestCoVeExtremeZeroRecall(t *testing.T) {
	candidates := []cove.CandidateMemory{
		{ID: "1", Score: 0.1},
		{ID: "2", Score: 0.2},
	}

	mockClient := &mockLLMClient{}
	cfg := &config.Config{
		CoVeEnabled:            true,
		CoVeConfidenceThreshold: 0.9,
	}

	verifier := cove.NewVerifier(cfg, mockClient)
	filtered, err := verifier.VerifyCandidates(candidates)
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	if len(filtered) != 0 {
		t.Fatalf("Expected zero results for high threshold, got %d", len(filtered))
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:    "CoVe_ExtremeZeroRecall",
		Passed:      true,
		InputCount:  len(candidates),
		OutputCount: 0,
	})
}

// TestCoVeStrongDeterminism verifies exact ordering across multiple runs
func TestCoVeStrongDeterminism(t *testing.T) {
	candidates := []cove.CandidateMemory{
		{ID: "A", Score: 0.9},
		{ID: "B", Score: 0.8},
		{ID: "C", Score: 0.7},
	}

	cfg := &config.Config{
		CoVeEnabled:            true,
		CoVeConfidenceThreshold: 0.5,
	}

	runOnce := func() []string {
		verifier := cove.NewVerifier(cfg, &mockLLMClient{})
		filtered, _ := verifier.VerifyCandidates(candidates)
		ids := make([]string, len(filtered))
		for i, c := range filtered {
			ids[i] = c.ID
		}
		return ids
	}

	first := runOnce()
	for i := 0; i < 5; i++ {
		next := runOnce()
		if len(first) != len(next) {
			t.Fatal("Determinism failed: different counts")
		}
		for j := range first {
			if first[j] != next[j] {
				t.Fatalf("Determinism failed at index %d: %s != %s", j, first[j], next[j])
			}
		}
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:      "CoVe_StrongDeterminism",
		Passed:        true,
		Deterministic: true,
	})
}

// TestMain runs all tests and outputs metrics
func TestMain(m *testing.M) {
	code := m.Run()

	// Output metrics to JSON
	metricsFile := filepath.Join("..", "results", "cove_metrics.json")
	os.MkdirAll(filepath.Join("..", "results"), 0755)

	data, _ := json.MarshalIndent(allMetrics, "", "  ")
	os.WriteFile(metricsFile, data, 0644)

	os.Exit(code)
}

func floatToString(f float64) string {
	return map[float64]string{
		0.3: "0.3",
		0.5: "0.5",
		0.7: "0.7",
		0.9: "0.9",
		0.99: "0.99",
	}[f]
}
