package inject

import (
	"context"
	"fmt"
	"github.com/daverage/tinymem/internal/cove"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/recall"
	"strconv"
	"strings"
)

// MemoryInjector handles injecting memories into prompts
type MemoryInjector struct {
	recallEngine recall.Recaller
	coveVerifier *cove.Verifier
}

// NewMemoryInjector creates a new memory injector
func NewMemoryInjector(recallEngine recall.Recaller) *MemoryInjector {
	return &MemoryInjector{
		recallEngine: recallEngine,
		coveVerifier: nil, // Can be set later via SetCoVeVerifier
	}
}

// SetCoVeVerifier sets the CoVe verifier for recall filtering
func (mi *MemoryInjector) SetCoVeVerifier(verifier *cove.Verifier) {
	mi.coveVerifier = verifier
}

// FilterRecallResults applies optional CoVe recall filtering (fail-safe).
func (mi *MemoryInjector) FilterRecallResults(ctx context.Context, results []recall.RecallResult, query string) []recall.RecallResult {
	if mi.coveVerifier == nil || len(results) == 0 {
		return results
	}

	recallMemories := recallResultsToCoVeMemories(results)
	filteredMemories, err := mi.coveVerifier.FilterRecall(ctx, recallMemories, query)
	if err != nil {
		return results
	}
	return filterRecallResultsByCoVe(results, filteredMemories)
}

// InjectMemoriesIntoPrompt injects relevant memories into a prompt
func (mi *MemoryInjector) InjectMemoriesIntoPrompt(prompt string, projectID string, maxItems int, maxTokens int) (string, error) {
	// Perform recall to find relevant memories
	results, err := mi.recallEngine.Recall(recall.RecallOptions{
		ProjectID: projectID,
		Query:     prompt,
		MaxItems:  maxItems,
		MaxTokens: maxTokens,
	})
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		// No relevant memories found, return original prompt
		return prompt, nil
	}

	// COVE INTEGRATION POINT 2: Optional recall filtering (advisory only)
	// This can only remove memories, never add new ones.
	results = mi.FilterRecallResults(context.Background(), results, prompt)

	if len(results) == 0 {
		// All memories filtered out, return original prompt
		return prompt, nil
	}

	// Build memory section
	var memorySection strings.Builder
	memorySection.WriteString("\n\n=== RELEVANT MEMORY ===\n")

	for i, result := range results {
		mem := result.Memory

		memorySection.WriteString(fmt.Sprintf("[%d] %s: %s\n", i+1, labelForType(mem.Type), mem.Summary))

		if mem.Detail != "" {
			memorySection.WriteString(fmt.Sprintf("DETAIL: %s\n", mem.Detail))
		}

		if mem.Key != nil {
			memorySection.WriteString(fmt.Sprintf("KEY: %s\n", *mem.Key))
		}

		if mem.Source != nil {
			memorySection.WriteString(fmt.Sprintf("SOURCE: %s\n", *mem.Source))
		}

		memorySection.WriteString("---\n")
	}

	memorySection.WriteString("=== END MEMORY ===\n")

	// Prepend memories to the original prompt
	injectedPrompt := memorySection.String() + prompt

	return injectedPrompt, nil
}

// FormatMemoriesForSystemMessage formats memories in a structured way for system messages
func (mi *MemoryInjector) FormatMemoriesForSystemMessage(memories []*memory.Memory) string {
	if len(memories) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n=== PROJECT MEMORY ===\n")

	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("[%d] %s: %s\n", i+1, labelForType(mem.Type), mem.Summary))

		if mem.Detail != "" {
			sb.WriteString(fmt.Sprintf("   DETAIL: %s\n", mem.Detail))
		}

		if mem.Key != nil {
			sb.WriteString(fmt.Sprintf("   KEY: %s\n", *mem.Key))
		}

		if mem.Source != nil {
			sb.WriteString(fmt.Sprintf("   SOURCE: %s\n", *mem.Source))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("=== END PROJECT MEMORY ===\n")

	return sb.String()
}

func labelForType(memType memory.Type) string {
	switch memType {
	case memory.Fact:
		return "FACT"
	case memory.Claim:
		return "CLAIM (unverified)"
	case memory.Decision:
		return "DECISION"
	case memory.Constraint:
		return "CONSTRAINT"
	case memory.Plan:
		return "PLAN"
	case memory.Observation:
		return "OBSERVATION"
	case memory.Note:
		return "NOTE"
	default:
		return "UNKNOWN"
	}
}

// recallResultsToCoVeMemories converts recall results to CoVe format
func recallResultsToCoVeMemories(results []recall.RecallResult) []cove.RecallMemory {
	memories := make([]cove.RecallMemory, 0, len(results))

	for i, result := range results {
		mem := result.Memory
		memories = append(memories, cove.RecallMemory{
			ID:      strconv.Itoa(i), // Use index as temporary ID
			Type:    string(mem.Type),
			Summary: mem.Summary,
			Detail:  mem.Detail,
		})
	}

	return memories
}

// filterRecallResultsByCoVe filters recall results based on CoVe decisions
func filterRecallResultsByCoVe(results []recall.RecallResult, filtered []cove.RecallMemory) []recall.RecallResult {
	// Build a set of accepted IDs
	acceptedIDs := make(map[string]bool)
	for _, mem := range filtered {
		acceptedIDs[mem.ID] = true
	}

	// Filter results based on accepted IDs
	filteredResults := make([]recall.RecallResult, 0, len(filtered))
	for i, result := range results {
		if acceptedIDs[strconv.Itoa(i)] {
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults
}
