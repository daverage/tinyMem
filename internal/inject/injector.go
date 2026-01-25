package inject

import (
	"fmt"
	"strings"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/recall"
)

// MemoryInjector handles injecting memories into prompts
type MemoryInjector struct {
	recallEngine *recall.Engine
}

// NewMemoryInjector creates a new memory injector
func NewMemoryInjector(recallEngine *recall.Engine) *MemoryInjector {
	return &MemoryInjector{
		recallEngine: recallEngine,
	}
}

// InjectMemoriesIntoPrompt injects relevant memories into a prompt
func (mi *MemoryInjector) InjectMemoriesIntoPrompt(prompt string, maxItems int, maxTokens int) (string, error) {
	// Perform recall to find relevant memories
	results, err := mi.recallEngine.Recall(recall.RecallOptions{
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

		memorySection.WriteString(fmt.Sprintf("[%d] Type: %s\n", i+1, string(mem.Type)))
		memorySection.WriteString(fmt.Sprintf("Summary: %s\n", mem.Summary))

		if mem.Detail != "" {
			memorySection.WriteString(fmt.Sprintf("Details: %s\n", mem.Detail))
		}

		if mem.Key != nil {
			memorySection.WriteString(fmt.Sprintf("Key: %s\n", *mem.Key))
		}

		if mem.Source != nil {
			memorySection.WriteString(fmt.Sprintf("Source: %s\n", *mem.Source))
		}

		// Add evidence info for facts
		if mem.Type == memory.Fact {
			memorySection.WriteString("(This fact has been verified with evidence)\n")
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
		sb.WriteString(fmt.Sprintf("[%d] %s: %s\n", i+1, string(mem.Type), mem.Summary))
		
		if mem.Detail != "" {
			sb.WriteString(fmt.Sprintf("   Details: %s\n", mem.Detail))
		}
		
		if mem.Key != nil {
			sb.WriteString(fmt.Sprintf("   Key: %s\n", *mem.Key))
		}
		
		if mem.Type == memory.Fact {
			sb.WriteString("   Status: VERIFIED\n")
		}
		
		sb.WriteString("\n")
	}
	
	sb.WriteString("=== END PROJECT MEMORY ===\n")
	
	return sb.String()
}