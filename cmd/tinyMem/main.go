package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrzejmarczewski/tinyMem/config"
	"github.com/andrzejmarczewski/tinyMem/internal/api"
	"github.com/andrzejmarczewski/tinyMem/internal/audit"
	"github.com/andrzejmarczewski/tinyMem/internal/entity"
	"github.com/andrzejmarczewski/tinyMem/internal/hydration"
	"github.com/andrzejmarczewski/tinyMem/internal/llm"
	"github.com/andrzejmarczewski/tinyMem/internal/logging"
	"github.com/andrzejmarczewski/tinyMem/internal/runtime"
	"github.com/andrzejmarczewski/tinyMem/internal/storage"
	"github.com/andrzejmarczewski/tinyMem/internal/embeddings"
)

var (
	configPath = flag.String("config", "config/config.toml", "Path to configuration file")
	version    = "v5.3-gold"
)

func main() {
	flag.Parse()

	// Startup banner (stdout only, before logger init)
	fmt.Printf("tinyMem (Transactional State-Ledger Proxy) %s\n", version)
	fmt.Println("Per Specification v5.3 (Gold)")
	fmt.Println()

	// ========================================================================
	// STARTUP PHASE 1: Load Configuration
	// ========================================================================
	fmt.Printf("Phase 1/5: Loading configuration from %s\n", *configPath)
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Configuration error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nConfiguration must include all required fields:\n")
		fmt.Fprintf(os.Stderr, "  - database.database_path\n")
		fmt.Fprintf(os.Stderr, "  - logging.log_path\n")
		fmt.Fprintf(os.Stderr, "  - logging.debug\n")
		fmt.Fprintf(os.Stderr, "  - llm.llm_provider\n")
		fmt.Fprintf(os.Stderr, "  - llm.llm_endpoint\n")
		fmt.Fprintf(os.Stderr, "  - llm.llm_api_key\n")
		fmt.Fprintf(os.Stderr, "  - llm.llm_model\n")
		fmt.Fprintf(os.Stderr, "  - proxy.listen_address\n")
		os.Exit(1)
	}
	fmt.Println("✓ Configuration validated")
	fmt.Println()

	// ========================================================================
	// STARTUP PHASE 2: Initialize Logger
	// ========================================================================
	fmt.Printf("Phase 2/5: Initializing logger (log_path=%s, debug=%v)\n", cfg.Logging.LogPath, cfg.Logging.Debug)
	logger, err := logging.New(cfg.Logging.LogPath, cfg.Logging.Debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()
	fmt.Println("✓ Logger initialized")
	fmt.Println()

	// From this point on, all logging goes to log file only (no stdout in production)
	logger.StartupPhase("1_config_loaded")
	logger.Info("tinyMem %s starting", version)
	logger.Info("Configuration loaded from: %s", *configPath)
	logger.Info("  Database: %s", cfg.Database.DatabasePath)
	logger.Info("  Log file: %s", cfg.Logging.LogPath)
	logger.Info("  Debug mode: %v", cfg.Logging.Debug)
	logger.Info("  LLM Provider: %s", cfg.LLM.Provider)
	logger.Info("  LLM Endpoint: %s", cfg.LLM.Endpoint)
	logger.Info("  LLM Model: %s", cfg.LLM.Model)
	logger.Info("  Proxy Address: %s", cfg.Proxy.ListenAddress)

	logger.StartupPhase("2_logger_initialized")

	// ========================================================================
	// STARTUP PHASE 3: Open Database
	// ========================================================================
	fmt.Printf("Phase 3/5: Opening database at %s\n", cfg.Database.DatabasePath)
	logger.StartupPhase("3_opening_database")
	logger.Info("Opening database: %s", cfg.Database.DatabasePath)

	db, err := storage.Open(cfg.Database.DatabasePath)
	if err != nil {
		logger.Error("FATAL: Failed to open database: %v", err)
		fmt.Fprintf(os.Stderr, "FATAL: Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("Database opened successfully")
	fmt.Println("✓ Database opened")
	fmt.Println()

	// ========================================================================
	// STARTUP PHASE 4: Run Migrations
	// ========================================================================
	fmt.Println("Phase 4/5: Running database migrations")
	logger.StartupPhase("4_running_migrations")
	logger.Info("Database migrations completed (WAL mode enabled)")
	fmt.Println("✓ Migrations complete (WAL mode enabled)")
	fmt.Println()

	// ========================================================================
	// STARTUP PHASE 5: Initialize Runtime and Start HTTP Server
	// ========================================================================
	fmt.Println("Phase 5/5: Starting HTTP server")
	logger.StartupPhase("5_starting_server")

	// Load symbols.json for regex fallback
	logger.Debug("Loading symbols.json patterns")
	if err := entity.LoadSymbolsConfig(); err != nil {
		logger.Warn("Failed to load symbols.json: %v (regex fallback will be unavailable)", err)
	} else {
		logger.Info("Loaded symbols.json for regex fallback")
	}

	// Initialize runtime components
	logger.Debug("Initializing runtime components")
	rt := runtime.New(db.Conn())

	// Initialize hydration engine with tracker and ETV consistency checker
	logger.Debug("Initializing hydration engine")
	hydrator := hydration.New(rt.GetVault(), rt.GetState(), rt.GetHydrationTracker(), rt.GetConsistencyChecker())

	// Initialize hybrid hydration engine with configuration
	logger.Debug("Initializing hybrid hydration engine")
	embedder, err := embeddings.GetEmbedder(cfg.Hydration.EmbeddingProvider, cfg.Hydration.EmbeddingModel, cfg.LLM.APIKey)
	if err != nil {
		logger.Warn("Failed to initialize embedder: %v (using basic hydration only)", err)
		embedder = nil
	}

	// Configure hybrid hydrator with configuration values
	rt.ConfigureHybridHydrator(logger, embedder, cfg.Hydration.EnableSemanticRanking, cfg.Hydration.SemanticThreshold)

	// Initialize LLM client (HTTP or CLI based on provider)
	logger.Debug("Initializing LLM client (provider=%s)", cfg.LLM.Provider)
	var llmClient interface {
		Chat(ctx context.Context, messages []llm.Message) (*llm.ChatResponse, error)
		GetModel() string
		CountMessagesTokens([]llm.Message) int
		CountTokens(string) int
	}

	if cfg.LLM.IsCLIProvider() {
		// CLI-based provider (claude, gemini, etc.)
		logger.Info("Using CLI provider: %s", cfg.LLM.Provider)
		cliClient, err := llm.NewCLIClient(cfg.LLM.Provider, cfg.LLM.Model)
		if err != nil {
			logger.Error("FATAL: Failed to create CLI client: %v", err)
			fmt.Fprintf(os.Stderr, "FATAL: Failed to create CLI client: %v\n", err)
			os.Exit(1)
		}
		llmClient = cliClient
	} else {
		// HTTP-based provider (lmstudio, openai, etc.)
		logger.Info("Using HTTP provider: %s at %s", cfg.LLM.Provider, cfg.LLM.Endpoint)
		llmClient = llm.NewClient(cfg.LLM.Endpoint, cfg.LLM.APIKey, cfg.LLM.Model)
	}

	// Initialize shadow auditor
	logger.Debug("Initializing shadow auditor")
	auditor := audit.NewAuditor(llmClient, rt.GetVault(), rt.GetLedger(), logger)

	// Initialize API server
	logger.Debug("Initializing API server on %s", cfg.Proxy.ListenAddress)
	server := api.NewServer(
		rt,
		llmClient,
		hydrator,
		auditor,
		logger,
		cfg.Proxy.ListenAddress,
		cfg.Database.DatabasePath,
		cfg.LLM.Provider,
		cfg.LLM.Endpoint,
		cfg.Logging.Debug,
		cfg.Hydration,
	)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening on %s", cfg.Proxy.ListenAddress)
		serverErrors <- server.Start()
	}()

	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)

	logger.StartupComplete(cfg.Proxy.ListenAddress)
	fmt.Println("✓ HTTP server started")
	fmt.Println()

	// Startup complete
	fmt.Println("========================================")
	fmt.Println("tinyMem Ready")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Core Principles:")
	fmt.Println("  • The LLM is stateless")
	fmt.Println("  • The Proxy is authoritative")
	fmt.Println("  • State advances only by structural proof")
	fmt.Println("  • Nothing is overwritten without acknowledgement")
	fmt.Println("  • Continuity is structural, not linguistic")
	fmt.Println("  • Truth is materialized, never inferred")
	fmt.Println()
	fmt.Printf("Endpoint: http://%s/v1/chat/completions\n", cfg.Proxy.ListenAddress)
	fmt.Printf("Log file: %s\n", cfg.Logging.LogPath)
	fmt.Println()
	fmt.Println("Press Ctrl+C to shutdown")
	fmt.Println()

	// ========================================================================
	// RUNTIME: Wait for shutdown signal or server error
	// ========================================================================
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("FATAL: Server error: %v", err)
		fmt.Fprintf(os.Stderr, "FATAL: Server error: %v\n", err)
		os.Exit(1)

	case sig := <-shutdown:
		fmt.Printf("\nReceived signal: %v\n", sig)
		logger.ShutdownInitiated(fmt.Sprintf("signal=%v", sig))

		fmt.Println("Initiating graceful shutdown...")

		// Graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Error during shutdown: %v", err)
			fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
			os.Exit(1)
		}

		logger.ShutdownComplete()
		fmt.Println("✓ Shutdown complete")
	}
}
