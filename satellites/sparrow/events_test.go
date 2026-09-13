package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEventsList(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[
			{"name":"user.signup","description":"a user signed up","active":true},
			{"name":"order.created","description":"an order","active":false}
		]}`))
	}))
	defer srv.Close()
	pushEnv(t, srv)

	var out bytes.Buffer
	root := newRootCmd(&out)
	root.SetArgs([]string{"events"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if gotPath != "/v1/event-types" {
		t.Errorf("path = %s", gotPath)
	}
	if !strings.Contains(gotQuery, "limit=200") {
		t.Errorf("query = %s, want limit", gotQuery)
	}
	s := out.String()
	// Sorted by name: order.created before user.signup.
	if i, j := strings.Index(s, "order.created"), strings.Index(s, "user.signup"); i < 0 || j < 0 || i > j {
		t.Errorf("names missing or unsorted:\n%s", s)
	}
	if !strings.Contains(s, "no ") || !strings.Contains(s, "yes ") {
		t.Errorf("active flags not rendered:\n%s", s)
	}
}

func TestEventsDetailShowsSchema(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"order.created","description":"an order","active":true,
			"event_schema":{"type":"object","properties":{"order_id":{"type":"string"}}},
			"sample_payload":{"order_id":"abc"}}`))
	}))
	defer srv.Close()
	pushEnv(t, srv)

	var out bytes.Buffer
	root := newRootCmd(&out)
	root.SetArgs([]string{"events", "order.created"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if gotPath != "/v1/event-types/order.created" {
		t.Errorf("path = %s", gotPath)
	}
	s := out.String()
	for _, want := range []string{"schema:", "order_id", "sample payload:", `"abc"`} {
		if !strings.Contains(s, want) {
			t.Errorf("detail missing %q:\n%s", want, s)
		}
	}
}
