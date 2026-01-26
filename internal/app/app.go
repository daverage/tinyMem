package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/doctor"
	"github.com/a-marczewski/tinymem/internal/logging"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/storage"
	"go.uber.org/zap"
)

// App holds the core components of the application.
type App struct {
	Config      *config.Config
	Logger      *zap.Logger
	DB          *storage.DB
	Memory      *memory.Service
	ProjectPath string
	ProjectID   string // New field for the current project's ID
	ServerMode  doctor.ServerMode // Track the server mode
}

// NewApp initializes and returns a new App instance.
func NewApp() (*App, error) {
	// 1. Determine project path
	projectPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// 2. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// 3. Initialize logger
	logFile := cfg.LogFile
	if logFile == "" {
		logDir := filepath.Join(cfg.TinyMemDir, "logs")
		logFile = filepath.Join(logDir, fmt.Sprintf("tinymem-%s.log", time.Now().Format("2006-01-02")))
	} else if !filepath.IsAbs(logFile) {
		logFile = filepath.Join(cfg.TinyMemDir, logFile)
	}
	logDir := filepath.Dir(logFile)

	// Ensure log directory exists before initializing logger
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	logger, err := logging.NewLogger(cfg.LogLevel, logFile) // Use cfg.LogLevel and constructed logFile
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// 4. Initialize database
	// storage.NewDB already handles migrations and uses cfg.DBPath
	db, err := storage.NewDB(cfg)
	if err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 5. Initialize memory service
	memoryService := memory.NewService(db)

	projectID := config.GenerateProjectID(projectPath) // Generate project ID

	return &App{
		Config:      cfg,
		Logger:      logger,
		DB:          db,
		Memory:      memoryService,
		ProjectPath: projectPath,
		ProjectID:   projectID, // Store the generated project ID
		ServerMode:  doctor.StandaloneMode, // Default to standalone mode
	}, nil
}

// Close gracefully shuts down the application resources.
func (a *App) Close() {
	if a.DB != nil {
		if err := a.DB.Close(); err != nil {
			a.Logger.Error("Failed to close database connection", zap.Error(err))
		} else {
			a.Logger.Info("Database connection closed.")
		}
	}
	if a.Logger != nil {
		if err := a.Logger.Sync(); err != nil {
			// Zap's Sync can return an error if the underlying writer fails
			// For os.Stderr or regular files, it's usually safe to ignore certain errors.
			// However, it's good practice to log unexpected errors.
			if !strings.Contains(err.Error(), "sync /dev/stderr: invalid argument") &&
				!strings.Contains(err.Error(), "sync <file descriptor>: bad file descriptor") &&
				!strings.Contains(err.Error(), "sync /dev/stderr: inappropriate ioctl for device") {
				fmt.Fprintf(os.Stderr, "Error syncing logger: %v\n", err)
			}
		}
	}
}

// ContextWithLogger returns a new context with the application's logger.
func (a *App) ContextWithLogger(ctx context.Context) context.Context {
	return logging.ContextWithLogger(ctx, a.Logger)
}

// LoggerFromContext retrieves the logger from the given context, or returns the default app logger.
func (a *App) LoggerFromContext(ctx context.Context) *zap.Logger {
	if logger, ok := logging.LoggerFromContext(ctx); ok {
		return logger
	}
	return a.Logger
}
