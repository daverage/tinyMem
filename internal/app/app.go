package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/a-marczewski/tinymem/internal/config"
	"github.com/a-marczewski/tinymem/internal/logging"
	"github.com/a-marczewski/tinymem/internal/memory"
	"github.com/a-marczewski/tinymem/internal/storage"
	"go.uber.org/zap"
)

// App holds the core components of the application.
type App struct {
	Config      *config.Config
	Logger      *zap.Logger
	DB          *sql.DB
	Memory      *memory.Service
	ProjectPath string
}

// NewApp initializes and returns a new App instance.
func NewApp() (*App, error) {
	// 1. Determine project path
	projectPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// 2. Load configuration
	cfg, err := config.Load(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// 3. Initialize logger
	logPath := filepath.Join(projectPath, cfg.Logging.File)
	logger, err := logging.NewLogger(cfg.Logging.Level, logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// 4. Initialize database
	dbPath := filepath.Join(projectPath, ".tinyMem", "store.sqlite3")
	db, err := storage.NewSQLiteDB(dbPath)
	if err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 5. Initialize memory service
	memoryService := memory.NewService(db, logger)

	return &App{
		Config:      cfg,
		Logger:      logger,
		DB:          db,
		Memory:      memoryService,
		ProjectPath: projectPath,
	}, nil
}

// Close gracefully shuts down the application resources.
func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
		a.Logger.Info("Database connection closed.")
	}
	if a.Logger != nil {
		a.Logger.Sync() // Flushes any buffered log entries
		a.Logger.Info("Logger synced and closed.")
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
