package evidence

import (
	"time"
	"tinymem/internal/memory"
	"tinymem/internal/storage"
)

// Service handles evidence operations
type Service struct {
	db *storage.DB
}

// NewService creates a new evidence service
func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// AddEvidence adds evidence for a memory
func (s *Service) AddEvidence(memoryID int64, evidenceType, content string) (*memory.Evidence, error) {
	query := `
		INSERT INTO evidence (memory_id, type, content, verified, created_at)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id
	`

	now := time.Now()
	row := s.db.GetConnection().QueryRow(query,
		memoryID,
		evidenceType,
		content,
		false, // Initially unverified
		now,
	)

	var evidenceID int64
	if err := row.Scan(&evidenceID); err != nil {
		return nil, err
	}

	evidence := &memory.Evidence{
		ID:        evidenceID,
		MemoryID:  memoryID,
		Type:      evidenceType,
		Content:   content,
		Verified:  false,
		CreatedAt: now,
	}

	return evidence, nil
}

// VerifyEvidenceForMemory verifies all evidence for a specific memory
func (s *Service) VerifyEvidenceForMemory(memoryID int64) (bool, error) {
	query := `
		SELECT id, type, content
		FROM evidence
		WHERE memory_id = ?
	`

	rows, err := s.db.GetConnection().Query(query, memoryID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	allVerified := true

	for rows.Next() {
		var id int64
		var evidenceType, content string

		if err := rows.Scan(&id, &evidenceType, &content); err != nil {
			return false, err
		}

		// Verify the evidence
		verified, err := VerifyEvidence(evidenceType, content)
		if err != nil {
			// Log the error but continue checking other evidence
			continue
		}

		if !verified {
			allVerified = false
		}

		// Update the verification status in the database
		updateQuery := `
			UPDATE evidence
			SET verified = ?
			WHERE id = ?
		`
		_, err = s.db.GetConnection().Exec(updateQuery, verified, id)
		if err != nil {
			return false, err
		}
	}

	return allVerified, nil
}

// GetEvidenceForMemory retrieves all evidence for a memory
func (s *Service) GetEvidenceForMemory(memoryID int64) ([]*memory.Evidence, error) {
	query := `
		SELECT id, memory_id, type, content, verified, created_at
		FROM evidence
		WHERE memory_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.GetConnection().Query(query, memoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evidences []*memory.Evidence
	for rows.Next() {
		var evidence memory.Evidence

		err := rows.Scan(
			&evidence.ID,
			&evidence.MemoryID,
			&evidence.Type,
			&evidence.Content,
			&evidence.Verified,
			&evidence.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		evidences = append(evidences, &evidence)
	}

	return evidences, nil
}

// IsMemoryValidated checks if a memory is properly validated based on its type and evidence
func (s *Service) IsMemoryValidated(mem *memory.Memory) (bool, error) {
	// Facts require evidence
	if mem.Type == memory.Fact {
		// Check if there's verified evidence for this fact
		evidences, err := s.GetEvidenceForMemory(mem.ID)
		if err != nil {
			return false, err
		}

		hasVerifiedEvidence := false
		for _, evidence := range evidences {
			if evidence.Verified {
				hasVerifiedEvidence = true
				break
			}
		}

		return hasVerifiedEvidence, nil
	}

	// Other types don't require evidence verification
	return true, nil
}