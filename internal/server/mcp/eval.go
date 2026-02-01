package mcp

import (
	"encoding/json"

	"github.com/daverage/tinymem/internal/llm"
)

// handleMemoryEvalStats handles the memory_eval_stats tool call
func (s *Server) handleMemoryEvalStats(req *MCPRequest) {
	stats := make(map[string]interface{})

	// Collect LLM metrics from all registered providers
	llmStats := llm.GetAllMetrics()
	stats["llm"] = llmStats

	// If we have a provider, get its specific metrics
	if s.llmProvider != nil {
		providerMetrics := s.llmProvider.GetMetrics()
		if providerMetrics != nil {
			stats["active_provider"] = providerMetrics.GetStats()
		}
	}

	// Add CoVe stats if available
	if s.coveVerifier != nil {
		stats["cove"] = map[string]interface{}{
			"enabled": true,
		}
	}

	resultJSON, _ := json.MarshalIndent(stats, "", "  ")

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		},
		"id": req.ID,
	}

	s.sendResponse(response)
}
