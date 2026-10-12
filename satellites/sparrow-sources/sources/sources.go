// Package sources is the core of sparrow-sources: a Pusher that publishes
// events to Sparrow, and Handle, which turns a provider's webhooks into
// Sparrow events. The built-in sources live in subpackages (stripe, github,
// cron), one per source so each keeps its own dependencies; a custom webhook
// source is one MapFunc passed to Handle. This package imports only the
// standard library.
package sources

import (
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

const maxWebhookBody = 1 << 20 // 1 MiB

// SparrowConfig is the Sparrow server connection (the sparrow: section).
type SparrowConfig struct {
	URL      string `yaml:"url"`
	APIKey   string `yaml:"api_key"`
	Consumer string `yaml:"consumer"` // default "default"
}

// ErrBadSignature is what a MapFunc returns when the request is not signed
// by the provider; Handle answers 401 without echoing anything back.
var ErrBadSignature = errors.New("bad signature")

// MapFunc verifies one provider request and maps it to the Sparrow event to
// push: its name and labels. The raw body becomes the event payload. Return
// ErrBadSignature for an unsigned request (401); any other error answers 400.
type MapFunc func(r *http.Request, body []byte) (event string, labels map[string]string, err error)

// Handle turns provider webhooks into Sparrow events: it reads the body,
// verifies and maps it with mapFn, pushes the event, and only then answers
// 2xx. A failed push answers 502 so the provider retries (at-least-once).
// Every built-in provider is built on it, and so is a custom one. name only
// labels log lines; a nil log means slog.Default().
func Handle(name string, p Pusher, log *slog.Logger, mapFn MapFunc) http.Handler {
	log = cmp.Or(log, slog.Default())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		event, labels, err := mapFn(r, body)
		if err != nil {
			if errors.Is(err, ErrBadSignature) {
				w.WriteHeader(http.StatusUnauthorized) // no body echo
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := p.PushEvent(r.Context(), event, json.RawMessage(body), labels); err != nil {
			log.Error("webhook push failed", "provider", name, "event", event, "error", err)
			http.Error(w, "push failed", http.StatusBadGateway)
			return
		}
		log.Info("webhook event pushed", "provider", name, "event", event)
		w.WriteHeader(http.StatusOK)
	})
}
