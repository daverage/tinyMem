package memory

import (
	"time"
)

// Type represents the type of memory entry
type Type string

const (
	Fact       Type = "fact"
	Claim      Type = "claim"
	Plan       Type = "plan"
	Decision   Type = "decision"
	Constraint Type = "constraint"
	Observation Type = "observation"
	Note       Type = "note"
)

// Memory represents a memory entry
type Memory struct {
	ID            int64     `json:"id"`
	ProjectID     string    `json:"project_id"`
	Type          Type      `json:"type"`
	Summary       string    `json:"summary"`
	Detail        string    `json:"detail"`
	Key           *string   `json:"key,omitempty"`
	Source        *string   `json:"source,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	SupersededBy  *int64    `json:"superseded_by,omitempty"`
}

// Evidence represents evidence for a memory
type Evidence struct {
	ID        int64     `json:"id"`
	MemoryID  int64     `json:"memory_id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
}

// Validation rules for memory types
func (t Type) RequiresEvidence() bool {
	return t == Fact
}

func (t Type) IsValid() bool {
	switch t {
	case Fact, Claim, Plan, Decision, Constraint, Observation, Note:
		return true
	default:
		return false
	}
}