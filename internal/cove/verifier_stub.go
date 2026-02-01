//go:build nollm

package cove

import (
	"context"

	"github.com/daverage/tinymem/internal/config"
)

// Verifier stub for no-llm builds
type Verifier struct{}

// NewVerifier returns a stub verifier for no-llm builds
func NewVerifier(cfg *config.Config, llmClient LLMClient) *Verifier {
	return &Verifier{}
}

// SetStatsStore is a no-op for no-llm builds
func (v *Verifier) SetStatsStore(store StatsStore, projectID string) {}

// VerifyCandidates is a no-op that returns all candidates unfiltered
func (v *Verifier) VerifyCandidates(candidates []CandidateMemory) ([]CandidateMemory, error) {
	// Without LLM support, CoVe is unavailable - return all candidates unverified
	return candidates, nil
}

// FilterRecall is a no-op that returns all memories unfiltered
func (v *Verifier) FilterRecall(ctx context.Context, memories []RecallMemory, query string) ([]RecallMemory, error) {
	// Without LLM support, CoVe recall filtering is unavailable - return all memories
	return memories, nil
}

// GetStats returns empty stats for no-llm builds
func (v *Verifier) GetStats() Stats {
	return Stats{}
}

// ResetStats is a no-op for no-llm builds
func (v *Verifier) ResetStats() {}
