package queue

import (
	"testing"

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
