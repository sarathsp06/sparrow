package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/internal/accessauth"
	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/memstore"
	"github.com/sarathsp06/sparrow/pkg/crypto"
)

type mintedToken struct {
	Token struct {
		ID string `json:"id"`
	} `json:"token"`
	Secret     string `json:"secret"`
	Reused     bool   `json:"reused"`
	PortalPath string `json:"portal_path"`
}

func mintToken(t *testing.T, h http.Handler, body string) mintedToken {
	t.Helper()
	rec := post(h, "/v1/tokens", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /v1/tokens %s = %d %s", body, rec.Code, rec.Body)
	}
	var out mintedToken
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func sealedRouter(t *testing.T) (http.Handler, *access.Service) {
	t.Helper()
	cryptoSvc, err := crypto.NewService(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := accessauth.NewWithStore(memstore.New(), "", accessauth.Sealer(cryptoSvc))
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	rest.Mount(r, nil, rest.AccessDeps{Service: svc})
	return r, svc
}

func TestConsumerTokenComesWithAPortalLink(t *testing.T) {
	h, svc := sealedRouter(t)

	out := mintToken(t, h, `{"name":"acme portal","consumer":"acme co"}`)
	if want := "/portal#token=" + out.Secret + "&consumer=acme%20co&expires="; !strings.HasPrefix(out.PortalPath, want) {
		t.Fatalf("portal_path = %q, want prefix %q", out.PortalPath, want)
	}
	p, err := svc.Authenticate(context.Background(), out.Secret)
	if err != nil || p.FullAccess() || *p.Scope != "acme co" {
		t.Fatalf("principal = %+v, %v", p, err)
	}

	if full := mintToken(t, h, `{"name":"ci"}`); full.PortalPath != "" {
		t.Fatalf("tenant-wide token got a portal link: %q", full.PortalPath)
	}
}

func TestIdempotencyKeyReturnsTheValidConsumerToken(t *testing.T) {
	h, svc := sealedRouter(t)

	first := mintToken(t, h, `{"name":"portal","consumer":"acme","idempotency_key":"user-1"}`)
	again := mintToken(t, h, `{"name":"portal","consumer":"acme","idempotency_key":"user-1","ttl_seconds":60}`)
	if first.Reused || !again.Reused || again.Secret != first.Secret || again.Token.ID != first.Token.ID || again.PortalPath != first.PortalPath {
		t.Fatalf("same key: first %+v, again %+v; want the same token reused", first, again)
	}
	if other := mintToken(t, h, `{"name":"portal","consumer":"acme","idempotency_key":"user-2"}`); other.Reused || other.Token.ID == first.Token.ID {
		t.Fatalf("other key reused %+v", other)
	}
	if otherConsumer := mintToken(t, h, `{"name":"portal","consumer":"globex","idempotency_key":"user-1"}`); otherConsumer.Reused {
		t.Fatalf("key reused across consumers %+v", otherConsumer)
	}
	if plain := mintToken(t, h, `{"name":"portal","consumer":"acme"}`); plain.Reused || plain.Token.ID == first.Token.ID {
		t.Fatalf("no key reused %+v", plain)
	}

	req := httptest.NewRequest(http.MethodDelete, "/v1/tokens/"+first.Token.ID, nil)
	del := httptest.NewRecorder()
	h.ServeHTTP(del, req)
	if del.Code != http.StatusNoContent {
		t.Fatalf("revoke = %d %s", del.Code, del.Body)
	}
	var authErr *access.AuthError
	if _, err := svc.Authenticate(context.Background(), first.Secret); !errors.As(err, &authErr) || authErr.Reason != access.ReasonRevoked {
		t.Fatalf("after revoke err = %v, want revoked", err)
	}
	if fresh := mintToken(t, h, `{"name":"portal","consumer":"acme","idempotency_key":"user-1"}`); fresh.Reused || fresh.Token.ID == first.Token.ID {
		t.Fatalf("after revoke %+v, want a new token", fresh)
	}
}

func TestIdempotencyKeyRules(t *testing.T) {
	h, _ := sealedRouter(t)
	if rec := post(h, "/v1/tokens", `{"name":"ci","idempotency_key":"k"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("tenant-wide with key = %d %s, want 400", rec.Code, rec.Body)
	}
	if rec := post(accessRouter(t, memstore.New()), "/v1/tokens", `{"name":"p","consumer":"acme","idempotency_key":"k"}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("without sealer = %d %s, want 503", rec.Code, rec.Body)
	}
}
