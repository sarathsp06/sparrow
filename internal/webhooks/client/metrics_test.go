package client

import (
	"sync"
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()

	if m == nil {
		t.Fatal("Expected non-nil metrics")
	}

	if m.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests 0, got %d", m.TotalRequests)
	}

	if m.MinResponseTime != time.Hour {
		t.Errorf("Expected MinResponseTime to be initialized to 1 hour, got %v", m.MinResponseTime)
	}
}

func TestRecordRequest(t *testing.T) {
	m := NewMetrics()
	m.RecordRequest()
	m.RecordRequest()

	if m.TotalRequests != 2 {
		t.Errorf("Expected TotalRequests 2, got %d", m.TotalRequests)
	}
}

func TestRecordSuccess(t *testing.T) {
	m := NewMetrics()
	m.RecordSuccess(100 * time.Millisecond)
	m.RecordSuccess(200 * time.Millisecond)

	if m.SuccessRequests != 2 {
		t.Errorf("Expected SuccessRequests 2, got %d", m.SuccessRequests)
	}

	if m.MinResponseTime > 100*time.Millisecond {
		t.Errorf("Expected MinResponseTime <= 100ms, got %v", m.MinResponseTime)
	}

	if m.MaxResponseTime < 200*time.Millisecond {
		t.Errorf("Expected MaxResponseTime >= 200ms, got %v", m.MaxResponseTime)
	}
}

func TestRecordFailure(t *testing.T) {
	m := NewMetrics()
	m.RecordFailure(150 * time.Millisecond)

	if m.FailedRequests != 1 {
		t.Errorf("Expected FailedRequests 1, got %d", m.FailedRequests)
	}
}

func TestRecordTimeout(t *testing.T) {
	m := NewMetrics()
	m.RecordTimeout()
	m.RecordTimeout()

	if m.TimeoutRequests != 2 {
		t.Errorf("Expected TimeoutRequests 2, got %d", m.TimeoutRequests)
	}
}

func TestRecordCache(t *testing.T) {
	m := NewMetrics()
	m.RecordCacheHit()
	m.RecordCacheHit()
	m.RecordCacheMiss()

	if m.CacheHits != 2 {
		t.Errorf("Expected CacheHits 2, got %d", m.CacheHits)
	}

	if m.CacheMisses != 1 {
		t.Errorf("Expected CacheMisses 1, got %d", m.CacheMisses)
	}
}

func TestRecordConnections(t *testing.T) {
	m := NewMetrics()
	m.RecordConnectionCreated()
	m.RecordConnectionReused()
	m.RecordConnectionReused()

	if m.ConnectionsCreated != 1 {
		t.Errorf("Expected ConnectionsCreated 1, got %d", m.ConnectionsCreated)
	}

	if m.ConnectionsReused != 2 {
		t.Errorf("Expected ConnectionsReused 2, got %d", m.ConnectionsReused)
	}
}

func TestGetStats(t *testing.T) {
	m := NewMetrics()
	m.RecordRequest()
	m.RecordRequest()
	m.RecordSuccess(100 * time.Millisecond)
	m.RecordFailure(200 * time.Millisecond)
	m.RecordCacheHit()
	m.RecordCacheMiss()

	stats := m.GetStats()

	if stats["total_requests"].(int64) != 2 {
		t.Errorf("Expected total_requests 2, got %v", stats["total_requests"])
	}

	if stats["success_requests"].(int64) != 1 {
		t.Errorf("Expected success_requests 1, got %v", stats["success_requests"])
	}

	if stats["failed_requests"].(int64) != 1 {
		t.Errorf("Expected failed_requests 1, got %v", stats["failed_requests"])
	}

	successRate := stats["success_rate"].(float64)
	if successRate != 50.0 {
		t.Errorf("Expected success_rate 50.0, got %v", successRate)
	}

	cacheHitRate := stats["cache_hit_rate"].(float64)
	if cacheHitRate != 50.0 {
		t.Errorf("Expected cache_hit_rate 50.0, got %v", cacheHitRate)
	}
}

func TestMetricsConcurrency(t *testing.T) {
	m := NewMetrics()
	var wg sync.WaitGroup

	// Simulate concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.RecordRequest()
			m.RecordSuccess(100 * time.Millisecond)
		}()
	}

	wg.Wait()

	if m.TotalRequests != 100 {
		t.Errorf("Expected TotalRequests 100, got %d", m.TotalRequests)
	}

	if m.SuccessRequests != 100 {
		t.Errorf("Expected SuccessRequests 100, got %d", m.SuccessRequests)
	}
}

func TestReset(t *testing.T) {
	m := NewMetrics()
	m.RecordRequest()
	m.RecordSuccess(100 * time.Millisecond)
	m.RecordFailure(200 * time.Millisecond)
	m.RecordTimeout()
	m.RecordCacheHit()
	m.RecordCacheMiss()
	m.RecordConnectionCreated()
	m.RecordConnectionReused()

	m.Reset()

	if m.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests 0 after reset, got %d", m.TotalRequests)
	}

	if m.SuccessRequests != 0 {
		t.Errorf("Expected SuccessRequests 0 after reset, got %d", m.SuccessRequests)
	}

	if m.FailedRequests != 0 {
		t.Errorf("Expected FailedRequests 0 after reset, got %d", m.FailedRequests)
	}

	if m.TimeoutRequests != 0 {
		t.Errorf("Expected TimeoutRequests 0 after reset, got %d", m.TimeoutRequests)
	}

	if m.CacheHits != 0 {
		t.Errorf("Expected CacheHits 0 after reset, got %d", m.CacheHits)
	}

	if m.CacheMisses != 0 {
		t.Errorf("Expected CacheMisses 0 after reset, got %d", m.CacheMisses)
	}

	if m.ConnectionsCreated != 0 {
		t.Errorf("Expected ConnectionsCreated 0 after reset, got %d", m.ConnectionsCreated)
	}

	if m.ConnectionsReused != 0 {
		t.Errorf("Expected ConnectionsReused 0 after reset, got %d", m.ConnectionsReused)
	}

	if m.MinResponseTime != time.Hour {
		t.Errorf("Expected MinResponseTime reset to 1 hour, got %v", m.MinResponseTime)
	}

	if m.MaxResponseTime != 0 {
		t.Errorf("Expected MaxResponseTime 0 after reset, got %v", m.MaxResponseTime)
	}
}

func BenchmarkRecordRequest(b *testing.B) {
	m := NewMetrics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordRequest()
	}
}

func BenchmarkRecordSuccess(b *testing.B) {
	m := NewMetrics()
	duration := 100 * time.Millisecond
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordSuccess(duration)
	}
}

func BenchmarkGetStats(b *testing.B) {
	m := NewMetrics()
	m.RecordRequest()
	m.RecordSuccess(100 * time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.GetStats()
	}
}
