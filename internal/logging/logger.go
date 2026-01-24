package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"tinymem/internal/config"
)

// Level represents the logging level
type Level int

const (
	Off Level = iota
	Error
	Warn
	Info
	Debug
)

// Logger wraps the standard logger with additional functionality
type Logger struct {
	logger     *log.Logger
	level      Level
	fileWriter io.Writer
	config     *config.Config
}

// NewLogger creates a new logger instance
func NewLogger(cfg *config.Config) (*Logger, error) {
	return NewLoggerWithStderr(cfg, true)
}

// NewLoggerWithStderr creates a new logger instance with optional stderr output
// When silent=true, logs only go to file (useful for MCP mode over stdio)
func NewLoggerWithStderr(cfg *config.Config, includeStderr bool) (*Logger, error) {
	logger := &Logger{
		level:  parseLogLevel(cfg.LogLevel),
		config: cfg,
	}

	// Set up file logging if enabled
	if cfg.LogLevel != "off" {
		logDir := filepath.Join(cfg.TinyMemDir, "logs")
		logFile := filepath.Join(logDir, fmt.Sprintf("tinymem-%s.log", time.Now().Format("2006-01-02")))

		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		logger.fileWriter = file

		// For MCP mode, only write to file; otherwise write to both stderr and file
		if includeStderr {
			logger.logger = log.New(io.MultiWriter(os.Stderr, file), "", log.LstdFlags|log.Lshortfile)
		} else {
			logger.logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
		}
	} else {
		if includeStderr {
			logger.logger = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
		} else {
			logger.logger = log.New(io.Discard, "", log.LstdFlags|log.Lshortfile)
		}
	}

	return logger, nil
}

// parseLogLevel converts string log level to Level enum
func parseLogLevel(levelStr string) Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return Debug
	case "info":
		return Info
	case "warn":
		return Warn
	case "error":
		return Error
	case "off":
		return Off
	default:
		return Info // Default to info if invalid
	}
}

// Debug logs a debug message
func (l *Logger) Debug(v ...interface{}) {
	if l.level >= Debug {
		l.logger.Print("[DEBUG] ", fmt.Sprint(v...))
	}
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.level >= Debug {
		l.logger.Print("[DEBUG] ", fmt.Sprintf(format, v...))
	}
}

// Info logs an info message
func (l *Logger) Info(v ...interface{}) {
	if l.level >= Info {
		l.logger.Print("[INFO] ", fmt.Sprint(v...))
	}
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, v ...interface{}) {
	if l.level >= Info {
		l.logger.Print("[INFO] ", fmt.Sprintf(format, v...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(v ...interface{}) {
	if l.level >= Warn {
		l.logger.Print("[WARN] ", fmt.Sprint(v...))
	}
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.level >= Warn {
		l.logger.Print("[WARN] ", fmt.Sprintf(format, v...))
	}
}

// Error logs an error message
func (l *Logger) Error(v ...interface{}) {
	if l.level >= Error {
		l.logger.Print("[ERROR] ", fmt.Sprint(v...))
	}
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.level >= Error {
		l.logger.Print("[ERROR] ", fmt.Sprintf(format, v...))
	}
}

// Close closes the logger and releases resources
func (l *Logger) Close() error {
	if closer, ok := l.fileWriter.(io.WriteCloser); ok {
		return closer.Close()
	}
	return nil
}

// ChangeLogLevel changes the logging level at runtime
func (l *Logger) ChangeLogLevel(levelStr string) {
	l.level = parseLogLevel(levelStr)
}

// GetLogLevel returns the current logging level
func (l *Logger) GetLogLevel() string {
	switch l.level {
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warn:
		return "warn"
	case Error:
		return "error"
	case Off:
		return "off"
	default:
		return "info"
	}
}