package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogsURL(t *testing.T) {
	cases := map[string]string{
		"http://collector:4318":          "http://collector:4318/v1/logs",
		"http://collector:4318/":         "http://collector:4318/v1/logs",
		"https://otlp.vendor.com/custom": "https://otlp.vendor.com/custom",
		"http://collector:4318/v1/logs":  "http://collector:4318/v1/logs",
	}
	for in, want := range cases {
		got, err := logsURL(in)
		if err != nil {
			t.Fatalf("logsURL(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("logsURL(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := logsURL("collector:4318"); err == nil {
		t.Error("expected error for scheme-less endpoint")
	}
}

func TestOTLPSinkDeliver(t *testing.T) {
	env := envelope{
		Version:   "1",
		EventID:   "evt_1",
		EventName: "user.signup",
		Timestamp: "2026-01-02T03:04:05Z",
		Attempt:   2,
		Payload:   json.RawMessage(`{"email":"a@b.c"}`),
	}

	var gotPath, gotAuth string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sink, err := newOTLPSink(otlpConfig{
		Endpoint: srv.URL,
		Headers:  map[string]string{"Authorization": "Bearer tok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.deliver(context.Background(), env, nil); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if gotPath != "/v1/logs" {
		t.Errorf("path = %q, want /v1/logs", gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q", gotAuth)
	}

	var req struct {
		ResourceLogs []struct {
			Resource struct {
				Attributes []struct {
					Key   string `json:"key"`
					Value struct {
						StringValue string `json:"stringValue"`
					} `json:"value"`
				} `json:"attributes"`
			} `json:"resource"`
			ScopeLogs []struct {
				LogRecords []struct {
					TimeUnixNano string `json:"timeUnixNano"`
					Body         struct {
						StringValue string `json:"stringValue"`
					} `json:"body"`
					Attributes []struct {
						Key   string `json:"key"`
						Value struct {
							StringValue string `json:"stringValue"`
							IntValue    string `json:"intValue"`
						} `json:"value"`
					} `json:"attributes"`
				} `json:"logRecords"`
			} `json:"scopeLogs"`
		} `json:"resourceLogs"`
	}
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatalf("unmarshal OTLP body: %v", err)
	}
	rec := req.ResourceLogs[0].ScopeLogs[0].LogRecords[0]
	if rec.Body.StringValue != `{"email":"a@b.c"}` {
		t.Errorf("body = %q", rec.Body.StringValue)
	}
	if rec.TimeUnixNano != "1767323045000000000" { // 2026-01-02T03:04:05Z
		t.Errorf("timeUnixNano = %q", rec.TimeUnixNano)
	}
	attrs := map[string]string{}
	for _, a := range rec.Attributes {
		attrs[a.Key] = a.Value.StringValue + a.Value.IntValue
	}
	if attrs["sparrow.event_id"] != "evt_1" || attrs["sparrow.event_name"] != "user.signup" || attrs["sparrow.attempt"] != "2" {
		t.Errorf("attributes = %v", attrs)
	}
	if req.ResourceLogs[0].Resource.Attributes[0].Value.StringValue != "sparrow" {
		t.Errorf("service.name default not applied: %v", req.ResourceLogs[0].Resource.Attributes)
	}
}

func TestOTLPSinkDeliverDownstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "storage full", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	sink, err := newOTLPSink(otlpConfig{Endpoint: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.deliver(context.Background(), envelope{EventID: "e", EventName: "n"}, nil); err == nil {
		t.Fatal("expected error on 503 so Sparrow retries")
	}
}
