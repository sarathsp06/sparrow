// Package stripe is the Stripe source: it verifies the Stripe-Signature
// header and republishes each Stripe event as a Sparrow event named
// <sparrow_event_prefix>.<stripe event type>.
package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources"
)

// tolerance is the max age of a Stripe-Signature timestamp.
const tolerance = 5 * time.Minute

// Config is the stripe: provider section of sources.yaml.
type Config struct {
	// Path is where the sparrow-sources binary mounts this provider; register
	// the matching URL in the Stripe dashboard. Default: /webhooks/stripe.
	// Handler ignores it: mount the handler wherever you like.
	Path string `yaml:"path"`
	// SigningSecret is the Stripe "whsec_..." secret used to verify the incoming
	// webhook's signature.
	SigningSecret string `yaml:"signing_secret"`
	// SparrowEventPrefix names the OUTBOUND event pushed to Sparrow (not the
	// incoming Stripe event): it is prefixed to the Stripe event type.
	// Default: stripe.
	SparrowEventPrefix string `yaml:"sparrow_event_prefix"`
}

// Handler returns the Stripe source: Stripe-Signature verified, each event
// pushed as <prefix>.<stripe event type> with source and livemode labels.
func Handler(cfg Config, p sources.Pusher, log *slog.Logger) (http.Handler, error) {
	if cfg.SigningSecret == "" {
		return nil, errors.New("stripe: signing_secret is required")
	}
	if cfg.SparrowEventPrefix == "" {
		cfg.SparrowEventPrefix = "stripe"
	}
	return sources.Handle("stripe", p, log, func(r *http.Request, body []byte) (string, map[string]string, error) {
		if err := verifySignature(r.Header.Get("Stripe-Signature"), body, cfg.SigningSecret, time.Now()); err != nil {
			return "", nil, sources.ErrBadSignature
		}
		return mapEvent(cfg.SparrowEventPrefix, body)
	}), nil
}

// verifySignature checks a Stripe-Signature header
// ("t=<unix>,v1=<hex hmac>") against HMAC-SHA256(secret, "{t}.{body}") with
// a 5-minute timestamp tolerance.
func verifySignature(header string, body []byte, secret string, now time.Time) error {
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
	if d := now.Sub(time.Unix(t, 0)); d > tolerance || d < -tolerance {
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

// mapEvent maps a Stripe event body to a Sparrow event name and labels:
// payment_intent.succeeded -> <prefix>.payment_intent.succeeded.
func mapEvent(prefix string, body []byte) (string, map[string]string, error) {
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
