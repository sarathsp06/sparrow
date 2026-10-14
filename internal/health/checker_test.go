package health

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// Liveness never depends on the database (a nil pool would panic if it were
// pinged), and readiness turns 503 once shutdown starts, before any ping.
func TestLivenessIsProcessOnlyAndReadinessDrains(t *testing.T) {
	hc := NewChecker(nil, time.Now())

	if resp, code := hc.Health(context.Background()); code != http.StatusOK || resp.Status != "healthy" {
		t.Fatalf("Health() = %d %q, want 200 healthy", code, resp.Status)
	}

	hc.SetDraining()
	if resp, code := hc.Ready(context.Background()); code != http.StatusServiceUnavailable || resp.Status != "draining" {
		t.Fatalf("Ready() while draining = %d %q, want 503 draining", code, resp.Status)
	}
	if _, code := hc.Health(context.Background()); code != http.StatusOK {
		t.Fatalf("Health() while draining = %d, want 200: the process is still alive", code)
	}
}
