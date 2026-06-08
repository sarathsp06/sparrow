package client

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
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
	SignatureType     string // "hmac" or "ed25519" — controls which signing scheme is used
	Timeout           time.Duration
	RetryCount        int
	MaxRetries        int
	EventID           uuid.UUID
	EventName         string
	Namespace         string
}

// WebhookEnvelope is the default JSON body sent to webhook endpoints.
// All fields use snake_case JSON tags.
// Namespace, WebhookID, and DeliveryID are conveyed via HTTP headers
// and are intentionally omitted from the body.
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
// sent as HTTP headers instead.
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

	// Standard Webhooks signing (https://www.standardwebhooks.com)
	// Uses webhook-id, webhook-timestamp, webhook-signature headers.
	// Message to sign: "{webhook-id}.{timestamp}.{payload}"
	// Only the scheme selected by SignatureType is used:
	//   "hmac" (default) -> v1, prefix (HMAC-SHA256)
	//   "ed25519"        -> v1a, prefix (Ed25519)
	if dr.Secret != "" || len(dr.Ed25519PrivateKey) > 0 {
		msgID := "msg_" + dr.DeliveryID
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		req.Header.Set("webhook-id", msgID)
		req.Header.Set("webhook-timestamp", timestamp)

		var signatures []string

		// Always include HMAC signature if secret is present
		if dr.Secret != "" {
			sig, err := generateHMACSignature(dr.Payload, dr.Secret, msgID, timestamp)
			if err == nil {
				signatures = append(signatures, "v1,"+sig)
			}
		}

		// Include Ed25519 signature if private key is present
		if len(dr.Ed25519PrivateKey) == ed25519.PrivateKeySize {
			signatures = append(signatures, "v1a,"+generateEd25519Signature(dr.Payload, dr.Ed25519PrivateKey, msgID, timestamp))
		}

		if len(signatures) > 0 {
			req.Header.Set("webhook-signature", strings.Join(signatures, " "))
		}
	}

	return req, nil
}

// generateHMACSignature generates an HMAC-SHA256 signature per Standard Webhooks spec.
// Signs "{msgID}.{timestamp}.{payload}" and returns base64-encoded signature.
// It handles Standard Webhook secrets (starts with whsec_ prefix and base64 encoded).
func generateHMACSignature(payload []byte, secret, msgID, timestamp string) (string, error) {
	rawSecret := []byte(secret)

	// Standard Webhooks: "The secret is a base64 encoded string.
	// A secret is a string that starts with whsec_ followed by a base64 encoded string."
	if strings.HasPrefix(secret, "whsec_") {
		var err error
		rawSecret, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
		if err != nil {
			return "", fmt.Errorf("failed to decode Standard Webhook secret: %w", err)
		}
	}

	h := hmac.New(sha256.New, rawSecret)
	h.Write([]byte(msgID))
	h.Write([]byte("."))
	h.Write([]byte(timestamp))
	h.Write([]byte("."))
	h.Write(payload)
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

// generateEd25519Signature signs the payload with an Ed25519 private key per Standard Webhooks spec.
// Signs "{msgID}.{timestamp}.{payload}" and returns base64-encoded signature.
func generateEd25519Signature(payload []byte, privateKey []byte, msgID, timestamp string) string {
	var buf bytes.Buffer
	buf.WriteString(msgID)
	buf.WriteString(".")
	buf.WriteString(timestamp)
	buf.WriteString(".")
	buf.Write(payload)

	sig := ed25519.Sign(ed25519.PrivateKey(privateKey), buf.Bytes())
	return base64.StdEncoding.EncodeToString(sig)
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
		SignatureType:     string(webhook.SignatureType),
		Timeout:           timeout,
		RetryCount:        0, // Initial attempt
		MaxRetries:        webhook.MaxRetries,
		EventID:           event.ID,
		EventName:         event.Event,
		Namespace:         event.Namespace,
	}
}
