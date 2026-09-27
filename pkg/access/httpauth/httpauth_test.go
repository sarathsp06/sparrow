package httpauth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/httpauth"
	"github.com/sarathsp06/sparrow/pkg/access/memstore"
)

func newSvc(t *testing.T, store access.Store) *access.Service {
	t.Helper()
	svc, err := access.New(access.Config{Store: store, RootKeys: []access.RootKey{{Secret: "root", Realm: "r1", Name: "master key"}}})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestCredential(t *testing.T) {
	cases := map[string]func(r *http.Request){
		"bearer":      func(r *http.Request) { r.Header.Set("Authorization", "Bearer abc") },
		"api key":     func(r *http.Request) { r.Header.Set("X-API-Key", " abc ") },
		"bearer wins": func(r *http.Request) { r.Header.Set("Authorization", "Bearer abc"); r.Header.Set("X-API-Key", "other") },
	}
	for name, set := range cases {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		set(r)
		if got := httpauth.Credential(r); got != "abc" {
			t.Errorf("%s: Credential = %q", name, got)
		}
	}
	r := httptest.NewRequest(http.MethodGet, "/?api_key=abc&token=abc", nil)
	if got := httpauth.Credential(r); got != "" {
		t.Errorf("query parameters must be ignored, got %q", got)
	}
}

func serve(h http.Handler, cred string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if cred != "" {
		r.Header.Set("Authorization", "Bearer "+cred)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestMiddleware(t *testing.T) {
	store := memstore.New()
	svc := newSvc(t, store)
	a := &httpauth.Authenticator{Service: svc}
	var seen access.Principal
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = httpauth.FromContext(r.Context())
	}))

	tok, secret, _ := svc.CreateToken(context.Background(), access.CreateTokenRequest{Realm: "r1", Name: "alice", CreatedBy: "x"})
	if w := serve(h, secret); w.Code != http.StatusOK || seen.TokenID != tok.ID {
		t.Fatalf("valid token: %d, principal %+v", w.Code, seen)
	}
	if w := serve(h, "root"); w.Code != http.StatusOK || !seen.Root {
		t.Fatalf("root: %d, principal %+v", w.Code, seen)
	}

	w := serve(h, "")
	var body httpauth.ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != http.StatusUnauthorized || body.Reason != "missing" {
		t.Fatalf("no credential: %d %+v", w.Code, body)
	}

	_ = svc.RevokeToken(context.Background(), "r1", tok.ID)
	w = serve(h, secret)
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != http.StatusUnauthorized || body.Reason != "revoked" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("revoked: %d %+v", w.Code, body)
	}
}

type downStore struct{ access.Store }

func (downStore) TokenByHash(context.Context, []byte) (access.Token, error) {
	return access.Token{}, errors.New("db down")
}

func TestMiddlewareOutageIs503(t *testing.T) {
	a := &httpauth.Authenticator{Service: newSvc(t, downStore{memstore.New()})}
	w := serve(a.Middleware(http.NotFoundHandler()), "tk_whatever")
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Retry-After") == "" || strings.Contains(w.Body.String(), "db down") {
		t.Fatalf("outage: %d %s", w.Code, w.Body)
	}
}

func TestExtraVerifier(t *testing.T) {
	legacy := func(_ context.Context, cred string) (access.Principal, bool, error) {
		if !strings.HasPrefix(cred, "legacy_") {
			return access.Principal{}, false, nil
		}
		if cred != "legacy_ok" {
			return access.Principal{}, true, &access.AuthError{Reason: access.ReasonInvalid}
		}
		return access.Principal{Realm: "r1", Name: "legacy"}, true, nil
	}
	a := &httpauth.Authenticator{Service: newSvc(t, memstore.New()), Extra: []httpauth.Verifier{legacy}}
	var seen access.Principal
	h := a.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen, _ = httpauth.FromContext(r.Context()) }))
	if w := serve(h, "legacy_ok"); w.Code != http.StatusOK || seen.Name != "legacy" {
		t.Fatalf("legacy ok: %d %+v", w.Code, seen)
	}
	if w := serve(h, "legacy_bad"); w.Code != http.StatusUnauthorized {
		t.Fatalf("legacy bad: %d", w.Code)
	}
	if w := serve(h, "root"); w.Code != http.StatusOK || !seen.Root {
		t.Fatalf("fallthrough to service: %d %+v", w.Code, seen)
	}
}

func TestRedeemHandler(t *testing.T) {
	svc := newSvc(t, memstore.New())
	_, invite, _ := svc.CreateInvite(context.Background(), access.CreateInviteRequest{Realm: "r1", Name: "bob", TTL: time.Hour, TokenTTL: time.Hour, CreatedBy: "alice"})
	h := httpauth.RedeemHandler(svc)
	post := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/redeem", strings.NewReader(body)))
		return w
	}

	w := post(`{"invite":"` + invite + `"}`)
	var res httpauth.RedeemResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if w.Code != http.StatusOK || res.Name != "bob" || res.TokenID == "" || res.ExpiresAt == nil || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("redeem: %d %s", w.Code, w.Body)
	}
	if p, err := svc.Authenticate(context.Background(), res.Token); err != nil || p.Name != "bob" {
		t.Fatalf("redeemed token does not authenticate: %+v %v", p, err)
	}

	if w := post(`{"invite":"` + invite + `"}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_invite") {
		t.Fatalf("reuse: %d %s", w.Code, w.Body)
	}
	if w := post(`nope`); w.Code != http.StatusBadRequest {
		t.Fatalf("bad body: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/redeem", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", w.Code)
	}
}
