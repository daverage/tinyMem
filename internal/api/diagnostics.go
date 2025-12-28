package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// DiagnosticsResponse structures for diagnostic endpoints
// Per requirements: Human-readable JSON, read-only, no side effects

// HealthResponse for GET /health
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// DoctorResponse for GET /doctor
type DoctorResponse struct {
	Database struct {
		Connected   bool   `json:"connected"`
		Path        string `json:"path"`
		VaultCount  int    `json:"vault_count"`
		StateCount  int    `json:"state_count"`
		LedgerCount int    `json:"ledger_count"`
	} `json:"database"`
	LLM struct {
		Provider string `json:"provider"`
		Endpoint string `json:"endpoint"`
		Model    string `json:"model"`
	} `json:"llm"`
	Proxy struct {
		ListenAddress string `json:"listen_address"`
		Uptime        int64  `json:"uptime_seconds"`
	} `json:"proxy"`
	ETV struct {
		StaleCount     int      `json:"stale_count"`
		FileReadErrors []string `json:"file_read_errors,omitempty"`
	} `json:"etv"`
}

// StateResponse for GET /state
type StateResponse struct {
	AuthoritativeCount int               `json:"authoritative_count"`
	ProposedCount      int               `json:"proposed_count"`
	SupersededCount    int               `json:"superseded_count"`
	TombstonedCount    int               `json:"tombstoned_count"`
	Entities           []EntityStateInfo `json:"entities,omitempty"`
}

// EntityStateInfo provides entity state information
// Per spec section 15.7: Each entity includes STALE flag
type EntityStateInfo struct {
	EntityKey    string `json:"entity_key"`
	Filepath     string `json:"filepath"`
	Symbol       string `json:"symbol"`
	State        string `json:"state"`
	Confidence   string `json:"confidence"`
	ArtifactHash string `json:"artifact_hash"`
	LastUpdated  int64  `json:"last_updated"`
	Stale        bool   `json:"stale"` // ETV: true if disk diverges from State Map
}

// handleHealth responds to health checks
// GET /health - Simple liveness check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDoctor provides detailed system health information
// GET /doctor - Comprehensive system check
// Per spec section 15.7: Reports STALE count and file access errors
func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := DoctorResponse{}

	// Database health
	response.Database.Path = s.databasePath
	vaultCount, err := s.runtime.GetVault().Count()
	if err != nil {
		response.Database.Connected = false
	} else {
		response.Database.Connected = true
		response.Database.VaultCount = vaultCount
	}
	if stateCount, err := s.runtime.GetState().Count(); err != nil {
		s.logger.Debug("Failed to count state entries: %v", err)
	} else {
		response.Database.StateCount = stateCount
	}

	if ledgerCount, err := s.runtime.GetLedger().CountEpisodes(); err != nil {
		s.logger.Debug("Failed to count ledger episodes: %v", err)
	} else {
		response.Database.LedgerCount = ledgerCount
	}

	// LLM configuration (from startup)
	response.LLM.Provider = s.llmProvider
	response.LLM.Endpoint = s.llmEndpoint
	response.LLM.Model = s.llmClient.GetModel()

	// Proxy information
	response.Proxy.ListenAddress = s.listenAddr
	if !s.startTime.IsZero() {
		response.Proxy.Uptime = int64(time.Since(s.startTime).Seconds())
	}

	// ETV: External Truth Verification status
	// Per spec section 15.7: Report STALE count and file read errors
	if s.runtime.GetConsistencyChecker() != nil {
		entities, err := s.runtime.GetState().GetAuthoritative()
		if err == nil {
			response.ETV.StaleCount = s.runtime.GetConsistencyChecker().CountStale(entities)
			response.ETV.FileReadErrors = s.runtime.GetConsistencyChecker().GetFileReadErrors(entities)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleState provides state map information
// GET /state - Current state map status
// Per spec section 15.7: Each entity includes STALE flag
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := StateResponse{}

	// Get authoritative entities
	entities, err := s.runtime.GetState().GetAuthoritative()
	if err != nil {
		http.Error(w, "Failed to query state", http.StatusInternalServerError)
		return
	}

	response.AuthoritativeCount = len(entities)

	// Build entity list with STALE status
	// Per spec section 15.7: Each entity includes STALE flag
	for _, e := range entities {
		entityInfo := EntityStateInfo{
			EntityKey:    e.EntityKey,
			Filepath:     e.Filepath,
			Symbol:       e.Symbol,
			State:        string(e.State),
			Confidence:   string(e.Confidence),
			ArtifactHash: e.ArtifactHash,
			LastUpdated:  e.LastUpdated.Unix(),
			Stale:        false, // Default to false
		}

		// ETV: Check if entity is STALE
		if s.runtime.GetConsistencyChecker() != nil {
			isStale, _, _ := s.runtime.GetConsistencyChecker().IsEntityStale(e)
			entityInfo.Stale = isStale
		}

		response.Entities = append(response.Entities, entityInfo)
	}

	// TODO: Count other states (PROPOSED, SUPERSEDED, TOMBSTONED)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RecentEpisodesResponse for GET /recent
type RecentEpisodesResponse struct {
	Episodes []EpisodeInfo `json:"episodes"`
}

// EpisodeInfo provides episode information without code content
type EpisodeInfo struct {
	EpisodeID             string                 `json:"episode_id"`
	Timestamp             int64                  `json:"timestamp"`
	UserPromptHash        *string                `json:"user_prompt_hash,omitempty"`
	AssistantResponseHash *string                `json:"assistant_response_hash,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// LastPromptResponse for GET /debug/last-prompt
type LastPromptResponse struct {
	EpisodeID      string `json:"episode_id"`
	Timestamp      int64  `json:"timestamp"`
	UserPromptHash string `json:"user_prompt_hash"`
	PromptContent  string `json:"prompt_content"`
}

// handleRecent shows recent episodes without code content
// GET /recent - Read-only, no LLM calls, no state mutation
func (s *Server) handleRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get recent episodes (default: last 10)
	episodes, err := s.runtime.GetLedger().GetRecentEpisodes(10)
	if err != nil {
		s.logger.Error("Failed to get recent episodes: %v", err)
		http.Error(w, "Failed to query episodes", http.StatusInternalServerError)
		return
	}

	response := RecentEpisodesResponse{
		Episodes: make([]EpisodeInfo, len(episodes)),
	}

	for i, e := range episodes {
		response.Episodes[i] = EpisodeInfo{
			EpisodeID:             e.EpisodeID,
			Timestamp:             e.Timestamp.Unix(),
			UserPromptHash:        e.UserPromptHash,
			AssistantResponseHash: e.AssistantResponseHash,
			Metadata:              e.Metadata,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDebugLastPrompt shows the last user prompt content
// GET /debug/last-prompt - Only available when debug=true
func (s *Server) handleDebugLastPrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the most recent episode
	episodes, err := s.runtime.GetLedger().GetRecentEpisodes(1)
	if err != nil {
		s.logger.Error("Failed to get recent episodes: %v", err)
		http.Error(w, "Failed to query episodes", http.StatusInternalServerError)
		return
	}

	if len(episodes) == 0 {
		http.Error(w, "No episodes found", http.StatusNotFound)
		return
	}

	episode := episodes[0]
	promptHash := episode.UserPromptHash
	if episode.Metadata != nil {
		if v, ok := episode.Metadata["llm_prompt_hash"]; ok {
			if str, ok := v.(string); ok && str != "" {
				promptHash = &str
			}
		}
	}

	if promptHash == nil {
		http.Error(w, "No prompt available in last episode", http.StatusNotFound)
		return
	}

	// Retrieve prompt content from vault
	artifact, err := s.runtime.GetVault().Get(*promptHash)
	if err != nil {
		s.logger.Error("Failed to get prompt artifact: %v", err)
		http.Error(w, "Failed to retrieve prompt content", http.StatusInternalServerError)
		return
	}

	if artifact == nil {
		http.Error(w, "Prompt artifact not found", http.StatusNotFound)
		return
	}

	response := LastPromptResponse{
		EpisodeID:      episode.EpisodeID,
		Timestamp:      episode.Timestamp.Unix(),
		UserPromptHash: *promptHash,
		PromptContent:  artifact.Content,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDebugReset truncates all persisted state (debug only)
// POST /debug/reset
func (s *Server) handleDebugReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.runtime.ResetAll(); err != nil {
		s.logger.Error("Failed to reset state: %v", err)
		http.Error(w, "Failed to reset state", http.StatusInternalServerError)
		return
	}

	response := map[string]string{"status": "reset"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
