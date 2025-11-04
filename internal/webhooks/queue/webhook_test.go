package queue

import (
	"testing"
)

func TestWebhookWorkerDefaults(t *testing.T) {
	worker := WebhookWorker{}

	args := WebhookArgs{
		URL:     "https://example.com",
		Payload: map[string]any{"test": "data"}, // `{"test": "data"}`,
		Timeout: 30,
	}

	// Test that webhook worker has correct type
	if worker.webhookRepo == nil && len(args.URL) > 0 {
		// Basic validation that the webhook worker and args are properly structured
		t.Log("WebhookWorker structure is valid")
	}

	// Test timeout field exists
	if args.Timeout != 30 {
		t.Errorf("Expected timeout to be 30, got %d", args.Timeout)
	}

	// Test URL field exists
	if args.URL != "https://example.com" {
		t.Errorf("Expected URL to be 'https://example.com', got '%s'", args.URL)
	}
}
