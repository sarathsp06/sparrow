package client

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	sparrow "github.com/sarathsp06/sparrow"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
)

func TestBuildRequest(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.New()
	webhookID := uuid.New()

	dr := &DeliveryRequest{
		WebhookID:  webhookID,
		DeliveryID: "delivery-123",
		URL:        "https://example.com/webhook",
		Method:     "POST",
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
		Payload:   []byte(`{"test": "data"}`),
		Secret:    "my-secret",
		EventID:   eventID,
		EventName: "user.created",
		Namespace: "default",
	}

	req, err := BuildRequest(ctx, dr)
	if err != nil {
		t.Fatalf("Unexpected error building request: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method)
	}

	if req.URL.String() != "https://example.com/webhook" {
		t.Errorf("Expected URL https://example.com/webhook, got %s", req.URL.String())
	}

	// Check default headers
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}

	expectedUA := "Sparrow-Webhook/" + sparrow.Version
	if req.Header.Get("User-Agent") != expectedUA {
		t.Errorf("Expected User-Agent %s, got %s", expectedUA, req.Header.Get("User-Agent"))
	}

	if req.Header.Get("X-Sparrow-Event-ID") != eventID.String() {
		t.Errorf("Expected X-Sparrow-Event-ID %s, got %s", eventID.String(), req.Header.Get("X-Sparrow-Event-ID"))
	}

	if req.Header.Get("X-Sparrow-Delivery-ID") != "delivery-123" {
		t.Errorf("Expected X-Sparrow-Delivery-ID delivery-123, got %s", req.Header.Get("X-Sparrow-Delivery-ID"))
	}

	if req.Header.Get("X-Sparrow-Webhook-ID") != webhookID.String() {
		t.Errorf("Expected X-Sparrow-Webhook-ID %s, got %s", webhookID.String(), req.Header.Get("X-Sparrow-Webhook-ID"))
	}

	// Check custom header
	if req.Header.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("Expected X-Custom-Header custom-value, got %s", req.Header.Get("X-Custom-Header"))
	}

	// Check Standard Webhooks signature headers
	if req.Header.Get("webhook-id") == "" {
		t.Error("Expected webhook-id header to be set")
	}

	if req.Header.Get("webhook-timestamp") == "" {
		t.Error("Expected webhook-timestamp header to be set")
	}

	sig := req.Header.Get("webhook-signature")
	if sig == "" {
		t.Error("Expected webhook-signature header to be set")
	}

	// HMAC signature should be present with v1 prefix
	if !strings.HasPrefix(sig, "v1,") {
		t.Errorf("Expected webhook-signature to start with v1, got %s", sig)
	}
}

func TestBuildRequestWithoutSecret(t *testing.T) {
	ctx := context.Background()

	dr := &DeliveryRequest{
		WebhookID:  uuid.New(),
		DeliveryID: "delivery-123",
		URL:        "https://example.com/webhook",
		Method:     "POST",
		Payload:    []byte(`{"test": "data"}`),
		Secret:     "", // No secret
		EventID:    uuid.New(),
	}

	req, err := BuildRequest(ctx, dr)
	if err != nil {
		t.Fatalf("Unexpected error building request: %v", err)
	}

	// Should not have Standard Webhooks signature headers
	if req.Header.Get("webhook-signature") != "" {
		t.Error("Expected no webhook-signature header without secret")
	}

	if req.Header.Get("webhook-timestamp") != "" {
		t.Error("Expected no webhook-timestamp header without secret")
	}
}

func TestBuildRequestInvalidURL(t *testing.T) {
	ctx := context.Background()

	dr := &DeliveryRequest{
		WebhookID:  uuid.New(),
		DeliveryID: "delivery-123",
		URL:        "://invalid-url",
		Method:     "POST",
		Payload:    []byte(`{"test": "data"}`),
		EventID:    uuid.New(),
	}

	_, err := BuildRequest(ctx, dr)
	if err == nil {
		t.Error("Expected error building request with invalid URL")
	}
}

func TestGenerateHMACSignature(t *testing.T) {
	payload := []byte(`{"test": "data"}`)
	secret := "my-secret"
	msgID := "msg_delivery-123"
	timestamp := "1234567890"

	sig1, err := generateHMACSignature(payload, secret, msgID, timestamp)
	if err != nil {
		t.Fatalf("generateHMACSignature failed: %v", err)
	}
	sig2, err := generateHMACSignature(payload, secret, msgID, timestamp)
	if err != nil {
		t.Fatalf("generateHMACSignature failed: %v", err)
	}

	if sig1 != sig2 {
		t.Error("Expected consistent signature for same inputs")
	}

	if sig1 == "" {
		t.Error("Expected non-empty signature")
	}

	// Different timestamp should produce different signature
	sig3, err := generateHMACSignature(payload, secret, msgID, "9999999999")
	if err != nil {
		t.Fatalf("generateHMACSignature failed: %v", err)
	}
	if sig1 == sig3 {
		t.Error("Expected different signature for different timestamp")
	}

	// Different payload should produce different signature
	sig4, err := generateHMACSignature([]byte(`{"different": "payload"}`), secret, msgID, timestamp)
	if err != nil {
		t.Fatalf("generateHMACSignature failed: %v", err)
	}
	if sig1 == sig4 {
		t.Error("Expected different signature for different payload")
	}

	// Different msgID should produce different signature
	sig5, err := generateHMACSignature(payload, secret, "msg_other-id", timestamp)
	if err != nil {
		t.Fatalf("generateHMACSignature failed: %v", err)
	}
	if sig1 == sig5 {
		t.Error("Expected different signature for different msgID")
	}
}

func TestPrepareDeliveryRequest(t *testing.T) {
	webhookID := uuid.New()
	eventID := uuid.New()

	// Create a crypto service for encrypting/decrypting the webhook secret
	_, key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	cryptoSvc, err := crypto.NewService(key)
	if err != nil {
		t.Fatalf("Failed to create crypto service: %v", err)
	}

	// Encrypt the webhook secret (simulates what the service layer does on register)
	encryptedSecret, err := cryptoSvc.EncryptString("secret123")
	if err != nil {
		t.Fatalf("Failed to encrypt secret: %v", err)
	}

	webhook := &store.WebhookRegistration{
		ID:                    webhookID,
		URL:                   "https://example.com/webhook",
		WebhookSecret:         encryptedSecret,
		RequestTimeoutSeconds: 15,
		MaxRetries:            3,
		Headers: map[string]string{
			"X-Webhook-Header": "webhook-value",
		},
	}

	sub := &store.EventSubscription{
		Method:  "POST",
		Timeout: 20,
		Headers: map[string]string{
			"X-Sub-Header":     "sub-value",
			"X-Webhook-Header": "override-value", // Should override webhook header
		},
	}

	event := &store.EventRecord{
		ID:        eventID,
		Event:     "user.created",
		Namespace: "default",
	}

	payload := []byte(`{"user_id": "123"}`)
	deliveryID := "delivery-456"

	dr := PrepareDeliveryRequest(webhook, sub, event, deliveryID, payload, cryptoSvc)

	if dr.WebhookID != webhookID {
		t.Errorf("Expected WebhookID %s, got %s", webhookID, dr.WebhookID)
	}

	if dr.DeliveryID != deliveryID {
		t.Errorf("Expected DeliveryID %s, got %s", deliveryID, dr.DeliveryID)
	}

	if dr.URL != webhook.URL {
		t.Errorf("Expected URL %s, got %s", webhook.URL, dr.URL)
	}

	if dr.Method != "POST" {
		t.Errorf("Expected Method POST, got %s", dr.Method)
	}

	if dr.Secret != "secret123" {
		t.Errorf("Expected Secret 'secret123', got %s", dr.Secret)
	}

	if dr.Timeout != 20*time.Second {
		t.Errorf("Expected Timeout 20s (from subscription), got %v", dr.Timeout)
	}

	if dr.MaxRetries != webhook.MaxRetries {
		t.Errorf("Expected MaxRetries %d, got %d", webhook.MaxRetries, dr.MaxRetries)
	}

	if dr.EventID != eventID {
		t.Errorf("Expected EventID %s, got %s", eventID, dr.EventID)
	}

	if dr.EventName != event.Event {
		t.Errorf("Expected EventName %s, got %s", event.Event, dr.EventName)
	}

	// Check header merging - subscription headers should override
	if dr.Headers["X-Webhook-Header"] != "override-value" {
		t.Errorf("Expected X-Webhook-Header override-value, got %s", dr.Headers["X-Webhook-Header"])
	}

	if dr.Headers["X-Sub-Header"] != "sub-value" {
		t.Errorf("Expected X-Sub-Header sub-value, got %s", dr.Headers["X-Sub-Header"])
	}
}

func TestPrepareDeliveryRequestWithoutSubscription(t *testing.T) {
	webhookID := uuid.New()
	eventID := uuid.New()

	webhook := &store.WebhookRegistration{
		ID:                    webhookID,
		URL:                   "https://example.com/webhook",
		RequestTimeoutSeconds: 15,
		MaxRetries:            3,
		Headers: map[string]string{
			"X-Webhook-Header": "webhook-value",
		},
	}

	event := &store.EventRecord{
		ID:        eventID,
		Event:     "user.created",
		Namespace: "default",
	}

	payload := []byte(`{"user_id": "123"}`)
	deliveryID := "delivery-456"

	dr := PrepareDeliveryRequest(webhook, nil, event, deliveryID, payload, nil)

	if dr.Method != http.MethodPost {
		t.Errorf("Expected default Method POST, got %s", dr.Method)
	}

	if dr.Timeout != 15*time.Second {
		t.Errorf("Expected Timeout 15s (from webhook), got %v", dr.Timeout)
	}

	// Check only webhook headers are present
	if dr.Headers["X-Webhook-Header"] != "webhook-value" {
		t.Errorf("Expected X-Webhook-Header webhook-value, got %s", dr.Headers["X-Webhook-Header"])
	}
}

func TestPrepareDeliveryRequestDefaultTimeout(t *testing.T) {
	webhook := &store.WebhookRegistration{
		ID:                    uuid.New(),
		URL:                   "https://example.com/webhook",
		RequestTimeoutSeconds: 0, // Zero timeout
	}

	event := &store.EventRecord{
		ID:        uuid.New(),
		Event:     "user.created",
		Namespace: "default",
	}

	dr := PrepareDeliveryRequest(webhook, nil, event, "delivery-123", []byte(`{}`), nil)

	if dr.Timeout != 30*time.Second {
		t.Errorf("Expected default Timeout 30s, got %v", dr.Timeout)
	}
}
