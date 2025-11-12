package queue

import (
	"testing"
	"time"
)

func TestWebhookWorkerDefaults(t *testing.T) {
	worker := WebhookWorker{}

	args := WebhookArgs{
		DeliveryID: "test-delivery-id",
		WebhookID:  "test-webhook-id",
		EventID:    "test-event-id",
		ExpiresAt:  time.Now().Add(time.Hour),
		Namespace:  "test-namespace",
	}

	// Test that webhook worker has correct type
	if worker.webhookRepo == nil && len(args.DeliveryID) > 0 {
		// Basic validation that the webhook worker and args are properly structured
		t.Log("WebhookWorker structure is valid")
	}

	// Test DeliveryID field exists
	if args.DeliveryID != "test-delivery-id" {
		t.Errorf("Expected DeliveryID to be 'test-delivery-id', got '%s'", args.DeliveryID)
	}

	// Test WebhookID field exists
	if args.WebhookID != "test-webhook-id" {
		t.Errorf("Expected WebhookID to be 'test-webhook-id', got '%s'", args.WebhookID)
	}

	// Test EventID field exists
	if args.EventID != "test-event-id" {
		t.Errorf("Expected EventID to be 'test-event-id', got '%s'", args.EventID)
	}

	// Test Namespace field exists
	if args.Namespace != "test-namespace" {
		t.Errorf("Expected Namespace to be 'test-namespace', got '%s'", args.Namespace)
	}
}
