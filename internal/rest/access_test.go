package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/internal/accessauth"
	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/memstore"
)

// brokenStore fails every write with a message that must never reach clients.
type brokenStore struct{ access.Store }

func (brokenStore) CreateToken(context.Context, access.Token, []byte) error {
	return errors.New("pq: password authentication failed for user secret-db-user")
}

func accessRouter(t *testing.T, store access.Store) http.Handler {
	t.Helper()
	svc, err := accessauth.NewWithStore(store, "")
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	rest.Mount(r, nil, nil, rest.AccessDeps{Service: svc})
	return r
}

func post(h http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAccessStoreErrorsAreNotLeaked(t *testing.T) {
	rec := post(accessRouter(t, brokenStore{memstore.New()}), "/v1/tokens", `{"name":"ci"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret-db-user") || strings.Contains(rec.Body.String(), "pq:") {
		t.Fatalf("store error leaked to client: %s", rec.Body)
	}
}

func TestAccessRejectsUnsafeConsumerNames(t *testing.T) {
	h := accessRouter(t, memstore.New())
	for _, body := range []string{
		`{"name":"x","consumer":"../../v1"}`,
		`{"name":"x","consumer":"a/b"}`,
		`{"name":"x","consumer":" acme"}`,
	} {
		if rec := post(h, "/v1/tokens", body); rec.Code != http.StatusBadRequest {
			t.Errorf("POST /v1/tokens %s = %d, want 400", body, rec.Code)
		}
		if rec := post(h, "/v1/invites", body); rec.Code != http.StatusBadRequest {
			t.Errorf("POST /v1/invites %s = %d, want 400", body, rec.Code)
		}
	}
	if rec := post(h, "/v1/tokens", `{"name":"x","consumer":"acme"}`); rec.Code != http.StatusCreated {
		t.Fatalf("valid consumer = %d %s", rec.Code, rec.Body)
	}
}

func TestInvitePathsPointAtConsoleOrPortal(t *testing.T) {
	h := accessRouter(t, memstore.New())
	if rec := post(h, "/v1/invites", `{"name":"alice"}`); rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"path":"/#invite=sparrow_inv_`) {
		t.Fatalf("console invite = %d %s", rec.Code, rec.Body)
	}
	if rec := post(h, "/v1/invites", `{"name":"acme","consumer":"acme"}`); rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"path":"/portal#invite=sparrow_inv_`) {
		t.Fatalf("consumer invite = %d %s", rec.Code, rec.Body)
	}
	if rec := post(h, "/v1/invites", `{"name":"x","ttl_seconds":`+strconv.Itoa(8*24*3600)+`}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invite above 7 days = %d", rec.Code)
	}
}

func TestTenantTokenLifetimeDefaultsAndNeverExpires(t *testing.T) {
	svc, err := accessauth.NewWithStore(memstore.New(), "")
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	rest.Mount(r, nil, nil, rest.AccessDeps{Service: svc, TokenDefaultTTL: accessauth.TenantTokenDefaultTTL})

	var out struct {
		Token struct {
			ExpiresAt *time.Time `json:"expires_at"`
		} `json:"token"`
	}
	decode := func(rec *httptest.ResponseRecorder) {
		t.Helper()
		out.Token.ExpiresAt = nil
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rec.Code, rec.Body)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
	}

	decode(post(r, "/v1/tokens", `{"name":"default"}`))
	if out.Token.ExpiresAt == nil || time.Until(*out.Token.ExpiresAt) < 89*24*time.Hour {
		t.Fatalf("default tenant token should expire in ~90 days, got %v", out.Token.ExpiresAt)
	}
	decode(post(r, "/v1/tokens", `{"name":"forever","never_expires":true}`))
	if out.Token.ExpiresAt != nil {
		t.Fatalf("never_expires token has expires_at %v", out.Token.ExpiresAt)
	}
	for _, body := range []string{
		`{"name":"x","never_expires":true,"ttl_seconds":60}`,
		`{"name":"x","consumer":"acme","never_expires":true}`,
	} {
		if rec := post(r, "/v1/tokens", body); rec.Code != http.StatusBadRequest {
			t.Errorf("POST /v1/tokens %s = %d, want 400", body, rec.Code)
		}
	}
	if rec := post(r, "/v1/invites", `{"name":"ops","token_never_expires":true}`); rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"token_ttl_seconds":null`) {
		t.Fatalf("invite with token_never_expires = %d %s", rec.Code, rec.Body)
	}
}
