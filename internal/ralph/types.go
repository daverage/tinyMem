package ralph

import (
	"time"
)

// Status represents the final state of a Ralph loop
type Status string

const (
	StatusSuccess Status = "success"
	StatusAborted Status = "aborted"
	StatusFailed  Status = "failed"
)

// Options represents the input configuration for memory_ralph
type Options struct {
	Task          string        `json:"task"`
	Command       string        `json:"command"`
	Evidence      []string      `json:"evidence"`
	MaxIterations int           `json:"max_iterations"`
	Recall        RecallOptions `json:"recall"`
	Safety        SafetyOptions `json:"safety"`
	HumanGate     HumanGate     `json:"human_gate"`
}

// RecallOptions configures how memories are retrieved during the loop
type RecallOptions struct {
	QueryTerms []string `json:"query_terms"`
	Limit      int      `json:"limit"`
}

// SafetyOptions defines restrictions for autonomous actions
type SafetyOptions struct {
	ForbidPaths       []string `json:"forbid_paths"`
	ForbidCommands    []string `json:"forbid_commands"`
	RequireDiffReview bool     `json:"require_diff_review"`
}

// HumanGate configures when to pause for user intervention
type HumanGate struct {
	OnAmbiguity     bool `json:"on_ambiguity"`
	AfterIterations int  `json:"after_iterations"`
}

// Result is the output of the memory_ralph tool
type Result struct {
	Status     Status                 `json:"status"`
	Iterations int                    `json:"iterations"`
	Evidence   map[string]interface{} `json:"evidence"`
	FinalDiff  string                 `json:"final_diff"`
	MemoryUsed []string               `json:"memory_used"`
	Log        []LogEntry             `json:"log"`
}

// LogEntry records the outcome of a single iteration
type LogEntry struct {
	Iteration int       `json:"iteration"`
	Result    string    `json:"result"`
	Duration  float64   `json:"duration_ms"`
	Error     string    `json:"error,omitempty"`
}

// IterationState holds the runtime data for the current loop iteration
type IterationState struct {
	Iteration int
	ExitCode  int
	Stdout    string
	Stderr    string
	Duration  time.Duration
	Diff      string
}
