package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Logger provides structured logging for TSLP
// Per spec: simple, deterministic, inspectable
type Logger struct {
	debug      bool
	file       *os.File
	infoLogger *log.Logger
	debugLogger *log.Logger
	errorLogger *log.Logger
}

// New creates a new logger instance
func New(logPath string, debug bool) (*Logger, error) {
	// Ensure log directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer for both file and stdout
	multiWriter := io.MultiWriter(file, os.Stdout)

	l := &Logger{
		debug:       debug,
		file:        file,
		infoLogger:  log.New(multiWriter, "[INFO]  ", log.Ldate|log.Ltime|log.Lmicroseconds),
		debugLogger: log.New(multiWriter, "[DEBUG] ", log.Ldate|log.Ltime|log.Lmicroseconds),
		errorLogger: log.New(multiWriter, "[ERROR] ", log.Ldate|log.Ltime|log.Lmicroseconds),
	}

	return l, nil
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	l.infoLogger.Printf(format, args...)
}

// Debug logs a debug message (only if debug mode is enabled)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debug {
		l.debugLogger.Printf(format, args...)
	}
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.errorLogger.Printf(format, args...)
}

// StateTransition logs a state transition event
// Per spec: all state transitions must be explicit and reviewable
func (l *Logger) StateTransition(episodeID, entityKey, fromState, toState, reason string) {
	l.Info("STATE_TRANSITION episode=%s entity=%s from=%s to=%s reason=%s",
		episodeID, entityKey, fromState, toState, reason)
}

// ArtifactStored logs when an artifact is stored in the vault
func (l *Logger) ArtifactStored(hash, contentType string, byteSize int) {
	l.Debug("ARTIFACT_STORED hash=%s type=%s size=%d", hash, contentType, byteSize)
}

// EntityResolved logs the result of entity resolution
func (l *Logger) EntityResolved(artifactHash, entityKey, confidence, method string) {
	l.Debug("ENTITY_RESOLVED artifact=%s entity=%s confidence=%s method=%s",
		artifactHash, entityKey, confidence, method)
}

// PromotionResult logs the outcome of artifact promotion evaluation
func (l *Logger) PromotionResult(artifactHash, entityKey string, promoted bool, reason string) {
	l.Info("PROMOTION_EVAL artifact=%s entity=%s promoted=%t reason=%s",
		artifactHash, entityKey, promoted, reason)
}

// HydrationStart logs when hydration begins
func (l *Logger) HydrationStart(entityCount int) {
	l.Debug("HYDRATION_START entity_count=%d", entityCount)
}

// EpisodeCreated logs when a new episode is created
func (l *Logger) EpisodeCreated(episodeID string) {
	l.Debug("EPISODE_CREATED episode_id=%s", episodeID)
}

// ProxyRequest logs an incoming proxy request
func (l *Logger) ProxyRequest(method, path string) {
	l.Debug("PROXY_REQUEST method=%s path=%s", method, path)
}

// AuditStarted logs when shadow audit begins
func (l *Logger) AuditStarted(episodeID, artifactHash string) {
	l.Debug("AUDIT_STARTED episode=%s artifact=%s", episodeID, artifactHash)
}

// AuditCompleted logs when shadow audit completes
func (l *Logger) AuditCompleted(episodeID, artifactHash, status string) {
	l.Info("AUDIT_COMPLETED episode=%s artifact=%s status=%s", episodeID, artifactHash, status)
}

// Timestamp returns the current timestamp in a consistent format
func (l *Logger) Timestamp() string {
	return time.Now().Format(time.RFC3339)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// WithField returns a formatted string with key=value pairs
// Useful for structured logging
func WithField(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

// WithFields combines multiple key-value pairs
func WithFields(fields map[string]string) string {
	result := ""
	first := true
	for k, v := range fields {
		if !first {
			result += " "
		}
		result += fmt.Sprintf("%s=%s", k, v)
		first = false
	}
	return result
}
