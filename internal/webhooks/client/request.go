package client

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	sparrow "github.com/sarathsp06/sparrow"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
)

// DeliveryRequest represents the data needed to send a webhook
type DeliveryRequest struct {
	WebhookID         uuid.UUID
	DeliveryID        string
	URL               string
	Method            string
	Headers           map[string]string
	Payload           []byte
	Secret            string
	Ed25519PrivateKey []byte // Raw Ed25519 private key (64 bytes) for asymmetric signing
	Timeout           time.Duration
	RetryCount        int
	MaxRetries        int
	EventID           uuid.UUID
	EventName         string
	Namespace         string
}

// WebhookEnvelope is the default JSON body sent to webhook endpoints.
// All fields use snake_case JSON tags.
// Namespace, WebhookID, and DeliveryID are conveyed via X-Sparrow-* HTTP
// headers and are intentionally omitted from the body.
type WebhookEnvelope struct {
	Version   string         `json:"version"`
	EventID   string         `json:"event_id"`
	EventName string         `json:"event_name"`
	Timestamp string         `json:"timestamp"`
	Attempt   int            `json:"attempt"`
	Payload   map[string]any `json:"payload"`
}

// EnvelopeVersion is the current version of the webhook envelope schema.
const EnvelopeVersion = "1"

// BuildEnvelopePayload constructs the default webhook body as a JSON envelope.
// Namespace, WebhookID, and DeliveryID are not included in the body; they are
// sent as X-Sparrow-* HTTP headers instead.
func BuildEnvelopePayload(
	eventID string,
	eventName string,
	attempt int,
	payload map[string]any,
) ([]byte, error) {
	envelope := WebhookEnvelope{
		Version:   EnvelopeVersion,
		EventID:   eventID,
		EventName: eventName,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Attempt:   attempt,
		Payload:   payload,
	}
	return json.Marshal(envelope)
}

// BuildRequest creates an HTTP request from the delivery request
func BuildRequest(ctx context.Context, dr *DeliveryRequest) (*http.Request, error) {
	// Use bytes.NewReader for zero-copy payload reading
	// This avoids creating a new buffer while allowing http to read the payload
	req, err := http.NewRequestWithContext(ctx, dr.Method, dr.URL, bytes.NewReader(dr.Payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Sparrow-Webhook/"+sparrow.Version)
	req.Header.Set("X-Sparrow-Event-ID", dr.EventID.String())
	req.Header.Set("X-Sparrow-Delivery-ID", dr.DeliveryID)
	req.Header.Set("X-Sparrow-Webhook-ID", dr.WebhookID.String())
	// Set custom headers (overriding defaults if needed)
	for k, v := range dr.Headers {
		req.Header.Set(k, v)
	}

	// Add HMAC signature if secret is present
	if dr.Secret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		signature := generateHMACSignature(dr.Payload, dr.Secret, timestamp)
		req.Header.Set("X-Sparrow-Signature-256", "sha256="+signature)
		req.Header.Set("X-Sparrow-Timestamp", timestamp)

		// Add Ed25519 signature if private key is present (dual signing)
		if len(dr.Ed25519PrivateKey) == ed25519.PrivateKeySize {
			ed25519Sig := generateEd25519Signature(dr.Payload, dr.Ed25519PrivateKey, timestamp)
			req.Header.Set("X-Sparrow-Signature-Ed25519", ed25519Sig)
		}
	}

	return req, nil
}

// generateHMACSignature generates an HMAC-SHA256 signature
func generateHMACSignature(payload []byte, secret, timestamp string) string {
	message := timestamp + "." + string(payload)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// generateEd25519Signature signs the payload with an Ed25519 private key.
// Returns the hex-encoded signature. Uses the same message format as HMAC: "timestamp.payload".
func generateEd25519Signature(payload []byte, privateKey []byte, timestamp string) string {
	message := []byte(timestamp + "." + string(payload))
	sig := ed25519.Sign(ed25519.PrivateKey(privateKey), message)
	return hex.EncodeToString(sig)
}

// PrepareDeliveryRequest creates a DeliveryRequest from subscription and event data.
// If cryptoSvc is provided and the webhook has encrypted secret headers, they are
// decrypted and merged after regular + subscription headers (secret headers win).
func PrepareDeliveryRequest(
	webhook *store.WebhookRegistration,
	sub *store.EventSubscription,
	event *store.EventRecord,
	deliveryID string,
	payload []byte,
	cryptoSvc *crypto.Service,
) *DeliveryRequest {

	// Merge headers: subscription headers override webhook headers
	// Use header map from pool
	headers := GetHeaderMap()
	maps.Copy(headers, webhook.Headers)
	if sub != nil {
		maps.Copy(headers, sub.Headers)
	}

	// Decrypt and merge secret headers (override regular + subscription headers)
	if cryptoSvc != nil && len(webhook.SecretHeaders) > 0 {
		var secretHeaders map[string]string
		if err := cryptoSvc.DecryptJSON(webhook.SecretHeaders, &secretHeaders); err == nil {
			for k, v := range secretHeaders {
				headers[k] = v
			}
		}
		// On decryption failure, silently skip — the webhook still delivers
		// with regular headers. The error is non-fatal because the encryption
		// key may have been rotated or removed.
	}

	// Determine method
	method := http.MethodPost
	if sub != nil && sub.Method != "" {
		method = sub.Method
	}

	// Determine timeout
	timeout := time.Duration(webhook.RequestTimeoutSeconds) * time.Second
	if sub != nil && sub.Timeout > 0 {
		timeout = time.Duration(sub.Timeout) * time.Second
	}
	if timeout == 0 {
		timeout = 30 * time.Second // Default
	}

	// Decrypt webhook secret for HMAC signing.
	// If decryption fails (e.g. key rotated), the webhook still delivers
	// without a signature — same resilience as secret header decryption.
	var webhookSecret string
	if cryptoSvc != nil && len(webhook.WebhookSecret) > 0 {
		if decrypted, err := cryptoSvc.DecryptString(webhook.WebhookSecret); err == nil {
			webhookSecret = decrypted
		}
	}

	// Decrypt Ed25519 private key for asymmetric signing.
	var ed25519PrivateKey []byte
	if cryptoSvc != nil && len(webhook.Ed25519PrivateKey) > 0 {
		if decrypted, err := cryptoSvc.DecryptString(webhook.Ed25519PrivateKey); err == nil {
			ed25519PrivateKey = []byte(decrypted)
		}
	}

	return &DeliveryRequest{
		WebhookID:         webhook.ID,
		DeliveryID:        deliveryID,
		URL:               webhook.URL,
		Method:            method,
		Headers:           headers,
		Payload:           payload,
		Secret:            webhookSecret,
		Ed25519PrivateKey: ed25519PrivateKey,
		Timeout:           timeout,
		RetryCount:        0, // Initial attempt
		MaxRetries:        webhook.MaxRetries,
		EventID:           event.ID,
		EventName:         event.Event,
		Namespace:         event.Namespace,
	}
}
