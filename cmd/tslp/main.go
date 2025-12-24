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

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := logging.New(cfg.Logging.LogPath, cfg.Logging.Debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("TSLP %s starting", version)
	logger.Info("Configuration loaded from: %s", *configPath)

	// Open database
	logger.Info("Opening database: %s", cfg.Database.DatabasePath)
	db, err := storage.Open(cfg.Database.DatabasePath)
	if err != nil {
		logger.Error("Failed to open database: %v", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Database initialized with WAL mode")

	// Initialize runtime
	logger.Info("Initializing runtime components")
	rt := runtime.New(db.Conn())

	// Initialize hydration engine
	hydrator := hydration.New(rt.GetVault(), rt.GetState())

	// Initialize LLM client
	logger.Info("Initializing LLM client: provider=%s model=%s endpoint=%s",
		cfg.LLM.Provider, cfg.LLM.Model, cfg.LLM.Endpoint)
	llmClient := llm.NewClient(cfg.LLM.Endpoint, cfg.LLM.APIKey, cfg.LLM.Model)

	// Initialize shadow auditor
	auditor := audit.NewAuditor(llmClient, rt.GetVault(), rt.GetLedger(), logger)

	// Initialize API server
	logger.Info("Initializing API server on %s", cfg.Proxy.ListenAddress)
	server := api.NewServer(rt, llmClient, hydrator, auditor, logger, cfg.Proxy.ListenAddress)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Start()
	}()

	logger.Info("TSLP proxy ready")
	logger.Info("OpenAI-compatible endpoint: http://%s/v1/chat/completions", cfg.Proxy.ListenAddress)
	logger.Info("Core principles:")
	logger.Info("  - The LLM is stateless")
	logger.Info("  - The Proxy is authoritative")
	logger.Info("  - State advances only by structural proof")
	logger.Info("  - Nothing is overwritten without acknowledgement")
	logger.Info("  - Continuity is structural, not linguistic")
	logger.Info("  - Truth is materialized, never inferred")
	fmt.Println()
	fmt.Printf("Proxy listening on http://%s\n", cfg.Proxy.ListenAddress)
	fmt.Println("Press Ctrl+C to shutdown")

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("Server error: %v", err)
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

		logger.Info("TSLP shutdown complete")
	}
}
