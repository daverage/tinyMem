package enforcement

import (
	"sync"
	"time"
)

// Outcome describes the enforcement disposition for an action.
type Outcome string

const (
	OutcomeAllow     Outcome = "ALLOW"
	OutcomeBlock     Outcome = "BLOCK"
	OutcomeViolation Outcome = "VIOLATION"
)

// Event captures a structured enforcement signal.
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Code      string    `json:"code"`
	Boundary  string    `json:"boundary"`
	Mode      string    `json:"mode"`
	Outcome   Outcome   `json:"outcome"`
	Details   string    `json:"details,omitempty"`
}

// RunMetadata exposes the enforcement contract for an execution run.
type RunMetadata struct {
	ExecutionMode        string  `json:"execution_mode"`
	EnforcementEvents    []Event `json:"enforcement_events"`
	AllowedActionsCount  int     `json:"allowed_actions_count"`
	BlockedActionsCount  int     `json:"blocked_actions_count"`
	ViolationsCount      int     `json:"violations_count"`
	ClaimedSuccessCount  int     `json:"claimed_success_count"`
	EnforcedSuccessCount int     `json:"enforced_success_count"`
}

// Recorder builds the enforcement metadata stream.
type Recorder struct {
	mu       sync.Mutex
	metadata RunMetadata
}

// NewRecorder creates a recorder initialized with the given mode.
func NewRecorder(mode string) *Recorder {
	if mode == "" {
		mode = "PASSIVE"
	}
	return &Recorder{metadata: RunMetadata{ExecutionMode: mode}}
}

// SetMode updates the recorded execution mode.
func (r *Recorder) SetMode(mode string) {
	if mode == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metadata.ExecutionMode = mode
}

// RecordEvent appends a structured enforcement signal and updates counters.
func (r *Recorder) RecordEvent(evt Event) {
	if evt.Mode == "" {
		evt.Mode = r.metadata.ExecutionMode
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.metadata.EnforcementEvents = append(r.metadata.EnforcementEvents, evt)

	switch evt.Outcome {
	case OutcomeAllow:
		r.metadata.AllowedActionsCount++
		r.metadata.EnforcedSuccessCount++
	case OutcomeBlock:
		r.metadata.BlockedActionsCount++
	case OutcomeViolation:
		r.metadata.ViolationsCount++
	}
}

// RecordClaimedSuccess increments the claimed success counter.
func (r *Recorder) RecordClaimedSuccess() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metadata.ClaimedSuccessCount++
}

// RecordEnforcedSuccess increments the enforced success counter independently.
func (r *Recorder) RecordEnforcedSuccess() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metadata.EnforcedSuccessCount++
}

// Metadata returns a snapshot of the enforcement metadata.
func (r *Recorder) Metadata() RunMetadata {
	r.mu.Lock()
	defer r.mu.Unlock()

	eventsCopy := make([]Event, len(r.metadata.EnforcementEvents))
	copy(eventsCopy, r.metadata.EnforcementEvents)

	meta := r.metadata
	meta.EnforcementEvents = eventsCopy

	return meta
}
