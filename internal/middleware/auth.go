package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/httpauth"
)

// APIKeyHeader is the HTTP header used to pass the API key (or a token).
const APIKeyHeader = httpauth.APIKeyHeader

// Auth guards the /v1 API. When SPARROW_API_KEY is set, every request needs
// either that master key or a full-access access token (both work in the
// X-API-Key header or as "Authorization: Bearer"). Consumer-scoped tokens are
// refused here: they may only use the portal gateway, which pins them to their
// consumer. When SPARROW_API_KEY is unset, everything is open.
//
// The authenticated principal is stored in the request context
// (httpauth.FromContext) so handlers can record who did what.
type Auth struct {
	// Enabled is true when SPARROW_API_KEY is set.
	Enabled bool
	// Realm is the tenant every credential must belong to.
	Realm string
	Authn *httpauth.Authenticator
	// ExcludedPathPrefixes are never checked (health, docs, spec).
	ExcludedPathPrefixes []string
}

// NewAuth builds the /v1 guard. apiKey is SPARROW_API_KEY ("" disables auth);
// svc must already know it as a root key.
func NewAuth(apiKey string, svc *access.Service, realm string, excluded ...string) *Auth {
	return &Auth{
		Enabled:              apiKey != "",
		Realm:                realm,
		Authn:                &httpauth.Authenticator{Service: svc},
		ExcludedPathPrefixes: excluded,
	}
}

// AnonymousName is the principal name recorded when authentication is off.
const AnonymousName = "anonymous (authentication disabled)"

// HTTPMiddleware enforces authentication on the wrapped routes.
func (a *Auth) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, prefix := range a.ExcludedPathPrefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
		}
		// The portal gateway already authenticated and scoped this request.
		if PortalAuthorized(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}
		if !a.Enabled {
			anon := access.Principal{Realm: a.Realm, Name: AnonymousName, Root: true}
			next.ServeHTTP(w, r.WithContext(httpauth.WithPrincipal(r.Context(), anon)))
			return
		}

		p, err := a.Authn.Authenticate(r)
		if err == nil && p.Realm != a.Realm {
			err = &access.AuthError{Reason: access.ReasonInvalid}
		}
		if err != nil {
			httpauth.WriteError(w, err)
			return
		}
		if !p.FullAccess() {
			writeJSONError(w, http.StatusForbidden, "forbidden", "consumer-scoped tokens can only call the portal API under /portal/api/")
			return
		}
		next.ServeHTTP(w, r.WithContext(httpauth.WithPrincipal(r.Context(), p)))
	})
}

// Errors returned by a PortalVerifier.
var (
	// ErrPortalFullAccessToken rejects full-access tokens at the portal
	// gateway: the portal needs a consumer to scope to.
	ErrPortalFullAccessToken = errors.New("full-access tokens cannot use the portal API; use /v1 instead")
	// ErrPortalTokenInvalid rejects a token from another realm.
	ErrPortalTokenInvalid = errors.New("invalid or expired portal token")
)

// PortalVerifier resolves a portal bearer token to the consumer it is scoped to.
type PortalVerifier func(ctx context.Context, token string) (consumer string, err error)

// NewPortalVerifier accepts consumer-scoped access tokens (minted by
// POST /v1/tokens with a consumer, or by consumer invites).
func NewPortalVerifier(svc *access.Service, realm string) PortalVerifier {
	return func(ctx context.Context, token string) (string, error) {
		p, err := svc.Authenticate(ctx, token)
		if err != nil {
			return "", err
		}
		if p.Realm != realm {
			return "", ErrPortalTokenInvalid
		}
		if p.FullAccess() {
			return "", ErrPortalFullAccessToken
		}
		return *p.Scope, nil
	}
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "message": message})
}
