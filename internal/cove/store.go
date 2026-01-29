package cove

import (
	"database/sql"
	"time"
)

// StatsStore persists CoVe stats.
type StatsStore interface {
	Save(projectID string, stats Stats) error
	Load(projectID string) (*Stats, error)
}

// SQLiteStatsStore stores CoVe stats in SQLite.
type SQLiteStatsStore struct {
	db *sql.DB
}

// NewSQLiteStatsStore creates a new SQLite-backed stats store.
func NewSQLiteStatsStore(db *sql.DB) *SQLiteStatsStore {
	return &SQLiteStatsStore{db: db}
}

// Save persists the given stats for a project.
func (s *SQLiteStatsStore) Save(projectID string, stats Stats) error {
	if s == nil || s.db == nil {
		return nil
	}
	lastUpdated := stats.LastUpdated.Format(time.RFC3339)
	_, err := s.db.Exec(`
		INSERT INTO cove_stats (
			project_id, candidates_evaluated, candidates_discarded, total_confidence, cove_errors, last_updated
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE SET
			candidates_evaluated = excluded.candidates_evaluated,
			candidates_discarded = excluded.candidates_discarded,
			total_confidence = excluded.total_confidence,
			cove_errors = excluded.cove_errors,
			last_updated = excluded.last_updated
	`, projectID, stats.CandidatesEvaluated, stats.CandidatesDiscarded, stats.TotalConfidence, stats.CoVeErrors, lastUpdated)
	return err
}

// Load retrieves stats for a project, or nil if none exist.
func (s *SQLiteStatsStore) Load(projectID string) (*Stats, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	var stats Stats
	var lastUpdatedStr sql.NullString
	err := s.db.QueryRow(`
		SELECT candidates_evaluated, candidates_discarded, total_confidence, cove_errors, last_updated
		FROM cove_stats
		WHERE project_id = ?
	`, projectID).Scan(
		&stats.CandidatesEvaluated,
		&stats.CandidatesDiscarded,
		&stats.TotalConfidence,
		&stats.CoVeErrors,
		&lastUpdatedStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if stats.CandidatesEvaluated > 0 {
		stats.AvgConfidence = stats.TotalConfidence / float64(stats.CandidatesEvaluated)
	}
	if lastUpdatedStr.Valid && lastUpdatedStr.String != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, lastUpdatedStr.String); parseErr == nil {
			stats.LastUpdated = parsed
		}
	}

	return &stats, nil
}
