package workers

import (
	"testing"

	"github.com/sarathsp06/httpqueue/internal/jobs"
)

func TestWebhookWorkerDefaults(t *testing.T) {
	worker := WebhookWorker{}

	args := jobs.WebhookArgs{
		URL:     "https://example.com",
		Payload: map[string]interface{}{"test": "data"},
	}

	// Test default method
	method := worker.getMethod(args)
	if method != "POST" {
		t.Errorf("Expected default method to be 'POST', got '%s'", method)
	}

	// Test custom method
	args.Method = "PUT"
	method = worker.getMethod(args)
	if method != "PUT" {
		t.Errorf("Expected method to be 'PUT', got '%s'", method)
	}

	// Test default timeout
	args.Timeout = 0
	timeout := worker.getTimeout(args)
	if timeout != 30 {
		t.Errorf("Expected default timeout to be 30, got %d", timeout)
	}

	// Test custom timeout
	args.Timeout = 10
	timeout = worker.getTimeout(args)
	if timeout != 10 {
		t.Errorf("Expected timeout to be 10, got %d", timeout)
	}
}
