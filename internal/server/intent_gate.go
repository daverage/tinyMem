package server

import (
	"errors"
	"fmt"

	"github.com/daverage/tinymem/internal/enforcement"
	"github.com/daverage/tinymem/internal/execution"
	"github.com/daverage/tinymem/internal/intent"
	"go.uber.org/zap"
)

// IntentGate centralizes mode, recall, and intention checks for both
// MCP and Proxy transports so there is a single enforcement boundary.
type IntentGate struct {
	modeCtrl   *execution.Controller
	enforcer   *enforcement.Recorder
	logger     *zap.Logger
	hasRecalled bool
}

// NewIntentGate constructs a gate that enforces the provided controller and recorder.
func NewIntentGate(modeCtrl *execution.Controller, enforcer *enforcement.Recorder, logger *zap.Logger) *IntentGate {
	return &IntentGate{
		modeCtrl: modeCtrl,
		enforcer: enforcer,
		logger:   logger,
	}
}

// EnsureIntent validates the given tool/action pair against the declarative registry.
func (g *IntentGate) EnsureIntent(tool string, action execution.Action) (intent.Definition, error) {
	def, ok := intent.Lookup(tool)
	if !ok {
		return def, fmt.Errorf("intent metadata missing for tool %s", tool)
	}

	minMode := def.RequiredMode
	if minMode == "" {
		minMode = execution.ModePassive
	}

	if err := g.requireMode(action, minMode); err != nil {
		return def, err
	}

	if def.RequiresRecall {
		if err := g.ensureRecallBeforeMutation(tool, action); err != nil {
			return def, err
		}
	}

	return def, nil
}

// RecordRecall marks that a recall tool was invoked.
func (g *IntentGate) RecordRecall() {
	g.hasRecalled = true
}

func (g *IntentGate) requireMode(action execution.Action, minimum execution.Mode) error {
	if g.modeCtrl == nil {
		return nil
	}

	if err := g.modeCtrl.RefreshTinyTasksState(); err != nil {
		if g.logger != nil {
			g.logger.Warn("Failed to inspect tinyTasks.md state", zap.Error(err))
		}
	}

	mode := g.modeCtrl.Mode()
	modeSet := g.modeCtrl.ModeSet()

	if err := g.modeCtrl.Enforce(action, minimum); err != nil {
		message := err.Error()

		code := enforcementCodeModeTooLow
		if !modeSet {
			code = enforcementCodeModeNotSet
		}
		if errors.Is(err, execution.ErrStrictRequired) {
			message = "STRICT mode required"
			code = enforcementCodeStrictRequired
		}

		if g.enforcer != nil {
			g.enforcer.RecordEvent(enforcement.Event{
				Code:     code,
				Boundary: string(action),
				Mode:     string(mode),
				Outcome:  enforcement.OutcomeBlock,
				Details:  message,
			})
		}

		return errors.New(message)
	}

	if g.enforcer != nil {
		g.enforcer.RecordEvent(enforcement.Event{
			Code:     enforcementCodeModeCompliance,
			Boundary: string(action),
			Mode:     string(mode),
			Outcome:  enforcement.OutcomeAllow,
		})
	}

	return nil
}

func (g *IntentGate) ensureRecallBeforeMutation(tool string, action execution.Action) error {
	if g.hasRecalled {
		return nil
	}

	currentMode := execution.ModePassive
	if g.modeCtrl != nil {
		currentMode = g.modeCtrl.Mode()
	}

	if currentMode >= execution.ModeGuarded {
		message := fmt.Sprintf("Memory recall required: Call memory_query or memory_recent before %s in GUARDED/STRICT modes", tool)
		if g.enforcer != nil {
			g.enforcer.RecordEvent(enforcement.Event{
				Code:     enforcementCodeRecallRequired,
				Boundary: string(action),
				Mode:     string(currentMode),
				Outcome:  enforcement.OutcomeBlock,
				Details:  message,
			})
		}
		return errors.New(message)
	}

	if g.enforcer != nil {
		g.enforcer.RecordEvent(enforcement.Event{
			Code:     enforcementCodeRecallRequired,
			Boundary: string(action),
			Mode:     string(currentMode),
			Outcome:  enforcement.OutcomeViolation,
			Details:  "Contract violation: Memory recall should precede mutations",
		})
	}

	return nil
}

// EnforceMode applies an explicit minimum mode check without re-running intent lookup.
func (g *IntentGate) EnforceMode(action execution.Action, minimum execution.Mode) error {
	return g.requireMode(action, minimum)
}
