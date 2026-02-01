//go:build nollm

package ralph

import (
	"context"
	"errors"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/memory"
	"go.uber.org/zap"
)

// Engine stub for no-llm builds
type Engine struct{}

// NewEngine returns a stub engine for no-llm builds
func NewEngine(cfg *config.Config, mem *memory.Service, projectID string, logger *zap.Logger) *Engine {
	return &Engine{}
}

// ExecuteLoop returns an error indicating Ralph is unavailable in no-llm builds
func (e *Engine) ExecuteLoop(ctx context.Context, opt Options) (*Result, error) {
	return nil, errors.New("Ralph (autonomous repair loop) is unavailable in no-llm builds")
}
