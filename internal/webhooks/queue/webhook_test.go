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

	// Exponential doubling: base * 2^(attempt-1).
	cases := []struct {
		base, attempt int
		want          time.Duration
	}{
		{60, 1, 60 * time.Second},
		{60, 2, 120 * time.Second},
		{60, 4, 480 * time.Second},
		{3600, 10, maxRetryDelay}, // capped
	}
	for _, c := range cases {
		got := time.Until(worker.NextRetry(job(c.base, c.attempt)))
		if diff := got - c.want; diff < -time.Second || diff > time.Second {
			t.Errorf("base=%d attempt=%d: expected delay ~%v, got %v", c.base, c.attempt, c.want, got)
		}
	}
}
