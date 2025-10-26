package jobs

import "testing"

func TestWebhookArgsKind(t *testing.T) {
	args := WebhookArgs{URL: "https://example.com/webhook", Payload: `{"test": "data"}`}

	if args.Kind() != "webhook_delivery" {
		t.Errorf("Expected Kind() to return 'webhook_delivery', got '%s'", args.Kind())
	}
}
