package ledger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// State represents the artifact state in the state machine
// Per spec section 5: PROPOSED → AUTHORITATIVE → SUPERSEDED, with TOMBSTONED branch
type State string

const (
	StateProposed      State = "PROPOSED"
	StateAuthoritative State = "AUTHORITATIVE"
	StateSuperseded    State = "SUPERSEDED"
	StateTombstoned    State = "TOMBSTONED"
)

// Episode represents a single interaction session
// Per spec section 3.2: chronological evidence, append-only
type Episode struct {
	ID                    string
	EpisodeID             string
	Timestamp             time.Time
	UserPromptHash        *string
	AssistantResponseHash *string
	Metadata              map[string]interface{}
}

// StateTransition records a change in entity state
// Per spec: all state transitions must be explicit and reviewable
type StateTransition struct {
	ID           int64
	EpisodeID    string
	EntityKey    string
	FromState    *State
	ToState      State
	ArtifactHash string
	Timestamp    time.Time
	Reason       string
}

// AuditResult records shadow audit outcomes
// Per spec section 10: async, non-blocking, metadata only
type AuditResult struct {
	ID            int64
	EpisodeID     string
	ArtifactHash  string
	EntityKey     *string
	Status        string // 'completed', 'partial', 'discussion'
	AuditResponse string // Full JSON response
	Timestamp     time.Time
}

// Ledger manages the append-only chronological log
// Per spec: never injected into prompts, pure evidence
type Ledger struct {
	db *sql.DB
}

// New creates a new Ledger instance
func New(db *sql.DB) *Ledger {
	return &Ledger{db: db}
}

// CreateEpisode creates a new episode entry
// Returns the generated episode ID (UUID)
func (l *Ledger) CreateEpisode(userPromptHash, assistantResponseHash *string, metadata map[string]interface{}) (string, error) {
	episodeID := uuid.New().String()
	now := time.Now().Unix()

	var metadataJSON []byte
	var err error
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return "", fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = l.db.Exec(`
		INSERT INTO ledger_episodes (episode_id, timestamp, user_prompt_hash, assistant_response_hash, metadata)
		VALUES (?, ?, ?, ?, ?)
	`, episodeID, now, userPromptHash, assistantResponseHash, metadataJSON)

	if err != nil {
		return "", fmt.Errorf("failed to create episode: %w", err)
	}

	return episodeID, nil
}

// CountEpisodes returns the number of episodes in the ledger
func (l *Ledger) CountEpisodes() (int, error) {
	var count int
	err := l.db.QueryRow("SELECT COUNT(*) FROM ledger_episodes").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count ledger episodes: %w", err)
	}
	return count, nil
}

// UpdateEpisodeMetadata merges metadata into the existing episode metadata.
func (l *Ledger) UpdateEpisodeMetadata(episodeID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	var metadataJSON []byte
	err := l.db.QueryRow(`
		SELECT metadata FROM ledger_episodes WHERE episode_id = ?
	`, episodeID).Scan(&metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to read episode metadata: %w", err)
	}

	metadata := make(map[string]interface{})
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return fmt.Errorf("failed to unmarshal episode metadata: %w", err)
		}
	}

	for key, value := range updates {
		metadata[key] = value
	}

	mergedJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal episode metadata: %w", err)
	}

	_, err = l.db.Exec(`
		UPDATE ledger_episodes
		SET metadata = ?
		WHERE episode_id = ?
	`, string(mergedJSON), episodeID)
	if err != nil {
		return fmt.Errorf("failed to update episode metadata: %w", err)
	}

	return nil
}

// UpdateEpisodeAssistantResponse stores the assistant response hash for the episode
func (l *Ledger) UpdateEpisodeAssistantResponse(episodeID, assistantResponseHash string) error {
	_, err := l.db.Exec(`
		UPDATE ledger_episodes
		SET assistant_response_hash = ?
		WHERE episode_id = ?
	`, assistantResponseHash, episodeID)

	if err != nil {
		return fmt.Errorf("failed to update episode response hash: %w", err)
	}

	return nil
}

// RecordStateTransition logs a state change for an entity
// Per spec: explicit state transitions only
func (l *Ledger) RecordStateTransition(episodeID, entityKey string, fromState *State, toState State, artifactHash, reason string) error {
	now := time.Now().Unix()

	var fromStateStr *string
	if fromState != nil {
		s := string(*fromState)
		fromStateStr = &s
	}

	_, err := l.db.Exec(`
		INSERT INTO ledger_state_transitions (episode_id, entity_key, from_state, to_state, artifact_hash, timestamp, reason)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, episodeID, entityKey, fromStateStr, toState, artifactHash, now, reason)

	if err != nil {
		return fmt.Errorf("failed to record state transition: %w", err)
	}

	return nil
}

// RecordAudit stores a shadow audit result
// Per spec section 10: non-blocking, metadata only
func (l *Ledger) RecordAudit(episodeID, artifactHash string, entityKey *string, status string, auditResponse string) error {
	now := time.Now().Unix()

	_, err := l.db.Exec(`
		INSERT INTO ledger_audit_results (episode_id, artifact_hash, entity_key, status, audit_response, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, episodeID, artifactHash, entityKey, status, auditResponse, now)

	if err != nil {
		return fmt.Errorf("failed to record audit: %w", err)
	}

	return nil
}

// GetEpisode retrieves an episode by ID
func (l *Ledger) GetEpisode(episodeID string) (*Episode, error) {
	var e Episode
	var timestamp int64
	var userPromptHash, assistantResponseHash sql.NullString
	var metadataJSON []byte

	err := l.db.QueryRow(`
		SELECT id, episode_id, timestamp, user_prompt_hash, assistant_response_hash, metadata
		FROM ledger_episodes
		WHERE episode_id = ?
	`, episodeID).Scan(&e.ID, &e.EpisodeID, &timestamp, &userPromptHash, &assistantResponseHash, &metadataJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get episode: %w", err)
	}

	e.Timestamp = time.Unix(timestamp, 0)
	if userPromptHash.Valid {
		e.UserPromptHash = &userPromptHash.String
	}
	if assistantResponseHash.Valid {
		e.AssistantResponseHash = &assistantResponseHash.String
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &e, nil
}

// GetStateTransitions retrieves all state transitions for an entity
func (l *Ledger) GetStateTransitions(entityKey string) ([]*StateTransition, error) {
	rows, err := l.db.Query(`
		SELECT id, episode_id, entity_key, from_state, to_state, artifact_hash, timestamp, reason
		FROM ledger_state_transitions
		WHERE entity_key = ?
		ORDER BY timestamp ASC
	`, entityKey)
	if err != nil {
		return nil, fmt.Errorf("failed to query state transitions: %w", err)
	}
	defer rows.Close()

	var transitions []*StateTransition
	for rows.Next() {
		var t StateTransition
		var timestamp int64
		var fromState sql.NullString
		var reason sql.NullString

		err := rows.Scan(&t.ID, &t.EpisodeID, &t.EntityKey, &fromState, &t.ToState, &t.ArtifactHash, &timestamp, &reason)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transition: %w", err)
		}

		t.Timestamp = time.Unix(timestamp, 0)
		if fromState.Valid {
			s := State(fromState.String)
			t.FromState = &s
		}
		if reason.Valid {
			t.Reason = reason.String
		}

		transitions = append(transitions, &t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transitions: %w", err)
	}

	return transitions, nil
}

// GetAuditResults retrieves audit results for an episode
func (l *Ledger) GetAuditResults(episodeID string) ([]*AuditResult, error) {
	rows, err := l.db.Query(`
		SELECT id, episode_id, artifact_hash, entity_key, status, audit_response, timestamp
		FROM ledger_audit_results
		WHERE episode_id = ?
		ORDER BY timestamp ASC
	`, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit results: %w", err)
	}
	defer rows.Close()

	var results []*AuditResult
	for rows.Next() {
		var r AuditResult
		var timestamp int64
		var entityKey sql.NullString

		err := rows.Scan(&r.ID, &r.EpisodeID, &r.ArtifactHash, &entityKey, &r.Status, &r.AuditResponse, &timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit result: %w", err)
		}

		r.Timestamp = time.Unix(timestamp, 0)
		if entityKey.Valid {
			r.EntityKey = &entityKey.String
		}

		results = append(results, &r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit results: %w", err)
	}

	return results, nil
}

// GetRecentEpisodes retrieves the N most recent episodes
// Per spec Step 7: diagnostic endpoint, read-only, no code content
func (l *Ledger) GetRecentEpisodes(limit int) ([]*Episode, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}

	rows, err := l.db.Query(`
		SELECT episode_id, timestamp, user_prompt_hash, assistant_response_hash, metadata
		FROM ledger_episodes
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent episodes: %w", err)
	}
	defer rows.Close()

	var episodes []*Episode
	for rows.Next() {
		var e Episode
		var timestamp int64
		var userPromptHash, assistantResponseHash sql.NullString
		var metadataJSON []byte

		err := rows.Scan(&e.EpisodeID, &timestamp, &userPromptHash, &assistantResponseHash, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan episode: %w", err)
		}

		e.Timestamp = time.Unix(timestamp, 0)
		if userPromptHash.Valid {
			e.UserPromptHash = &userPromptHash.String
		}
		if assistantResponseHash.Valid {
			e.AssistantResponseHash = &assistantResponseHash.String
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		episodes = append(episodes, &e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating episodes: %w", err)
	}

	return episodes, nil
}

// GetRecentEpisodesBefore retrieves the N most recent episodes before a timestamp.
func (l *Ledger) GetRecentEpisodesBefore(timestamp int64, limit int) ([]*Episode, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := l.db.Query(`
		SELECT episode_id, timestamp, user_prompt_hash, assistant_response_hash, metadata
		FROM ledger_episodes
		WHERE timestamp < ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, timestamp, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent episodes: %w", err)
	}
	defer rows.Close()

	var episodes []*Episode
	for rows.Next() {
		var e Episode
		var ts int64
		var userPromptHash, assistantResponseHash sql.NullString
		var metadataJSON []byte

		err := rows.Scan(&e.EpisodeID, &ts, &userPromptHash, &assistantResponseHash, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan episode: %w", err)
		}

		e.Timestamp = time.Unix(ts, 0)
		if userPromptHash.Valid {
			e.UserPromptHash = &userPromptHash.String
		}
		if assistantResponseHash.Valid {
			e.AssistantResponseHash = &assistantResponseHash.String
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		episodes = append(episodes, &e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating episodes: %w", err)
	}

	return episodes, nil
}
