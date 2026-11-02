package queue

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/riverqueue/river"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// TestStatusForFailure guards F-004: a failed attempt must report "retrying"
// while River still has attempts left, and only report the terminal "failed"
// once retries are exhausted. Regression: previously every failed attempt
// wrote "failed" immediately, then flipped back to "success" on a later
// retry — making "failed" a non-terminal, misleading status.
func TestStatusForFailure(t *testing.T) {
	cases := []struct {
		name        string
		attempt     int
		maxAttempts int
		want        store.WebhookDeliveryStatus
	}{
		{"first of many attempts", 1, 3, store.StatusRetrying},
		{"middle attempt", 2, 3, store.StatusRetrying},
		{"final attempt", 3, 3, store.StatusFailed},
		{"attempt exceeds max", 4, 3, store.StatusFailed},
		{"legacy job with unset max attempts", 1, 0, store.StatusFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := statusForFailure(tc.attempt, tc.maxAttempts); got != tc.want {
				t.Errorf("statusForFailure(%d, %d) = %q, want %q", tc.attempt, tc.maxAttempts, got, tc.want)
			}
		})
	}
}

// The job must outlive the slowest permitted receiver, or River cancels it
// before the HTTP client times out and no attempt is recorded.
func TestWebhookWorkerTimeoutCoversMaxRequestTimeout(t *testing.T) {
	w := &WebhookWorker{}
	if got := w.Timeout(&river.Job[WebhookArgs]{}); got <= client.MaxRequestTimeout {
		t.Fatalf("Timeout() = %v, want more than the %v request timeout cap", got, client.MaxRequestTimeout)
	}
}

func TestLimitErrorText(t *testing.T) {
	if got := limitErrorText("HTTP 500: 500 Internal Server Error"); got != "HTTP 500: 500 Internal Server Error" {
		t.Fatalf("short text changed: %q", got)
	}
	long := strings.Repeat("ü", 2000) // 4000 bytes
	got := limitErrorText(long)
	if len(got) > maxErrorTextBytes+64 || !utf8.ValidString(got) || !strings.HasSuffix(got, "(truncated, 4000 bytes in all)") {
		t.Fatalf("got %d bytes, valid=%v, suffix %q", len(got), utf8.ValidString(got), got[len(got)-40:])
	}
}
