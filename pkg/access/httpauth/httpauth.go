// Package httpauth adapts an access.Service to net/http: it reads
// credentials from requests, puts the authenticated principal in the request
// context, writes consistent JSON errors, and serves invite redemption.
//
// Deciding what a principal may do (in particular what a non-nil Scope
// allows) is left to the application.
package httpauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sarathsp06/sparrow/pkg/access"
)

// APIKeyHeader is accepted as an alternative to "Authorization: Bearer".
const APIKeyHeader = "X-API-Key"

// Credential returns the bearer token, or else the X-API-Key header value.
// Query parameters are never read: URLs end up in logs and browser history.
func Credential(r *http.Request) string {
	if v, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(r.Header.Get(APIKeyHeader))
}

type ctxKey struct{}

// WithPrincipal returns ctx carrying p.
func WithPrincipal(ctx context.Context, p access.Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext returns the principal stored by Middleware or WithPrincipal.
func FromContext(ctx context.Context) (access.Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(access.Principal)
	return p, ok
}

// Verifier authenticates credentials the Service does not know (e.g. an
// application's legacy tokens). It returns handled=false to pass.
type Verifier func(ctx context.Context, credential string) (p access.Principal, handled bool, err error)

// Authenticator resolves request credentials: Extra verifiers first, then Service.
type Authenticator struct {
	Service *access.Service
	Extra   []Verifier
}

// Authenticate resolves r's credential to a principal.
func (a *Authenticator) Authenticate(r *http.Request) (access.Principal, error) {
	cred := Credential(r)
	for _, v := range a.Extra {
		if p, handled, err := v(r.Context(), cred); handled {
			return p, err
		}
	}
	return a.Service.Authenticate(r.Context(), cred)
}

// Middleware requires a valid credential and stores the principal in the
// request context. Failures get WriteError.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := a.Authenticate(r)
		if err != nil {
			WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
	})
}

// ErrorBody is the JSON body of authentication failures.
type ErrorBody struct {
	Error   string `json:"error"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message"`
}

// WriteError writes 401 for *access.AuthError (with its reason), 503 for
// store outages (so clients keep their credential and retry), and 500 otherwise.
func WriteError(w http.ResponseWriter, err error) {
	var authErr *access.AuthError
	switch {
	case errors.As(err, &authErr):
		writeJSON(w, http.StatusUnauthorized, ErrorBody{Error: "unauthorized", Reason: string(authErr.Reason), Message: authMessage(authErr.Reason)})
	case errors.Is(err, access.ErrUnavailable):
		w.Header().Set("Retry-After", "5")
		writeJSON(w, http.StatusServiceUnavailable, ErrorBody{Error: "unavailable", Message: "authentication is temporarily unavailable, try again shortly"})
	default:
		writeJSON(w, http.StatusInternalServerError, ErrorBody{Error: "internal", Message: "authentication failed"})
	}
}

func authMessage(r access.Reason) string {
	switch r {
	case access.ReasonMissing:
		return "missing API key or token"
	case access.ReasonExpired:
		return "this token has expired"
	case access.ReasonRevoked:
		return "this token has been revoked"
	default:
		return "invalid API key or token"
	}
}

// RedeemRequest is the body of the redeem endpoint.
type RedeemRequest struct {
	Invite string `json:"invite"`
}

// RedeemResponse returns the new token's secret, once.
type RedeemResponse struct {
	Token     string     `json:"token"`
	TokenID   string     `json:"token_id"`
	Name      string     `json:"name"`
	Scope     *string    `json:"scope"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// RedeemHandler serves POST {"invite": "<secret>"} -> RedeemResponse. The
// invite is the credential, so mount it outside authenticated routes.
// Unknown, expired, cancelled, and used invites all get 400 invalid_invite.
func RedeemHandler(svc *access.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, ErrorBody{Error: "method_not_allowed", Message: "use POST"})
			return
		}
		var req RedeemRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil || req.Invite == "" {
			writeJSON(w, http.StatusBadRequest, ErrorBody{Error: "bad_request", Message: `body must be {"invite": "..."}`})
			return
		}
		t, secret, err := svc.RedeemInvite(r.Context(), req.Invite)
		if errors.Is(err, access.ErrInvalidInvite) {
			writeJSON(w, http.StatusBadRequest, ErrorBody{Error: "invalid_invite", Message: "this invite is invalid, expired, cancelled, or has already been used"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorBody{Error: "internal", Message: "could not redeem the invite"})
			return
		}
		writeJSON(w, http.StatusOK, RedeemResponse{Token: secret, TokenID: t.ID, Name: t.Name, Scope: t.Scope, ExpiresAt: t.ExpiresAt})
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	// Responses here can carry secrets or describe credentials: never cache.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
