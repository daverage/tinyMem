package app

import (
	"context"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/cove"
	"github.com/daverage/tinymem/internal/doctor"
	"github.com/daverage/tinymem/internal/embedding"
	"github.com/daverage/tinymem/internal/evidence"
	"github.com/daverage/tinymem/internal/extract"
	"github.com/daverage/tinymem/internal/llm"
	"github.com/daverage/tinymem/internal/memory"
	"github.com/daverage/tinymem/internal/recall"
	"github.com/daverage/tinymem/internal/semantic"
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
	Core      CoreModule
	Project   ProjectModule
	Server    ServerModule
	Memory    *memory.Service
	Ctx       context.Context
	Cancel    context.CancelFunc
}

// RecallServices holds the shared recall-related services
type RecallServices struct {
	EvidenceService *evidence.Service
	RecallEngine    recall.Recaller
	Extractor       *extract.Extractor
	CoVeVerifier    *cove.Verifier   // May be nil if CoVe is disabled
	LLMProvider     llm.Provider     // Provider interface for LLM (local, external, or calling AI)
}

// InitializeRecallServices creates and configures the shared recall-related services
// This includes evidence service, recall engine (lexical or semantic), extractor, and CoVe if enabled
func (a *App) InitializeRecallServices() *RecallServices {
	// Create evidence service
	evidenceService := evidence.NewService(a.Core.DB, a.Core.Config)

	// Create recall engine (semantic or lexical based on config)
	var recallEngine recall.Recaller
	if a.Core.Config.SemanticEnabled {
		// Auto-detect embedder: try local first, fallback to HTTP
		var embedder embedding.Embedder
		var embedderType string

		// Try local embedder first (only available if built with -tags embeddings)
		localEmbedder, err := embedding.NewLocalEmbedder()
		if err == nil {
			embedder = localEmbedder
			embedderType = "local (embedded model)"
		} else if a.Core.Config.EmbeddingBaseURL != "" {
			// Fallback to HTTP if local unavailable and URL configured
			embedder = embedding.NewHTTPEmbedder(a.Core.Config)
			embedderType = "HTTP"
		} else {
			// No embedder available - log warning and fallback to lexical
			a.Core.Logger.Warn("Semantic search enabled but no embedder available",
				zap.Error(err),
				zap.String("hint", "build with -tags embeddings for local model, or configure embedding_base_url for HTTP"),
			)
			recallEngine = recall.NewEngine(a.Memory, evidenceService, a.Core.Config, a.Core.Logger, a.Core.DB.GetConnection())
		}

		if embedder != nil {
			a.Core.Logger.Info("Semantic search enabled",
				zap.String("embedder", embedderType),
			)
			recallEngine = semantic.NewSemanticEngine(a.Core.DB, a.Memory, evidenceService, a.Core.Config, a.Core.Logger, embedder)
		}
	} else {
		recallEngine = recall.NewEngine(a.Memory, evidenceService, a.Core.Config, a.Core.Logger, a.Core.DB.GetConnection())
	}

	// Create extractor
	extractor := extract.NewExtractor(evidenceService)

	// Initialize LLM provider for Ralph (generative repairs)
	// Strategy:
	// 1. MCP mode: Use CallingAI provider (Claude/Gemini generates repairs)
	// 2. Standalone: Use external HTTP LLM if configured (Ollama, etc.)
	// 3. Otherwise: No LLM (Ralph disabled)
	//
	// Note: CoVe does NOT require LLM - uses semantic similarity scoring instead
	var llmProvider llm.Provider
	var llmProviderType string

	if a.Core.Config.LLMBaseURL != "" {
		// External HTTP LLM configured (Ollama, etc.)
		llmProvider = llm.NewClient(a.Core.Config)
		llmProviderType = "external HTTP"
	} else if a.Server.Mode == doctor.MCPMode {
		// In MCP mode, fallback to CallingAI provider for Ralph repairs
		llmProvider = llm.NewCallingAIProvider()
		llmProviderType = "calling-ai (MCP mode simplified)"
	}

	if llmProvider != nil {
		a.Core.Logger.Info("LLM provider initialized",
			zap.String("provider", llmProviderType),
			zap.String("mode", string(a.Server.Mode)),
			zap.String("usage", "Ralph autonomous repair"),
		)
	} else {
		a.Core.Logger.Info("No LLM provider - Ralph disabled, CoVe uses semantic scoring",
			zap.String("mode", string(a.Server.Mode)),
		)
	}

	// Initialize CoVe if enabled (uses semantic similarity scoring, no LLM needed)
	var coveVerifier *cove.Verifier
	if a.Core.Config.CoVeEnabled {
		coveVerifier = cove.NewVerifier(a.Core.Config, llmProvider) // llmProvider can be nil
		coveVerifier.SetStatsStore(cove.NewSQLiteStatsStore(a.Core.DB.GetConnection()), a.Project.ID)
		extractor.SetCoVeVerifier(coveVerifier)

		scoringMethod := "semantic similarity"
		if llmProvider != nil {
			scoringMethod = "semantic similarity (LLM available for legacy FilterRecall)"
		}

		a.Core.Logger.Info("CoVe enabled",
			zap.String("mode", string(a.Server.Mode)),
			zap.String("scoring", scoringMethod),
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