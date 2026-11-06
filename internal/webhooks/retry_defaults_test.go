package webhooks

import (
	"testing"
	"time"
)

// TestDefaultRetryWindowSpansDays pins the intent behind the retry defaults: a
// receiver that is down for a weekend still gets its deliveries. The schedule
// mirrors queue.WebhookWorker.NextRetry (base * 2^(N-1), capped at 24 hours);
// jitter only shortens it, by at most 20%.
func TestDefaultRetryWindowSpansDays(t *testing.T) {
	cfg := DefaultWebhookHTTPConfig()
	if err := cfg.ValidateConfig(); err != nil {
		t.Fatalf("default config does not validate: %v", err)
	}

	var window time.Duration
	for attempt := 1; attempt <= *cfg.MaxRetries; attempt++ {
		delay := time.Duration(cfg.RetryBackoffSeconds) * time.Second << uint(attempt-1)
		window += min(delay, 24*time.Hour)
	}

	minWindow := window * 8 / 10 // largest jitter on every delay
	if minWindow < 48*time.Hour {
		t.Errorf("default retries can end after %v, want at least two days", minWindow)
	}
	if window > 4*24*time.Hour {
		t.Errorf("default retries last up to %v, want under four days", window)
	}
}
