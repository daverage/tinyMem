package app

import (
	"context"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/cove"
	"github.com/daverage/tinymem/internal/doctor"
	"github.com/daverage/tinymem/internal/enforcement"
	"github.com/daverage/tinymem/internal/evidence"
	"github.com/daverage/tinymem/internal/execution"
	"github.com/daverage/tinymem/internal/extract"
	"github.com/daverage/tinymem/internal/llm"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/recall"
	"github.com/daverage/tinymem/internal/storage"
	"go.uber.org/zap"
)

// CoreModule holds the core application components
type CoreModule struct {
	Config *config.Config
	Logger *zap.Logger
	DB     *storage.DB
}

// ProjectModule holds project-specific information
type ProjectModule struct {
	Path string
	ID   string
}

// ServerModule holds server-specific information
type ServerModule struct {
	Mode doctor.ServerMode
}

// App holds the core components of the application with better separation of concerns.
type App struct {
	Core        CoreModule
	Project     ProjectModule
	Server      ServerModule
	Memory      *memory.Service
	Execution   *execution.Controller
	Enforcement *enforcement.Recorder
	Ctx         context.Context
	Cancel      context.CancelFunc
}

// RecallServices holds the shared recall-related services
type RecallServices struct {
	EvidenceService *evidence.Service
	RecallEngine    recall.Recaller
	Extractor       *extract.Extractor
	CoVeVerifier    *cove.Verifier // May be nil if CoVe is disabled
	LLMProvider     llm.Provider   // Provider interface for LLM (local, external, or calling AI)
}

// InitializeRecallServices creates and configures the shared recall-related services
// This includes evidence service, recall engine (lexical-only), extractor, and CoVe if enabled
func (a *App) InitializeRecallServices() *RecallServices {
	// Create evidence service
	evidenceService := evidence.NewService(a.Core.DB, a.Core.Config)

	// Create recall engine (lexical-only with FTS5)
	recallEngine := recall.NewEngine(a.Memory, evidenceService, a.Core.Config, a.Core.Logger, a.Core.DB.GetConnection())

	// Create extractor
	extractor := extract.NewExtractor(evidenceService, a.Execution)

	// Initialize LLM provider for CoVe (Chain-of-Verification)
	// Strategy:
	// 1. MCP mode: Use CallingAI provider (Claude/Gemini via MCP)
	// 2. Standalone: Use external HTTP LLM if configured (Ollama, etc.)
	// 3. Otherwise: No LLM (CoVe disabled)
	var llmProvider llm.Provider
	var llmProviderType string

	if a.Core.Config.LLMBaseURL != "" {
		// External HTTP LLM configured (Ollama, etc.)
		llmProvider = llm.NewClient(a.Core.Config)
		llmProviderType = "external HTTP"
	} else if a.Server.Mode == doctor.MCPMode {
		// In MCP mode, use CallingAI provider for CoVe
		llmProvider = llm.NewCallingAIProvider()
		llmProviderType = "calling-ai (MCP mode)"
	}

	if llmProvider != nil {
		a.Core.Logger.Info("LLM provider initialized",
			zap.String("provider", llmProviderType),
			zap.String("mode", string(a.Server.Mode)),
			zap.String("usage", "CoVe verification"),
		)
	} else {
		a.Core.Logger.Info("No LLM provider - CoVe disabled",
			zap.String("mode", string(a.Server.Mode)),
		)
	}

	// Initialize CoVe if enabled (requires LLM for verification)
	var coveVerifier *cove.Verifier
	if a.Core.Config.CoVeEnabled {
		coveVerifier = cove.NewVerifier(a.Core.Config, llmProvider)
		coveVerifier.SetStatsStore(cove.NewSQLiteStatsStore(a.Core.DB.GetConnection()), a.Project.ID)
		extractor.SetCoVeVerifier(coveVerifier)

		a.Core.Logger.Info("CoVe enabled",
			zap.String("mode", string(a.Server.Mode)),
			zap.Float64("confidence_threshold", a.Core.Config.CoVeConfidenceThreshold),
			zap.Int("max_candidates", a.Core.Config.CoVeMaxCandidates),
		)
	}

	return &RecallServices{
		EvidenceService: evidenceService,
		RecallEngine:    recallEngine,
		Extractor:       extractor,
		CoVeVerifier:    coveVerifier,
		LLMProvider:     llmProvider,
	}
}
