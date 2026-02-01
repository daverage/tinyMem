package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Mode represents the execution mode of tinyMem.
type Mode string

const (
	ModePassive Mode = "PASSIVE"
	ModeGuarded Mode = "GUARDED"
	ModeStrict  Mode = "STRICT"
)

var modeHierarchy = map[Mode]int{
	ModePassive: 0,
	ModeGuarded: 1,
	ModeStrict:  2,
}

var modeLookup = map[string]Mode{
	"passive": ModePassive,
	"guarded": ModeGuarded,
	"strict":  ModeStrict,
}

// ErrStrictRequired is returned when the current execution requires STRICT mode.
var ErrStrictRequired = errors.New("STRICT mode required")

// Action represents a guarded action that is regulated by the execution mode controller.
type Action string

const (
	ActionMemoryQuery    Action = "memory_query"
	ActionMemoryWrite    Action = "memory_write"
	ActionTaskMutation   Action = "task_mutation"
	ActionFactPromotion  Action = "fact_promotion"
	ActionTinyTasksWrite Action = "tinyTasks_edit"
)

// Controller tracks the current execution mode, escalation state, and tinyTasks mutations.
type Controller struct {
	mu                sync.RWMutex
	currentMode       Mode
	modeExplicitlySet bool
	strictRequired    bool
	strictReason      string
	failureCounts     map[string]int
	fileEditCounts    map[string]int
	tinyTasksPath     string
	tinyTasksHash     string
	tinyTasksModTime  time.Time
	failureThreshold  int
	fileEditThreshold int
}

// NewController creates a controller with the provided initial mode and tinyTasks path.
func NewController(initial Mode, tinyTasksPath string) *Controller {

	ctrl := &Controller{
		currentMode:       initial,
		modeExplicitlySet: initial != "",
		failureCounts:     map[string]int{},
		fileEditCounts:    map[string]int{},
		tinyTasksPath:     tinyTasksPath,
		failureThreshold:  2,
		fileEditThreshold: 2,
	}
	if initial != "" {
		_ = ctrl.setModeNoLock(initial)
	}
	ctrl.loadTinyTasksSnapshot()
	return ctrl
}

// Mode returns the current execution mode.
func (c *Controller) Mode() Mode {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentMode
}

// ModeSet reports whether a mode has been explicitly set via SetMode.
func (c *Controller) ModeSet() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.modeExplicitlySet
}

// SetMode updates the controller's mode and updates the environment variable.
func (c *Controller) SetMode(mode Mode) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.setModeNoLock(mode); err != nil {
		return err
	}
	c.modeExplicitlySet = true
	return nil
}

func (c *Controller) setModeNoLock(mode Mode) error {
	if mode == "" {
		return fmt.Errorf("mode cannot be empty")
	}
	c.currentMode = mode

	if err := os.Setenv("TINYMEMP_MODE", string(mode)); err != nil {
		return fmt.Errorf("failed to set TINYMEMP_MODE: %w", err)
	}

	if mode == ModeStrict {
		c.strictRequired = false
		c.strictReason = ""
		c.failureCounts = map[string]int{}
		c.fileEditCounts = map[string]int{}
	}

	return nil
}

// ParseMode converts a string to a Mode value.
func ParseMode(value string) (Mode, error) {
	if value == "" {
		return ModePassive, nil
	}
	normalized := strings.ToLower(strings.TrimSpace(value))
	if mode, ok := modeLookup[normalized]; ok {
		return mode, nil
	}
	return "", fmt.Errorf("unsupported mode %q", value)
}

// Enforce checks whether an action can proceed given the requested minimum mode.
func (c *Controller) Enforce(action Action, minimum Mode) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.modeExplicitlySet {
		return fmt.Errorf("execution mode not explicitly set")
	}

	if c.strictRequired && c.currentMode != ModeStrict {
		return ErrStrictRequired
	}

	if minimum == "" {
		minimum = ModePassive
	}

	if modeHierarchy[c.currentMode] < modeHierarchy[minimum] {
		return fmt.Errorf("action %s requires %s mode (current: %s)", action, minimum, c.currentMode)
	}

	return nil
}

// RequiresStrict reports whether the controller currently forces STRICT mode.
func (c *Controller) RequiresStrict() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.strictRequired
}

// StrictReason returns the reason STRICT mode is required.
func (c *Controller) StrictReason() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.strictReason
}

// TriggerStrict marks that STRICT mode is required for future actions.
func (c *Controller) TriggerStrict(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strictRequired = true
	c.strictReason = reason
}

// RecordFailure records a failing tool invocation and checks for escalation.
func (c *Controller) RecordFailure(tool string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failureCounts["__total__"]++
	if tool != "" {
		c.failureCounts[tool]++
	}

	if c.failureCounts["__total__"] >= c.failureThreshold {
		c.strictRequired = true
		c.strictReason = "multiple tool failures"
		return
	}

	if tool != "" && c.failureCounts[tool] >= c.failureThreshold {
		c.strictRequired = true
		c.strictReason = fmt.Sprintf("multiple failures for %s", tool)
	}
}

// RecordFileEdit records an edit attempt for the given path.
func (c *Controller) RecordFileEdit(path string) {
	if path == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	normalized := filepath.Clean(path)
	c.fileEditCounts[normalized]++
	if c.fileEditCounts[normalized] >= c.fileEditThreshold {
		c.strictRequired = true
		c.strictReason = fmt.Sprintf("multiple edits detected to %s", normalized)
	}
}

// MarkFactPromotion marks that fact promotion requires STRICT mode.
func (c *Controller) MarkFactPromotion() {
	c.TriggerStrict("claim promotion attempted")
}

// TinyTasksPath returns the configured tinyTasks file path.
func (c *Controller) TinyTasksPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tinyTasksPath
}

// RefreshTinyTasksState checks the tinyTasks file for changes.
func (c *Controller) RefreshTinyTasksState() error {
	if c.tinyTasksPath == "" {
		return nil
	}

	info, err := os.Stat(c.tinyTasksPath)
	if os.IsNotExist(err) {
		// Reset state if the file was removed.
		c.mu.Lock()
		c.tinyTasksHash = ""
		c.tinyTasksModTime = time.Time{}
		c.mu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !info.ModTime().After(c.tinyTasksModTime) && c.tinyTasksHash != "" {
		return nil
	}

	hash, modTime, err := hashFileSnapshot(c.tinyTasksPath)
	if err != nil {
		return err
	}

	if hash != c.tinyTasksHash {
		c.tinyTasksHash = hash
		c.tinyTasksModTime = modTime
		c.strictRequired = true
		c.strictReason = "tinyTasks.md modified"
	}

	return nil
}

func (c *Controller) loadTinyTasksSnapshot() {
	if c.tinyTasksPath == "" {
		return
	}

	hash, modTime, err := hashFileSnapshot(c.tinyTasksPath)
	if err != nil {
		return
	}

	c.mu.Lock()
	c.tinyTasksHash = hash
	c.tinyTasksModTime = modTime
	c.mu.Unlock()
}

func hashFileSnapshot(path string) (string, time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", time.Time{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", time.Time{}, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", time.Time{}, err
	}

	return hex.EncodeToString(hasher.Sum(nil)), info.ModTime(), nil
}
