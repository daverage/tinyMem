package cove

import "time"

// CandidateResult represents the CoVe verification result for a single candidate memory
type CandidateResult struct {
	ID         string  `json:"id"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// RecallFilterResult represents the CoVe relevance check result for recall
type RecallFilterResult struct {
	ID      string `json:"id"`
	Include bool   `json:"include"`
}

// Stats tracks CoVe operation statistics
type Stats struct {
	CandidatesEvaluated int
	CandidatesDiscarded int
	TotalConfidence     float64
	AvgConfidence       float64
	CoVeErrors          int
	LastUpdated         time.Time
}

// VerificationRequest is the internal request structure for batch verification
type VerificationRequest struct {
	Candidates []CandidateMemory
	Timeout    time.Duration
	Model      string
}

// CandidateMemory represents a candidate memory to be verified
type CandidateMemory struct {
	ID      string
	Type    string
	Summary string
	Detail  string
}

// RecallFilterRequest is the internal request structure for recall filtering
type RecallFilterRequest struct {
	Memories []RecallMemory
	Query    string
	Timeout  time.Duration
	Model    string
}

// RecallMemory represents a memory to be filtered for recall
type RecallMemory struct {
	ID      string
	Type    string
	Summary string
	Detail  string
}
