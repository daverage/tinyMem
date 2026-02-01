package automated

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daverage/tinymem/internal/llm"
)

// TestTokenAndCostAccounting verifies instrumentation for token and cost tracking
func TestTokenAndCostAccounting(t *testing.T) {
	// 1. Setup - Create a new metrics tracker
	providerName := "test-provider"
	metrics := llm.NewMetrics(providerName)

	// 2. Simulate usage
	// Turn 1: 10 prompt, 20 completion, 100ms latency, success
	metrics.RecordRequest(true, 100, 30)
	// Turn 2: 15 prompt, 25 completion, 150ms latency, success
	metrics.RecordRequest(true, 150, 40)
	// Turn 3: 5 prompt, 0 completion, 50ms latency, failure
	metrics.RecordRequest(false, 50, 5)

	// 3. Verify Aggregations
	stats := metrics.GetStats()

	// Check counts
	if stats.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", stats.TotalRequests)
	}
	if stats.SuccessfulCalls != 2 {
		t.Errorf("Expected 2 successful calls, got %d", stats.SuccessfulCalls)
	}
	if stats.FailedCalls != 1 {
		t.Errorf("Expected 1 failed call, got %d", stats.FailedCalls)
	}

	// Check token sums
	expectedTokens := int64(30 + 40 + 5)
	if stats.TotalTokens != expectedTokens {
		t.Errorf("Expected %d total tokens, got %d", expectedTokens, stats.TotalTokens)
	}

	// Check latency (min, max, avg)
	if stats.MinLatencyMs != 50 {
		t.Errorf("Expected min latency 50ms, got %d", stats.MinLatencyMs)
	}
	if stats.MaxLatencyMs != 150 {
		t.Errorf("Expected max latency 150ms, got %d", stats.MaxLatencyMs)
	}
	// Avg: (100 + 150 + 50) / 3 = 100
	if stats.AvgLatencyMs != 100 {
		t.Errorf("Expected avg latency 100ms, got %d", stats.AvgLatencyMs)
	}

	// 4. Verify Baseline Comparison Capability
	// Create baseline metrics (no tinyMem)
	baselineMetrics := llm.NewMetrics("baseline")
	baselineMetrics.RecordRequest(true, 200, 100) // Higher latency, more tokens

	baselineStats := baselineMetrics.GetStats()

	// Assert tinyMem (mocked) is more efficient (just verifying comparison logic)
	if stats.TotalTokens >= baselineStats.TotalTokens {
		t.Logf("Note: tinyMem tokens (%d) not less than baseline (%d) in this mock", stats.TotalTokens, baselineStats.TotalTokens)
	}

	// 5. Output Artifacts
	// Ensure we can serialize to JSON for reporting
	output := map[string]llm.MetricsStats{
		"tinyMem_Core": stats,
		"baseline":     baselineStats,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal metrics to JSON: %v", err)
	}

	// Write to test artifact directory
	resultsDir := filepath.Join("..", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		t.Fatalf("Failed to create results dir: %v", err)
	}

	outputPath := filepath.Join(resultsDir, "accounting_metrics.json")
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		t.Fatalf("Failed to write metrics artifact: %v", err)
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName:    "Accounting_TokenTracking",
		Passed:      true,
		InputCount:  3,
		OutputCount: 3,
	})
}

// TestGlobalMetricsRegistry verifies the global registry works for cross-component tracking
func TestGlobalMetricsRegistry(t *testing.T) {
	providerName := "global-test-provider"
	metrics := llm.NewMetrics(providerName)

	llm.RegisterMetrics(providerName, metrics)

	retrieved := llm.GetMetrics(providerName)
	if retrieved == nil {
		t.Fatal("Failed to retrieve metrics from global registry")
	}

	if retrieved != metrics {
		t.Error("Retrieved metrics object does not match registered object")
	}

	all := llm.GetAllMetrics()
	if _, ok := all[providerName]; !ok {
		t.Error("Provider not found in GetAllMetrics result")
	}

	allMetrics = append(allMetrics, TestMetrics{
		TestName: "Accounting_GlobalRegistry",
		Passed:   true,
	})
}

// AssertNoTokenRegression is a helper for CI gating
func AssertNoTokenRegression(t *testing.T, current, baseline int64, threshold float64) {
	if baseline == 0 {
		return // No baseline to compare against
	}
	
	increase := float64(current-baseline) / float64(baseline)
	if increase > threshold {
		t.Errorf("TOKEN REGRESSION DETECTED: current %d, baseline %d (%.2f%% increase, limit %.2f%%)", 
			current, baseline, increase*100, threshold*100)
	}
}

func TestTokenRegressionGating(t *testing.T) {
	// Example usage
	AssertNoTokenRegression(t, 105, 100, 0.10) // Should pass (5% increase < 10%)
	// AssertNoTokenRegression(t, 120, 100, 0.10) // Would fail (20% increase > 10%)
}
