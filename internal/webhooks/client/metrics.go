package client

import (
	"sync/atomic"
	"time"
)

// Metrics tracks webhook client performance metrics
type Metrics struct {
	// Request counters
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TimeoutRequests int64

	// Error category counters
	ClientErrors      int64 // 4xx responses
	ServerErrors      int64 // 5xx responses
	DNSErrors         int64
	TLSErrors         int64
	ConnectionRefused int64
	NetworkErrors     int64 // other network errors

	// DNS cache metrics
	CacheHits   int64
	CacheMisses int64

	// Connection metrics
	ConnectionsCreated int64
	ConnectionsReused  int64

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
	atomic.AddInt64(&m.TotalRequests, 1)
}

// RecordSuccess increments success counter
func (m *Metrics) RecordSuccess(duration time.Duration) {
	atomic.AddInt64(&m.SuccessRequests, 1)
	m.recordResponseTime(duration)
}

// RecordFailure increments failure counter
func (m *Metrics) RecordFailure(duration time.Duration) {
	atomic.AddInt64(&m.FailedRequests, 1)
	m.recordResponseTime(duration)
}

// RecordTimeout increments timeout counter
func (m *Metrics) RecordTimeout() {
	atomic.AddInt64(&m.TimeoutRequests, 1)
}

// RecordErrorCategory increments the counter for a specific error category
func (m *Metrics) RecordErrorCategory(category string) {
	switch category {
	case "client_error":
		atomic.AddInt64(&m.ClientErrors, 1)
	case "server_error":
		atomic.AddInt64(&m.ServerErrors, 1)
	case "timeout":
		atomic.AddInt64(&m.TimeoutRequests, 1)
	case "dns_error":
		atomic.AddInt64(&m.DNSErrors, 1)
	case "tls_error":
		atomic.AddInt64(&m.TLSErrors, 1)
	case "connection_refused":
		atomic.AddInt64(&m.ConnectionRefused, 1)
	case "network_error":
		atomic.AddInt64(&m.NetworkErrors, 1)
	}
}

// RecordCacheHit increments cache hit counter
func (m *Metrics) RecordCacheHit() {
	atomic.AddInt64(&m.CacheHits, 1)
}

// RecordCacheMiss increments cache miss counter
func (m *Metrics) RecordCacheMiss() {
	atomic.AddInt64(&m.CacheMisses, 1)
}

// RecordConnectionCreated increments connection created counter
func (m *Metrics) RecordConnectionCreated() {
	atomic.AddInt64(&m.ConnectionsCreated, 1)
}

// RecordConnectionReused increments connection reused counter
func (m *Metrics) RecordConnectionReused() {
	atomic.AddInt64(&m.ConnectionsReused, 1)
}

// recordResponseTime updates response time statistics
func (m *Metrics) recordResponseTime(duration time.Duration) {
	// Update total response time
	for {
		old := atomic.LoadInt64((*int64)(&m.TotalResponseTime))
		new := old + int64(duration)
		if atomic.CompareAndSwapInt64((*int64)(&m.TotalResponseTime), old, new) {
			break
		}
	}

	// Update min response time
	for {
		old := atomic.LoadInt64((*int64)(&m.MinResponseTime))
		if duration >= time.Duration(old) {
			break
		}
		if atomic.CompareAndSwapInt64((*int64)(&m.MinResponseTime), old, int64(duration)) {
			break
		}
	}

	// Update max response time
	for {
		old := atomic.LoadInt64((*int64)(&m.MaxResponseTime))
		if duration <= time.Duration(old) {
			break
		}
		if atomic.CompareAndSwapInt64((*int64)(&m.MaxResponseTime), old, int64(duration)) {
			break
		}
	}
}

// GetStats returns current metrics as a map
func (m *Metrics) GetStats() map[string]interface{} {
	total := atomic.LoadInt64(&m.TotalRequests)
	success := atomic.LoadInt64(&m.SuccessRequests)
	failed := atomic.LoadInt64(&m.FailedRequests)
	timeout := atomic.LoadInt64(&m.TimeoutRequests)
	clientErrors := atomic.LoadInt64(&m.ClientErrors)
	serverErrors := atomic.LoadInt64(&m.ServerErrors)
	dnsErrors := atomic.LoadInt64(&m.DNSErrors)
	tlsErrors := atomic.LoadInt64(&m.TLSErrors)
	connRefused := atomic.LoadInt64(&m.ConnectionRefused)
	networkErrors := atomic.LoadInt64(&m.NetworkErrors)
	cacheHits := atomic.LoadInt64(&m.CacheHits)
	cacheMisses := atomic.LoadInt64(&m.CacheMisses)
	connCreated := atomic.LoadInt64(&m.ConnectionsCreated)
	connReused := atomic.LoadInt64(&m.ConnectionsReused)

	var successRate, cacheHitRate float64
	resolved := success + failed + timeout
	if resolved > 0 {
		successRate = float64(success) / float64(resolved) * 100
	}
	if (cacheHits + cacheMisses) > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
	}

	var avgResponseTime time.Duration
	completedWithTime := success + failed // only success and failure record response times
	if completedWithTime > 0 {
		avgResponseTime = time.Duration(atomic.LoadInt64((*int64)(&m.TotalResponseTime))) / time.Duration(completedWithTime)
	}

	return map[string]interface{}{
		"total_requests":      total,
		"success_requests":    success,
		"failed_requests":     failed,
		"timeout_requests":    timeout,
		"client_errors":       clientErrors,
		"server_errors":       serverErrors,
		"dns_errors":          dnsErrors,
		"tls_errors":          tlsErrors,
		"connection_refused":  connRefused,
		"network_errors":      networkErrors,
		"success_rate":        successRate,
		"cache_hits":          cacheHits,
		"cache_misses":        cacheMisses,
		"cache_hit_rate":      cacheHitRate,
		"connections_created": connCreated,
		"connections_reused":  connReused,
		"avg_response_time":   avgResponseTime,
		"min_response_time":   time.Duration(atomic.LoadInt64((*int64)(&m.MinResponseTime))),
		"max_response_time":   time.Duration(atomic.LoadInt64((*int64)(&m.MaxResponseTime))),
	}
}

// Reset resets all metrics to zero
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.TotalRequests, 0)
	atomic.StoreInt64(&m.SuccessRequests, 0)
	atomic.StoreInt64(&m.FailedRequests, 0)
	atomic.StoreInt64(&m.TimeoutRequests, 0)
	atomic.StoreInt64(&m.ClientErrors, 0)
	atomic.StoreInt64(&m.ServerErrors, 0)
	atomic.StoreInt64(&m.DNSErrors, 0)
	atomic.StoreInt64(&m.TLSErrors, 0)
	atomic.StoreInt64(&m.ConnectionRefused, 0)
	atomic.StoreInt64(&m.NetworkErrors, 0)
	atomic.StoreInt64(&m.CacheHits, 0)
	atomic.StoreInt64(&m.CacheMisses, 0)
	atomic.StoreInt64(&m.ConnectionsCreated, 0)
	atomic.StoreInt64(&m.ConnectionsReused, 0)
	atomic.StoreInt64((*int64)(&m.TotalResponseTime), 0)
	atomic.StoreInt64((*int64)(&m.MinResponseTime), int64(time.Hour))
	atomic.StoreInt64((*int64)(&m.MaxResponseTime), 0)
}
