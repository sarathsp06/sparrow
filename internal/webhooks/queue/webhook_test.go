package queue

import (
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func TestWebhookWorkerNextRetry(t *testing.T) {
	worker := &WebhookWorker{}

	job := func(backoffSeconds, attempt int) *river.Job[WebhookArgs] {
		return &river.Job[WebhookArgs]{
			JobRow: &rivertype.JobRow{Attempt: attempt},
			Args:   WebhookArgs{RetryBackoffSeconds: backoffSeconds},
		}
	}

	// Zero base (pre-existing jobs) defers to River's default policy.
	if got := worker.NextRetry(job(0, 1)); !got.IsZero() {
		t.Errorf("expected zero time for unset backoff, got %v", got)
	}

	// Exponential doubling: base * 2^(attempt-1) is the maximum; jitter takes
	// off up to 20%, never more, never adds.
	cases := []struct {
		base, attempt int
		max           time.Duration
	}{
		{60, 1, 60 * time.Second},
		{60, 2, 120 * time.Second},
		{60, 4, 480 * time.Second},
		{3600, 10, maxRetryDelay}, // capped
	}
	for _, c := range cases {
		minDelay := c.max - c.max/5
		for range 200 {
			got := time.Until(worker.NextRetry(job(c.base, c.attempt)))
			if got < minDelay-time.Second || got > c.max {
				t.Fatalf("base=%d attempt=%d: delay %v outside [%v, %v]", c.base, c.attempt, got, minDelay, c.max)
			}
		}
	}

	// Jitter spreads retries: 50 deliveries failing together must not all
	// get the same delay.
	seen := map[time.Duration]bool{}
	for range 50 {
		seen[time.Until(worker.NextRetry(job(60, 1))).Round(time.Second)] = true
	}
	if len(seen) < 5 {
		t.Errorf("expected jittered delays to spread, got %d distinct values", len(seen))
	}
}
