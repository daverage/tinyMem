package api

import (
	"encoding/json"
	"net/http"

	"github.com/andrzejmarczewski/tinyMem/internal/vault"
)

// UserCodeRequest represents a request to submit user-pasted code
type UserCodeRequest struct {
	Content  string  `json:"content"`
	Filepath *string `json:"filepath,omitempty"` // Optional filepath for entity resolution
}

// UserCodeResponse represents the response after submitting user code
type UserCodeResponse struct {
	ArtifactHash string `json:"artifact_hash"`
	EntityKey    string `json:"entity_key,omitempty"`
	Confidence   string `json:"confidence"`
	State        string `json:"state"`
	Promoted     bool   `json:"promoted"`
	Reason       string `json:"reason"`
}

// handleUserCode handles user-pasted code submission
// POST /v1/user/code
// Per spec section 9.1: User as Write-Head
// User input always wins and supersedes all prior LLM artifacts
func (s *Server) handleUserCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Info("User code submission received")

	// Parse request
	var req UserCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Failed to decode user code request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Create episode for this user action
	episodeID, err := s.runtime.GetLedger().CreateEpisode(nil, nil, map[string]interface{}{
		"type": "user_code_paste",
	})
	if err != nil {
		s.logger.Error("Failed to create episode: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Process artifact with isUserPaste=true
	// This bypasses promotion gates and goes directly to AUTHORITATIVE
	result, err := s.runtime.ProcessArtifact(req.Content, vault.ContentTypeUserInput, episodeID, true)
	if err != nil {
		s.logger.Error("Failed to process user code: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Build response
	response := UserCodeResponse{
		ArtifactHash: result.ArtifactHash,
		Confidence:   result.Confidence,
		State:        string(result.State),
		Promoted:     result.Promoted,
		Reason:       result.Reason,
	}

	if result.EntityKey != nil {
		response.EntityKey = *result.EntityKey
	}

	s.logger.Info("User code processed: promoted=%v state=%s", result.Promoted, result.State)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
