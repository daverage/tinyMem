package llm

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks LLM provider performance and usage
type Metrics struct {
	// Counters
	TotalRequests    atomic.Int64
	SuccessfulCalls  atomic.Int64
	FailedCalls      atomic.Int64

	// Latency tracking
	mu              sync.RWMutex
	totalLatencyMs  int64
	minLatencyMs    int64
	maxLatencyMs    int64

	// Token usage
	TotalTokens     atomic.Int64
	PromptTokens    atomic.Int64
	CompletionTokens atomic.Int64

	// Provider info
	ProviderName    string
	StartTime       time.Time
}

// NewMetrics creates a new metrics tracker
func NewMetrics(providerName string) *Metrics {
	return &Metrics{
		ProviderName: providerName,
		StartTime:    time.Now(),
		minLatencyMs: -1, // -1 indicates not yet set
	}
}

// RecordRequest records a request with its outcome
func (m *Metrics) RecordRequest(success bool, latencyMs int64, tokens int) {
	m.TotalRequests.Add(1)

	if success {
		m.SuccessfulCalls.Add(1)
	} else {
		m.FailedCalls.Add(1)
	}

	// Update latency stats
	m.mu.Lock()
	m.totalLatencyMs += latencyMs
	if m.minLatencyMs < 0 || latencyMs < m.minLatencyMs {
		m.minLatencyMs = latencyMs
	}
	if latencyMs > m.maxLatencyMs {
		m.maxLatencyMs = latencyMs
	}
	m.mu.Unlock()

	// Update token count
	if tokens > 0 {
		m.TotalTokens.Add(int64(tokens))
	}
}

// GetStats returns current statistics
func (m *Metrics) GetStats() MetricsStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.TotalRequests.Load()
	var avgLatency int64
	if total > 0 {
		avgLatency = m.totalLatencyMs / total
	}

	return MetricsStats{
		ProviderName:     m.ProviderName,
		TotalRequests:    total,
		SuccessfulCalls:  m.SuccessfulCalls.Load(),
		FailedCalls:      m.FailedCalls.Load(),
		SuccessRate:      m.calculateSuccessRate(),
		AvgLatencyMs:     avgLatency,
		MinLatencyMs:     m.minLatencyMs,
		MaxLatencyMs:     m.maxLatencyMs,
		TotalTokens:      m.TotalTokens.Load(),
		PromptTokens:     m.PromptTokens.Load(),
		CompletionTokens: m.CompletionTokens.Load(),
		Uptime:           time.Since(m.StartTime),
	}
}

// calculateSuccessRate returns success rate as percentage (0-100)
func (m *Metrics) calculateSuccessRate() float64 {
	total := m.TotalRequests.Load()
	if total == 0 {
		return 0
	}
	success := m.SuccessfulCalls.Load()
	return float64(success) / float64(total) * 100.0
}

// MetricsStats holds snapshot of metrics
type MetricsStats struct {
	ProviderName     string
	TotalRequests    int64
	SuccessfulCalls  int64
	FailedCalls      int64
	SuccessRate      float64 // Percentage
	AvgLatencyMs     int64
	MinLatencyMs     int64
	MaxLatencyMs     int64
	TotalTokens      int64
	PromptTokens     int64
	CompletionTokens int64
	Uptime           time.Duration
}

// Global metrics registry
var (
	metricsRegistry = make(map[string]*Metrics)
	metricsLock     sync.RWMutex
)

// RegisterMetrics registers metrics for a provider
func RegisterMetrics(providerName string, metrics *Metrics) {
	metricsLock.Lock()
	defer metricsLock.Unlock()
	metricsRegistry[providerName] = metrics
}

// GetMetrics retrieves metrics for a provider
func GetMetrics(providerName string) *Metrics {
	metricsLock.RLock()
	defer metricsLock.RUnlock()
	return metricsRegistry[providerName]
}

// GetAllMetrics returns metrics for all providers
func GetAllMetrics() map[string]MetricsStats {
	metricsLock.RLock()
	defer metricsLock.RUnlock()

	result := make(map[string]MetricsStats, len(metricsRegistry))
	for name, metrics := range metricsRegistry {
		result[name] = metrics.GetStats()
	}
	return result
}
