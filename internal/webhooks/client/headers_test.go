package client

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateHeaders(t *testing.T) {
	ok := []map[string]string{
		nil,
		{"Authorization": "Bearer x", "Content-Type": "application/x-www-form-urlencoded", "User-Agent": "acme", "X-Tenant": "a\tb"},
		{"X-Sparrow-Env": "prod"},
	}
	for _, h := range ok {
		if err := ValidateHeaders("headers", h); err != nil {
			t.Errorf("ValidateHeaders(%v) = %v, want ok", h, err)
		}
	}
	bad := map[string]map[string]string{
		"host":        {"host": "evil.example"},
		"length":      {"Content-Length": "1"},
		"te":          {"Transfer-Encoding": "chunked"},
		"crlf":        {"X-A": "v\r\nX-Injected: 1"},
		"lf":          {"X-A": "v\nX"},
		"bad name":    {"X A": "v"},
		"colon name":  {"X-A:": "v"},
		"empty name":  {"": "v"},
		"huge value":  {"X-A": strings.Repeat("a", maxHeaderValueBytes+1)},
		"connection":  {"connection": "close"},
		"upgrade":     {"Upgrade": "websocket"},
		"nul in name": {"X-\x00": "v"},
	}
	for name, h := range bad {
		if err := ValidateHeaders("headers", h); err == nil {
			t.Errorf("%s: ValidateHeaders(%q) = nil, want error", name, h)
		}
	}
}

func TestBuildRequestHeaderPrecedence(t *testing.T) {
	dr := &DeliveryRequest{
		Method:     "POST",
		URL:        "https://example.com/hook",
		Payload:    []byte(`{}`),
		EventID:    uuid.New(),
		WebhookID:  uuid.New(),
		DeliveryID: "d1",
		Headers: map[string]string{
			"Content-Type":          "text/plain",
			"X-Sparrow-Delivery-ID": "spoofed",
			"X-Sparrow-Env":         "prod",
			"Host":                  "stored-before-validation.example",
		},
	}
	req, err := BuildRequest(context.Background(), dr)
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("Content-Type"); got != "text/plain" {
		t.Errorf("Content-Type = %q, want custom override", got)
	}
	if got := req.Header.Get("X-Sparrow-Delivery-ID"); got != "d1" {
		t.Errorf("X-Sparrow-Delivery-ID = %q, want Sparrow's own value", got)
	}
	if got := req.Header.Get("X-Sparrow-Env"); got != "prod" {
		t.Errorf("custom X-Sparrow-Env dropped: %q", got)
	}
	if _, set := req.Header["Host"]; set || req.Host != "" && req.Host != "example.com" {
		t.Errorf("reserved Host header leaked into request: header=%v host=%q", req.Header["Host"], req.Host)
	}
}
