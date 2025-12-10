package benchmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
)

// ResourceMetrics tracks resource usage during benchmarks
type ResourceMetrics struct {
	StartMemStats   runtime.MemStats
	EndMemStats     runtime.MemStats
	PeakMemory      uint64
	AllocatedMemory uint64
	GCPauses        uint64
	Goroutines      int
	StartTime       time.Time
	EndTime         time.Time
}

// BandwidthMetrics tracks network bandwidth usage
type BandwidthMetrics struct {
	TotalBytesSent     int64
	TotalBytesReceived int64
	RequestCount       int64
	AvgRequestSize     int64
	AvgResponseSize    int64
}

// captureResourceMetrics captures memory and goroutine statistics
func captureResourceMetrics() *ResourceMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return &ResourceMetrics{
		StartMemStats: m,
		Goroutines:    runtime.NumGoroutine(),
		StartTime:     time.Now(),
	}
}

func (rm *ResourceMetrics) Stop() {
	runtime.ReadMemStats(&rm.EndMemStats)
	rm.EndTime = time.Now()
	rm.AllocatedMemory = rm.EndMemStats.TotalAlloc - rm.StartMemStats.TotalAlloc
	rm.PeakMemory = rm.EndMemStats.Sys
	rm.GCPauses = rm.EndMemStats.PauseTotalNs - rm.StartMemStats.PauseTotalNs
}

func (rm *ResourceMetrics) Print() {
	duration := rm.EndTime.Sub(rm.StartTime)
	fmt.Printf("\n=== Resource Usage ===\n")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Memory Allocated: %.2f MB\n", float64(rm.AllocatedMemory)/1024/1024)
	fmt.Printf("Peak Memory: %.2f MB\n", float64(rm.PeakMemory)/1024/1024)
	fmt.Printf("Heap In Use: %.2f MB\n", float64(rm.EndMemStats.HeapInuse)/1024/1024)
	fmt.Printf("Heap Allocated: %.2f MB\n", float64(rm.EndMemStats.HeapAlloc)/1024/1024)
	fmt.Printf("GC Pauses: %v ms\n", rm.GCPauses/1e6)
	fmt.Printf("Goroutines: %d\n", rm.Goroutines)
	if rm.Goroutines > 0 {
		fmt.Printf("Memory/Goroutine: %.2f KB\n", float64(rm.AllocatedMemory)/1024/float64(rm.Goroutines))
	}
}

// BenchmarkWebhookDelivery_SmallPayload benchmarks delivery with small payloads (1KB)
func BenchmarkWebhookDelivery_SmallPayload(b *testing.B) {
	benchmarkWebhookDelivery(b, 1024, 1) // 1KB payload
}

// BenchmarkWebhookDelivery_MediumPayload benchmarks delivery with medium payloads (10KB)
func BenchmarkWebhookDelivery_MediumPayload(b *testing.B) {
	benchmarkWebhookDelivery(b, 10*1024, 1) // 10KB payload
}

// BenchmarkWebhookDelivery_LargePayload benchmarks delivery with large payloads (100KB)
func BenchmarkWebhookDelivery_LargePayload(b *testing.B) {
	benchmarkWebhookDelivery(b, 100*1024, 1) // 100KB payload
}

// BenchmarkWebhookDelivery_Concurrent benchmarks concurrent deliveries
func BenchmarkWebhookDelivery_Concurrent(b *testing.B) {
	benchmarkWebhookDelivery(b, 10*1024, 10) // 10KB payload, 10 concurrent
}

// BenchmarkWebhookDelivery_HighConcurrency benchmarks high concurrency
func BenchmarkWebhookDelivery_HighConcurrency(b *testing.B) {
	benchmarkWebhookDelivery(b, 10*1024, 100) // 10KB payload, 100 concurrent
}

// benchmarkWebhookDelivery is a helper function for webhook delivery benchmarks
func benchmarkWebhookDelivery(b *testing.B, payloadSize int, concurrency int) {
	// Create test server
	var bandwidthMu sync.Mutex
	bandwidth := &BandwidthMetrics{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track request size
		if r.ContentLength > 0 {
			bandwidthMu.Lock()
			bandwidth.TotalBytesReceived += r.ContentLength
			bandwidth.RequestCount++
			bandwidthMu.Unlock()
		}

		response := []byte(`{"status":"success"}`)
		bandwidthMu.Lock()
		bandwidth.TotalBytesSent += int64(len(response))
		bandwidthMu.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}))
	defer server.Close()

	// Create client
	webhookClient := client.NewWebhookClient(nil)
	defer webhookClient.Close()

	// Generate test payload
	payload := make(map[string]interface{})
	payload["data"] = string(make([]byte, payloadSize))
	payload["timestamp"] = time.Now().Unix()
	payload["id"] = uuid.New().String()

	payloadBytes, _ := json.Marshal(payload)

	// Create delivery request
	req := &client.DeliveryRequest{
		WebhookID:  uuid.New(),
		DeliveryID: uuid.New().String(),
		EventID:    uuid.New(),
		URL:        server.URL,
		Method:     "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Payload: payloadBytes,
		Timeout: 30 * time.Second,
	}

	// Capture resource metrics
	metrics := captureResourceMetrics()

	b.ResetTimer()
	b.SetParallelism(concurrency)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = webhookClient.Send(context.Background(), req)
		}
	})
	b.StopTimer()

	metrics.Stop()

	// Calculate bandwidth metrics
	if bandwidth.RequestCount > 0 {
		bandwidth.AvgRequestSize = bandwidth.TotalBytesReceived / bandwidth.RequestCount
		bandwidth.AvgResponseSize = bandwidth.TotalBytesSent / bandwidth.RequestCount
	}

	// Report custom metrics
	b.ReportMetric(float64(metrics.AllocatedMemory)/float64(b.N)/1024, "KB/op")
	b.ReportMetric(float64(bandwidth.TotalBytesSent+bandwidth.TotalBytesReceived)/float64(b.N)/1024, "KB-transferred/op")
	b.ReportMetric(float64(metrics.Goroutines), "goroutines")

	if !testing.Short() {
		metrics.Print()
		fmt.Printf("\n=== Bandwidth Usage ===\n")
		fmt.Printf("Total Sent: %.2f MB\n", float64(bandwidth.TotalBytesSent)/1024/1024)
		fmt.Printf("Total Received: %.2f MB\n", float64(bandwidth.TotalBytesReceived)/1024/1024)
		fmt.Printf("Avg Request Size: %.2f KB\n", float64(bandwidth.AvgRequestSize)/1024)
		fmt.Printf("Avg Response Size: %.2f KB\n", float64(bandwidth.AvgResponseSize)/1024)
		fmt.Printf("Requests/sec: %.2f\n", float64(b.N)/metrics.EndTime.Sub(metrics.StartTime).Seconds())
	}
}

// BenchmarkTemplateTransformation benchmarks template transformation overhead
func BenchmarkTemplateTransformation_Small(b *testing.B) {
	benchmarkTemplateTransformation(b, 1024)
}

func BenchmarkTemplateTransformation_Large(b *testing.B) {
	benchmarkTemplateTransformation(b, 100*1024)
}

func benchmarkTemplateTransformation(b *testing.B, payloadSize int) {
	engine := client.NewTemplateEngine()

	// Create test payload
	payload := map[string]interface{}{
		"data":      string(make([]byte, payloadSize)),
		"timestamp": time.Now().Unix(),
		"metadata": map[string]interface{}{
			"source": "benchmark",
			"type":   "test",
		},
	}

	// Template with transformations
	tmpl := `{
		"transformed": true,
		"original_size": {{ len .Payload.data }},
		"timestamp": "{{ now | formatTime "2006-01-02T15:04:05Z07:00" }}",
		"data": {{ .Payload | json }},
		"uppercase": "{{ .Payload.metadata.type | upper }}"
	}`

	testData := map[string]interface{}{
		"Payload": payload,
		"Event": map[string]interface{}{
			"Name": "test.event",
		},
	}

	metrics := captureResourceMetrics()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := engine.Execute(tmpl, testData)
		if err != nil {
			b.Fatalf("Template execution failed: %v", err)
		}
	}

	b.StopTimer()
	metrics.Stop()

	b.ReportMetric(float64(metrics.AllocatedMemory)/float64(b.N)/1024, "KB/op")

	if !testing.Short() {
		metrics.Print()
	}
}

// BenchmarkCachePerformance benchmarks template cache hit/miss performance
func BenchmarkCachePerformance_Hits(b *testing.B) {
	engine := client.NewTemplateEngine()

	tmpl := `{"transformed": {{ .value | json }}}`
	data := map[string]interface{}{"value": "test"}

	// Warm up cache
	engine.Execute(tmpl, data)

	metrics := captureResourceMetrics()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.Execute(tmpl, data)
	}

	b.StopTimer()
	metrics.Stop()

	size, maxSize := engine.CacheStats()
	b.ReportMetric(float64(size), "cached-templates")
	b.ReportMetric(float64(maxSize), "max-cache-size")
	b.ReportMetric(float64(metrics.AllocatedMemory)/float64(b.N), "bytes/op")
}

// BenchmarkMemoryProfile performs a comprehensive memory profile
func BenchmarkMemoryProfile_SustainedLoad(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping memory profile in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	webhookClient := client.NewWebhookClient(nil)
	defer webhookClient.Close()

	// Simulate sustained load
	duration := 10 * time.Second
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	done := make(chan bool)
	metrics := captureResourceMetrics()

	go func() {
		time.Sleep(duration)
		done <- true
	}()

	payloadSizes := []int{1024, 10 * 1024, 100 * 1024}
	sizeIndex := 0

	b.ResetTimer()
	requestCount := int64(0)

loop:
	for {
		select {
		case <-ticker.C:
			payload := make(map[string]interface{})
			payload["data"] = string(make([]byte, payloadSizes[sizeIndex]))
			payloadBytes, _ := json.Marshal(payload)

			req := &client.DeliveryRequest{
				WebhookID:  uuid.New(),
				DeliveryID: uuid.New().String(),
				EventID:    uuid.New(),
				URL:        server.URL,
				Method:     "POST",
				Payload:    payloadBytes,
				Timeout:    5 * time.Second,
			}

			webhookClient.Send(context.Background(), req)
			atomic.AddInt64(&requestCount, 1)
			sizeIndex = (sizeIndex + 1) % len(payloadSizes)

		case <-done:
			break loop
		}
	}

	b.StopTimer()
	metrics.Stop()

	// Force GC and get final stats
	runtime.GC()
	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)

	fmt.Printf("\n=== Sustained Load Profile ===\n")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Requests: %d\n", requestCount)
	fmt.Printf("Requests/sec: %.2f\n", float64(requestCount)/duration.Seconds())
	metrics.Print()
	fmt.Printf("Final Heap Size: %.2f MB\n", float64(finalMem.HeapInuse)/1024/1024)
	fmt.Printf("Final Goroutines: %d\n", runtime.NumGoroutine())
}
