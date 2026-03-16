package client

import (
	"sync"
	"time"
)

// Metrics tracks webhook client performance metrics
type Metrics struct {
	mu sync.Mutex

	// Request counters
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64

	// Response time tracking
	TotalResponseTime time.Duration
	MinResponseTime   time.Duration
	MaxResponseTime   time.Duration
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		MinResponseTime: time.Hour, // Initialize to a high value
	}
}

// RecordRequest increments total request counter
func (m *Metrics) RecordRequest() {
	m.mu.Lock()
	m.TotalRequests++
	m.mu.Unlock()
}

// RecordSuccess increments success counter
func (m *Metrics) RecordSuccess(duration time.Duration) {
	m.mu.Lock()
	m.SuccessRequests++
	m.TotalResponseTime += duration
	if duration < m.MinResponseTime {
		m.MinResponseTime = duration
	}
	if duration > m.MaxResponseTime {
		m.MaxResponseTime = duration
	}
	m.mu.Unlock()
}

// RecordFailure increments failure counter
func (m *Metrics) RecordFailure(duration time.Duration) {
	m.mu.Lock()
	m.FailedRequests++
	m.TotalResponseTime += duration
	if duration < m.MinResponseTime {
		m.MinResponseTime = duration
	}
	if duration > m.MaxResponseTime {
		m.MaxResponseTime = duration
	}
	m.mu.Unlock()
}
