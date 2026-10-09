package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// E2EConfig drives the pipeline benchmark: events are published through the
// REST API of a running Sparrow and received by a local HTTP receiver, so the
// measured path is API ingest → Postgres → River → worker → HTTP delivery.
type E2EConfig struct {
	SparrowURL      string
	APIKey          string
	Consumer        string
	EventName       string
	Webhooks        int // event types; each gets its own receiver URL(s) and an equal share of the events
	Subscribers     int // webhooks per event type (fan-out); an event counts as delivered when all of them got it
	Burst           int // >0: publish exactly this many events as fast as possible instead of pacing for Duration
	ReceiverAddr    string
	ReceiverURL     string // URL Sparrow uses to reach the receiver (defaults to the listener address)
	ReceiverDelay   time.Duration
	Duration        time.Duration
	TargetRPS       int
	Publishers      int
	PayloadSizeKB   int
	DrainTimeout    time.Duration
	KeepWebhook     bool
	SamplingPercent float64
	ResourceConfig  ResourceConfig
}

// eventRecord matches a publish to its delivery. Either side may arrive
// first: the delivery can reach the receiver before the publisher has read
// the 201 response that carries the event id.
type eventRecord struct {
	sent  time.Time
	recv  time.Time // when the last required subscriber received it
	count int       // deliveries seen
}

// E2EResults is the pipeline benchmark report.
type E2EResults struct {
	Config E2EConfig `json:"config"`

	PublishDuration time.Duration `json:"publish_duration"`
	TotalDuration   time.Duration `json:"total_duration"`
	DrainDuration   time.Duration `json:"drain_duration"`
	Drained         bool          `json:"drained"`

	Accepted   int64 `json:"accepted"`
	Rejected   int64 `json:"rejected"`
	Delivered  int64 `json:"delivered"`
	Duplicates int64 `json:"duplicates"`
	Unmatched  int64 `json:"unmatched"`
	Foreign    int64 `json:"foreign"` // deliveries for events this run did not publish (leftover backlog)

	PublishRPS        float64 `json:"publish_rps"`
	DeliveryRPS       float64 `json:"delivery_rps"`
	SteadyDeliveryRPS float64 `json:"steady_delivery_rps"`

	Ingest LatencySummary `json:"ingest_latency"`
	E2E    LatencySummary `json:"end_to_end_latency"`

	PeakBacklog int64            `json:"peak_backlog"`
	Backlog     []BacklogSample  `json:"backlog_timeline"`
	StatusCodes map[string]int64 `json:"publish_status_codes"`

	ResourceUsage *ResourceUsage `json:"resource_usage,omitempty"`
}

// BacklogSample is one second of the accepted-vs-delivered timeline.
type BacklogSample struct {
	Second    int   `json:"second"`
	Accepted  int64 `json:"accepted"`
	Delivered int64 `json:"delivered"`
	Backlog   int64 `json:"backlog"`
}

// E2ERunner owns the receiver, the publishers and the bookkeeping.
type E2ERunner struct {
	cfg  E2EConfig
	http *http.Client

	mu      sync.Mutex
	records map[string]*eventRecord

	accepted   atomic.Int64
	rejected   atomic.Int64
	delivered  atomic.Int64
	duplicates atomic.Int64
	unmatched  atomic.Int64
	foreign    atomic.Int64
	published  atomic.Int64 // ids recorded by publishers (accepted events whose id is known)

	statusMu    sync.Mutex
	statusCodes map[string]int64

	ingest *LatencyReservoir
	e2e    *LatencyReservoir

	payload    []byte
	eventNames []string
	webhookIDs []string
	rr         atomic.Uint64
	receiver   *http.Server
	listener   net.Listener
}

// NewE2ERunner prepares a runner; nothing is started yet.
func NewE2ERunner(cfg E2EConfig) *E2ERunner {
	return &E2ERunner{
		cfg: cfg,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        cfg.Publishers * 2,
				MaxIdleConnsPerHost: cfg.Publishers * 2,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		records:     make(map[string]*eventRecord, cfg.TargetRPS*int(cfg.Duration.Seconds())+1024),
		statusCodes: make(map[string]int64),
		ingest:      NewLatencyReservoir(20000),
		e2e:         NewLatencyReservoir(20000),
		payload:     bytes.Repeat([]byte("x"), cfg.PayloadSizeKB*1024),
	}
}

// Run executes the whole benchmark: receiver up, webhook registered,
// publish for Duration, drain, tear down.
func (r *E2ERunner) Run(ctx context.Context) (*E2EResults, error) {
	// A fresh event type per run: fan-out resolves webhooks at processing
	// time, so events still queued from an earlier run would otherwise be
	// delivered to this run's receiver.
	if r.cfg.Webhooks < 1 {
		r.cfg.Webhooks = 1
	}
	if r.cfg.Subscribers < 1 {
		r.cfg.Subscribers = 1
	}
	base := fmt.Sprintf("%s.%d", strings.TrimSuffix(r.cfg.EventName, "."), time.Now().Unix())
	for i := 0; i < r.cfg.Webhooks; i++ {
		r.eventNames = append(r.eventNames, fmt.Sprintf("%s.w%d", base, i))
	}
	r.cfg.EventName = base + ".w{0..N}"
	if err := r.startReceiver(); err != nil {
		return nil, err
	}
	defer r.stopReceiver()

	if err := r.registerPipeline(ctx); err != nil {
		return nil, err
	}
	if !r.cfg.KeepWebhook {
		defer r.deleteWebhooks()
	}

	// Warm-up: one event end to end so the first measured second does not
	// pay for cold connection pools and auto-registration.
	if err := r.warmUp(ctx); err != nil {
		return nil, err
	}

	if r.cfg.Burst > 0 {
		log.Printf("publishing a burst of %d %s events with %d publishers across %d event types × %d subscribers → %s", r.cfg.Burst, r.cfg.EventName, r.cfg.Publishers, r.cfg.Webhooks, r.cfg.Subscribers, r.cfg.SparrowURL)
	} else {
		log.Printf("publishing %s at %d rps with %d publishers across %d event types × %d subscribers → %s", r.cfg.EventName, r.cfg.TargetRPS, r.cfg.Publishers, r.cfg.Webhooks, r.cfg.Subscribers, r.cfg.SparrowURL)
	}

	start := time.Now()
	timeline := make([]BacklogSample, 0, int(r.cfg.Duration.Seconds())+int(r.cfg.DrainTimeout.Seconds())+2)
	var timelineMu sync.Mutex
	backlogSampler := r.startBacklogSampler(start, &timeline, &timelineMu)

	resSampler, err := NewResourceSampler(r.cfg.ResourceConfig)
	if err != nil {
		log.Printf("resource sampler init: %v", err)
	}
	resSampler.Start(ctx)

	window := r.cfg.Duration
	if r.cfg.Burst > 0 {
		window = r.cfg.DrainTimeout // the burst ends when the events are out, not on a clock
	}
	publishCtx, cancelPublish := context.WithTimeout(ctx, window)
	r.publish(publishCtx)
	cancelPublish()
	publishEnd := time.Now()

	// Delivery throughput while the publishers were still running.
	steadyDelivered := r.delivered.Load()

	drained := r.waitForDrain(ctx)
	end := time.Now()
	backlogSampler()
	resUsage := resSampler.Stop(r.delivered.Load())

	r.mu.Lock()
	for _, rec := range r.records {
		if rec.sent.IsZero() && !rec.recv.IsZero() {
			r.foreign.Add(1)
		}
	}
	r.mu.Unlock()

	timelineMu.Lock()
	tl := append([]BacklogSample(nil), timeline...)
	timelineMu.Unlock()

	res := &E2EResults{
		Config:            r.cfg,
		PublishDuration:   publishEnd.Sub(start),
		TotalDuration:     end.Sub(start),
		DrainDuration:     end.Sub(publishEnd),
		Drained:           drained,
		Accepted:          r.accepted.Load(),
		Rejected:          r.rejected.Load(),
		Delivered:         r.delivered.Load(),
		Duplicates:        r.duplicates.Load(),
		Unmatched:         r.unmatched.Load(),
		Foreign:           r.foreign.Load(),
		Ingest:            r.ingest.Summary(),
		E2E:               r.e2e.Summary(),
		Backlog:           tl,
		StatusCodes:       r.snapshotStatusCodes(),
		PublishRPS:        float64(r.accepted.Load()) / publishEnd.Sub(start).Seconds(),
		DeliveryRPS:       float64(r.delivered.Load()) / end.Sub(start).Seconds(),
		SteadyDeliveryRPS: float64(steadyDelivered) / publishEnd.Sub(start).Seconds(),
		ResourceUsage:     resUsage,
	}
	for _, s := range tl {
		if s.Backlog > res.PeakBacklog {
			res.PeakBacklog = s.Backlog
		}
	}
	return res, nil
}

// --- receiver -------------------------------------------------------------

func (r *E2ERunner) startReceiver() error {
	ln, err := net.Listen("tcp", r.cfg.ReceiverAddr)
	if err != nil {
		return fmt.Errorf("listen receiver: %w", err)
	}
	r.listener = ln
	if r.cfg.ReceiverURL == "" {
		r.cfg.ReceiverURL = "http://" + ln.Addr().String()
	}
	r.receiver = &http.Server{
		Handler:           http.HandlerFunc(r.handleDelivery),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := r.receiver.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("receiver: %v", err)
		}
	}()
	log.Printf("receiver listening on %s", r.cfg.ReceiverURL)
	return nil
}

func (r *E2ERunner) stopReceiver() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = r.receiver.Shutdown(ctx)
}

func (r *E2ERunner) handleDelivery(w http.ResponseWriter, req *http.Request) {
	now := time.Now()
	_, _ = io.Copy(io.Discard, req.Body)
	_ = req.Body.Close()

	if r.cfg.ReceiverDelay > 0 {
		time.Sleep(r.cfg.ReceiverDelay)
	}

	eventID := req.Header.Get("X-Sparrow-Event-ID")
	if eventID == "" {
		r.unmatched.Add(1)
	} else {
		r.mu.Lock()
		rec, ok := r.records[eventID]
		if !ok {
			rec = &eventRecord{}
			r.records[eventID] = rec
		}
		rec.count++
		complete := rec.count == r.cfg.Subscribers
		dup := rec.count > r.cfg.Subscribers
		if complete {
			rec.recv = now
		}
		sent := rec.sent
		r.mu.Unlock()

		switch {
		case dup:
			r.duplicates.Add(1)
		case complete && !sent.IsZero():
			r.delivered.Add(1)
			r.e2e.Add(now.Sub(sent))
		default:
			// Not all subscribers yet, or the publisher has not recorded
			// this id (publishOne settles that case).
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// --- Sparrow API ----------------------------------------------------------

func (r *E2ERunner) do(ctx context.Context, method, path string, body any, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(r.cfg.SparrowURL, "/")+path, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", r.cfg.APIKey)
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if out != nil && resp.StatusCode < 300 {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode %s %s: %w", method, path, err)
		}
	} else {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	}
	return resp.StatusCode, nil
}

func (r *E2ERunner) registerPipeline(ctx context.Context) error {
	for i, name := range r.eventNames {
		code, err := r.do(ctx, http.MethodPost, "/v1/event-types", map[string]any{"name": name, "active": true}, nil)
		if err != nil {
			return fmt.Errorf("register event type: %w", err)
		}
		if code != http.StatusCreated && code != http.StatusConflict {
			return fmt.Errorf("register event type: HTTP %d", code)
		}

		for sub := 0; sub < r.cfg.Subscribers; sub++ {
			var out struct {
				WebhookID string `json:"webhook_id"`
			}
			code, err = r.do(ctx, http.MethodPost, "/v1/consumers/"+r.cfg.Consumer+"/webhooks", map[string]any{
				"events": []string{name},
				"url":    fmt.Sprintf("%s/webhook/%d/%d", r.cfg.ReceiverURL, i, sub),
				"active": true,
				"http_config": map[string]any{
					"max_retries":             3,
					"retry_backoff_seconds":   1,
					"request_timeout_seconds": 10,
				},
			}, &out)
			if err != nil {
				return fmt.Errorf("register webhook: %w", err)
			}
			if code != http.StatusCreated {
				return fmt.Errorf("register webhook: HTTP %d (is SPARROW_ALLOW_PRIVATE_NETWORKS=true on the server?)", code)
			}
			r.webhookIDs = append(r.webhookIDs, out.WebhookID)
		}
	}
	log.Printf("registered %d webhook(s) over %d event type(s) → %s", len(r.webhookIDs), len(r.eventNames), r.cfg.ReceiverURL)
	return nil
}

func (r *E2ERunner) deleteWebhooks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, id := range r.webhookIDs {
		code, err := r.do(ctx, http.MethodDelete, "/v1/consumers/"+r.cfg.Consumer+"/webhooks/"+id, nil, nil)
		if err != nil || code >= 300 {
			log.Printf("delete webhook %s: code=%d err=%v (left in place)", id, code, err)
		}
	}
}

func (r *E2ERunner) warmUp(ctx context.Context) error {
	wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := r.publishOne(wctx, false); err != nil {
		return fmt.Errorf("warm-up publish: %w", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if r.delivered.Load() >= 1 {
			// Reset counters so the warm-up event is not part of the report.
			r.mu.Lock()
			r.records = make(map[string]*eventRecord, len(r.records))
			r.mu.Unlock()
			r.accepted.Store(0)
			r.rejected.Store(0)
			r.delivered.Store(0)
			r.duplicates.Store(0)
			r.unmatched.Store(0)
			r.published.Store(0)
			r.ingest = NewLatencyReservoir(r.ingest.capacity)
			r.e2e = NewLatencyReservoir(r.e2e.capacity)
			r.statusMu.Lock()
			r.statusCodes = make(map[string]int64)
			r.statusMu.Unlock()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("warm-up event was accepted but never delivered within 30s: check the server's worker logs and SPARROW_ALLOW_PRIVATE_NETWORKS")
}

// publishOne pushes a single event and records the publish timestamp.
func (r *E2ERunner) publishOne(ctx context.Context, sample bool) error {
	body := map[string]any{
		"payload": map[string]any{
			"sent_at": time.Now().UnixNano(),
			"data":    string(r.payload),
		},
		"ttl_seconds": 600,
	}
	var out struct {
		EventID string `json:"event_id"`
	}
	name := r.eventNames[int(r.rr.Add(1)-1)%len(r.eventNames)]
	sent := time.Now()
	code, err := r.do(ctx, http.MethodPost, "/v1/consumers/"+r.cfg.Consumer+"/events?event="+name, body, &out)
	lat := time.Since(sent)

	r.statusMu.Lock()
	if err != nil && ctx.Err() != nil {
		// window closed; don't record
	} else if err != nil {
		r.statusCodes["error"]++
	} else {
		r.statusCodes[fmt.Sprint(code)]++
	}
	r.statusMu.Unlock()

	if err != nil {
		if ctx.Err() != nil {
			// Publish window closed mid-request: not a rejection.
			return err
		}
		r.rejected.Add(1)
		return err
	}
	if code != http.StatusCreated || out.EventID == "" {
		r.rejected.Add(1)
		return fmt.Errorf("push event: HTTP %d", code)
	}

	r.accepted.Add(1)
	if sample {
		r.ingest.Add(lat)
	}

	r.mu.Lock()
	rec, ok := r.records[out.EventID]
	if !ok {
		rec = &eventRecord{}
		r.records[out.EventID] = rec
	}
	rec.sent = sent
	recv := rec.recv // set only once all subscribers delivered
	r.mu.Unlock()
	r.published.Add(1)

	// Deliveries beat us to it: settle it now.
	if !recv.IsZero() {
		r.delivered.Add(1)
		if sample {
			r.e2e.Add(recv.Sub(sent))
		}
	}
	return nil
}

// publish runs the paced publishers until ctx expires.
func (r *E2ERunner) publish(ctx context.Context) {
	var limiter *RateLimiter
	if r.cfg.Burst <= 0 {
		limiter = NewRateLimiter(r.cfg.TargetRPS, r.cfg.Publishers)
		defer limiter.Stop()
	}
	var remaining atomic.Int64
	remaining.Store(int64(r.cfg.Burst))

	var wg sync.WaitGroup
	var firstErr sync.Once
	for i := 0; i < r.cfg.Publishers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				if limiter != nil {
					if err := limiter.Wait(ctx); err != nil {
						return
					}
				} else if remaining.Add(-1) < 0 {
					return
				}
				if err := r.publishOne(ctx, true); err != nil && ctx.Err() == nil {
					firstErr.Do(func() { log.Printf("first publish failure: %v", err) })
				}
			}
		}()
	}
	wg.Wait()
}

// waitForDrain blocks until every accepted event has been delivered, or the
// drain timeout passes. Returns whether the backlog reached zero.
func (r *E2ERunner) waitForDrain(ctx context.Context) bool {
	deadline := time.Now().Add(r.cfg.DrainTimeout)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		if r.delivered.Load() >= r.published.Load() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return r.delivered.Load() >= r.published.Load()
}

// startBacklogSampler records accepted/delivered once a second; the returned
// func stops it.
func (r *E2ERunner) startBacklogSampler(start time.Time, out *[]BacklogSample, mu *sync.Mutex) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case now := <-t.C:
				a, d := r.accepted.Load(), r.delivered.Load()
				mu.Lock()
				*out = append(*out, BacklogSample{Second: int(now.Sub(start).Seconds()), Accepted: a, Delivered: d, Backlog: a - d})
				mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { cancel(); <-done }
}

func (r *E2ERunner) snapshotStatusCodes() map[string]int64 {
	r.statusMu.Lock()
	defer r.statusMu.Unlock()
	out := make(map[string]int64, len(r.statusCodes))
	for k, v := range r.statusCodes {
		out[k] = v
	}
	return out
}

// Print writes the human-readable report.
func (res *E2EResults) Print() {
	fmt.Printf("\n=== End-to-End Pipeline Results ===\n\n")
	fmt.Printf("Sparrow:          %s  (event %s, consumer %s)\n", res.Config.SparrowURL, res.Config.EventName, res.Config.Consumer)
	fmt.Printf("Receiver:         %s, %d event type(s) × %d subscriber(s) = %d webhooks (simulated delay %v)\n", res.Config.ReceiverURL, res.Config.Webhooks, res.Config.Subscribers, res.Config.Webhooks*res.Config.Subscribers, res.Config.ReceiverDelay)
	if res.Config.Burst > 0 {
		fmt.Printf("Publish window:   %v for a burst of %d events, %d publishers, %d KB payload\n", res.PublishDuration.Round(time.Millisecond), res.Config.Burst, res.Config.Publishers, res.Config.PayloadSizeKB)
	} else {
		fmt.Printf("Publish window:   %v at target %d rps, %d publishers, %d KB payload\n", res.PublishDuration.Round(time.Millisecond), res.Config.TargetRPS, res.Config.Publishers, res.Config.PayloadSizeKB)
	}
	fmt.Printf("Drain:            %v (%s)\n\n", res.DrainDuration.Round(time.Millisecond), map[bool]string{true: "backlog reached zero", false: "TIMED OUT with backlog"}[res.Drained])

	fmt.Printf("Events:\n")
	fmt.Printf("  Accepted (201): %d\n", res.Accepted)
	fmt.Printf("  Rejected:       %d  %v\n", res.Rejected, res.StatusCodes)
	fmt.Printf("  Delivered:      %d events (all %d subscriber(s) reached)\n", res.Delivered, res.Config.Subscribers)
	fmt.Printf("  Duplicates:     %d (retries that re-hit the receiver)\n", res.Duplicates)
	fmt.Printf("  Unmatched:      %d (deliveries without an event id)\n", res.Unmatched)
	fmt.Printf("  Foreign:        %d (deliveries of events this run did not publish: leftover backlog)\n\n", res.Foreign)

	fmt.Printf("Throughput:\n")
	fmt.Printf("  Publish:        %.1f events/s accepted\n", res.PublishRPS)
	fmt.Printf("  Delivery:       %.1f/s while publishing, %.1f/s over the whole run\n", res.SteadyDeliveryRPS, res.DeliveryRPS)
	fmt.Printf("  Peak backlog:   %d events\n\n", res.PeakBacklog)

	printLatency("Ingest latency (POST /events → 201)", res.Ingest)
	printLatency("End-to-end latency (POST sent → receiver got it)", res.E2E)

	if res.ResourceUsage != nil {
		fmt.Printf("Resource Utilization:\n")
		if res.ResourceUsage.SparrowPID > 0 {
			fmt.Printf("  Sparrow CPU:    avg %.1f%%, peak %.1f%%\n", res.ResourceUsage.SparrowCPUAvg, res.ResourceUsage.SparrowCPUPeak)
			fmt.Printf("  Sparrow RSS:    peak %.2f MB\n", res.ResourceUsage.SparrowPeakRSS)
		}
		if res.ResourceUsage.PostgresPID > 0 {
			fmt.Printf("  Postgres CPU:   avg %.1f%%, peak %.1f%%\n", res.ResourceUsage.PostgresCPUAvg, res.ResourceUsage.PostgresCPUPeak)
		}
		if res.ResourceUsage.WALBytesWritten > 0 {
			fmt.Printf("  WAL written:    %.2f MB (%.2f MB/s)\n", float64(res.ResourceUsage.WALBytesWritten)/1024/1024, res.ResourceUsage.WALMBPerSec)
		}
		if res.ResourceUsage.CPUMsPerDelivery > 0 {
			fmt.Printf("  Efficiency:     %.2f CPU-ms / delivery\n", res.ResourceUsage.CPUMsPerDelivery)
		}
		fmt.Println()
	}

	if len(res.Backlog) > 0 {
		fmt.Printf("Backlog timeline (sec: accepted/delivered/backlog):\n  ")
		for i, s := range res.Backlog {
			if i > 0 && i%6 == 0 {
				fmt.Printf("\n  ")
			}
			fmt.Printf("%3ds: %d/%d/%d   ", s.Second, s.Accepted, s.Delivered, s.Backlog)
		}
		fmt.Println()
	}
}

func printLatency(title string, s LatencySummary) {
	fmt.Printf("%s:\n", title)
	fmt.Printf("  samples=%d  avg=%v  p50=%v  p90=%v  p95=%v  p99=%v  p99.9=%v  max=%v\n\n",
		s.Samples, s.Avg.Round(time.Microsecond), s.P50.Round(time.Microsecond), s.P90.Round(time.Microsecond),
		s.P95.Round(time.Microsecond), s.P99.Round(time.Microsecond), s.P999.Round(time.Microsecond), s.Max.Round(time.Microsecond))
}
