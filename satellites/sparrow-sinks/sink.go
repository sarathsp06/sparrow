package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/sarathsp06/sparrow/pkg/signature"
)

// envelope is the untransformed Sparrow delivery body. Sinks require it:
// subscriptions with transform_enabled rewrite the body into an arbitrary
// shape the sink cannot interpret.
type envelope struct {
	Version   string          `json:"version"`
	EventID   string          `json:"event_id"`
	EventName string          `json:"event_name"`
	Timestamp string          `json:"timestamp"`
	Attempt   int             `json:"attempt"`
	Payload   json.RawMessage `json:"payload"`
}

// deliverFunc is a sink's delivery function: non-nil error -> 502 -> Sparrow retries.
type deliverFunc func(ctx context.Context, env envelope, raw []byte) error

// sinkDef describes one sink: how to detect it in config, validate its
// section, and build its deliver func. Adding a sink = one file with a
// sinkDef plus one entry in sinkDefs; config validation, signature secrets,
// mounting, and logging all follow from it.
type sinkDef struct {
	name       string
	configured func(*config) bool
	validate   func(*config) error
	// secret returns the per-sink webhook_secret override ("" -> top-level).
	secret func(*config) string
	// build returns the deliver func and extra key/value pairs for the
	// "sink enabled" log line.
	build func(ctx context.Context, c *config) (deliverFunc, []any, error)
}

var sinkDefs = []sinkDef{emailDef, s3Def, otlpDef}

// sinkHandler wraps a sink's deliver func with Standard Webhooks signature
// verification and envelope parsing. Non-2xx responses make Sparrow retry the
// delivery, so sinks stay stateless.
func sinkHandler(name, secret string, logger *slog.Logger, deliver func(ctx context.Context, env envelope, raw []byte) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := signature.VerifyHMAC(raw, r.Header, secret); err != nil {
			logger.Warn("signature rejected", "sink", name, "error", err)
			http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
			return
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil || env.EventID == "" || env.EventName == "" {
			http.Error(w, "body is not a Sparrow envelope (event_id/event_name missing); sinks require the subscription to have transforms disabled", http.StatusUnprocessableEntity)
			return
		}
		if err := deliver(r.Context(), env, raw); err != nil {
			logger.Error("sink delivery failed", "sink", name, "event_id", env.EventID, "error", err)
			http.Error(w, "downstream failure: "+err.Error(), http.StatusBadGateway)
			return
		}
		logger.Info("delivered", "sink", name, "event_id", env.EventID, "event_name", env.EventName)
		w.WriteHeader(http.StatusOK)
	})
}

// templateContext exposes the envelope to text/template with the same
// snake_case keys Sparrow's server-side transform templates use.
func templateContext(env envelope) map[string]any {
	var payload any
	_ = json.Unmarshal(env.Payload, &payload) // nil payload renders as <no value>
	pretty, _ := json.MarshalIndent(payload, "", "  ")
	return map[string]any{
		"version":        env.Version,
		"event_id":       env.EventID,
		"event_name":     env.EventName,
		"timestamp":      env.Timestamp,
		"attempt":        env.Attempt,
		"payload":        payload,
		"payload_pretty": string(pretty),
	}
}
