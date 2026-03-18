package client

import (
	"bytes"
	"context"
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

	"github.com/sarathsp06/sparrow"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// DeliveryRequest represents the data needed to send a webhook
type DeliveryRequest struct {
	WebhookID  uuid.UUID
	DeliveryID string
	URL        string
	Method     string
	Headers    map[string]string
	Payload    []byte
	Secret     string
	Timeout    time.Duration
	RetryCount int
	MaxRetries int
	EventID    uuid.UUID
	EventName  string
	Namespace  string
}

// WebhookEnvelope is the default JSON body sent to webhook endpoints.
// All fields use snake_case JSON tags.
type WebhookEnvelope struct {
	Version    string         `json:"version"`
	EventID    string         `json:"event_id"`
	EventName  string         `json:"event_name"`
	Namespace  string         `json:"namespace"`
	WebhookID  string         `json:"webhook_id"`
	DeliveryID string         `json:"delivery_id"`
	Timestamp  string         `json:"timestamp"`
	Attempt    int            `json:"attempt"`
	Payload    map[string]any `json:"payload"`
}

// EnvelopeVersion is the current version of the webhook envelope schema.
const EnvelopeVersion = "1"

// BuildEnvelopePayload constructs the default webhook body as a JSON envelope.
func BuildEnvelopePayload(
	eventID string,
	eventName string,
	namespace string,
	webhookID string,
	deliveryID string,
	attempt int,
	payload map[string]any,
) ([]byte, error) {
	envelope := WebhookEnvelope{
		Version:    EnvelopeVersion,
		EventID:    eventID,
		EventName:  eventName,
		Namespace:  namespace,
		WebhookID:  webhookID,
		DeliveryID: deliveryID,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Attempt:    attempt,
		Payload:    payload,
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

	// Add signature if secret is present
	if dr.Secret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		signature := generateHMACSignature(dr.Payload, dr.Secret, timestamp)
		req.Header.Set("X-Sparrow-Signature-256", "sha256="+signature)
		req.Header.Set("X-Sparrow-Timestamp", timestamp)
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

// PrepareDeliveryRequest creates a DeliveryRequest from subscription and event data
func PrepareDeliveryRequest(
	webhook *store.WebhookRegistration,
	sub *store.EventSubscription,
	event *store.EventRecord,
	deliveryID string,
	payload []byte,
) *DeliveryRequest {

	// Merge headers: subscription headers override webhook headers
	// Use header map from pool
	headers := GetHeaderMap()
	maps.Copy(headers, webhook.Headers)
	if sub != nil {
		maps.Copy(headers, sub.Headers)
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

	return &DeliveryRequest{
		WebhookID:  webhook.ID,
		DeliveryID: deliveryID,
		URL:        webhook.URL,
		Method:     method,
		Headers:    headers,
		Payload:    payload,
		Secret:     webhook.WebhookSecret,
		Timeout:    timeout,
		RetryCount: 0, // Initial attempt
		MaxRetries: webhook.MaxRetries,
		EventID:    event.ID,
		EventName:  event.Event,
		Namespace:  event.Namespace,
	}
}
