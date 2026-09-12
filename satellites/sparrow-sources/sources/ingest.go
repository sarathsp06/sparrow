package sources

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxIngestBody = 1 << 20 // 1 MiB

// stripeTolerance is the max age of a Stripe-Signature timestamp.
const stripeTolerance = 5 * time.Minute

// NewIngestHandler returns the ingest HTTP handler: POST /ingest/stripe and
// POST /ingest/github for configured providers. Unknown paths 404, bad
// signatures 401, push failures 502 (so the provider retries).
func NewIngestHandler(cfg IngestConfig, p Pusher, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	if s := cfg.Providers.Stripe; s != nil {
		mux.HandleFunc("POST /ingest/stripe", func(w http.ResponseWriter, r *http.Request) {
			handleIngest(w, r, p, log, "stripe", func(body []byte) (string, map[string]string, error) {
				if err := verifyStripeSignature(r.Header.Get("Stripe-Signature"), body, s.SigningSecret, time.Now()); err != nil {
					return "", nil, errBadSignature
				}
				return mapStripeEvent(s.EventPrefix, body)
			})
		})
	}
	if g := cfg.Providers.GitHub; g != nil {
		mux.HandleFunc("POST /ingest/github", func(w http.ResponseWriter, r *http.Request) {
			handleIngest(w, r, p, log, "github", func(body []byte) (string, map[string]string, error) {
				if !verifyGitHubSignature(r.Header.Get("X-Hub-Signature-256"), body, g.Secret) {
					return "", nil, errBadSignature
				}
				return mapGitHubEvent(g.EventPrefix, r.Header.Get("X-GitHub-Event"), body)
			})
		})
	}
	return mux
}

var errBadSignature = fmt.Errorf("bad signature")

// handleIngest reads the body, verifies+maps it via fn, pushes the event,
// and only then responds 2xx (at-least-once: the provider retries non-2xx).
func handleIngest(w http.ResponseWriter, r *http.Request, p Pusher, log *slog.Logger, provider string, fn func(body []byte) (string, map[string]string, error)) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxIngestBody))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	event, labels, err := fn(body)
	if err != nil {
		if err == errBadSignature {
			w.WriteHeader(http.StatusUnauthorized) // no body echo
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := p.PushEvent(r.Context(), event, json.RawMessage(body), labels); err != nil {
		log.Error("ingest push failed", "provider", provider, "event", event, "error", err)
		http.Error(w, "push failed", http.StatusBadGateway)
		return
	}
	log.Info("ingest event pushed", "provider", provider, "event", event)
	w.WriteHeader(http.StatusOK)
}

// verifyStripeSignature checks a Stripe-Signature header
// ("t=<unix>,v1=<hex hmac>") against HMAC-SHA256(secret, "{t}.{body}") with
// a 5-minute timestamp tolerance.
func verifyStripeSignature(header string, body []byte, secret string, now time.Time) error {
	var ts string
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			ts = v
		case "v1":
			sigs = append(sigs, v)
		}
	}
	if ts == "" || len(sigs) == 0 {
		return fmt.Errorf("missing t or v1")
	}
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("bad timestamp")
	}
	if d := now.Sub(time.Unix(t, 0)); d > stripeTolerance || d < -stripeTolerance {
		return fmt.Errorf("timestamp outside tolerance")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, sig := range sigs {
		if hmac.Equal([]byte(expected), []byte(sig)) {
			return nil
		}
	}
	return fmt.Errorf("no matching v1 signature")
}

// mapStripeEvent maps a Stripe event body to a Sparrow event name and labels:
// payment_intent.succeeded -> <prefix>.payment_intent.succeeded.
func mapStripeEvent(prefix string, body []byte) (string, map[string]string, error) {
	var evt struct {
		Type     string `json:"type"`
		Livemode bool   `json:"livemode"`
	}
	if err := json.Unmarshal(body, &evt); err != nil || evt.Type == "" {
		return "", nil, fmt.Errorf("invalid stripe event body")
	}
	return prefix + "." + evt.Type, map[string]string{
		"source":   "stripe",
		"livemode": strconv.FormatBool(evt.Livemode),
	}, nil
}

// verifyGitHubSignature checks an X-Hub-Signature-256 header
// ("sha256=<hex hmac>") against HMAC-SHA256(secret, body).
func verifyGitHubSignature(header string, body []byte, secret string) bool {
	sig, ok := strings.CutPrefix(header, "sha256=")
	if !ok {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

// mapGitHubEvent maps X-GitHub-Event (+ body "action" when present) to
// <prefix>.<event>[.<action>], with source/repo labels.
func mapGitHubEvent(prefix, ghEvent string, body []byte) (string, map[string]string, error) {
	if ghEvent == "" {
		return "", nil, fmt.Errorf("missing X-GitHub-Event header")
	}
	var evt struct {
		Action     string `json:"action"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &evt); err != nil {
		return "", nil, fmt.Errorf("invalid github event body")
	}
	name := prefix + "." + ghEvent
	if evt.Action != "" {
		name += "." + evt.Action
	}
	labels := map[string]string{"source": "github"}
	if evt.Repository.FullName != "" {
		labels["repo"] = evt.Repository.FullName
	}
	return name, labels, nil
}
