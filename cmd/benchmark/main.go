// Command benchmark load-tests Sparrow in two modes.
//
//	-mode client  drives the outbound delivery client (internal/webhooks/client)
//	              against an in-process HTTP server. No database, no queue: it
//	              isolates the HTTP sending path.
//	-mode e2e     publishes events through the REST API of a running Sparrow
//	              and receives the deliveries on a local receiver, measuring the
//	              full pipeline: API ingest → Postgres → River → worker → HTTP.
//
// Both modes print a report and can write it as JSON with -json.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
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

// ---------------------------------------------------------------------------
// Shared: latency reservoir and rate limiter
// ---------------------------------------------------------------------------

// LatencySummary is the percentile report of one latency series.
type LatencySummary struct {
	Samples int           `json:"samples"`
	Avg     time.Duration `json:"avg"`
	P50     time.Duration `json:"p50"`
	P90     time.Duration `json:"p90"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
	P999    time.Duration `json:"p999"`
	Max     time.Duration `json:"max"`
}

// LatencyReservoir keeps a fixed-size uniform random sample of a latency
// stream (Algorithm R), so memory stays bounded however long the run is.
type LatencyReservoir struct {
	mu       sync.Mutex
	samples  []time.Duration
	capacity int
	seen     uint64
	max      time.Duration
}

// NewLatencyReservoir creates a reservoir holding at most capacity samples.
func NewLatencyReservoir(capacity int) *LatencyReservoir {
	return &LatencyReservoir{samples: make([]time.Duration, 0, capacity), capacity: capacity}
}

// Add records one latency.
func (r *LatencyReservoir) Add(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen++
	if d > r.max {
		r.max = d
	}
	if len(r.samples) < r.capacity {
		r.samples = append(r.samples, d)
		return
	}
	// Keep the new value with probability capacity/seen, replacing a
	// uniformly chosen existing sample.
	if j := rand.Uint64N(r.seen); j < uint64(r.capacity) {
		r.samples[j] = d
	}
}

// Summary computes the percentile report.
func (r *LatencyReservoir) Summary() LatencySummary {
	r.mu.Lock()
	sorted := make([]time.Duration, len(r.samples))
	copy(sorted, r.samples)
	seen, maxLat := r.seen, r.max
	r.mu.Unlock()

	if len(sorted) == 0 {
		return LatencySummary{}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	pct := func(p float64) time.Duration {
		idx := int(float64(len(sorted)) * p)
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		return sorted[idx]
	}
	var total time.Duration
	for _, s := range sorted {
		total += s
	}
	return LatencySummary{
		Samples: int(seen),
		Avg:     total / time.Duration(len(sorted)),
		P50:     pct(0.50),
		P90:     pct(0.90),
		P95:     pct(0.95),
		P99:     pct(0.99),
		P999:    pct(0.999),
		Max:     maxLat,
	}
}

// RateLimiter is a token bucket that paces callers at a fixed rate.
type RateLimiter struct {
	tokens chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewRateLimiter creates a token bucket refilled rps times per second with
// the given burst capacity.
func NewRateLimiter(rps int, burst int) *RateLimiter {
	if rps < 1 {
		rps = 1
	}
	if burst < 1 {
		burst = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{tokens: make(chan struct{}, burst), ctx: ctx, cancel: cancel}
	for i := 0; i < burst; i++ {
		rl.tokens <- struct{}{}
	}
	rl.wg.Add(1)
	go rl.refill(time.Second / time.Duration(rps))
	return rl
}

func (rl *RateLimiter) refill(interval time.Duration) {
	defer rl.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
			}
		case <-rl.ctx.Done():
			return
		}
	}
}

// Wait blocks until a token is available or ctx is done.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop halts the refill goroutine.
func (rl *RateLimiter) Stop() {
	rl.cancel()
	rl.wg.Wait()
}

// ---------------------------------------------------------------------------
// Client mode
// ---------------------------------------------------------------------------

// ClientConfig defines the client-mode load test.
type ClientConfig struct {
	Duration      time.Duration `json:"duration"`
	TargetRPS     int           `json:"target_rps"`
	PayloadSizeKB int           `json:"payload_kb"`
	Concurrency   int           `json:"concurrency"`
	TargetURL     string        `json:"target_url"`
}

// ClientResults is the client-mode report.
type ClientResults struct {
	Config          ClientConfig   `json:"config"`
	Duration        time.Duration  `json:"duration"`
	TotalRequests   int64          `json:"total_requests"`
	SuccessfulReqs  int64          `json:"successful"`
	FailedReqs      int64          `json:"failed"`
	ActualRPS       float64        `json:"actual_rps"`
	BytesSent       int64          `json:"bytes_sent"`
	BytesReceived   int64          `json:"bytes_received"`
	Latency         LatencySummary `json:"latency"`
	PeakHeapMB      float64        `json:"peak_heap_mb"`
	PeakGoroutines  int            `json:"peak_goroutines"`
	FirstError      string         `json:"first_error,omitempty"`
	ServerConnsSeen int64          `json:"server_connections_seen"`
	ResourceUsage   *ResourceUsage `json:"resource_usage,omitempty"`
}

type clientTester struct {
	cfg     ClientConfig
	client  *client.WebhookClient
	payload []byte
	target  string
	server  *httptest.Server

	total, ok, failed   atomic.Int64
	bytesSent, bytesRcv atomic.Int64
	serverConns         atomic.Int64
	firstErr            sync.Once
	firstErrMsg         string
	latency             *LatencyReservoir
	peakHeapMB          float64
	peakGoroutines      int
}

func newClientTester(cfg ClientConfig) *clientTester {
	t := &clientTester{
		cfg:     cfg,
		latency: NewLatencyReservoir(20000),
		client: client.NewWebhookClient(&client.Config{
			Timeout:         30 * time.Second,
			MaxIdleConns:    cfg.Concurrency * 2,
			MaxConnsPerHost: cfg.Concurrency * 2,
			IdleConnTimeout: 90 * time.Second,
			// The in-process test server listens on loopback, which the
			// client's SSRF guard blocks by default.
			AllowPrivateNetworks: true,
		}),
	}
	payload, _ := json.Marshal(map[string]any{
		"data":      string(make([]byte, cfg.PayloadSizeKB*1024)),
		"timestamp": time.Now().Unix(),
		"metadata":  map[string]any{"source": "load-test", "type": "benchmark"},
	})
	t.payload = payload

	t.target = cfg.TargetURL
	if t.target == "" {
		t.server = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"status":"success","id":%q}`, uuid.NewString())
		}))
		// Count accepted TCP connections: a healthy keep-alive client opens
		// about one per worker, a broken one opens one per request.
		t.server.Config.ConnState = func(_ net.Conn, st http.ConnState) {
			if st == http.StateNew {
				t.serverConns.Add(1)
			}
		}
		t.server.Start()
		t.target = t.server.URL
	}
	return t
}

func (t *clientTester) oneRequest(ctx context.Context) {
	req := &client.DeliveryRequest{
		WebhookID:  uuid.New(),
		DeliveryID: uuid.NewString(),
		EventID:    uuid.New(),
		URL:        t.target,
		Method:     http.MethodPost,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Payload:    t.payload,
		Timeout:    30 * time.Second, // production always sets a per-request timeout; keep that path hot
	}
	start := time.Now()
	resp, _, err := t.client.Send(ctx, req)
	lat := time.Since(start)

	t.total.Add(1)
	t.bytesSent.Add(int64(len(t.payload)))
	if err != nil {
		t.failed.Add(1)
		if ctx.Err() == nil {
			t.firstErr.Do(func() { t.firstErrMsg = err.Error(); log.Printf("first request failure: %v", err) })
		}
		return
	}
	n, _ := io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	t.bytesRcv.Add(n)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.ok.Add(1)
		t.latency.Add(lat)
	} else {
		t.failed.Add(1)
	}
}

func (t *clientTester) run(ctx context.Context) *ClientResults {
	ctx, cancel := context.WithTimeout(ctx, t.cfg.Duration)
	defer cancel()

	limiter := NewRateLimiter(t.cfg.TargetRPS, t.cfg.Concurrency)
	defer limiter.Stop()

	monCtx, stopMon := context.WithCancel(context.Background())
	monDone := make(chan struct{})
	go func() {
		defer close(monDone)
		tk := time.NewTicker(time.Second)
		defer tk.Stop()
		for {
			select {
			case <-tk.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if mb := float64(m.HeapInuse) / 1024 / 1024; mb > t.peakHeapMB {
					t.peakHeapMB = mb
				}
				if g := runtime.NumGoroutine(); g > t.peakGoroutines {
					t.peakGoroutines = g
				}
			case <-monCtx.Done():
				return
			}
		}
	}()

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < t.cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for limiter.Wait(ctx) == nil {
				t.oneRequest(ctx)
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	stopMon()
	<-monDone

	return &ClientResults{
		Config:          t.cfg,
		Duration:        elapsed,
		TotalRequests:   t.total.Load(),
		SuccessfulReqs:  t.ok.Load(),
		FailedReqs:      t.failed.Load(),
		ActualRPS:       float64(t.total.Load()) / elapsed.Seconds(),
		BytesSent:       t.bytesSent.Load(),
		BytesReceived:   t.bytesRcv.Load(),
		Latency:         t.latency.Summary(),
		PeakHeapMB:      t.peakHeapMB,
		PeakGoroutines:  t.peakGoroutines,
		FirstError:      t.firstErrMsg,
		ServerConnsSeen: t.serverConns.Load(),
	}
}

func (t *clientTester) close() {
	_ = t.client.Close()
	if t.server != nil {
		t.server.Close()
	}
}

// Print writes the client-mode report.
func (r *ClientResults) Print() {
	fmt.Printf("\n=== Client Load Test Results ===\n\n")
	if r.ResourceUsage != nil {
		fmt.Printf("Resource Utilization:\n")
		if r.ResourceUsage.SparrowPID > 0 {
			fmt.Printf("  Sparrow CPU:    avg %.1f%%, peak %.1f%%\n", r.ResourceUsage.SparrowCPUAvg, r.ResourceUsage.SparrowCPUPeak)
			fmt.Printf("  Sparrow RSS:    peak %.2f MB\n", r.ResourceUsage.SparrowPeakRSS)
		}
		if r.ResourceUsage.CPUMsPerDelivery > 0 {
			fmt.Printf("  Efficiency:     %.2f CPU-ms / request\n", r.ResourceUsage.CPUMsPerDelivery)
		}
		fmt.Println()
	}
	fmt.Printf("Duration: %v   target %d rps, %d workers, %d KB payload\n\n", r.Duration.Round(time.Millisecond), r.Config.TargetRPS, r.Config.Concurrency, r.Config.PayloadSizeKB)
	fmt.Printf("Requests:\n")
	fmt.Printf("  Total:           %d\n", r.TotalRequests)
	if r.TotalRequests > 0 {
		fmt.Printf("  Successful:      %d (%.2f%%)\n", r.SuccessfulReqs, 100*float64(r.SuccessfulReqs)/float64(r.TotalRequests))
		fmt.Printf("  Failed:          %d (%.2f%%)\n", r.FailedReqs, 100*float64(r.FailedReqs)/float64(r.TotalRequests))
	}
	if r.FirstError != "" {
		fmt.Printf("  First error:     %s\n", r.FirstError)
	}
	fmt.Printf("  Actual RPS:      %.2f\n", r.ActualRPS)
	if r.ServerConnsSeen > 0 {
		fmt.Printf("  TCP connections: %d opened at the test server (%.2f requests per connection)\n", r.ServerConnsSeen, float64(r.TotalRequests)/float64(r.ServerConnsSeen))
	}
	fmt.Println()
	printLatency("Latency (Send → response headers)", r.Latency)
	secs := r.Duration.Seconds()
	fmt.Printf("Bandwidth:\n")
	fmt.Printf("  Sent:            %.2f MB (%.2f MB/s)\n", float64(r.BytesSent)/1024/1024, float64(r.BytesSent)/1024/1024/secs)
	fmt.Printf("  Received:        %.2f MB (%.2f MB/s)\n\n", float64(r.BytesReceived)/1024/1024, float64(r.BytesReceived)/1024/1024/secs)
	fmt.Printf("Process:\n")
	fmt.Printf("  Peak heap:       %.2f MB\n", r.PeakHeapMB)
	fmt.Printf("  Peak goroutines: %d\n\n", r.PeakGoroutines)
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	mode := flag.String("mode", "client", "client (delivery client vs in-process server) or e2e (REST API → running Sparrow → local receiver)")
	duration := flag.Duration("duration", time.Minute, "Publish/request window")
	rps := flag.Int("rps", 100, "Target requests (or events) per second")
	payloadKB := flag.Int("payload", 10, "Payload size in KB")
	concurrency := flag.Int("concurrency", 10, "Concurrent workers (client mode) or publishers (e2e mode)")
	jsonOut := flag.String("json", "", "Write the report as JSON to this file")

	// client mode
	targetURL := flag.String("url", "", "client mode: target URL (empty = in-process test server)")

	// resource sampling flags
	sparrowPID := flag.Int("sparrow-pid", 0, "PID of target Sparrow process to sample CPU % and RSS")
	postgresPID := flag.Int("postgres-pid", 0, "PID of Postgres process to sample CPU %")
	postgresDSN := flag.String("postgres-dsn", "", "Postgres DSN connection string to sample WAL bytes")

	// e2e mode
	sparrowURL := flag.String("sparrow-url", "http://localhost:8080", "e2e mode: base URL of the running Sparrow")
	apiKey := flag.String("api-key", os.Getenv("SPARROW_API_KEY"), "e2e mode: X-API-Key for Sparrow (default $SPARROW_API_KEY)")
	consumer := flag.String("consumer", "bench", "e2e mode: consumer id to publish under")
	eventName := flag.String("event", "bench.event", "e2e mode: event type name prefix (a unique suffix is added per run)")
	webhooks := flag.Int("webhooks", 1, "e2e mode: number of event types to spread the events over (each with its own receiver URL)")
	subscribers := flag.Int("subscribers", 1, "e2e mode: webhooks subscribed to each event type (fan-out); typical is 1 or 2")
	burst := flag.Int("burst", 0, "e2e mode: publish exactly this many events as fast as possible, then wait for the drain (ignores -duration and -rps)")
	receiverAddr := flag.String("receiver-addr", "127.0.0.1:0", "e2e mode: local receiver listen address")
	receiverURL := flag.String("receiver-url", "", "e2e mode: URL Sparrow should call (default: the receiver listen address; set when Sparrow runs in a container)")
	receiverDelay := flag.Duration("receiver-delay", 0, "e2e mode: simulated processing time inside the receiver")
	drainTimeout := flag.Duration("drain-timeout", 2*time.Minute, "e2e mode: how long to wait for the backlog to drain after publishing stops")
	keepWebhook := flag.Bool("keep-webhook", false, "e2e mode: leave the benchmark webhook registered")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Printf("interrupt: shutting down")
		cancel()
	}()

	var report any
	resCfg := ResourceConfig{
		SparrowPID:  *sparrowPID,
		PostgresPID: *postgresPID,
		PostgresDSN: *postgresDSN,
	}

	switch *mode {
	case "client":
		cfg := ClientConfig{Duration: *duration, TargetRPS: *rps, PayloadSizeKB: *payloadKB, Concurrency: *concurrency, TargetURL: *targetURL}
		fmt.Printf("Client mode: %v at %d rps, %d workers, %d KB payload, target=%q\n", cfg.Duration, cfg.TargetRPS, cfg.Concurrency, cfg.PayloadSizeKB, cfg.TargetURL)
		sampler, err := NewResourceSampler(resCfg)
		if err != nil {
			log.Printf("resource sampler init: %v", err)
		}
		sampler.Start(ctx)
		t := newClientTester(cfg)
		res := t.run(ctx)
		t.close()
		res.ResourceUsage = sampler.Stop(res.SuccessfulReqs)
		res.Print()
		report = res
	case "e2e":
		cfg := E2EConfig{
			SparrowURL: *sparrowURL, APIKey: *apiKey, Consumer: *consumer, EventName: *eventName, Webhooks: *webhooks, Subscribers: *subscribers, Burst: *burst,
			ReceiverAddr: *receiverAddr, ReceiverURL: *receiverURL, ReceiverDelay: *receiverDelay,
			Duration: *duration, TargetRPS: *rps, Publishers: *concurrency, PayloadSizeKB: *payloadKB,
			DrainTimeout: *drainTimeout, KeepWebhook: *keepWebhook, ResourceConfig: resCfg,
		}
		res, err := NewE2ERunner(cfg).Run(ctx)
		if err != nil {
			log.Fatalf("e2e benchmark: %v", err)
		}
		res.Print()
		report = res
	default:
		log.Fatalf("unknown -mode %q (client|e2e)", *mode)
	}

	if *jsonOut != "" {
		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Fatalf("marshal report: %v", err)
		}
		if err := os.WriteFile(*jsonOut, b, 0o644); err != nil {
			log.Fatalf("write %s: %v", *jsonOut, err)
		}
		log.Printf("report written to %s", *jsonOut)
	}
}
