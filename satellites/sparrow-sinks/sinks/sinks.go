// Package sinks is the core of sparrow-sinks: it turns a DeliverFunc into an
// http.Handler that verifies the Standard Webhooks signature and parses the
// Sparrow envelope. The built-in sinks live in subpackages (email, otlp, s3),
// each a Config plus a Handler you can mount on your own server; a custom
// sink is one DeliverFunc passed to Handle. This package imports only the
// standard library and pkg/signature.
//
// Sinks are stateless: any downstream failure answers non-2xx so Sparrow's
// retry machinery redelivers.
package sinks

import (
	"cmp"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/sarathsp06/sparrow/pkg/signature"
)

// Envelope is the untransformed Sparrow delivery body. Sinks require it:
// subscriptions with transform_enabled rewrite the body into an arbitrary
// shape a sink cannot interpret.
type Envelope struct {
	Version   string          `json:"version"`
	EventID   string          `json:"event_id"`
	EventName string          `json:"event_name"`
	Timestamp string          `json:"timestamp"`
	Attempt   int             `json:"attempt"`
	Payload   json.RawMessage `json:"payload"`
}

// Verifier checks a delivery's Standard Webhooks signature against the raw
// body and headers. Build one with HMAC or Ed25519.
type Verifier func(raw []byte, h http.Header) error

// HMAC verifies the "v1," HMAC-SHA256 signature with the webhook secret
// ("whsec_..." or raw). Every Sparrow delivery carries it.
func HMAC(secret string) (Verifier, error) {
	if secret == "" {
		return nil, signature.ErrEmptySecret
	}
	return func(raw []byte, h http.Header) error { return signature.VerifyHMAC(raw, h, secret) }, nil
}

// Ed25519 verifies the "v1a," Ed25519 signature with the hex public key from
// the webhook's signing_public_key field (webhooks with signature_type
// ed25519).
func Ed25519(publicKeyHex string) (Verifier, error) {
	key, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("signing_public_key: decode hex: %w", err)
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("signing_public_key: must be %d bytes, got %d", ed25519.PublicKeySize, len(key))
	}
	return func(raw []byte, h http.Header) error { return signature.VerifyEd25519(raw, h, publicKeyHex) }, nil
}

// DeliverFunc forwards one delivery. raw is the exact signed body. A non-nil
// error answers 502 so Sparrow retries.
type DeliverFunc func(ctx context.Context, env Envelope, raw []byte) error

// Handle wraps deliver with signature verification (verify) and envelope
// parsing: 401 on a bad signature, 422 on a non-envelope body, 502
// when deliver fails, 200 otherwise. Every built-in sink is built on it, and
// so is a custom one. name only labels log lines; a nil logger means
// slog.Default().
func Handle(name string, verify Verifier, logger *slog.Logger, deliver DeliverFunc) http.Handler {
	logger = cmp.Or(logger, slog.Default())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := verify(raw, r.Header); err != nil {
			logger.Warn("signature rejected", "sink", name, "error", err)
			http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
			return
		}
		var env Envelope
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

// VerifierFor picks the verifier the built-in sinks use: Ed25519 when
// publicKeyHex is set, else HMAC with secret. Both empty is an error.
func VerifierFor(secret, publicKeyHex string) (Verifier, error) {
	switch {
	case publicKeyHex != "":
		return Ed25519(publicKeyHex)
	case secret != "":
		return HMAC(secret)
	default:
		return nil, errors.New("webhook_secret or signing_public_key is required (top-level or per-sink)")
	}
}
