package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type otlpConfig struct {
	WebhookSecret string            `yaml:"webhook_secret"` // optional per-sink override
	Endpoint      string            `yaml:"endpoint"`       // http(s)://host:4318; /v1/logs appended when no path
	Headers       map[string]string `yaml:"headers"`        // extra headers, e.g. Authorization
	ServiceName   string            `yaml:"service_name"`   // resource service.name, default "sparrow"
}

var otlpDef = sinkDef{
	name:       "otlp",
	configured: func(c *config) bool { return c.OTLP != nil },
	validate: func(c *config) error {
		if c.OTLP.Endpoint == "" {
			return fmt.Errorf("otlp: endpoint is required")
		}
		if _, err := logsURL(c.OTLP.Endpoint); err != nil {
			return fmt.Errorf("otlp: %w", err)
		}
		return nil
	},
	secret: func(c *config) string { return c.OTLP.WebhookSecret },
	build: func(_ context.Context, c *config) (deliverFunc, []any, error) {
		sink, err := newOTLPSink(*c.OTLP)
		if err != nil {
			return nil, nil, err
		}
		return sink.deliver, []any{"endpoint", sink.url}, nil
	},
}

// otlpSink exports each delivery as one OTLP/HTTP JSON log record, making
// Sparrow events visible in any OpenTelemetry-compatible backend (an OTel
// Collector, or a vendor's native OTLP endpoint).
type otlpSink struct {
	cfg    otlpConfig
	url    string
	client *http.Client
}

func newOTLPSink(cfg otlpConfig) (*otlpSink, error) {
	u, err := logsURL(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("otlp: %w", err)
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "sparrow"
	}
	return &otlpSink{cfg: cfg, url: u, client: &http.Client{Timeout: 15 * time.Second}}, nil
}

// logsURL normalizes the endpoint: a bare host/port gets the standard
// OTLP/HTTP logs path; an explicit path is kept verbatim.
func logsURL(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse endpoint: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("endpoint must be http(s), got %q", endpoint)
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/v1/logs"
	}
	return u.String(), nil
}

// otlpPayload builds an ExportLogsServiceRequest in OTLP/JSON: one log
// record per delivery, envelope fields as attributes, the event payload as
// the body.
// ponytail: hand-rolled OTLP/JSON keeps this stdlib-only; switch to the
// otlploghttp exporter if proto/gzip or structured (kvlist) bodies matter.
func otlpPayload(env envelope, serviceName string) ([]byte, error) {
	ts := time.Now()
	if t, err := time.Parse(time.RFC3339Nano, env.Timestamp); err == nil {
		ts = t
	}
	str := func(v string) map[string]any { return map[string]any{"stringValue": v} }
	attr := func(k string, v map[string]any) map[string]any { return map[string]any{"key": k, "value": v} }
	record := map[string]any{
		"timeUnixNano":   strconv.FormatInt(ts.UnixNano(), 10),
		"severityNumber": 9, // INFO
		"severityText":   "INFO",
		"body":           str(string(env.Payload)),
		"attributes": []map[string]any{
			attr("sparrow.event_id", str(env.EventID)),
			attr("sparrow.event_name", str(env.EventName)),
			attr("sparrow.attempt", map[string]any{"intValue": strconv.Itoa(env.Attempt)}),
		},
	}
	return json.Marshal(map[string]any{
		"resourceLogs": []map[string]any{{
			"resource": map[string]any{
				"attributes": []map[string]any{attr("service.name", str(serviceName))},
			},
			"scopeLogs": []map[string]any{{
				"scope":      map[string]any{"name": "sparrow-sinks"},
				"logRecords": []map[string]any{record},
			}},
		}},
	})
}

func (s *otlpSink) deliver(ctx context.Context, env envelope, _ []byte) error {
	body, err := otlpPayload(env, s.cfg.ServiceName)
	if err != nil {
		return fmt.Errorf("otlp: encode: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("otlp: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range s.cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("otlp: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	// ponytail: any 2xx counts as delivered; OTLP partialSuccess (a 200 that
	// rejected records) is ignored — inspect the response body if that bites.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("otlp: endpoint returned %d: %s", resp.StatusCode, msg)
	}
	return nil
}
