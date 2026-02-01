package cove

import (
	"strconv"

	"github.com/daverage/tinymem/internal/memory"
)

// MemoriesToCandidates converts memory.Memory objects to CandidateMemory for CoVe verification
func MemoriesToCandidates(memories []*memory.Memory) []CandidateMemory {
	candidates := make([]CandidateMemory, 0, len(memories))

	for i, mem := range memories {
		candidates = append(candidates, CandidateMemory{
			ID:      strconv.Itoa(i),
			Type:    string(mem.Type),
			Summary: mem.Summary,
			Detail:  mem.Detail,
			Score:   1.0, // Lexical recall (no semantic scoring)
		})
	}

	return candidates
}
