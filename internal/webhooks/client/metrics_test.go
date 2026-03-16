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
		return
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
