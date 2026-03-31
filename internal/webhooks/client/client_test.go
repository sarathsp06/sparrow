package client

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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

	if client.metrics == nil {
		t.Error("Expected metrics to be initialized")
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

	// Check metrics
	if client.metrics.TotalRequests != 1 {
		t.Errorf("Expected TotalRequests 1, got %d", client.metrics.TotalRequests)
	}

	if client.metrics.SuccessRequests != 1 {
		t.Errorf("Expected SuccessRequests 1, got %d", client.metrics.SuccessRequests)
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

	// Check metrics
	if client.metrics.TotalRequests != 1 {
		t.Errorf("Expected TotalRequests 1, got %d", client.metrics.TotalRequests)
	}

	if client.metrics.FailedRequests != 1 {
		t.Errorf("Expected FailedRequests 1, got %d", client.metrics.FailedRequests)
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

func TestTransformPayload(t *testing.T) {
	client := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		data     WebhookTemplateContext
		expected string
		wantErr  bool
	}{
		{
			name:     "simple template",
			template: `{"event": "{{.event_name}}"}`,
			data: WebhookTemplateContext{
				"event_name": "user.created",
			},
			expected: `{"event": "user.created"}`,
			wantErr:  false,
		},
		{
			name:     "with payload",
			template: `{"user_id": "{{.payload.user_id}}"}`,
			data: WebhookTemplateContext{
				"payload": map[string]any{"user_id": "123"},
			},
			expected: `{"user_id": "123"}`,
			wantErr:  false,
		},
		{
			name:     "empty template",
			template: "",
			data:     WebhookTemplateContext{},
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.TransformPayload(tt.template, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got error=%v", tt.wantErr, err)
			}

			if tt.template == "" && result != nil {
				t.Error("Expected nil result for empty template")
			}

			if tt.template != "" && string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
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

func BenchmarkTransformPayload(b *testing.B) {
	client := NewTemplateEngine()
	templates := []string{
		`{"event": "{{.event_name}}", "id": "{{.event_id}}"}`,
		`{"event": "{{.payload.event}}", "id": "{{.event_id}}"}`,
		`{"event": "{{.event_name | upper}}", "id": "{{.event_id | lower}}"}`,
	}

	data := WebhookTemplateContext{
		"event_id":   "event-123",
		"event_name": "user.created",
		"payload": map[string]any{
			"event": "user.created",
		},
	}

	// Create duplicate templates to simulate real-world usage
	duplicatedTemplates := make([]string, b.N*len(templates))
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(templates); j++ {
			duplicatedTemplates[i*len(templates)+j] = templates[j]
		}
	}

	// Shuffle the templates to ensure randomness
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(duplicatedTemplates), func(i, j int) {
		duplicatedTemplates[i], duplicatedTemplates[j] = duplicatedTemplates[j], duplicatedTemplates[i]
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.TransformPayload(duplicatedTemplates[i], data)
	}
}
