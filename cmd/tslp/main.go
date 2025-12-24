package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrzejmarczewski/tslp/config"
	"github.com/andrzejmarczewski/tslp/internal/api"
	"github.com/andrzejmarczewski/tslp/internal/audit"
	"github.com/andrzejmarczewski/tslp/internal/hydration"
	"github.com/andrzejmarczewski/tslp/internal/llm"
	"github.com/andrzejmarczewski/tslp/internal/logging"
	"github.com/andrzejmarczewski/tslp/internal/runtime"
	"github.com/andrzejmarczewski/tslp/internal/storage"
)

var (
	configPath = flag.String("config", "config/config.toml", "Path to configuration file")
	version    = "v5.3-gold"
)

func main() {
	flag.Parse()

	fmt.Printf("TSLP (Transactional State-Ledger Proxy) %s\n", version)
	fmt.Println("Per Specification v5.3 (Gold)")
	fmt.Println()

	// Load and validate configuration
	// Per spec: fail fast on missing or invalid config, config is immutable after startup
	fmt.Printf("Loading configuration from: %s\n", *configPath)
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

	// Configuration is now immutable - passed by value to prevent modification
	// Initialize logger
	logger, err := logging.New(cfg.Logging.LogPath, cfg.Logging.Debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("TSLP %s starting", version)
	logger.Info("Configuration:")
	logger.Info("  Database: %s", cfg.Database.DatabasePath)
	logger.Info("  Log file: %s", cfg.Logging.LogPath)
	logger.Info("  Debug mode: %v", cfg.Logging.Debug)
	logger.Info("  LLM Provider: %s", cfg.LLM.Provider)
	logger.Info("  LLM Endpoint: %s", cfg.LLM.Endpoint)
	logger.Info("  LLM Model: %s", cfg.LLM.Model)
	logger.Info("  Proxy Address: %s", cfg.Proxy.ListenAddress)

	// Open database
	logger.Info("Initializing database: %s", cfg.Database.DatabasePath)
	db, err := storage.Open(cfg.Database.DatabasePath)
	if err != nil {
		logger.Error("FATAL: Failed to open database: %v", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("✓ Database initialized with WAL mode")

	// Initialize runtime
	logger.Info("Initializing runtime components")
	rt := runtime.New(db.Conn())

	// Initialize hydration engine
	hydrator := hydration.New(rt.GetVault(), rt.GetState())

	// Initialize LLM client
	logger.Info("Initializing LLM client")
	llmClient := llm.NewClient(cfg.LLM.Endpoint, cfg.LLM.APIKey, cfg.LLM.Model)

	// Initialize shadow auditor
	auditor := audit.NewAuditor(llmClient, rt.GetVault(), rt.GetLedger(), logger)

	// Initialize API server
	logger.Info("Initializing API server")
	server := api.NewServer(rt, llmClient, hydrator, auditor, logger, cfg.Proxy.ListenAddress)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Start()
	}()

	logger.Info("✓ TSLP proxy ready")
	logger.Info("")
	logger.Info("Core Principles:")
	logger.Info("  • The LLM is stateless")
	logger.Info("  • The Proxy is authoritative")
	logger.Info("  • State advances only by structural proof")
	logger.Info("  • Nothing is overwritten without acknowledgement")
	logger.Info("  • Continuity is structural, not linguistic")
	logger.Info("  • Truth is materialized, never inferred")
	logger.Info("")
	fmt.Println()
	fmt.Printf("✓ Proxy listening on http://%s\n", cfg.Proxy.ListenAddress)
	fmt.Printf("  OpenAI-compatible endpoint: http://%s/v1/chat/completions\n", cfg.Proxy.ListenAddress)
	fmt.Println()
	fmt.Println("Press Ctrl+C to shutdown")

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("FATAL: Server error: %v", err)
		os.Exit(1)

	case sig := <-shutdown:
		logger.Info("Received signal: %v", sig)
		logger.Info("Initiating graceful shutdown")

		// Graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Error during shutdown: %v", err)
			os.Exit(1)
		}

		logger.Info("✓ TSLP shutdown complete")
	}
}
