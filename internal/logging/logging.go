package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Logger provides structured logging for tinyMem
// Per spec: simple, deterministic, inspectable
// Log levels: INFO, WARN, ERROR, DEBUG
// DEBUG controlled only by config.debug
type Logger struct {
	debug       bool
	file        *os.File
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

// New creates a new logger instance
// Per requirements: logs written to log_path only, no stdout in production
func New(logPath string, debug bool) (*Logger, error) {
	// Ensure log directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file for append
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Write to file only (no stdout in production per requirements)
	writer := io.Writer(file)

	// Create loggers for each level
	l := &Logger{
		debug:       debug,
		file:        file,
		infoLogger:  log.New(writer, "[INFO]  ", log.Ldate|log.Ltime|log.Lmicroseconds),
		warnLogger:  log.New(writer, "[WARN]  ", log.Ldate|log.Ltime|log.Lmicroseconds),
		errorLogger: log.New(writer, "[ERROR] ", log.Ldate|log.Ltime|log.Lmicroseconds),
		debugLogger: log.New(writer, "[DEBUG] ", log.Ldate|log.Ltime|log.Lmicroseconds),
	}

	return l, nil
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	l.infoLogger.Printf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.warnLogger.Printf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.errorLogger.Printf(format, args...)
}

// Debug logs a debug message (only if debug mode is enabled)
// Per requirements: DEBUG controlled only by config.debug
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debug {
		l.debugLogger.Printf(format, args...)
	}
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// IsDebug returns whether debug mode is enabled
func (l *Logger) IsDebug() bool {
	return l.debug
}

// --- Domain-Specific Logging Methods ---

// StateTransition logs a state transition event
// Per spec: all state transitions must be explicit and reviewable
func (l *Logger) StateTransition(episodeID, entityKey, fromState, toState, reason string) {
	l.Info("STATE_TRANSITION episode=%s entity=%s from=%s to=%s reason=%q",
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

// PromotionEvaluated logs the outcome of artifact promotion evaluation
func (l *Logger) PromotionEvaluated(artifactHash, entityKey string, promoted bool, reason string) {
	l.Info("PROMOTION_EVAL artifact=%s entity=%s promoted=%t reason=%q",
		artifactHash, entityKey, promoted, reason)
}

// HydrationStarted logs when hydration begins
func (l *Logger) HydrationStarted(entityCount int) {
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

// StartupPhase logs a startup lifecycle phase
// Per requirements: startup sequence must be explicit and ordered
func (l *Logger) StartupPhase(phase string) {
	l.Info("STARTUP_PHASE phase=%s", phase)
}

// StartupComplete logs successful startup
func (l *Logger) StartupComplete(listenAddr string) {
	l.Info("STARTUP_COMPLETE listen_addr=%s", listenAddr)
}

// ShutdownInitiated logs graceful shutdown start
func (l *Logger) ShutdownInitiated(reason string) {
	l.Info("SHUTDOWN_INITIATED reason=%q", reason)
}

// ShutdownComplete logs successful shutdown
func (l *Logger) ShutdownComplete() {
	l.Info("SHUTDOWN_COMPLETE")
}
