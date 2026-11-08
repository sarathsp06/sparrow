package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/pkg/access"
)

// forTenant runs the request as tenant id, as server.WithTenant does.
func forTenant(id uuid.UUID) func(*http.Request) {
	return func(r *http.Request) { *r = *r.WithContext(tenant.WithTenant(r.Context(), id)) }
}

func both(sets ...func(*http.Request)) func(*http.Request) {
	return func(r *http.Request) {
		for _, set := range sets {
			set(r)
		}
	}
}

func TestAuthRefusesATokenOfTenantAOnTenantB(t *testing.T) {
	auth, svc := newAuth(t, "secret")
	tenantA, tenantB := uuid.New(), uuid.New()
	tok, secret, err := svc.CreateToken(context.Background(), access.CreateTokenRequest{Realm: tenantA.String(), Name: "a", CreatedBy: "x"})
	if err != nil {
		t.Fatal(err)
	}

	if code, p, _ := guarded(auth, both(forTenant(tenantA), apiKey(secret))); code != http.StatusNoContent || p.TokenID != tok.ID {
		t.Fatalf("token on its own tenant: %d %+v", code, p)
	}
	if code, _, body := guarded(auth, both(forTenant(tenantB), apiKey(secret))); code != http.StatusUnauthorized {
		t.Fatalf("token of tenant A on tenant B: %d %s, want 401", code, body)
	}
	if code, _, body := guarded(auth, apiKey(secret)); code != http.StatusUnauthorized {
		t.Fatalf("token of tenant A on the default tenant: %d %s, want 401", code, body)
	}
	// The master key is the server's, not a tenant's.
	if code, p, _ := guarded(auth, both(forTenant(tenantB), apiKey("secret"))); code != http.StatusNoContent || !p.Root {
		t.Fatalf("master key on tenant B: %d %+v", code, p)
	}
}

func TestAuthDisabledRunsAsTheRequestTenant(t *testing.T) {
	auth, _ := newAuth(t, "")
	tenantB := uuid.New()
	if code, p, _ := guarded(auth, forTenant(tenantB)); code != http.StatusNoContent || p.Realm != tenantB.String() {
		t.Fatalf("disabled auth on tenant B: %d %+v", code, p)
	}
}

func TestPortalGatewayRefusesATokenOfTenantAOnTenantB(t *testing.T) {
	svc, gw, seen := newPortalGateway(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	acme := "acme"
	_, secret, err := svc.CreateToken(context.Background(), access.CreateTokenRequest{Realm: tenantA.String(), Scope: &acme, Name: "portal", CreatedBy: "x"})
	if err != nil {
		t.Fatal(err)
	}
	call := func(id uuid.UUID) int {
		req := httptest.NewRequest(http.MethodGet, "/portal/api/webhooks", nil)
		req = req.WithContext(tenant.WithTenant(req.Context(), id))
		req.Header.Set("Authorization", "Bearer "+secret)
		rr := httptest.NewRecorder()
		gw.ServeHTTP(rr, req)
		return rr.Code
	}

	if code := call(tenantA); code != http.StatusNoContent || *seen != "/v1/consumers/acme/webhooks" {
		t.Fatalf("token on its own tenant: %d %q", code, *seen)
	}
	*seen = ""
	if code := call(tenantB); code != http.StatusUnauthorized || *seen != "" {
		t.Fatalf("token of tenant A on tenant B: %d, reached %q; want 401 and nothing reached", code, *seen)
	}
	if code := call(tenant.DefaultTenantID); code != http.StatusUnauthorized || *seen != "" {
		t.Fatalf("token of tenant A on the default tenant: %d, reached %q; want 401", code, *seen)
	}
}
