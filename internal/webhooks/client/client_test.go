package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewWebhookClient(t *testing.T) {
	config := &Config{
		Timeout:         10 * time.Second,
		MaxIdleConns:    50,
		MaxConnsPerHost: 5,
	}

	client := NewWebhookClient(config)

	if client.httpClient == nil {
		t.Error("Expected http client to be initialized")
	}

	if client.tmpl == nil {
		t.Error("Expected template engine to be initialized")
	}

	if client.config != config {
		t.Error("Expected config to be set")
	}
}

func TestNewWebhookClientWithNilConfig(t *testing.T) {
	client := NewWebhookClient(nil)

	if client.config == nil {
		t.Fatal("Expected default config to be set")
	}

	if client.config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", client.config.Timeout)
	}
}

func TestSend(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("X-Sparrow-Event-ID") == "" {
			t.Error("Expected X-Sparrow-Event-ID header")
		}

		if r.Header.Get("X-Sparrow-Delivery-ID") == "" {
			t.Error("Expected X-Sparrow-Delivery-ID header")
		}

		// Verify payload
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "test") {
			t.Error("Expected payload to contain 'test'")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewWebhookClient(&Config{
		Timeout:              30 * time.Second,
		MaxIdleConns:         100,
		MaxConnsPerHost:      10,
		IdleConnTimeout:      90 * time.Second,
		AllowPrivateNetworks: true, // httptest.NewServer binds to 127.0.0.1
	})
	ctx := context.Background()

	req := &DeliveryRequest{
		WebhookID:  uuid.New(),
		DeliveryID: "delivery-123",
		URL:        server.URL,
		Method:     "POST",
		Headers:    map[string]string{},
		Payload:    []byte(`{"test": "data"}`),
		EventID:    uuid.New(),
		EventName:  "test.event",
	}

	resp, duration, err := client.Send(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
		return
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestSendFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/final", http.StatusFound)
	}))
	defer server.Close()

	client := NewWebhookClient(&Config{
		Timeout:              30 * time.Second,
		MaxIdleConns:         100,
		MaxConnsPerHost:      10,
		IdleConnTimeout:      90 * time.Second,
		AllowPrivateNetworks: true, // httptest.NewServer binds to 127.0.0.1
	})
	ctx := context.Background()

	for _, tc := range []struct {
		follow     bool
		wantStatus int
	}{
		{follow: false, wantStatus: http.StatusFound},
		{follow: true, wantStatus: http.StatusOK},
	} {
		req := &DeliveryRequest{
			WebhookID:       uuid.New(),
			DeliveryID:      "delivery-123",
			URL:             server.URL,
			Method:          "POST",
			Headers:         map[string]string{},
			Payload:         []byte(`{}`),
			EventID:         uuid.New(),
			EventName:       "test.event",
			FollowRedirects: tc.follow,
		}
		resp, _, err := client.Send(ctx, req)
		if err != nil {
			t.Fatalf("follow=%v: unexpected error: %v", tc.follow, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != tc.wantStatus {
			t.Errorf("follow=%v: expected status %d, got %d", tc.follow, tc.wantStatus, resp.StatusCode)
		}
	}
}

func TestSendSkipTLSVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(&Config{
		Timeout:              30 * time.Second,
		MaxIdleConns:         100,
		MaxConnsPerHost:      10,
		IdleConnTimeout:      90 * time.Second,
		AllowPrivateNetworks: true, // httptest binds to 127.0.0.1
	})
	ctx := context.Background()

	newReq := func(skip bool) *DeliveryRequest {
		return &DeliveryRequest{
			WebhookID:     uuid.New(),
			DeliveryID:    "delivery-tls",
			URL:           server.URL,
			Method:        "POST",
			Headers:       map[string]string{},
			Payload:       []byte(`{}`),
			EventID:       uuid.New(),
			EventName:     "test.event",
			SkipTLSVerify: skip,
		}
	}

	// Default (verify): self-signed cert must be rejected.
	if resp, _, err := client.Send(ctx, newReq(false)); err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected TLS verification failure against self-signed cert, got success")
	}

	// verify_ssl=false: delivery succeeds despite self-signed cert.
	resp, _, err := client.Send(ctx, newReq(true))
	if err != nil {
		t.Fatalf("SkipTLSVerify=true: unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestSendFailure(t *testing.T) {
	client := NewWebhookClient(nil)
	ctx := context.Background()

	// Use invalid URL to trigger error
	req := &DeliveryRequest{
		WebhookID:  uuid.New(),
		DeliveryID: "delivery-123",
		URL:        "http://invalid-host-that-does-not-exist:9999",
		Method:     "POST",
		Headers:    map[string]string{},
		Payload:    []byte(`{"test": "data"}`),
		EventID:    uuid.New(),
		EventName:  "test.event",
	}

	// Set a very short timeout to speed up the test
	client.httpClient.Timeout = 100 * time.Millisecond

	_, duration, err := client.Send(ctx, req)
	if err == nil {
		t.Error("Expected error for invalid host")
	}

	if duration <= 0 {
		t.Error("Expected positive duration even on failure")
	}
}

func TestSendInvalidRequest(t *testing.T) {
	client := NewWebhookClient(nil)
	ctx := context.Background()

	req := &DeliveryRequest{
		WebhookID:  uuid.New(),
		DeliveryID: "delivery-123",
		URL:        "://invalid-url",
		Method:     "POST",
		Payload:    []byte(`{"test": "data"}`),
		EventID:    uuid.New(),
	}

	_, _, err := client.Send(ctx, req)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestClientClose(t *testing.T) {
	client := NewWebhookClient(nil)

	err := client.Close()
	if err != nil {
		t.Errorf("Unexpected error closing client: %v", err)
	}
}

func TestReadBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		limit    int64
		expected string
	}{
		{
			name:     "read full body",
			body:     "Hello World",
			limit:    0,
			expected: "Hello World",
		},
		{
			name:     "read with limit",
			body:     "Hello World",
			limit:    5,
			expected: "Hello",
		},
		{
			name:     "read empty body",
			body:     "",
			limit:    0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(tt.body)),
			}

			result, err := ReadBody(resp, tt.limit)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestReadBodyNil(t *testing.T) {
	result, err := ReadBody(nil, 0)
	if err != nil {
		t.Errorf("Unexpected error with nil response: %v", err)
	}

	if result != nil {
		t.Error("Expected nil result for nil response")
	}
}

func TestReadBodyNilBody(t *testing.T) {
	resp := &http.Response{Body: nil}

	result, err := ReadBody(resp, 0)
	if err != nil {
		t.Errorf("Unexpected error with nil body: %v", err)
	}

	if result != nil {
		t.Error("Expected nil result for nil body")
	}
}

func BenchmarkSend(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(&Config{
		Timeout:              30 * time.Second,
		MaxIdleConns:         100,
		MaxConnsPerHost:      10,
		IdleConnTimeout:      90 * time.Second,
		AllowPrivateNetworks: true, // httptest.NewServer binds to 127.0.0.1
	})
	ctx := context.Background()

	// Use a relatively bigger payload
	payload := []byte{}
	for range 1000 {
		payload = append(payload, []byte(`{"test": "data"}`)...)
	}

	req := &DeliveryRequest{
		WebhookID:  uuid.New(),
		DeliveryID: "delivery-123",
		URL:        server.URL,
		Method:     "POST",
		Payload:    payload,
		EventID:    uuid.New(),
	}

	for b.Loop() {
		_, _, _ = client.Send(ctx, req)
	}
}

func TestRedactURL(t *testing.T) {
	cases := map[string]string{
		"https://hooks.slack.com/services/T000/B000/XXXXSECRET": "https://hooks.slack.com/…",
		"https://user:pass@example.com/hook?token=abc":          "https://example.com/…",
		"https://example.com":                                   "https://example.com",
		"https://example.com/":                                  "https://example.com",
		"http://10.0.0.5:8080/webhook":                          "http://10.0.0.5:8080/…",
		"https://discord.com/api/webhooks/123/SECRET#frag":      "https://discord.com/…",
		"not a url with spaces and a secret://":                 "[redacted url]",
		"://missing-scheme/secret":                              "[redacted url]",
	}
	for in, want := range cases {
		if got := RedactURL(in); got != want {
			t.Errorf("RedactURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSendRedactsURLInErrors(t *testing.T) {
	// A refused connection: net/http wraps it in *url.Error whose message
	// embeds the request URL, secret path and query included.
	client := NewWebhookClient(&Config{Timeout: 2 * time.Second, AllowPrivateNetworks: true})
	secretURL := "http://127.0.0.1:1/services/T000/B000/SUPERSECRET?token=ALSOSECRET"
	_, _, err := client.Send(context.Background(), &DeliveryRequest{
		WebhookID: uuid.New(), DeliveryID: "d", URL: secretURL, Method: "POST",
		Headers: map[string]string{}, Payload: []byte(`{}`), EventID: uuid.New(), EventName: "e",
	})
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if strings.Contains(err.Error(), "SUPERSECRET") || strings.Contains(err.Error(), "ALSOSECRET") {
		t.Fatalf("error leaks the webhook URL: %v", err)
	}
	var urlErr *url.Error
	if !errors.As(err, &urlErr) || urlErr.URL != "http://127.0.0.1:1/…" {
		t.Fatalf("error type or redacted URL lost: %#v", err)
	}
	// The underlying cause is kept for error classification.
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("wrapped network error lost: %v", err)
	}
}

func TestDeliverySpansDoNotExportSecretURLs(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer server.Close()

	client := NewWebhookClient(&Config{Timeout: 5 * time.Second, AllowPrivateNetworks: true})
	resp, _, err := client.Send(context.Background(), &DeliveryRequest{
		WebhookID: uuid.New(), DeliveryID: "d", URL: server.URL + "/services/T000/B000/SUPERSECRET?token=ALSOSECRET",
		Method: "POST", Headers: map[string]string{}, Payload: []byte(`{}`), EventID: uuid.New(), EventName: "e",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	spans := recorder.Ended()
	if len(spans) == 0 {
		t.Fatal("no delivery span recorded")
	}
	for _, s := range spans {
		for _, kv := range s.Attributes() {
			if v := kv.Value.Emit(); strings.Contains(v, "SUPERSECRET") || strings.Contains(v, "ALSOSECRET") {
				t.Fatalf("span %q attribute %s leaks the webhook URL: %s", s.Name(), kv.Key, v)
			}
			if kv.Key == "url.full" && !strings.HasSuffix(kv.Value.AsString(), "/…") {
				t.Fatalf("url.full = %q, want the redacted form", kv.Value.AsString())
			}
		}
	}
}
