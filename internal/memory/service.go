package memory

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"github.com/a-marczewski/tinymem/internal/storage"
)

// Service handles memory operations
type Service struct {
	db *storage.DB
}

// NewService creates a new memory service
func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// CreateMemory creates a new memory entry
func (s *Service) CreateMemory(memory *Memory) error {
	// Enforce evidence requirement at storage layer
	if memory.Type == Fact {
		return fmt.Errorf("facts cannot be created directly - use PromoteToFact after evidence verification")
	}

	query := `
		INSERT INTO memories (project_id, type, summary, detail, key, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	row := s.db.GetConnection().QueryRow(query,
		memory.ProjectID,
		string(memory.Type),
		memory.Summary,
		memory.Detail,
		memory.Key,
		memory.Source,
		time.Now(),
		time.Now(),
	)

	if err := row.Scan(&memory.ID); err != nil {
		return err
	}

	memory.CreatedAt = time.Now()
	memory.UpdatedAt = time.Now()

	return nil
}

// UpdateMemory updates an existing memory entry
func (s *Service) UpdateMemory(memory *Memory) error {
	// Enforce evidence requirement at storage layer
	if memory.Type == Fact {
		return fmt.Errorf("facts cannot be updated directly - use PromoteToFact after evidence verification")
	}

	query := `
		UPDATE memories
		SET type = ?, summary = ?, detail = ?, key = ?, source = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := s.db.GetConnection().Exec(query,
		string(memory.Type),
		memory.Summary,
		memory.Detail,
		memory.Key,
		memory.Source,
		time.Now(),
		memory.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memory with ID %d not found", memory.ID)
	}

	memory.UpdatedAt = time.Now()

	return nil
}

// GetMemory retrieves a memory by ID
func (s *Service) GetMemory(id int64) (*Memory, error) {
	query := `
		SELECT id, project_id, type, summary, detail, key, source, created_at, updated_at, superseded_by
		FROM memories
		WHERE id = ? AND (superseded_by IS NULL OR superseded_by = 0)
	`

	row := s.db.GetConnection().QueryRow(query, id)

	var memory Memory
	var key, source sql.NullString
	var supersededByID sql.NullInt64

	err := row.Scan(
		&memory.ID,
		&memory.ProjectID,
		&memory.Type,
		&memory.Summary,
		&memory.Detail,
		&key,
		&source,
		&memory.CreatedAt,
		&memory.UpdatedAt,
		&supersededByID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memory with ID %d not found", id)
		}
		return nil, err
	}

	if key.Valid {
		memory.Key = &key.String
	}
	if source.Valid {
		memory.Source = &source.String
	}
	if supersededByID.Valid {
		memory.SupersededBy = &supersededByID.Int64
	}

	return &memory, nil
}

// SearchMemories performs a full-text search on memories
func (s *Service) SearchMemories(searchTerm string, limit int) ([]*Memory, error) {
	// Sanitize the search term to prevent SQL injection
	searchTerm = strings.ReplaceAll(searchTerm, "\"", "")
	searchTerm = strings.TrimSpace(searchTerm)
	if searchTerm == "" {
		return []*Memory{}, nil
	}

	// First, try FTS5 search if available
	ftsAvailable, err := s.isFTSAvailable()
	if err != nil {
		return nil, err
	}

	if ftsAvailable {
		// Use FTS5 for better search results
		return s.searchWithFTS(searchTerm, limit)
	} else {
		// Fall back to LIKE-based search
		return s.searchWithLike(searchTerm, limit)
	}
}

// isFTSAvailable checks if FTS5 is available in the database
func (s *Service) isFTSAvailable() (bool, error) {
	// Try to query the sqlite_master table to see if the FTS table exists
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name='memories_fts'"

	rows, err := s.db.GetConnection().Query(query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

// searchWithFTS performs a search using FTS5
func (s *Service) searchWithFTS(searchTerm string, limit int) ([]*Memory, error) {
	// Split search terms and join with OR for broader search
	terms := strings.Fields(searchTerm)
	ftsQuery := strings.Join(terms, " OR ")

	query := `
		SELECT m.id, m.project_id, m.type, m.summary, m.detail, m.key, m.source, m.created_at, m.updated_at, m.superseded_by
		FROM memories m
		JOIN memories_fts f ON m.id = f.rowid
		WHERE f.memories_fts MATCH ?
		  AND (m.superseded_by IS NULL OR m.superseded_by = 0)
		ORDER BY m.created_at DESC
		LIMIT ?
	`

	rows, err := s.db.GetConnection().Query(query, ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		var memory Memory
		var key, source sql.NullString
		var supersededByID sql.NullInt64

		err := rows.Scan(
			&memory.ID,
			&memory.ProjectID,
			&memory.Type,
			&memory.Summary,
			&memory.Detail,
			&key,
			&source,
			&memory.CreatedAt,
			&memory.UpdatedAt,
			&supersededByID,
		)

		if err != nil {
			return nil, err
		}

		if key.Valid {
			memory.Key = &key.String
		}
		if source.Valid {
			memory.Source = &source.String
		}
		if supersededByID.Valid {
			memory.SupersededBy = &supersededByID.Int64
		}

		memories = append(memories, &memory)
	}

	return memories, nil
}

// searchWithLike performs a search using LIKE operator as fallback
func (s *Service) searchWithLike(searchTerm string, limit int) ([]*Memory, error) {
	// Create a search pattern with wildcards
	searchPattern := "%" + searchTerm + "%"

	query := `
		SELECT id, project_id, type, summary, detail, key, source, created_at, updated_at, superseded_by
		FROM memories
		WHERE (summary LIKE ? OR detail LIKE ?)
		  AND (superseded_by IS NULL OR superseded_by = 0)
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.GetConnection().Query(query, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		var memory Memory
		var key, source sql.NullString
		var supersededByID sql.NullInt64

		err := rows.Scan(
			&memory.ID,
			&memory.ProjectID,
			&memory.Type,
			&memory.Summary,
			&memory.Detail,
			&key,
			&source,
			&memory.CreatedAt,
			&memory.UpdatedAt,
			&supersededByID,
		)

		if err != nil {
			return nil, err
		}

		if key.Valid {
			memory.Key = &key.String
		}
		if source.Valid {
			memory.Source = &source.String
		}
		if supersededByID.Valid {
			memory.SupersededBy = &supersededByID.Int64
		}

		memories = append(memories, &memory)
	}

	return memories, nil
}

// GetAllMemories retrieves all memories for a project
func (s *Service) GetAllMemories(projectID string) ([]*Memory, error) {
	query := `
		SELECT id, project_id, type, summary, detail, key, source, created_at, updated_at, superseded_by
		FROM memories
		WHERE project_id = ? AND (superseded_by IS NULL OR superseded_by = 0)
		ORDER BY created_at DESC
	`

	rows, err := s.db.GetConnection().Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		var memory Memory
		var key, source sql.NullString
		var supersededByID sql.NullInt64

		err := rows.Scan(
			&memory.ID,
			&memory.ProjectID,
			&memory.Type,
			&memory.Summary,
			&memory.Detail,
			&key,
			&source,
			&memory.CreatedAt,
			&memory.UpdatedAt,
			&supersededByID,
		)

		if err != nil {
			return nil, err
		}

		if key.Valid {
			memory.Key = &key.String
		}
		if source.Valid {
			memory.Source = &source.String
		}
		if supersededByID.Valid {
			memory.SupersededBy = &supersededByID.Int64
		}

		memories = append(memories, &memory)
	}

	return memories, nil
}

// PromoteToFact promotes a memory to fact type after verifying evidence
func (s *Service) PromoteToFact(memoryID int64, isValidated bool) error {
	// First, get the memory to verify it exists and check its current type
	memory, err := s.GetMemory(memoryID)
	if err != nil {
		return fmt.Errorf("failed to get memory: %w", err)
	}

	// Only allow promotion from certain types (e.g., claim, observation)
	// Don't allow promotion from fact to fact
	if memory.Type == Fact {
		return fmt.Errorf("memory is already a fact")
	}

	// Verify that evidence exists and is valid for this memory
	// This validation should be done externally before calling this function
	if !isValidated {
		return fmt.Errorf("memory cannot be promoted to fact: lacks valid evidence")
	}

	// Handle supersession of conflicting memories before promoting
	if err := s.handleConflictingMemories(memory, memoryID); err != nil {
		return fmt.Errorf("failed to handle conflicting memories: %w", err)
	}

	// Update the memory type to fact
	query := `
		UPDATE memories
		SET type = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := s.db.GetConnection().Exec(query, string(Fact), time.Now(), memoryID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memory with ID %d not found", memoryID)
	}

	return nil
}

// handleConflictingMemories marks conflicting memories as superseded when promoting a new one
func (s *Service) handleConflictingMemories(newMemory *Memory, newMemoryID int64) error {
	// Only handle supersession for important memory types that shouldn't conflict
	if newMemory.Type != Fact && newMemory.Type != Decision && newMemory.Type != Constraint {
		return nil
	}

	// Look for memories with the same key in the same project that might conflict
	var conflictingQuery string
	var params []interface{}

	if newMemory.Key != nil {
		// If the new memory has a key, find others with the same key
		conflictingQuery = `
			SELECT id FROM memories
			WHERE project_id = ? AND key = ? AND id != ? AND superseded_by IS NULL
		`
		params = []interface{}{newMemory.ProjectID, *newMemory.Key, newMemoryID}
	} else {
		// If no key, look for similar content that might be conflicting
		conflictingQuery = `
			SELECT id FROM memories
			WHERE project_id = ? AND type = ? AND summary = ? AND id != ? AND superseded_by IS NULL
		`
		params = []interface{}{newMemory.ProjectID, string(newMemory.Type), newMemory.Summary, newMemoryID}
	}

	rows, err := s.db.GetConnection().Query(conflictingQuery, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Mark all conflicting memories as superseded by the new one
	for rows.Next() {
		var conflictingID int64
		if err := rows.Scan(&conflictingID); err != nil {
			return err
		}

		// Mark the conflicting memory as superseded
		if err := s.MarkAsSuperseded(conflictingID, newMemoryID); err != nil {
			return fmt.Errorf("failed to mark conflicting memory as superseded: %w", err)
		}
	}

	return nil
}

// MarkAsSuperseded marks a memory as superseded by another
func (s *Service) MarkAsSuperseded(oldID, newID int64) error {
	query := `
		UPDATE memories
		SET superseded_by = ?
		WHERE id = ?
	`

	result, err := s.db.GetConnection().Exec(query, newID, oldID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memory with ID %d not found", oldID)
	}

	return nil
}