package memory

import (
	"time"
)

// Type represents the type of memory entry
type Type string

const (
	Fact        Type = "fact"
	Claim       Type = "claim"
	Plan        Type = "plan"
	Decision    Type = "decision"
	Constraint  Type = "constraint"
	Observation Type = "observation"
	Note        Type = "note"
)

// RecallTier represents the recall tier of a memory
type RecallTier string

const (
	Always       RecallTier = "always"
	Contextual   RecallTier = "contextual"
	Opportunistic RecallTier = "opportunistic"
)

// TruthState represents the truth state of a memory
type TruthState string

const (
	Tentative TruthState = "tentative"
	Asserted  TruthState = "asserted"
	Verified  TruthState = "verified"
)

// Memory represents a memory entry
type Memory struct {
	ID           int64      `json:"id"`
	ProjectID    string     `json:"project_id"`
	Type         Type       `json:"type"`
	Summary      string     `json:"summary"`
	Detail       string     `json:"detail"`
	Key          *string    `json:"key,omitempty"`
	Source       *string    `json:"source,omitempty"`
	RecallTier   RecallTier `json:"recall_tier"`
	TruthState   TruthState `json:"truth_state"`
	Classification *string  `json:"classification,omitempty"`  // Optional classification for better recall precision
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	SupersededBy *int64     `json:"superseded_by,omitempty"`
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

// EvidenceInput represents evidence input for creating a fact.
type EvidenceInput struct {
	Type    string `json:"type"`
	Content string `json:"content"`
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
