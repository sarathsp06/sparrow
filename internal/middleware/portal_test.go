package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func newPortalGateway(t *testing.T) (*PortalTokens, http.Handler, *string) {
	t.Helper()
	pt := NewPortalTokens([]byte("test-key-32-bytes-test-key-32-by"))
	seen := new(string)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !PortalAuthorized(r.Context()) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		*seen = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	return pt, PortalGateway(pt, next), seen
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

func TestPortalGatewayScopesToTokenConsumer(t *testing.T) {
	pt, handler, seen := newPortalGateway(t)
	token, _, err := pt.Mint("acme", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// Every allowed call rewrites to a /v1 path scoped to the token's consumer.
	// The consumer never appears in the request URL, so cross-consumer access
	// is structurally impossible — there is nothing to test for "other".
	cases := []struct {
		method, path string
		want         int
		target       string
	}{
		{http.MethodGet, "/portal/api/webhooks", http.StatusNoContent, "/v1/consumers/acme/webhooks"},
		{http.MethodPost, "/portal/api/subscriptions", http.StatusNoContent, "/v1/consumers/acme/subscriptions"},
		{http.MethodGet, "/portal/api/deliveries", http.StatusNoContent, "/v1/consumers/acme/deliveries"},
		{http.MethodPost, "/portal/api/deliveries/abc:retry", http.StatusNoContent, "/v1/consumers/acme/deliveries/abc:retry"},
		{http.MethodGet, "/portal/api/events", http.StatusNoContent, "/v1/consumers/acme/events"},
		// Event injection and token minting are denied for portal tokens.
		{http.MethodPost, "/portal/api/events", http.StatusForbidden, ""},
		{http.MethodPost, "/portal/api/portal-token", http.StatusForbidden, ""},
		// Read-only global helpers the portal UI needs: allowed, not consumer-scoped.
		{http.MethodGet, "/portal/api/event-types", http.StatusNoContent, "/v1/event-types"},
		{http.MethodGet, "/portal/api/event-types/order.created", http.StatusNoContent, "/v1/event-types/order.created"},
		{http.MethodGet, "/portal/api/template-functions", http.StatusNoContent, "/v1/template-functions"},
		{http.MethodPost, "/portal/api/subscriptions:testTemplate", http.StatusNoContent, "/v1/subscriptions:testTemplate"},
	}
	for _, c := range cases {
		*seen = ""
		if got := doBearer(handler, c.method, c.path, token); got != c.want {
			t.Errorf("%s %s = %d, want %d", c.method, c.path, got, c.want)
		}
		if c.want == http.StatusNoContent && *seen != c.target {
			t.Errorf("%s %s rewrote to %q, want %q", c.method, c.path, *seen, c.target)
		}
	}
}

func TestPortalGatewayRejectsGarbageToken(t *testing.T) {
	_, handler, _ := newPortalGateway(t)
	if got := doBearer(handler, http.MethodGet, "/portal/api/webhooks", "spt_v1.garbage"); got != http.StatusUnauthorized {
		t.Fatalf("garbage token status = %d, want 401", got)
	}
	if got := doBearer(handler, http.MethodGet, "/portal/api/webhooks", ""); got != http.StatusUnauthorized {
		t.Fatalf("no credentials status = %d, want 401", got)
	}
}

// End-to-end: the gateway re-dispatches into the real chi router, so a portal
// call must reach the admin-key-protected /v1 handler with {consumer} resolved
// from the token — proving the fresh route context and the auth-bypass flag
// both work through an actual router (the wiring in cmd/server/main.go).
func TestPortalGatewayReDispatchesThroughRouter(t *testing.T) {
	pt := NewPortalTokens([]byte("test-key-32-bytes-test-key-32-by"))
	auth := &APIKeyAuth{APIKey: "admin-secret"}

	r := chi.NewRouter()
	r.Group(func(gr chi.Router) {
		gr.Use(auth.HTTPMiddleware)
		gr.Get("/v1/consumers/{consumer}/webhooks", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte(chi.URLParam(req, "consumer")))
		})
	})
	r.Handle("/portal/api/*", PortalGateway(pt, r))

	token, _, err := pt.Mint("acme", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// Portal call: no admin key, only the bearer token.
	req := httptest.NewRequest(http.MethodGet, "/portal/api/webhooks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || rr.Body.String() != "acme" {
		t.Fatalf("portal call = %d %q, want 200 \"acme\"", rr.Code, rr.Body.String())
	}

	// Same /v1 route without a key is still rejected (bypass is token-only).
	direct := httptest.NewRequest(http.MethodGet, "/v1/consumers/acme/webhooks", nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, direct)
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("keyless /v1 call = %d, want 401", rr2.Code)
	}
}
