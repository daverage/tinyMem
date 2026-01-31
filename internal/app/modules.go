package app

import (
	"context"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/cove"
	"github.com/daverage/tinymem/internal/doctor"
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
	CoVeVerifier    *cove.Verifier // May be nil if CoVe is disabled
	LLMClient       *llm.Client    // May be nil if CoVe is disabled
}

// InitializeRecallServices creates and configures the shared recall-related services
// This includes evidence service, recall engine (lexical or semantic), extractor, and CoVe if enabled
func (a *App) InitializeRecallServices() *RecallServices {
	// Create evidence service
	evidenceService := evidence.NewService(a.Core.DB, a.Core.Config)

	// Create recall engine (semantic or lexical based on config)
	var recallEngine recall.Recaller
	if a.Core.Config.SemanticEnabled {
		recallEngine = semantic.NewSemanticEngine(a.Core.DB, a.Memory, evidenceService, a.Core.Config, a.Core.Logger)
	} else {
		recallEngine = recall.NewEngine(a.Memory, evidenceService, a.Core.Config, a.Core.Logger, a.Core.DB.GetConnection())
	}

	// Create extractor
	extractor := extract.NewExtractor(evidenceService)

	// Initialize CoVe if enabled
	var coveVerifier *cove.Verifier
	var llmClient *llm.Client
	if a.Core.Config.CoVeEnabled {
		llmClient = llm.NewClient(a.Core.Config)
		coveVerifier = cove.NewVerifier(a.Core.Config, llmClient)
		coveVerifier.SetStatsStore(cove.NewSQLiteStatsStore(a.Core.DB.GetConnection()), a.Project.ID)
		extractor.SetCoVeVerifier(coveVerifier)

		a.Core.Logger.Info("CoVe enabled (extraction + recall filtering)",
			zap.Float64("confidence_threshold", a.Core.Config.CoVeConfidenceThreshold),
			zap.Int("max_candidates", a.Core.Config.CoVeMaxCandidates),
		)
	}

	return &RecallServices{
		EvidenceService: evidenceService,
		RecallEngine:    recallEngine,
		Extractor:       extractor,
		CoVeVerifier:    coveVerifier,
		LLMClient:       llmClient,
	}
}