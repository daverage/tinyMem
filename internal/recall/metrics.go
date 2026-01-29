package recall

import (
	"database/sql"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const (
	metricsBufferSize = 100
)

// TierStats holds the breakdown of memories by recall tier.
type TierStats struct {
	Always        int `json:"always"`
	Contextual    int `json:"contextual"`
	Opportunistic int `json:"opportunistic"`
}

// RecallMetric represents a single recall operation's metrics.
type RecallMetric struct {
	ProjectID     string
	Query         string
	QueryType     string // "empty" or "search"
	MemoryIDs     []int64
	MemoryCount   int
	TotalTokens   int
	TierBreakdown TierStats
	DurationMs    int64
	CreatedAt     time.Time
}

// MetricsWriter handles async writing of recall metrics to the database.
type MetricsWriter struct {
	db      *sql.DB
	logger  *zap.Logger
	metrics chan RecallMetric
	wg      sync.WaitGroup
	done    chan struct{}
	closeOnce sync.Once
	closed    atomic.Bool
}

// NewMetricsWriter creates a new async metrics writer.
// Pass nil for db to disable metrics writing.
func NewMetricsWriter(db *sql.DB, logger *zap.Logger) *MetricsWriter {
	if db == nil {
		return nil
	}

	mw := &MetricsWriter{
		db:      db,
		logger:  logger,
		metrics: make(chan RecallMetric, metricsBufferSize),
		done:    make(chan struct{}),
	}

	mw.wg.Add(1)
	go mw.writeLoop()

	return mw
}

// Write queues a metric for async writing. Non-blocking; drops if buffer full.
func (mw *MetricsWriter) Write(metric RecallMetric) {
	if mw == nil || mw.closed.Load() {
		return
	}

	select {
	case mw.metrics <- metric:
		// Successfully queued
	default:
		// Buffer full, drop the metric
		if mw.logger != nil {
			mw.logger.Debug("Metrics buffer full, dropping metric",
				zap.String("project_id", metric.ProjectID),
				zap.String("query", metric.Query),
			)
		}
	}
}

// Close gracefully shuts down the metrics writer, flushing pending writes.
func (mw *MetricsWriter) Close() {
	if mw == nil {
		return
	}

	mw.closeOnce.Do(func() {
		mw.closed.Store(true)
		close(mw.done)
	})
	mw.wg.Wait()
}

// writeLoop runs in a background goroutine, writing metrics to the database.
func (mw *MetricsWriter) writeLoop() {
	defer mw.wg.Done()

	for {
		select {
		case metric := <-mw.metrics:
			mw.writeMetric(metric)
		case <-mw.done:
			// Drain any remaining metrics
			for {
				select {
				case metric := <-mw.metrics:
					mw.writeMetric(metric)
				default:
					return
				}
			}
		}
	}
}

// writeMetric performs the actual database insert.
func (mw *MetricsWriter) writeMetric(metric RecallMetric) {
	// Serialize memory IDs as JSON array
	memoryIDsJSON, err := json.Marshal(metric.MemoryIDs)
	if err != nil {
		if mw.logger != nil {
			mw.logger.Error("Failed to serialize memory IDs", zap.Error(err))
		}
		memoryIDsJSON = []byte("[]")
	}

	// Serialize tier breakdown as JSON
	tierBreakdownJSON, err := json.Marshal(metric.TierBreakdown)
	if err != nil {
		if mw.logger != nil {
			mw.logger.Error("Failed to serialize tier breakdown", zap.Error(err))
		}
		tierBreakdownJSON = []byte("{}")
	}

	_, err = mw.db.Exec(`
		INSERT INTO recall_metrics (
			project_id, query, query_type, memory_ids, memory_count,
			total_tokens, tier_breakdown, duration_ms, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		metric.ProjectID,
		metric.Query,
		metric.QueryType,
		string(memoryIDsJSON),
		metric.MemoryCount,
		metric.TotalTokens,
		string(tierBreakdownJSON),
		metric.DurationMs,
		metric.CreatedAt,
	)

	if err != nil {
		if mw.logger != nil {
			mw.logger.Error("Failed to write recall metric",
				zap.Error(err),
				zap.String("project_id", metric.ProjectID),
			)
		}
	}
}
