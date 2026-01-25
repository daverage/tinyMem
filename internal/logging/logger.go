package logging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new zap.Logger instance with file output.
// It returns a *zap.Logger, which is now the standard logger type across the app.
func NewLogger(levelStr, logPath string) (*zap.Logger, error) {
	return NewLoggerWithStderr(levelStr, logPath, true)
}

// NewLoggerWithStderr creates a new zap.Logger instance with optional stderr output.
// When includeStderr is false, logs only go to file (useful for MCP mode over stdio).
func NewLoggerWithStderr(levelStr, logPath string, includeStderr bool) (*zap.Logger, error) {
	// Configure log level
	var level zapcore.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "off":
		level = zap.FatalLevel + 1 // Effectively "off" for non-fatal levels
	default:
		level = zap.InfoLevel
	}

	// Configure file output
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", logPath, err)
	}

	// Combine outputs
	var core zapcore.Core
	if includeStderr {
		// MultiWriteSyncer combines file and stderr
		ws := zapcore.NewMultiWriteSyncer(zapcore.AddSync(logFile), zapcore.AddSync(os.Stderr))
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), // Use JSON encoder for structured logging
			ws,
			level,
		)
	} else {
		// Only write to file
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(logFile),
			level,
		)
	}

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	return logger, nil
}

type loggerContextKey struct{}

// ContextWithLogger returns a new context with the given logger.
func ContextWithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext retrieves the logger from the given context, or returns false if not found.
func LoggerFromContext(ctx context.Context) (*zap.Logger, bool) {
	logger, ok := ctx.Value(loggerContextKey{}).(*zap.Logger)
	return logger, ok
}

// EnsureLogFile ensures the log file and its directory exist.
func EnsureLogFile(logPath string) error {
	logDir := filepath.Dir(logPath)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
		}
	}
	// Check if log file exists and create if not (OpenFile handles this, but good to ensure path)
	_, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		file, err := os.Create(logPath)
		if err != nil {
			return fmt.Errorf("failed to create log file %s: %w", logPath, err)
		}
		file.Close()
	}
	return nil
}
