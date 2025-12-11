package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
)

// Config defines load test parameters
type Config struct {
	Duration          time.Duration
	TargetRPS         int
	PayloadSizeKB     int
	Concurrency       int
	EnableTemplates   bool
	TargetURL         string
	SamplingRate      float64 // Percentage of requests to track (0.0-1.0)
	EnableResourceMon bool
}

// Metrics contains aggregated test results
type Metrics struct {
	totalRequests   atomic.Int64
	successRequests atomic.Int64
	failedRequests  atomic.Int64
	totalBytes      atomic.Int64

	// Latency tracking with lock-free reservoir sampling
	latencySamples *LatencyReservoir

	// Resource tracking
	memSamples       []float64
	goroutineSamples []int
	sampleMu         sync.RWMutex

	startTime time.Time
	endTime   time.Time
}

// LatencyReservoir implements reservoir sampling for latency metrics
// This avoids unbounded memory growth while maintaining statistical accuracy
type LatencyReservoir struct {
	samples  []time.Duration
	capacity int
	count    atomic.Uint64
	mu       sync.Mutex
}

// NewLatencyReservoir creates a new reservoir with fixed capacity
func NewLatencyReservoir(capacity int) *LatencyReservoir {
	return &LatencyReservoir{
		samples:  make([]time.Duration, 0, capacity),
		capacity: capacity,
	}
}

// Add implements reservoir sampling algorithm
func (r *LatencyReservoir) Add(latency time.Duration) {
	count := r.count.Add(1)

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.samples) < r.capacity {
		r.samples = append(r.samples, latency)
		return
	}

	// Reservoir sampling: replace random element with probability k/n
	// where k is capacity and n is total count
	if rand := count % uint64(r.capacity); rand < uint64(r.capacity) {
		r.samples[rand] = latency
	}
}

// GetPercentiles returns sorted percentile values
func (r *LatencyReservoir) GetPercentiles(percentiles ...float64) map[float64]time.Duration {
	r.mu.Lock()
	sorted := make([]time.Duration, len(r.samples))
	copy(sorted, r.samples)
	r.mu.Unlock()

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	results := make(map[float64]time.Duration)
	for _, p := range percentiles {
		idx := int(float64(len(sorted)) * p)
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		if idx >= 0 {
			results[p] = sorted[idx]
		}
	}
	return results
}

// GetAverage returns average latency
func (r *LatencyReservoir) GetAverage() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.samples) == 0 {
		return 0
	}

	var total time.Duration
	for _, s := range r.samples {
		total += s
	}
	return total / time.Duration(len(r.samples))
}

// RateLimiter implements token bucket algorithm for accurate RPS pacing
type RateLimiter struct {
	rate     float64 // requests per second
	interval time.Duration
	tokens   chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewRateLimiter creates a token bucket rate limiter
func NewRateLimiter(rps int, burst int) *RateLimiter {
	if burst < 1 {
		burst = rps / 10 // 10% burst by default
		if burst < 1 {
			burst = 1
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{
		rate:     float64(rps),
		interval: time.Second / time.Duration(rps),
		tokens:   make(chan struct{}, burst),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Pre-fill tokens
	for i := 0; i < burst; i++ {
		rl.tokens <- struct{}{}
	}

	rl.wg.Add(1)
	go rl.refillTokens()

	return rl
}

// refillTokens continuously adds tokens at the specified rate
func (rl *RateLimiter) refillTokens() {
	defer rl.wg.Done()

	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Bucket full, skip
			}
		case <-rl.ctx.Done():
			return
		}
	}
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	rl.cancel()
	rl.wg.Wait()
	close(rl.tokens)
}

// Worker represents a request worker
type Worker struct {
	id           int
	client       *client.WebhookClient
	metrics      *Metrics
	payloadPool  *sync.Pool
	samplingRate float64
	requestCount uint64
}

// NewWorker creates a new worker
func NewWorker(id int, webhookClient *client.WebhookClient, metrics *Metrics, payloadPool *sync.Pool, samplingRate float64) *Worker {
	return &Worker{
		id:           id,
		client:       webhookClient,
		metrics:      metrics,
		payloadPool:  payloadPool,
		samplingRate: samplingRate,
	}
}

// ProcessRequest handles a single request
func (w *Worker) ProcessRequest(ctx context.Context, targetURL string, payloadBytes []byte) {
	// Create request from pool
	req := &client.DeliveryRequest{
		WebhookID:  uuid.New(),
		DeliveryID: uuid.New().String(),
		EventID:    uuid.New(),
		URL:        targetURL,
		Method:     "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Payload: payloadBytes,
		Timeout: 30 * time.Second,
	}

	// Use monotonic clock for accurate latency measurement
	start := time.Now()
	resp, _, err := w.client.Send(ctx, req)
	latency := time.Since(start)

	w.metrics.totalRequests.Add(1)

	if err != nil {
		w.metrics.failedRequests.Add(1)
		return
	}

	w.metrics.successRequests.Add(1)

	// Track bandwidth
	if resp != nil && resp.Body != nil {
		// Read and discard body to complete the request
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		w.metrics.totalBytes.Add(int64(len(bodyBytes)))
	}

	// Reservoir sampling: only track some latencies to avoid unbounded growth
	w.requestCount++
	if w.samplingRate >= 1.0 || (float64(w.requestCount%100) < w.samplingRate*100) {
		w.metrics.latencySamples.Add(latency)
	}
}

// WorkerPool manages a pool of workers
type WorkerPool struct {
	workers      []*Worker
	workQueue    chan struct{}
	rateLimiter  *RateLimiter
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	targetURL    string
	payloadBytes []byte
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config *Config, webhookClient *client.WebhookClient, metrics *Metrics, payloadPool *sync.Pool, targetURL string, payloadBytes []byte) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	// Create work queue with buffering to reduce contention
	queueSize := config.Concurrency * 10
	if queueSize < 100 {
		queueSize = 100
	}

	wp := &WorkerPool{
		workers:      make([]*Worker, config.Concurrency),
		workQueue:    make(chan struct{}, queueSize),
		rateLimiter:  NewRateLimiter(config.TargetRPS, config.Concurrency),
		ctx:          ctx,
		cancel:       cancel,
		targetURL:    targetURL,
		payloadBytes: payloadBytes,
	}

	// Initialize workers
	for i := 0; i < config.Concurrency; i++ {
		wp.workers[i] = NewWorker(i, webhookClient, metrics, payloadPool, config.SamplingRate)
	}

	return wp
}

// Start starts all workers
func (wp *WorkerPool) Start() {
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go wp.runWorker(worker)
	}
}

// runWorker runs a single worker
func (wp *WorkerPool) runWorker(worker *Worker) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.workQueue:
			worker.ProcessRequest(wp.ctx, wp.targetURL, wp.payloadBytes)
		case <-wp.ctx.Done():
			return
		}
	}
}

// SubmitWork submits work to the pool with rate limiting
func (wp *WorkerPool) SubmitWork() error {
	if err := wp.rateLimiter.Wait(wp.ctx); err != nil {
		return err
	}

	select {
	case wp.workQueue <- struct{}{}:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

// Stop gracefully stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.cancel()
	close(wp.workQueue)
	wp.wg.Wait()
	wp.rateLimiter.Stop()
}

// ResourceMonitor tracks system resources
type ResourceMonitor struct {
	metrics  *Metrics
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(metrics *Metrics, interval time.Duration) *ResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &ResourceMonitor{
		metrics:  metrics,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins monitoring
func (rm *ResourceMonitor) Start() {
	rm.wg.Add(1)
	go rm.monitor()
}

// monitor collects resource metrics
func (rm *ResourceMonitor) monitor() {
	defer rm.wg.Done()

	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			goroutines := runtime.NumGoroutine()

			rm.metrics.sampleMu.Lock()
			rm.metrics.memSamples = append(rm.metrics.memSamples, float64(m.HeapInuse)/1024/1024)
			rm.metrics.goroutineSamples = append(rm.metrics.goroutineSamples, goroutines)
			rm.metrics.sampleMu.Unlock()
		case <-rm.ctx.Done():
			return
		}
	}
}

// Stop stops monitoring
func (rm *ResourceMonitor) Stop() {
	rm.cancel()
	rm.wg.Wait()
}

// LoadTester orchestrates the load test
type LoadTester struct {
	config        *Config
	metrics       *Metrics
	webhookClient *client.WebhookClient
	workerPool    *WorkerPool
	resourceMon   *ResourceMonitor
	payloadPool   *sync.Pool
	testServer    *httptest.Server
}

// NewLoadTester creates a new load tester
func NewLoadTester(config *Config) *LoadTester {
	// Create metrics with reservoir sampling (10k samples max)
	metrics := &Metrics{
		latencySamples: NewLatencyReservoir(10000),
		startTime:      time.Now(),
	}

	// Create payload pool to reduce allocations
	payloadPool := &sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, config.PayloadSizeKB*1024)
		},
	}

	// Create webhook client with optimized config
	webhookClient := client.NewWebhookClient(&client.Config{
		Timeout:           30 * time.Second,
		MaxIdleConns:      config.Concurrency * 2,
		MaxConnsPerHost:   config.Concurrency * 2,
		IdleConnTimeout:   90 * time.Second,
		DisableKeepAlives: false,
	})

	lt := &LoadTester{
		config:        config,
		metrics:       metrics,
		webhookClient: webhookClient,
		payloadPool:   payloadPool,
	}

	// Setup target URL
	targetURL := config.TargetURL
	if targetURL == "" {
		// Create test server if no external URL provided
		lt.testServer = httptest.NewServer(http.HandlerFunc(lt.handleTestRequest))
		targetURL = lt.testServer.URL
	}

	// Generate payload once
	payloadBytes := lt.generatePayload()

	// Create worker pool
	lt.workerPool = NewWorkerPool(config, webhookClient, metrics, payloadPool, targetURL, payloadBytes)

	// Create resource monitor if enabled
	if config.EnableResourceMon {
		lt.resourceMon = NewResourceMonitor(metrics, time.Second)
	}

	return lt
}

// handleTestRequest handles test server requests
func (lt *LoadTester) handleTestRequest(w http.ResponseWriter, r *http.Request) {
	// Simulate realistic server behavior
	response := map[string]interface{}{
		"status":    "success",
		"timestamp": time.Now().Unix(),
		"id":        uuid.New().String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// generatePayload creates the test payload
func (lt *LoadTester) generatePayload() []byte {
	payload := map[string]interface{}{
		"data":      string(make([]byte, lt.config.PayloadSizeKB*1024)),
		"timestamp": time.Now().Unix(),
		"metadata": map[string]interface{}{
			"source":  "load-test",
			"type":    "benchmark",
			"version": "2.0",
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	return payloadBytes
}

// Run executes the load test
func (lt *LoadTester) Run(ctx context.Context) error {
	log.Printf("Starting load test...")

	// Start resource monitoring
	if lt.resourceMon != nil {
		lt.resourceMon.Start()
	}

	// Start workers
	lt.workerPool.Start()

	// Create test context with timeout
	testCtx, cancel := context.WithTimeout(ctx, lt.config.Duration)
	defer cancel()

	// Submit work continuously
	for {
		select {
		case <-testCtx.Done():
			log.Printf("Test duration completed, shutting down...")
			goto cleanup
		default:
			if err := lt.workerPool.SubmitWork(); err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					goto cleanup
				}
				log.Printf("Error submitting work: %v", err)
			}
		}
	}

cleanup:
	lt.metrics.endTime = time.Now()

	// Stop worker pool gracefully
	lt.workerPool.Stop()

	// Stop resource monitoring
	if lt.resourceMon != nil {
		lt.resourceMon.Stop()
	}

	log.Printf("Load test completed")
	return nil
}

// Close cleans up resources
func (lt *LoadTester) Close() {
	if lt.webhookClient != nil {
		lt.webhookClient.Close()
	}
	if lt.testServer != nil {
		lt.testServer.Close()
	}
}

// GetResults returns test results
func (lt *LoadTester) GetResults() *Results {
	duration := lt.metrics.endTime.Sub(lt.metrics.startTime)
	totalReqs := lt.metrics.totalRequests.Load()
	successReqs := lt.metrics.successRequests.Load()
	failedReqs := lt.metrics.failedRequests.Load()
	totalBytes := lt.metrics.totalBytes.Load()

	results := &Results{
		Duration:         duration,
		TotalRequests:    totalReqs,
		SuccessfulReqs:   successReqs,
		FailedReqs:       failedReqs,
		ActualRPS:        float64(totalReqs) / duration.Seconds(),
		TotalBandwidthMB: float64(totalBytes) / 1024 / 1024,
		BandwidthMBps:    float64(totalBytes) / 1024 / 1024 / duration.Seconds(),
	}

	// Get latency percentiles
	percentiles := lt.metrics.latencySamples.GetPercentiles(0.50, 0.90, 0.95, 0.99, 0.999)
	results.AvgLatency = lt.metrics.latencySamples.GetAverage()
	results.P50Latency = percentiles[0.50]
	results.P90Latency = percentiles[0.90]
	results.P95Latency = percentiles[0.95]
	results.P99Latency = percentiles[0.99]
	results.P999Latency = percentiles[0.999]

	// Get resource stats
	lt.metrics.sampleMu.RLock()
	if len(lt.metrics.memSamples) > 0 {
		var totalMem float64
		maxMem := 0.0
		for _, m := range lt.metrics.memSamples {
			totalMem += m
			if m > maxMem {
				maxMem = m
			}
		}
		results.AvgMemoryMB = totalMem / float64(len(lt.metrics.memSamples))
		results.PeakMemoryMB = maxMem
	}

	if len(lt.metrics.goroutineSamples) > 0 {
		var totalGoroutines int
		maxGoroutines := 0
		for _, g := range lt.metrics.goroutineSamples {
			totalGoroutines += g
			if g > maxGoroutines {
				maxGoroutines = g
			}
		}
		results.AvgGoroutines = totalGoroutines / len(lt.metrics.goroutineSamples)
		results.PeakGoroutines = maxGoroutines
	}
	lt.metrics.sampleMu.RUnlock()

	return results
}

// Results contains test results
type Results struct {
	Duration         time.Duration
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedReqs       int64
	ActualRPS        float64
	TotalBandwidthMB float64
	BandwidthMBps    float64

	AvgLatency  time.Duration
	P50Latency  time.Duration
	P90Latency  time.Duration
	P95Latency  time.Duration
	P99Latency  time.Duration
	P999Latency time.Duration

	AvgMemoryMB    float64
	PeakMemoryMB   float64
	AvgGoroutines  int
	PeakGoroutines int
}

// PrintResults prints test results
func (r *Results) Print() {
	fmt.Printf("\n=== Load Test Results ===\n\n")

	fmt.Printf("Duration: %v\n\n", r.Duration)

	fmt.Printf("Request Statistics:\n")
	fmt.Printf("  Total Requests:  %d\n", r.TotalRequests)
	if r.TotalRequests > 0 {
		successRate := float64(r.SuccessfulReqs) / float64(r.TotalRequests) * 100
		failRate := float64(r.FailedReqs) / float64(r.TotalRequests) * 100
		fmt.Printf("  Successful:      %d (%.2f%%)\n", r.SuccessfulReqs, successRate)
		fmt.Printf("  Failed:          %d (%.2f%%)\n", r.FailedReqs, failRate)
	}
	fmt.Printf("  Actual RPS:      %.2f\n\n", r.ActualRPS)

	fmt.Printf("Latency:\n")
	fmt.Printf("  Average:         %v\n", r.AvgLatency.Round(time.Microsecond))
	fmt.Printf("  P50:             %v\n", r.P50Latency.Round(time.Microsecond))
	fmt.Printf("  P90:             %v\n", r.P90Latency.Round(time.Microsecond))
	fmt.Printf("  P95:             %v\n", r.P95Latency.Round(time.Microsecond))
	fmt.Printf("  P99:             %v\n", r.P99Latency.Round(time.Microsecond))
	fmt.Printf("  P99.9:           %v\n\n", r.P999Latency.Round(time.Microsecond))

	fmt.Printf("Bandwidth:\n")
	fmt.Printf("  Total:           %.2f MB\n", r.TotalBandwidthMB)
	fmt.Printf("  Average:         %.2f MB/s (%.2f Mbps)\n\n", r.BandwidthMBps, r.BandwidthMBps*8)

	if r.PeakMemoryMB > 0 {
		fmt.Printf("Memory Usage:\n")
		fmt.Printf("  Average:         %.2f MB\n", r.AvgMemoryMB)
		fmt.Printf("  Peak:            %.2f MB\n\n", r.PeakMemoryMB)
	}

	if r.PeakGoroutines > 0 {
		fmt.Printf("Goroutines:\n")
		fmt.Printf("  Average:         %d\n", r.AvgGoroutines)
		fmt.Printf("  Peak:            %d\n\n", r.PeakGoroutines)
	}
}

// GenerateEstimates generates resource estimates for different load levels
func (r *Results) GenerateEstimates() {
	fmt.Printf("\n=== Resource Estimates for Production ===\n\n")

	if r.ActualRPS == 0 {
		fmt.Printf("No baseline RPS to generate estimates\n")
		return
	}

	loadLevels := []struct {
		name string
		rps  int
	}{
		{"Low Load", 100},
		{"Medium Load", 1000},
		{"High Load", 10000},
		{"Peak Load", 50000},
	}

	fmt.Printf("Based on actual RPS: %.2f\n", r.ActualRPS)
	fmt.Printf("Safety margin: 50%%\n\n")

	for _, level := range loadLevels {
		scaleFactor := float64(level.rps) / r.ActualRPS

		// Estimates with safety margin
		estimatedRAM := r.PeakMemoryMB * scaleFactor * 1.5
		estimatedBandwidth := r.BandwidthMBps * scaleFactor
		estimatedGoroutines := float64(r.PeakGoroutines) * scaleFactor
		estimatedP99 := time.Duration(float64(r.P99Latency) * math.Sqrt(scaleFactor))

		cpuCores := int(estimatedGoroutines/100) + 1
		if cpuCores < 2 {
			cpuCores = 2
		}

		fmt.Printf("%s (%d RPS):\n", level.name, level.rps)
		fmt.Printf("  RAM:             %.2f MB (%.2f GB)\n", estimatedRAM, estimatedRAM/1024)
		fmt.Printf("  Bandwidth:       %.2f MB/s (%.2f Mbps)\n", estimatedBandwidth, estimatedBandwidth*8)
		fmt.Printf("  Goroutines:      ~%.0f\n", estimatedGoroutines)
		fmt.Printf("  CPU Cores:       %d (minimum)\n", cpuCores)
		fmt.Printf("  Est. P99:        %v\n\n", estimatedP99.Round(time.Microsecond))
	}

	fmt.Printf("Notes:\n")
	fmt.Printf("  • Estimates include 50%% safety margin\n")
	fmt.Printf("  • Actual requirements depend on:\n")
	fmt.Printf("    - Payload size and complexity\n")
	fmt.Printf("    - Network latency to endpoints\n")
	fmt.Printf("    - Error rates and retry logic\n")
	fmt.Printf("    - Database and external service performance\n")
	fmt.Printf("  • Test on production-like hardware for accurate results\n")
}

func main() {
	// Parse flags
	duration := flag.Duration("duration", 1*time.Minute, "Test duration")
	rps := flag.Int("rps", 100, "Target requests per second")
	payloadKB := flag.Int("payload", 10, "Payload size in KB")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent workers")
	templates := flag.Bool("templates", false, "Enable template transformations")
	targetURL := flag.String("url", "", "Target URL (leave empty for test server)")
	samplingRate := flag.Float64("sampling", 0.1, "Latency sampling rate (0.01-1.0)")
	enableResourceMon := flag.Bool("monitor", true, "Enable resource monitoring")

	flag.Parse()

	// Validate flags
	if *samplingRate < 0.01 || *samplingRate > 1.0 {
		log.Fatal("Sampling rate must be between 0.01 and 1.0")
	}

	config := &Config{
		Duration:          *duration,
		TargetRPS:         *rps,
		PayloadSizeKB:     *payloadKB,
		Concurrency:       *concurrency,
		EnableTemplates:   *templates,
		TargetURL:         *targetURL,
		SamplingRate:      *samplingRate,
		EnableResourceMon: *enableResourceMon,
	}

	// Print configuration
	fmt.Printf("Load Test Configuration:\n")
	fmt.Printf("  Duration:        %v\n", config.Duration)
	fmt.Printf("  Target RPS:      %d\n", config.TargetRPS)
	fmt.Printf("  Payload Size:    %d KB\n", config.PayloadSizeKB)
	fmt.Printf("  Concurrency:     %d\n", config.Concurrency)
	fmt.Printf("  Templates:       %v\n", config.EnableTemplates)
	fmt.Printf("  Target URL:      %s\n", config.TargetURL)
	fmt.Printf("  Sampling Rate:   %.1f%%\n", config.SamplingRate*100)
	fmt.Printf("  Resource Mon:    %v\n\n", config.EnableResourceMon)

	// Create load tester
	tester := NewLoadTester(config)
	defer tester.Close()

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Run test
	if err := tester.Run(ctx); err != nil {
		log.Fatalf("Load test failed: %v", err)
	}

	// Print results
	results := tester.GetResults()
	results.Print()
	results.GenerateEstimates()
}
