package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/memstore"
)

func newPortalService(t *testing.T) *access.Service {
	t.Helper()
	svc, err := access.New(access.Config{Store: memstore.New(), TokenPrefix: "sparrow_tk_"})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// mintConsumerToken returns the secret of a new token scoped to consumer.
func mintConsumerToken(t *testing.T, svc *access.Service, consumer string) string {
	t.Helper()
	_, secret, err := svc.CreateToken(context.Background(), access.CreateTokenRequest{Realm: testRealm, Scope: &consumer, Name: "portal", CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return secret
}

func newPortalGateway(t *testing.T) (*access.Service, http.Handler, *string) {
	t.Helper()
	svc := newPortalService(t)
	seen := new(string)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !PortalAuthorized(r.Context()) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		*seen = r.URL.Path
		if r.URL.RawQuery != "" {
			*seen += "?" + r.URL.RawQuery
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return svc, PortalGateway(NewPortalVerifier(svc, testRealm), next), seen
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

func TestPortalGatewayScopesToTokenConsumer(t *testing.T) {
	svc, handler, seen := newPortalGateway(t)
	token := mintConsumerToken(t, svc, "acme")

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
		{http.MethodPost, "/portal/api/alert-configs", http.StatusNoContent, "/v1/consumers/acme/alert-configs"},
		{http.MethodGet, "/portal/api/deliveries", http.StatusNoContent, "/v1/consumers/acme/deliveries"},
		{http.MethodPost, "/portal/api/deliveries/abc:retry", http.StatusNoContent, "/v1/consumers/acme/deliveries/abc:retry"},
		{http.MethodGet, "/portal/api/events", http.StatusNoContent, "/v1/consumers/acme/events"},
		// Event injection is denied for portal tokens (token minting lives
		// under the global /v1/tokens, which no portal path maps to).
		{http.MethodPost, "/portal/api/events", http.StatusForbidden, ""},
		{http.MethodPost, "/portal/api/tokens", http.StatusNoContent, "/v1/consumers/acme/tokens"},
		// Read-only global helpers the portal UI needs: allowed, not consumer-scoped.
		// The event-type list is pinned to the token's consumer, whatever the
		// client asks for.
		{http.MethodGet, "/portal/api/event-types", http.StatusNoContent, "/v1/event-types?consumer=acme"},
		{http.MethodGet, "/portal/api/event-types?consumer=_sparrow&active_only=true", http.StatusNoContent, "/v1/event-types?active_only=true&consumer=acme"},
		{http.MethodGet, "/portal/api/event-types/order.created", http.StatusNoContent, "/v1/event-types/order.created"},
		{http.MethodGet, "/portal/api/template-functions", http.StatusNoContent, "/v1/template-functions"},
		{http.MethodGet, "/portal/api/capabilities", http.StatusNoContent, "/v1/capabilities"},
		{http.MethodPost, "/portal/api/subscriptions:testTemplate", http.StatusNoContent, "/v1/subscriptions:testTemplate"},
		// Dot segments, empty segments and backslashes never reach the router,
		// so the consumer pin can't be escaped even if paths get cleaned later.
		{http.MethodGet, "/portal/api/../other/webhooks", http.StatusForbidden, ""},
		{http.MethodGet, "/portal/api/webhooks/../../other/webhooks", http.StatusForbidden, ""},
		{http.MethodGet, "/portal/api/./webhooks", http.StatusForbidden, ""},
		{http.MethodGet, "/portal/api/webhooks//x", http.StatusForbidden, ""},
		{http.MethodGet, "/portal/api/..%5Cother", http.StatusForbidden, ""},
		{http.MethodGet, "/portal/api/", http.StatusForbidden, ""},
		{http.MethodGet, "/portal/api/webhooks/", http.StatusNoContent, "/v1/consumers/acme/webhooks/"},
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
	if got := doBearer(handler, http.MethodGet, "/portal/api/webhooks", "sparrow_tk_garbage"); got != http.StatusUnauthorized {
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
	auth, svc := newAuth(t, "admin-secret")

	r := chi.NewRouter()
	r.Group(func(gr chi.Router) {
		gr.Use(auth.HTTPMiddleware)
		gr.Get("/v1/consumers/{consumer}/webhooks", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte(chi.URLParam(req, "consumer")))
		})
	})
	r.Handle("/portal/api/*", PortalGateway(NewPortalVerifier(svc, testRealm), r))

	token := mintConsumerToken(t, svc, "acme")

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

// downStore simulates a token store outage.
type downStore struct{ access.Store }

func (downStore) TokenByHash(context.Context, []byte) (access.Token, error) {
	return access.Token{}, errors.New("db down")
}

func TestPortalGatewayAcceptsConsumerAccessTokens(t *testing.T) {
	svc := newPortalService(t)
	seen := new(string)
	gw := PortalGateway(NewPortalVerifier(svc, testRealm), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = r.URL.Path
		if r.URL.RawQuery != "" {
			*seen += "?" + r.URL.RawQuery
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	ctx := context.Background()
	acme := "acme"

	consumerTok, consumerSecret, _ := svc.CreateToken(ctx, access.CreateTokenRequest{Realm: testRealm, Scope: &acme, Name: "acme staff", CreatedBy: "x"})
	if code := doBearer(gw, http.MethodGet, "/portal/api/webhooks", consumerSecret); code != http.StatusNoContent || *seen != "/v1/consumers/acme/webhooks" {
		t.Fatalf("consumer token: %d %q", code, *seen)
	}
	// The portal allow-list still applies to access tokens.
	if code := doBearer(gw, http.MethodPost, "/portal/api/events", consumerSecret); code != http.StatusForbidden {
		t.Fatalf("event push via portal: %d, want 403", code)
	}

	_, fullSecret, _ := svc.CreateToken(ctx, access.CreateTokenRequest{Realm: testRealm, Name: "admin", CreatedBy: "x"})
	if code := doBearer(gw, http.MethodGet, "/portal/api/webhooks", fullSecret); code != http.StatusForbidden {
		t.Fatalf("full-access token via portal: %d, want 403", code)
	}

	_, foreignSecret, _ := svc.CreateToken(ctx, access.CreateTokenRequest{Realm: "other", Scope: &acme, Name: "x", CreatedBy: "x"})
	if code := doBearer(gw, http.MethodGet, "/portal/api/webhooks", foreignSecret); code != http.StatusUnauthorized {
		t.Fatalf("foreign realm: %d, want 401", code)
	}

	_ = svc.RevokeToken(ctx, testRealm, consumerTok.ID)
	req := httptest.NewRequest(http.MethodGet, "/portal/api/webhooks", nil)
	req.Header.Set("Authorization", "Bearer "+consumerSecret)
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || !strings.Contains(rr.Body.String(), `"reason":"revoked"`) {
		t.Fatalf("revoked: %d %s", rr.Code, rr.Body)
	}
}

func TestPortalGatewayStoreOutageIs503(t *testing.T) {
	svc, _ := access.New(access.Config{Store: downStore{memstore.New()}, TokenPrefix: "sparrow_tk_"})
	gw := PortalGateway(NewPortalVerifier(svc, testRealm), http.NotFoundHandler())
	if code := doBearer(gw, http.MethodGet, "/portal/api/webhooks", "sparrow_tk_anything"); code != http.StatusServiceUnavailable {
		t.Fatalf("outage: %d, want 503", code)
	}
}

func TestPortalGatewayRejectsUnsafeTokenConsumers(t *testing.T) {
	// pkg/access does not validate scope names (the REST layer does); the
	// gateway must still never build a multi-segment /v1 path from one.
	svc, gw, seen := newPortalGateway(t)
	for _, consumer := range []string{"a/../b", "..", "a%2Fb"} {
		token := mintConsumerToken(t, svc, consumer)
		*seen = ""
		if code := doBearer(gw, http.MethodGet, "/portal/api/webhooks", token); code != http.StatusForbidden || *seen != "" {
			t.Errorf("consumer %q = %d (reached %q), want 403", consumer, code, *seen)
		}
	}
}
