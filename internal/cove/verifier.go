package cove

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/llm"
)

// LLMClient is an interface for LLM clients (for testing)
type LLMClient interface {
	ChatCompletions(ctx context.Context, req llm.ChatCompletionRequest) (*llm.ChatCompletionResponse, error)
}

// Verifier provides CoVe (Chain-of-Verification) filtering for memory candidates
type Verifier struct {
	llmClient  LLMClient
	config     *config.Config
	stats      *StatsTracker
	statsStore StatsStore
	projectID  string
}

// NewVerifier creates a new CoVe verifier
func NewVerifier(cfg *config.Config, llmClient LLMClient) *Verifier {
	return &Verifier{
		llmClient: llmClient,
		config:    cfg,
		stats:     NewStatsTracker(),
	}
}

// SetStatsStore enables persistent stats storage.
func (v *Verifier) SetStatsStore(store StatsStore, projectID string) {
	v.statsStore = store
	v.projectID = projectID
}

// VerifyCandidates performs CoVe verification on candidate memories
// Returns filtered candidates that meet the confidence threshold
// IMPORTANT: This NEVER changes memory types or creates facts
func (v *Verifier) VerifyCandidates(candidates []CandidateMemory) ([]CandidateMemory, error) {
	// Safety check: CoVe must be enabled
	if !v.config.CoVeEnabled {
		return candidates, nil
	}

	// Safety check: if no candidates, return empty
	if len(candidates) == 0 {
		return candidates, nil
	}

	// Enforce max candidates limit (bounded processing)
	if len(candidates) > v.config.CoVeMaxCandidates {
		// Truncate to max candidates (bounded processing)
		candidates = candidates[:v.config.CoVeMaxCandidates]
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.config.CoVeTimeoutSeconds)*time.Second)
	defer cancel()

	// Call LLM for verification
	results, err := v.callLLMForCandidateVerification(ctx, candidates)
	if err != nil {
		// FAIL-SAFE: On error, return all candidates unfiltered
		v.stats.RecordError()
		v.persistStats()
		return candidates, nil
	}

	// Filter candidates based on confidence threshold
	filtered := make([]CandidateMemory, 0, len(candidates))
	resultMap := make(map[string]CandidateResult)
	for _, result := range results {
		resultMap[result.ID] = result
	}

	for _, candidate := range candidates {
		result, exists := resultMap[candidate.ID]

		// Default to keeping if no result (safety fallback)
		if !exists {
			filtered = append(filtered, candidate)
			continue
		}

		discarded := result.Confidence < v.config.CoVeConfidenceThreshold
		v.stats.RecordEvaluation(result.Confidence, discarded)

		if !discarded {
			filtered = append(filtered, candidate)
		}
		// Discarded candidates are simply not added to filtered slice
	}

	v.persistStats()
	return filtered, nil
}

// callLLMForCandidateVerification makes a single batched LLM call for all candidates
func (v *Verifier) callLLMForCandidateVerification(ctx context.Context, candidates []CandidateMemory) ([]CandidateResult, error) {
	prompt := v.buildVerificationPrompt(candidates)

	// Determine model to use
	model := v.config.CoVeModel
	if model == "" {
		model = v.config.LLMModel
	}
	if model == "" {
		// Final fallback for cloud/generic environments
		model = "gpt-3.5-turbo"
	}

	req := llm.ChatCompletionRequest{
		Model: model,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are a verification assistant that assesses memory candidate quality. Respond only with valid JSON.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	resp, err := v.llmClient.ChatCompletions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse JSON response
	content := resp.Choices[0].Message.Content
	results, err := v.parseVerificationResponse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return results, nil
}

// buildVerificationPrompt constructs the CoVe prompt for candidate verification
func (v *Verifier) buildVerificationPrompt(candidates []CandidateMemory) string {
	var sb strings.Builder

	sb.WriteString("You are verifying candidate memory items extracted from an LLM response.\n\n")
	sb.WriteString("IMPORTANT RULES:\n")
	sb.WriteString("- Do NOT assume any item is true\n")
	sb.WriteString("- Do NOT invent evidence\n")
	sb.WriteString("- Only assess internal consistency and confidence\n")
	sb.WriteString("- Look for speculation, hedging, or uncertainty markers\n\n")
	sb.WriteString("For each item below, assess:\n")
	sb.WriteString("1. Is this a concrete claim, plan, decision, or note?\n")
	sb.WriteString("2. Does it appear speculative or uncertain?\n")
	sb.WriteString("3. Could this be hallucinated or overconfident?\n")
	sb.WriteString("4. Should this be kept (confidence >= 0.6) or discarded?\n\n")

	sb.WriteString("CANDIDATES:\n\n")

	for _, candidate := range candidates {
		sb.WriteString(fmt.Sprintf("ID: %s\n", candidate.ID))
		sb.WriteString(fmt.Sprintf("Type: %s\n", candidate.Type))
		sb.WriteString(fmt.Sprintf("Summary: %s\n", candidate.Summary))
		if candidate.Detail != "" {
			sb.WriteString(fmt.Sprintf("Detail: %s\n", candidate.Detail))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Respond in strict JSON format:\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\n")
	sb.WriteString("    \"id\": \"<candidate_id>\",\n")
	sb.WriteString("    \"confidence\": 0.0–1.0,\n")
	sb.WriteString("    \"reason\": \"<short explanation>\"\n")
	sb.WriteString("  }\n")
	sb.WriteString("]\n")

	return sb.String()
}

// parseVerificationResponse parses the JSON response from the LLM
func (v *Verifier) parseVerificationResponse(content string) ([]CandidateResult, error) {
	// Try to extract JSON from the response (it might be wrapped in markdown)
	jsonStart := strings.Index(content, "[")
	jsonEnd := strings.LastIndex(content, "]")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	var results []CandidateResult
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Validate results
	for i, result := range results {
		if result.ID == "" {
			return nil, fmt.Errorf("result %d missing ID", i)
		}
		if result.Confidence < 0 || result.Confidence > 1 {
			return nil, fmt.Errorf("result %d has invalid confidence: %.2f", i, result.Confidence)
		}
	}

	return results, nil
}

// FilterRecall performs optional CoVe-based relevance filtering on recall results
// This is an advisory filter that can only remove items, never add
func (v *Verifier) FilterRecall(ctx context.Context, memories []RecallMemory, query string) ([]RecallMemory, error) {
	// Safety check: recall filtering must be explicitly enabled
	if !v.config.CoVeEnabled || !v.config.CoVeRecallFilterEnabled {
		return memories, nil
	}

	// Safety check: if no memories, return empty
	if len(memories) == 0 {
		return memories, nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(v.config.CoVeTimeoutSeconds)*time.Second)
	defer cancel()

	// Call LLM for relevance filtering
	results, err := v.callLLMForRecallFilter(ctx, memories, query)
	if err != nil {
		// FAIL-SAFE: On error, return all memories unfiltered
		v.stats.RecordError()
		v.persistStats()
		return memories, nil
	}

	// Filter memories based on include/exclude decisions
	filtered := make([]RecallMemory, 0, len(memories))
	resultMap := make(map[string]bool)
	for _, result := range results {
		resultMap[result.ID] = result.Include
	}

	for _, memory := range memories {
		include, exists := resultMap[memory.ID]

		// Default to including if no result (safety fallback)
		if !exists {
			filtered = append(filtered, memory)
			continue
		}

		if include {
			filtered = append(filtered, memory)
		}
		// Excluded memories are simply not added to filtered slice
	}

	v.persistStats()
	return filtered, nil
}

// callLLMForRecallFilter makes a single LLM call for recall filtering
func (v *Verifier) callLLMForRecallFilter(ctx context.Context, memories []RecallMemory, query string) ([]RecallFilterResult, error) {
	prompt := v.buildRecallFilterPrompt(memories, query)

	// Determine model to use
	model := v.config.CoVeModel
	if model == "" {
		model = v.config.LLMModel
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	req := llm.ChatCompletionRequest{
		Model: model,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are a memory relevance filter. Determine which memories are relevant to the user's query. Respond only with valid JSON.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	resp, err := v.llmClient.ChatCompletions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse JSON response
	content := resp.Choices[0].Message.Content
	results, err := v.parseRecallFilterResponse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return results, nil
}

// buildRecallFilterPrompt constructs the CoVe prompt for recall filtering
func (v *Verifier) buildRecallFilterPrompt(memories []RecallMemory, query string) string {
	var sb strings.Builder

	sb.WriteString("You are selecting which memories are relevant to the current user request.\n\n")
	sb.WriteString("IMPORTANT RULES:\n")
	sb.WriteString("- Do NOT judge truth or accuracy\n")
	sb.WriteString("- Do NOT modify content\n")
	sb.WriteString("- Only assess relevance to the query\n\n")
	sb.WriteString(fmt.Sprintf("USER QUERY: %s\n\n", query))
	sb.WriteString("MEMORIES:\n\n")

	for _, memory := range memories {
		sb.WriteString(fmt.Sprintf("ID: %s\n", memory.ID))
		sb.WriteString(fmt.Sprintf("Type: %s\n", memory.Type))
		sb.WriteString(fmt.Sprintf("Summary: %s\n", memory.Summary))
		if memory.Detail != "" {
			sb.WriteString(fmt.Sprintf("Detail: %s\n", memory.Detail))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("For each memory, answer:\n")
	sb.WriteString("- Is this directly relevant to the user's query?\n")
	sb.WriteString("- Is it outdated or superseded?\n")
	sb.WriteString("- Should it be included or skipped?\n\n")
	sb.WriteString("Respond with:\n")
	sb.WriteString("[\n")
	sb.WriteString("  { \"id\": \"...\", \"include\": true|false }\n")
	sb.WriteString("]\n")

	return sb.String()
}

// parseRecallFilterResponse parses the JSON response from recall filtering
func (v *Verifier) parseRecallFilterResponse(content string) ([]RecallFilterResult, error) {
	// Try to extract JSON from the response
	jsonStart := strings.Index(content, "[")
	jsonEnd := strings.LastIndex(content, "]")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	var results []RecallFilterResult
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Validate results
	for i, result := range results {
		if result.ID == "" {
			return nil, fmt.Errorf("result %d missing ID", i)
		}
	}

	return results, nil
}

// GetStats returns current CoVe statistics
func (v *Verifier) GetStats() Stats {
	return v.stats.GetStats()
}

// ResetStats clears all statistics
func (v *Verifier) ResetStats() {
	v.stats.Reset()
}

func (v *Verifier) persistStats() {
	if v.statsStore == nil || v.projectID == "" {
		return
	}
	_ = v.statsStore.Save(v.projectID, v.stats.GetStats())
}
