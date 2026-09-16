package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newPortalHandler(t *testing.T) (*PortalTokens, http.Handler) {
	t.Helper()
	pt := NewPortalTokens([]byte("test-key-32-bytes-test-key-32-by"))
	auth := &APIKeyAuth{APIKey: "admin-secret", Portal: pt}
	handler := auth.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	return pt, handler
}

func doBearer(handler http.Handler, method, path, token string) int {
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr.Code
}

func TestPortalTokenMintVerifyRoundTrip(t *testing.T) {
	pt := NewPortalTokens([]byte("key"))
	token, exp, err := pt.Mint("acme", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if until := time.Until(exp); until <= 0 || until > time.Hour {
		t.Fatalf("expiry out of range: %v", exp)
	}
	consumer, err := pt.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if consumer != "acme" {
		t.Fatalf("consumer = %q, want acme", consumer)
	}
}

func TestPortalTokenRejectsTamperedAndExpired(t *testing.T) {
	pt := NewPortalTokens([]byte("key"))
	token, _, _ := pt.Mint("acme", time.Hour)

	// Tampered consumer segment: signature no longer matches.
	parts := strings.Split(token, ".")
	parts[1] = "ZXZpbA" // base64url("evil")
	if _, err := pt.Verify(strings.Join(parts, ".")); err == nil {
		t.Fatal("tampered token verified")
	}

	// Different key: signature invalid.
	other := NewPortalTokens([]byte("other"))
	if _, err := other.Verify(token); err == nil {
		t.Fatal("cross-key token verified")
	}

	// Expired: mint with 1s TTL then verify past expiry is covered by unit
	// clock; simulate by hand-crafting an already-expired token.
	expired, _, _ := pt.Mint("acme", time.Second)
	partsExp := strings.Split(expired, ".")
	partsExp[2] = "1000000000" // year 2001 — signature won't match, also expired
	if _, err := pt.Verify(strings.Join(partsExp, ".")); err == nil {
		t.Fatal("expired/tampered token verified")
	}
}

func TestPortalBearerScopedToConsumerPaths(t *testing.T) {
	pt, handler := newPortalHandler(t)
	token, _, err := pt.Mint("acme", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		method, path string
		want         int
	}{
		// Own consumer: allowed.
		{http.MethodGet, "/v1/consumers/acme/webhooks", http.StatusNoContent},
		{http.MethodPost, "/v1/consumers/acme/subscriptions", http.StatusNoContent},
		{http.MethodGet, "/v1/consumers/acme/deliveries", http.StatusNoContent},
		{http.MethodPost, "/v1/consumers/acme/deliveries/abc:retry", http.StatusNoContent},
		// Denied inside own consumer: event injection and token minting.
		{http.MethodPost, "/v1/consumers/acme/events", http.StatusUnauthorized},
		{http.MethodPost, "/v1/consumers/acme/portal-token", http.StatusUnauthorized},
		// Listing occurrences (GET events) is fine.
		{http.MethodGet, "/v1/consumers/acme/events", http.StatusNoContent},
		// Other consumers and global routes: denied.
		{http.MethodGet, "/v1/consumers/other/webhooks", http.StatusUnauthorized},
		{http.MethodGet, "/v1/webhooks", http.StatusUnauthorized},
		{http.MethodGet, "/v1/stats", http.StatusUnauthorized},
		{http.MethodPost, "/v1/event-types", http.StatusUnauthorized},
		// Read-only helpers the portal UI needs: allowed.
		{http.MethodGet, "/v1/event-types", http.StatusNoContent},
		{http.MethodGet, "/v1/event-types/order.created", http.StatusNoContent},
		{http.MethodGet, "/v1/template-functions", http.StatusNoContent},
		{http.MethodPost, "/v1/subscriptions:testTemplate", http.StatusNoContent},
	}
	for _, c := range cases {
		if got := doBearer(handler, c.method, c.path, token); got != c.want {
			t.Errorf("%s %s = %d, want %d", c.method, c.path, got, c.want)
		}
	}
}

func TestPortalBearerRejectsGarbageToken(t *testing.T) {
	_, handler := newPortalHandler(t)
	if got := doBearer(handler, http.MethodGet, "/v1/consumers/acme/webhooks", "spt_v1.garbage"); got != http.StatusUnauthorized {
		t.Fatalf("garbage token status = %d, want 401", got)
	}
	if got := doBearer(handler, http.MethodGet, "/v1/consumers/acme/webhooks", ""); got != http.StatusUnauthorized {
		t.Fatalf("no credentials status = %d, want 401", got)
	}
}
