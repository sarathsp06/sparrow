package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/httpauth"
	"github.com/sarathsp06/sparrow/pkg/access/memstore"
)

const testRealm = "00000000-0000-0000-0000-000000000001"

// newAuth builds the /v1 guard over an in-memory token store, with apiKey as
// the master key ("" = authentication disabled).
func newAuth(t testing.TB, apiKey string) (*Auth, *access.Service) {
	t.Helper()
	var roots []access.RootKey
	if apiKey != "" {
		roots = []access.RootKey{{Secret: apiKey, Realm: testRealm, Name: "master key"}}
	}
	svc, err := access.New(access.Config{Store: memstore.New(), RootKeys: roots, TokenPrefix: "sparrow_tk_"})
	if err != nil {
		t.Fatal(err)
	}
	return NewAuth(apiKey, svc, testRealm), svc
}

// guarded runs one request through auth and reports the status and the
// principal the handler saw.
func guarded(auth *Auth, set func(*http.Request)) (int, access.Principal, string) {
	var seen access.Principal
	h := auth.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = httpauth.FromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks", nil)
	if set != nil {
		set(req)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code, seen, rr.Body.String()
}

func bearer(v string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+v) }
}

func apiKey(v string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(APIKeyHeader, v) }
}

func TestAuthMasterKeyAndFullAccessTokens(t *testing.T) {
	auth, svc := newAuth(t, "secret")
	if code, p, _ := guarded(auth, bearer("secret")); code != http.StatusNoContent || !p.Root {
		t.Fatalf("master key as bearer: %d %+v", code, p)
	}

	tok, secret, err := svc.CreateToken(context.Background(), access.CreateTokenRequest{Realm: testRealm, Name: "alice", CreatedBy: "master key"})
	if err != nil {
		t.Fatal(err)
	}
	for name, set := range map[string]func(*http.Request){"X-API-Key": apiKey(secret), "Bearer": bearer(secret)} {
		code, p, _ := guarded(auth, set)
		if code != http.StatusNoContent || p.TokenID != tok.ID || p.Name != "alice" {
			t.Fatalf("token via %s: %d %+v", name, code, p)
		}
	}

	_ = svc.RevokeToken(context.Background(), testRealm, tok.ID)
	code, _, body := guarded(auth, apiKey(secret))
	var e httpauth.ErrorBody
	_ = json.Unmarshal([]byte(body), &e)
	if code != http.StatusUnauthorized || e.Reason != "revoked" {
		t.Fatalf("revoked token: %d %s", code, body)
	}
}

func TestAuthRejectsConsumerTokensOnV1(t *testing.T) {
	auth, svc := newAuth(t, "secret")
	consumer := "acme"
	_, secret, _ := svc.CreateToken(context.Background(), access.CreateTokenRequest{Realm: testRealm, Scope: &consumer, Name: "acme ci", TTL: time.Hour, CreatedBy: "x"})
	if code, _, body := guarded(auth, apiKey(secret)); code != http.StatusForbidden {
		t.Fatalf("consumer token on /v1: %d %s, want 403", code, body)
	}
}

func TestAuthRejectsOtherRealms(t *testing.T) {
	auth, svc := newAuth(t, "secret")
	_, secret, _ := svc.CreateToken(context.Background(), access.CreateTokenRequest{Realm: "another-tenant", Name: "x", CreatedBy: "x"})
	if code, _, _ := guarded(auth, apiKey(secret)); code != http.StatusUnauthorized {
		t.Fatalf("foreign realm: %d, want 401", code)
	}
}

func TestAuthDisabledIsOpenWithAnonymousPrincipal(t *testing.T) {
	auth, _ := newAuth(t, "")
	code, p, _ := guarded(auth, nil)
	if code != http.StatusNoContent || p.Name != AnonymousName || !p.FullAccess() {
		t.Fatalf("disabled: %d %+v", code, p)
	}
}

func TestAPIKeyHTTPMiddlewareAcceptsHeader(t *testing.T) {
	auth, _ := newAuth(t, "secret")
	called := false
	handler := auth.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/webhook.EventService/ListEvents", nil)
	req.Header.Set(APIKeyHeader, "secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestAPIKeyHTTPMiddlewareRejectsQueryParameter(t *testing.T) {
	auth, _ := newAuth(t, "secret")
	called := false
	handler := auth.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/webhook.EventService/ListEvents?api_key=secret", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("next handler was called")
	}
}

func TestAPIKeyHTTPMiddlewareBypassesExcludedPath(t *testing.T) {
	auth, _ := newAuth(t, "secret")
	auth.ExcludedPathPrefixes = []string{"/health"}
	called := false
	handler := auth.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}
