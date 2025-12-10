package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
)

// LoadTestConfig defines load test parameters
type LoadTestConfig struct {
	Duration          time.Duration
	RequestsPerSecond int
	PayloadSizeKB     int
	Concurrency       int
	EnableTemplates   bool
}

// LoadTestResults contains test results and resource metrics
type LoadTestResults struct {
	TotalRequests     int64
	SuccessfulReqs    int64
	FailedReqs        int64
	TotalBandwidthMB  float64
	AvgResponseTimeMs float64
	P50ResponseTimeMs float64
	P95ResponseTimeMs float64
	P99ResponseTimeMs float64

	PeakMemoryMB   float64
	AvgMemoryMB    float64
	PeakGoroutines int
	AvgGoroutines  int

	RequestsPerSecond float64
	BandwidthMBps     float64
}

func main() {
	// Parse command line flags
	duration := flag.Duration("duration", 1*time.Minute, "Test duration")
	rps := flag.Int("rps", 100, "Target requests per second")
	payloadKB := flag.Int("payload", 10, "Payload size in KB")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent workers")
	templates := flag.Bool("templates", false, "Enable template transformations")

	flag.Parse()

	config := &LoadTestConfig{
		Duration:          *duration,
		RequestsPerSecond: *rps,
		PayloadSizeKB:     *payloadKB,
		Concurrency:       *concurrency,
		EnableTemplates:   *templates,
	}

	fmt.Printf("Starting load test with configuration:\n")
	fmt.Printf("  Duration: %v\n", config.Duration)
	fmt.Printf("  Target RPS: %d\n", config.RequestsPerSecond)
	fmt.Printf("  Payload Size: %d KB\n", config.PayloadSizeKB)
	fmt.Printf("  Concurrency: %d\n", config.Concurrency)
	fmt.Printf("  Templates: %v\n\n", config.EnableTemplates)

	results := runLoadTest(config)

	printResults(results, config)
	generateResourceEstimates(results, config)
}

func runLoadTest(config *LoadTestConfig) *LoadTestResults {
	results := &LoadTestResults{}

	var requestCount int64
	var successCount int64
	var failCount int64
	var totalBandwidth int64
	var responseTimes []time.Duration
	var responseTimesMu sync.Mutex

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&totalBandwidth, r.ContentLength)
		response := []byte(`{"status":"success","timestamp":` + fmt.Sprintf("%d", time.Now().Unix()) + `}`)
		atomic.AddInt64(&totalBandwidth, int64(len(response)))
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}))
	defer server.Close()

	// Create webhook client
	webhookClient := client.NewWebhookClient(nil)
	defer webhookClient.Close()

	// Track resource usage
	memSamples := []float64{}
	goroutineSamples := []int{}
	var sampleMu sync.Mutex

	// Start resource monitoring
	stopMonitoring := make(chan bool)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				goroutines := runtime.NumGoroutine()

				sampleMu.Lock()
				memSamples = append(memSamples, float64(m.HeapInuse)/1024/1024)
				goroutineSamples = append(goroutineSamples, goroutines)
				sampleMu.Unlock()
			case <-stopMonitoring:
				return
			}
		}
	}()

	// Generate test payload
	payload := make(map[string]any)
	payload["data"] = string(make([]byte, config.PayloadSizeKB*1024))
	payload["timestamp"] = time.Now().Unix()
	payload["metadata"] = map[string]any{
		"source": "load-test",
		"type":   "benchmark",
	}
	payloadBytes, _ := json.Marshal(payload)

	// Start load test
	startTime := time.Now()
	ticker := time.NewTicker(time.Second / time.Duration(config.RequestsPerSecond))
	defer ticker.Stop()

	done := make(chan bool)
	go func() {
		time.Sleep(config.Duration)
		done <- true
	}()

	var wg sync.WaitGroup
	requestChan := make(chan bool, config.Concurrency*10)

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestChan {
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

				start := time.Now()
				_, _, err := webhookClient.Send(context.Background(), req)
				duration := time.Since(start)

				atomic.AddInt64(&requestCount, 1)
				if err != nil {
					atomic.AddInt64(&failCount, 1)
					log.Printf("Request failed: %v", err)
				} else {
					atomic.AddInt64(&successCount, 1)
				}

				responseTimesMu.Lock()
				responseTimes = append(responseTimes, duration)
				responseTimesMu.Unlock()
			}
		}()
	}

	// Send requests
loop:
	for {
		select {
		case <-ticker.C:
			select {
			case requestChan <- true:
			default:
				// Channel full, skip this tick
			}
		case <-done:
			break loop
		}
	}

	close(requestChan)
	wg.Wait()
	close(stopMonitoring)

	endTime := time.Now()
	actualDuration := endTime.Sub(startTime)

	// Calculate results
	results.TotalRequests = requestCount
	results.SuccessfulReqs = successCount
	results.FailedReqs = failCount
	results.TotalBandwidthMB = float64(totalBandwidth) / 1024 / 1024
	results.RequestsPerSecond = float64(requestCount) / actualDuration.Seconds()
	results.BandwidthMBps = results.TotalBandwidthMB / actualDuration.Seconds()

	// Calculate response time percentiles
	if len(responseTimes) > 0 {
		sort.Slice(responseTimes, func(i, j int) bool {
			return responseTimes[i] < responseTimes[j]
		})

		total := time.Duration(0)
		for _, rt := range responseTimes {
			total += rt
		}
		results.AvgResponseTimeMs = float64(total.Milliseconds()) / float64(len(responseTimes))

		p50Idx := int(float64(len(responseTimes)) * 0.50)
		p95Idx := int(float64(len(responseTimes)) * 0.95)
		p99Idx := int(float64(len(responseTimes)) * 0.99)

		if p50Idx < len(responseTimes) {
			results.P50ResponseTimeMs = float64(responseTimes[p50Idx].Milliseconds())
		}
		if p95Idx < len(responseTimes) {
			results.P95ResponseTimeMs = float64(responseTimes[p95Idx].Milliseconds())
		}
		if p99Idx < len(responseTimes) {
			results.P99ResponseTimeMs = float64(responseTimes[p99Idx].Milliseconds())
		}
	}

	// Calculate memory stats
	if len(memSamples) > 0 {
		var totalMem float64
		maxMem := 0.0
		for _, m := range memSamples {
			totalMem += m
			if m > maxMem {
				maxMem = m
			}
		}
		results.AvgMemoryMB = totalMem / float64(len(memSamples))
		results.PeakMemoryMB = maxMem
	}

	// Calculate goroutine stats
	if len(goroutineSamples) > 0 {
		var totalGoroutines int
		maxGoroutines := 0
		for _, g := range goroutineSamples {
			totalGoroutines += g
			if g > maxGoroutines {
				maxGoroutines = g
			}
		}
		results.AvgGoroutines = totalGoroutines / len(goroutineSamples)
		results.PeakGoroutines = maxGoroutines
	}

	return results
}

func printResults(results *LoadTestResults, config *LoadTestConfig) {
	fmt.Printf("\n=== Load Test Results ===\n\n")

	fmt.Printf("Request Statistics:\n")
	fmt.Printf("  Total Requests: %d\n", results.TotalRequests)
	if results.TotalRequests > 0 {
		fmt.Printf("  Successful: %d (%.2f%%)\n", results.SuccessfulReqs,
			float64(results.SuccessfulReqs)/float64(results.TotalRequests)*100)
		fmt.Printf("  Failed: %d (%.2f%%)\n", results.FailedReqs,
			float64(results.FailedReqs)/float64(results.TotalRequests)*100)
	}
	fmt.Printf("  Actual RPS: %.2f\n\n", results.RequestsPerSecond)

	fmt.Printf("Response Time:\n")
	fmt.Printf("  Average: %.2f ms\n", results.AvgResponseTimeMs)
	fmt.Printf("  P50: %.2f ms\n", results.P50ResponseTimeMs)
	fmt.Printf("  P95: %.2f ms\n", results.P95ResponseTimeMs)
	fmt.Printf("  P99: %.2f ms\n\n", results.P99ResponseTimeMs)

	fmt.Printf("Bandwidth:\n")
	fmt.Printf("  Total: %.2f MB\n", results.TotalBandwidthMB)
	fmt.Printf("  Average: %.2f MB/s\n\n", results.BandwidthMBps)

	fmt.Printf("Memory Usage:\n")
	fmt.Printf("  Average: %.2f MB\n", results.AvgMemoryMB)
	fmt.Printf("  Peak: %.2f MB\n\n", results.PeakMemoryMB)

	fmt.Printf("Goroutines:\n")
	fmt.Printf("  Average: %d\n", results.AvgGoroutines)
	fmt.Printf("  Peak: %d\n\n", results.PeakGoroutines)
}

func generateResourceEstimates(results *LoadTestResults, config *LoadTestConfig) {
	fmt.Printf("\n=== Resource Estimates for Production ===\n\n")

	// Estimate for different load levels
	loadLevels := []struct {
		name string
		rps  int
	}{
		{"Low Load (100 RPS)", 100},
		{"Medium Load (1,000 RPS)", 1000},
		{"High Load (10,000 RPS)", 10000},
		{"Peak Load (50,000 RPS)", 50000},
	}

	for _, level := range loadLevels {
		if results.RequestsPerSecond == 0 {
			continue
		}
		scaleFactor := float64(level.rps) / results.RequestsPerSecond

		estimatedRAM := results.PeakMemoryMB * scaleFactor * 1.5 // 50% safety margin
		estimatedBandwidth := results.BandwidthMBps * scaleFactor
		estimatedGoroutines := float64(results.PeakGoroutines) * scaleFactor

		fmt.Printf("%s:\n", level.name)
		fmt.Printf("  Estimated RAM: %.2f MB (%.2f GB)\n", estimatedRAM, estimatedRAM/1024)
		fmt.Printf("  Estimated Bandwidth: %.2f MB/s (%.2f Mbps)\n",
			estimatedBandwidth, estimatedBandwidth*8)
		fmt.Printf("  Estimated Goroutines: %.0f\n", estimatedGoroutines)
		cpuCores := int(estimatedGoroutines/1000) + 1
		if cpuCores < 1 {
			cpuCores = 1
		}
		fmt.Printf("  Recommended CPU Cores: %d\n\n", cpuCores)
	}

	fmt.Printf("Notes:\n")
	fmt.Printf("  - Estimates include 50%% safety margin\n")
	fmt.Printf("  - Actual requirements may vary based on:\n")
	fmt.Printf("    * Payload complexity and size\n")
	fmt.Printf("    * Template transformation overhead\n")
	fmt.Printf("    * Database query performance\n")
	fmt.Printf("    * Network latency to webhook endpoints\n")
	fmt.Printf("    * Retry logic and failure rates\n")
}
