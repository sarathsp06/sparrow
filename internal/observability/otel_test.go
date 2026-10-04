package observability

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// traceCollector is a minimal OTLP/gRPC trace receiver.
type traceCollector struct {
	coltracepb.UnimplementedTraceServiceServer
	spans chan string
}

func (c *traceCollector) Export(_ context.Context, req *coltracepb.ExportTraceServiceRequest) (*coltracepb.ExportTraceServiceResponse, error) {
	for _, rs := range req.GetResourceSpans() {
		for _, ss := range rs.GetScopeSpans() {
			for _, s := range ss.GetSpans() {
				c.spans <- s.GetName()
			}
		}
	}
	return &coltracepb.ExportTraceServiceResponse{}, nil
}

func startGRPCCollector(t *testing.T) (addr string, spans chan string) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	spans = make(chan string, 16)
	srv := grpc.NewServer()
	coltracepb.RegisterTraceServiceServer(srv, &traceCollector{spans: spans})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String(), spans
}

func startHTTPCollector(t *testing.T) (addr string, spans chan string) {
	t.Helper()
	spans = make(chan string, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" {
			w.WriteHeader(http.StatusOK) // metrics/logs: accept and drop
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req coltracepb.ExportTraceServiceRequest
		if err := proto.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, rs := range req.GetResourceSpans() {
			for _, ss := range rs.GetScopeSpans() {
				for _, s := range ss.GetSpans() {
					spans <- s.GetName()
				}
			}
		}
		w.Header().Set("Content-Type", "application/x-protobuf")
		_, _ = w.Write(nil)
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://"), spans
}

// exportOneSpan runs Setup with the given endpoint and protocol, records a
// span, flushes via shutdown, and returns the span name the collector saw.
func exportOneSpan(t *testing.T, endpoint, protocol string, spans chan string) string {
	t.Helper()
	// Setup gates on config.OTLPEndpoint; the exporters read the env var.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", endpoint)

	cfg := DefaultConfig()
	cfg.OTLPEndpoint = endpoint
	cfg.OTLPProtocol = protocol
	shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	_, span := GetTracer("otel-test").Start(context.Background(), "test-span")
	span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Metrics/logs may be rejected by these trace-only collectors; only the
	// trace delivery matters here.
	_ = shutdown(ctx)

	select {
	case name := <-spans:
		return name
	case <-time.After(5 * time.Second):
		t.Fatal("collector received no span")
		return ""
	}
}

func TestSetupExportsSpans(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		start    func(*testing.T) (string, chan string)
		endpoint func(addr string) string
	}{
		{"grpc url", ProtocolGRPC, startGRPCCollector, func(a string) string { return "http://" + a }},
		{"grpc bare host:port", ProtocolGRPC, startGRPCCollector, func(a string) string { return a }},
		{"http url", ProtocolHTTPProtobuf, startHTTPCollector, func(a string) string { return "http://" + a }},
		{"http bare host:port", ProtocolHTTPProtobuf, startHTTPCollector, func(a string) string { return a }},
		{"default protocol is http", "", startHTTPCollector, func(a string) string { return "http://" + a }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, spans := tt.start(t)
			if got := exportOneSpan(t, tt.endpoint(addr), tt.protocol, spans); got != "test-span" {
				t.Fatalf("span name = %q, want test-span", got)
			}
		})
	}
}

func TestSetupRejectsUnknownProtocol(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OTLPEndpoint = "http://localhost:4318"
	cfg.OTLPProtocol = "http/json"
	if _, err := Setup(context.Background(), cfg); err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
}

func TestSetupServesPrometheusWithoutOTLP(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Prometheus = true
	shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()); metricsHandler = nil })

	handler := MetricsHandler()
	if handler == nil {
		t.Fatal("MetricsHandler() = nil with Prometheus enabled")
	}
	counter, err := GetMeter("prom-test").Int64Counter("sparrow_test_things_total")
	if err != nil {
		t.Fatal(err)
	}
	counter.Add(context.Background(), 3)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	for _, want := range []string{"sparrow_test_things_total", "go_goroutines"} {
		if !strings.Contains(body, want) {
			t.Errorf("scrape output lacks %s:\n%s", want, body)
		}
	}
	if strings.Contains(body, "_total_total") {
		t.Errorf("counter name got a doubled _total suffix:\n%s", body)
	}
}

func TestSetupWithoutExportersHasNoMetricsHandler(t *testing.T) {
	shutdown, err := Setup(context.Background(), DefaultConfig())
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	_ = shutdown(context.Background())
	if MetricsHandler() != nil {
		t.Fatal("MetricsHandler() != nil with Prometheus and OTLP both off")
	}
}
