package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestObjectKey(t *testing.T) {
	got := objectKey("events/", "order.paid", "2026-09-12T10:00:00Z", "evt_42")
	want := "events/order.paid/2026/09/12/evt_42.json"
	if got != want {
		t.Errorf("objectKey = %q, want %q", got, want)
	}
	// No prefix, unparseable timestamp: key still ends with name/date/id shape.
	got = objectKey("", "x", "not-a-time", "e1")
	if got == "" || got[len(got)-len("/e1.json"):] != "/e1.json" {
		t.Errorf("fallback key = %q", got)
	}
}

// TestS3Sink_PutObject exercises the sink against an httptest server stubbing
// PutObject (custom endpoint + static creds); MinIO-via-testcontainers is
// skipped because the repo pins no MinIO module.
func TestS3Sink_PutObject(t *testing.T) {
	type putReq struct {
		method, path, contentType string
		body                      []byte
	}
	got := make(chan putReq, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got <- putReq{r.Method, r.URL.Path, r.Header.Get("Content-Type"), body}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_SESSION_TOKEN", "")
	sink, err := newS3Sink(context.Background(), s3Config{
		Endpoint: srv.URL,
		Region:   "us-east-1",
		Bucket:   "sparrow-events",
		Prefix:   "events/",
	})
	if err != nil {
		t.Fatal(err)
	}

	raw := []byte(`{"version":"1","event_id":"evt_42","event_name":"order.paid","timestamp":"2026-09-12T10:00:00Z","attempt":1,"payload":{}}`)
	env := envelope{EventID: "evt_42", EventName: "order.paid", Timestamp: "2026-09-12T10:00:00Z"}
	if err := sink.deliver(context.Background(), env, raw); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	req := <-got
	if req.method != http.MethodPut {
		t.Errorf("method = %s", req.method)
	}
	if want := "/sparrow-events/events/order.paid/2026/09/12/evt_42.json"; req.path != want {
		t.Errorf("path = %q, want %q", req.path, want)
	}
	if req.contentType != "application/json" {
		t.Errorf("content-type = %q", req.contentType)
	}
	var check map[string]any
	if err := json.Unmarshal(req.body, &check); err != nil || check["event_id"] != "evt_42" {
		t.Errorf("stored body = %s (err %v)", req.body, err)
	}
}
