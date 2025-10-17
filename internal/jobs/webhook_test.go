package jobs

import "testing"

func TestWebhookArgsKind(t *testing.T) {
	args := WebhookArgs{URL: "https://example.com/webhook", Payload: map[string]interface{}{"test": "data"}}

	if args.Kind() != "webhook" {
		t.Errorf("Expected Kind() to return 'webhook', got '%s'", args.Kind())
	}
}
