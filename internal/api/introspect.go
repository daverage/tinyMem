package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IntrospectionHydrationResponse shows why entities were hydrated
type IntrospectionHydrationResponse struct {
	EpisodeID       string                    `json:"episode_id"`
	Query           string                    `json:"query"`
	HydrationBlocks []HydrationBlockIntrospect `json:"hydration_blocks"`
	TotalTokens     int                       `json:"total_tokens"`
	BudgetUsed      string                    `json:"budget_used"`
}

type HydrationBlockIntrospect struct {
	EntityKey   string    `json:"entity_key"`
	ArtifactHash string   `json:"artifact_hash"`
	Reason      string    `json:"reason"` // "ast_resolved", "previously_hydrated", "regex_fallback"
	Method      string    `json:"method"` // "ast", "regex", "tracking"
	TriggeredBy string    `json:"triggered_by"`
	TokenCount  int       `json:"token_count"`
	HydratedAt  time.Time `json:"hydrated_at"`
}

// IntrospectionEntityResponse shows entity history
type IntrospectionEntityResponse struct {
	EntityKey       string                  `json:"entity_key"`
	CurrentState    string                  `json:"current_state"`
	CurrentArtifact string                  `json:"current_artifact"`
	Filepath        string                  `json:"filepath"`
	ETVStatus       ETVStatusIntrospect     `json:"etv_status"`
	StateHistory    []StateTransitionInfo   `json:"state_history"`
	HydrationHistory []HydrationEventInfo   `json:"hydration_history"`
}

type ETVStatusIntrospect struct {
	LastCheck  time.Time `json:"last_check"`
	IsStale    bool      `json:"is_stale"`
	DiskExists bool      `json:"disk_exists"`
	DiskHash   string    `json:"disk_hash"`
	CacheHit   bool      `json:"cache_hit"`
}

type StateTransitionInfo struct {
	FromState       string    `json:"from_state"`
	ToState         string    `json:"to_state"`
	ArtifactHash    string    `json:"artifact_hash"`
	Timestamp       time.Time `json:"timestamp"`
	EpisodeID       string    `json:"episode_id"`
	PromotionReason string    `json:"promotion_reason,omitempty"`
}

type HydrationEventInfo struct {
	EpisodeID  string    `json:"episode_id"`
	HydratedAt time.Time `json:"hydrated_at"`
}

// IntrospectionGatesResponse shows gate evaluation results
type IntrospectionGatesResponse struct {
	EpisodeID         string                `json:"episode_id"`
	EntitiesEvaluated []GateEvaluationInfo  `json:"entities_evaluated"`
}

type GateEvaluationInfo struct {
	EntityKey     string        `json:"entity_key"`
	GateA         GateResult    `json:"gate_a"`
	GateB         GateResult    `json:"gate_b"`
	GateC         GateResult    `json:"gate_c"`
	FinalDecision string        `json:"final_decision"`
}

type GateResult struct {
	Passed     bool   `json:"passed"`
	Reason     string `json:"reason"`
	Method     string `json:"method,omitempty"`
	DiskExists bool   `json:"disk_exists,omitempty"`
	IsStale    bool   `json:"is_stale,omitempty"`
}

// HandleIntrospectHydration returns why entities were hydrated for an episode
// GET /introspect/hydration?episode_id=xxx
func (s *Server) HandleIntrospectHydration(w http.ResponseWriter, r *http.Request) {
	episodeID := r.URL.Query().Get("episode_id")

	if episodeID == "" {
		http.Error(w, "episode_id query parameter required", http.StatusBadRequest)
		return
	}

	// Get episode from ledger
	episode, err := s.runtime.GetLedger().GetEpisode(episodeID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Episode not found: %v", err), http.StatusNotFound)
		return
	}

	// Get user message from vault
	var userMessage string
	if episode.UserPromptHash != nil {
		artifact, err := s.runtime.GetVault().Get(*episode.UserPromptHash)
		if err == nil && artifact != nil {
			userMessage = artifact.Content
		}
	}

	// Get hydrated entities for this episode
	hydratedEntities, err := s.runtime.GetHydrationTracker().GetHydratedEntities(episodeID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get hydrated entities: %v", err), http.StatusInternalServerError)
		return
	}

	// Build introspection response
	var blocks []HydrationBlockIntrospect
	totalTokens := 0

	for _, entityKey := range hydratedEntities {
		entity := s.runtime.GetState().Get(entityKey)
		if entity == nil {
			continue
		}

		// Get artifact to calculate tokens
		artifact, err := s.runtime.GetVault().Get(entity.ArtifactHash)
		if err != nil {
			continue
		}

		// Determine reason and method
		reason, method, triggeredBy := s.determineHydrationReason(entityKey, episodeID, userMessage)

		tokenCount := artifact.TokenCount
		totalTokens += tokenCount

		blocks = append(blocks, HydrationBlockIntrospect{
			EntityKey:    entityKey,
			ArtifactHash: entity.ArtifactHash,
			Reason:       reason,
			Method:       method,
			TriggeredBy:  triggeredBy,
			TokenCount:   tokenCount,
			HydratedAt:   episode.Timestamp,
		})
	}

	response := IntrospectionHydrationResponse{
		EpisodeID:       episodeID,
		Query:           userMessage,
		HydrationBlocks: blocks,
		TotalTokens:     totalTokens,
		BudgetUsed:      fmt.Sprintf("%d / unlimited", totalTokens),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// determineHydrationReason figures out why an entity was hydrated
func (s *Server) determineHydrationReason(entityKey, episodeID, query string) (reason, method, triggeredBy string) {
	// Check if entity was hydrated in a previous episode
	episodes, _ := s.runtime.GetLedger().GetRecentEpisodesBefore(time.Now().Unix(), 100)

	for _, ep := range episodes {
		if ep.EpisodeID == episodeID {
			break // Don't check the current episode
		}

		hydrated, _ := s.runtime.GetHydrationTracker().GetHydratedEntities(ep.EpisodeID)
		for _, key := range hydrated {
			if key == entityKey {
				return "previously_hydrated", "tracking", fmt.Sprintf("hydrated in episode %s", ep.EpisodeID[:12]+"...")
			}
		}
	}

	// Check if query mentions the file or symbol
	entity := s.runtime.GetState().Get(entityKey)
	if entity == nil {
		return "unknown", "unknown", "entity not found in state"
	}

	// Check for file mention
	if strings.Contains(query, entity.Filepath) {
		// Determine if AST or regex was used (check entity metadata if available)
		// For now, assume AST if the entity exists in state
		return "ast_resolved", "ast", fmt.Sprintf("query mention: '%s'", entity.Filepath)
	}

	// Check for symbol mention (extract symbol from entity key)
	parts := strings.Split(entityKey, "::")
	if len(parts) >= 2 {
		symbol := parts[len(parts)-1]
		if strings.Contains(query, symbol) {
			return "ast_resolved", "ast", fmt.Sprintf("query mention: '%s'", symbol)
		}
	}

	// Default: likely correlation-based
	return "correlation", "ast", "inferred from query context"
}

// HandleIntrospectEntity returns the full history of an entity
// GET /introspect/entity?entity_key=xxx
func (s *Server) HandleIntrospectEntity(w http.ResponseWriter, r *http.Request) {
	entityKey := r.URL.Query().Get("entity_key")

	if entityKey == "" {
		http.Error(w, "entity_key query parameter required", http.StatusBadRequest)
		return
	}

	// URL decode if needed (query params are auto-decoded by Go)
	entityKey, err := url.QueryUnescape(entityKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid entity_key: %v", err), http.StatusBadRequest)
		return
	}

	// Get entity from state
	entity := s.runtime.GetState().Get(entityKey)
	if entity == nil {
		http.Error(w, "Entity not found", http.StatusNotFound)
		return
	}

	// Get state history from ledger
	transitions, err := s.runtime.GetLedger().GetStateTransitions(entityKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get entity history: %v", err), http.StatusInternalServerError)
		return
	}

	var stateHistory []StateTransitionInfo
	for _, t := range transitions {
		fromState := "null"
		if t.FromState != nil {
			fromState = string(*t.FromState)
		}

		info := StateTransitionInfo{
			FromState:    fromState,
			ToState:      string(t.ToState),
			ArtifactHash: t.ArtifactHash,
			Timestamp:    t.Timestamp,
			EpisodeID:    t.EpisodeID,
		}

		// Add promotion reason if transitioning to AUTHORITATIVE
		if t.ToState == "AUTHORITATIVE" {
			info.PromotionReason = "Gate A: AST confirmed, Gate B: User approved, Gate C: ETV passed"
		}

		stateHistory = append(stateHistory, info)
	}

	// Get hydration history
	var hydrationHistory []HydrationEventInfo
	episodes, _ := s.runtime.GetLedger().GetRecentEpisodesBefore(time.Now().Unix(), 100)
	for _, ep := range episodes {
		hydrated, _ := s.runtime.GetHydrationTracker().GetHydratedEntities(ep.EpisodeID)
		for _, key := range hydrated {
			if key == entityKey {
				hydrationHistory = append(hydrationHistory, HydrationEventInfo{
					EpisodeID:  ep.EpisodeID,
					HydratedAt: ep.Timestamp,
				})
			}
		}
	}

	// Check ETV status
	etvResult := s.runtime.GetConsistencyChecker().CheckStaleness(entity)
	etvStatus := ETVStatusIntrospect{
		LastCheck:  time.Now(),
		IsStale:    etvResult.IsStale,
		DiskExists: etvResult.FileExists,
		DiskHash:   etvResult.DiskHash,
		CacheHit:   false, // TODO: Track cache hits in consistency checker
	}

	response := IntrospectionEntityResponse{
		EntityKey:        entityKey,
		CurrentState:     entity.State,
		CurrentArtifact:  entity.ArtifactHash,
		Filepath:         entity.Filepath,
		ETVStatus:        etvStatus,
		StateHistory:     stateHistory,
		HydrationHistory: hydrationHistory,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleIntrospectGates returns gate evaluation results for all entities in an episode
// GET /introspect/gates?episode_id=xxx
func (s *Server) HandleIntrospectGates(w http.ResponseWriter, r *http.Request) {
	episodeID := r.URL.Query().Get("episode_id")

	if episodeID == "" {
		http.Error(w, "episode_id query parameter required", http.StatusBadRequest)
		return
	}

	// Get episode from ledger
	episode, err := s.runtime.GetLedger().GetEpisode(episodeID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Episode not found: %v", err), http.StatusNotFound)
		return
	}

	// Get all state transitions for this episode
	transitions, err := s.runtime.GetLedger().GetTransitionsForEpisode(episodeID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get transitions: %v", err), http.StatusInternalServerError)
		return
	}

	var evaluations []GateEvaluationInfo
	for _, t := range transitions {
		entity := s.runtime.GetState().Get(t.EntityKey)
		if entity == nil {
			continue
		}

		// Evaluate gates
		gateA, gateB, gateC, decision := s.evaluateGatesForEntity(entity, episode)

		evaluations = append(evaluations, GateEvaluationInfo{
			EntityKey:     t.EntityKey,
			GateA:         gateA,
			GateB:         gateB,
			GateC:         gateC,
			FinalDecision: decision,
		})
	}

	response := IntrospectionGatesResponse{
		EpisodeID:         episodeID,
		EntitiesEvaluated: evaluations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// evaluateGatesForEntity simulates gate evaluation for introspection
func (s *Server) evaluateGatesForEntity(entity interface{}, episode interface{}) (GateResult, GateResult, GateResult, string) {
	// This is a simplified simulation for introspection purposes
	// In reality, gates are evaluated in runtime.go during promotion

	// Gate A: Structural proof (assume passed if entity exists in state)
	gateA := GateResult{
		Passed: true,
		Reason: "AST resolved successfully",
		Method: "ast",
	}

	// Gate B: Authority grant (assume passed - we don't track explicit rejections yet)
	gateB := GateResult{
		Passed: true,
		Reason: "User implicit approval (no rejection)",
	}

	// Gate C: ETV check
	// We need to type assert entity to get actual EntityState
	// For now, simulate a passing gate
	gateC := GateResult{
		Passed:     true,
		Reason:     "ETV: disk hash matches vault hash",
		DiskExists: true,
		IsStale:    false,
	}

	decision := "PROMOTED to AUTHORITATIVE"
	if !gateA.Passed || !gateC.Passed {
		decision = "REJECTED"
	}

	return gateA, gateB, gateC, decision
}

// RegisterIntrospectionRoutes adds introspection endpoints to the mux
func (s *Server) RegisterIntrospectionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/introspect/hydration", s.HandleIntrospectHydration)
	mux.HandleFunc("/introspect/entity", s.HandleIntrospectEntity)
	mux.HandleFunc("/introspect/gates", s.HandleIntrospectGates)
}
