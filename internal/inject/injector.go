package inject

import (
	"fmt"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/recall"
	"strings"
)

// MemoryInjector handles injecting memories into prompts
type MemoryInjector struct {
	recallEngine recall.Recaller
}

// NewMemoryInjector creates a new memory injector
func NewMemoryInjector(recallEngine recall.Recaller) *MemoryInjector {
	return &MemoryInjector{
		recallEngine: recallEngine,
	}
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
