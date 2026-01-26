package cove

import (
	"sync"
	"time"
)

// StatsTracker tracks CoVe operation statistics in memory
type StatsTracker struct {
	mu                  sync.RWMutex
	candidatesEvaluated int
	candidatesDiscarded int
	totalConfidence     float64
	coveErrors          int
	lastUpdated         time.Time
}

// NewStatsTracker creates a new stats tracker
func NewStatsTracker() *StatsTracker {
	return &StatsTracker{
		lastUpdated: time.Now(),
	}
}

// RecordEvaluation records a single candidate evaluation
func (s *StatsTracker) RecordEvaluation(confidence float64, discarded bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.candidatesEvaluated++
	s.totalConfidence += confidence
	if discarded {
		s.candidatesDiscarded++
	}
	s.lastUpdated = time.Now()
}

// RecordError records a CoVe error
func (s *StatsTracker) RecordError() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.coveErrors++
	s.lastUpdated = time.Now()
}

// GetStats returns the current statistics
func (s *StatsTracker) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	avgConfidence := 0.0
	if s.candidatesEvaluated > 0 {
		avgConfidence = s.totalConfidence / float64(s.candidatesEvaluated)
	}

	return Stats{
		CandidatesEvaluated: s.candidatesEvaluated,
		CandidatesDiscarded: s.candidatesDiscarded,
		TotalConfidence:     s.totalConfidence,
		AvgConfidence:       avgConfidence,
		CoVeErrors:          s.coveErrors,
		LastUpdated:         s.lastUpdated,
	}
}

// Reset clears all statistics
func (s *StatsTracker) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.candidatesEvaluated = 0
	s.candidatesDiscarded = 0
	s.totalConfidence = 0
	s.coveErrors = 0
	s.lastUpdated = time.Now()
}
