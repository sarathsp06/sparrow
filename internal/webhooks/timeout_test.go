package webhooks

import "testing"

// A subscription's timeout override is bounded like request_timeout_seconds:
// the delivery job's River timeout is sized from that bound.
func TestValidateSubscriptionTimeout(t *testing.T) {
	for _, tc := range []struct {
		timeout int
		ok      bool
	}{
		{0, true}, {1, true}, {300, true}, {301, false}, {-1, false}, {86400, false},
	} {
		if err := validateSubscriptionTimeout(tc.timeout); (err == nil) != tc.ok {
			t.Errorf("validateSubscriptionTimeout(%d) err = %v, want ok=%v", tc.timeout, err, tc.ok)
		}
	}
}
