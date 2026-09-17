package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// portalAPIPrefix is the single public entry point for every portal API call.
// The consumer is carried by the bearer token, never the URL, so an operator
// forwarding the portal only needs to allowlist this one static prefix (plus
// the static /portal page and /_app assets) — no per-consumer proxy rules.
const portalAPIPrefix = "/portal/api/"

// portalCtxKey marks a request the PortalGateway has already authenticated so
// the shared API-key middleware lets it through without an admin key.
type portalCtxKey struct{}

// PortalAuthorized reports whether the PortalGateway pre-authenticated this
// request. APIKeyAuth honors it so portal traffic reuses the real /v1 handlers
// without holding the admin key.
func PortalAuthorized(ctx context.Context) bool {
	v, _ := ctx.Value(portalCtxKey{}).(bool)
	return v
}

// PortalGateway serves the portal API under one static prefix (/portal/api/).
// It verifies the consumer-scoped bearer token, maps the portal-relative path
// to its real /v1 path, and re-dispatches into next (the main router) so the
// existing handlers run unchanged. The consumer never appears in the public
// URL, which collapses the operator's proxy allowlist to a single prefix and
// makes cross-consumer access structurally impossible.
func PortalGateway(pt *PortalTokens, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pt == nil {
			http.Error(w, `{"error":"unauthorized","message":"portal tokens not configured"}`, http.StatusUnauthorized)
			return
		}
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			http.Error(w, `{"error":"unauthorized","message":"missing portal token"}`, http.StatusUnauthorized)
			return
		}
		consumer, err := pt.Verify(strings.TrimSpace(token))
		if err != nil {
			http.Error(w, `{"error":"unauthorized","message":"invalid or expired portal token"}`, http.StatusUnauthorized)
			return
		}
		target, ok := portalTarget(r.Method, strings.TrimPrefix(r.URL.Path, portalAPIPrefix), consumer)
		if !ok {
			http.Error(w, `{"error":"forbidden","message":"not permitted for portal tokens"}`, http.StatusForbidden)
			return
		}

		// Re-dispatch into the main router on the rewritten path. A fresh chi
		// route context is required so routing keys off the new URL.Path rather
		// than the stale /portal/api/* match.
		ctx := context.WithValue(r.Context(), portalCtxKey{}, true)
		ctx = context.WithValue(ctx, chi.RouteCtxKey, chi.NewRouteContext())
		r2 := r.Clone(ctx)
		r2.URL.Path = target
		r2.URL.RawPath = ""
		r2.RequestURI = ""
		next.ServeHTTP(w, r2)
	})
}

// portalTarget is the entire portal authorization model, in one place: it maps
// a portal-relative path to its real /v1 path, or returns ok=false to deny.
// A token for consumer C may reach anything under /v1/consumers/C/ except
// minting further tokens and injecting events (the producer's job), plus the
// read-only global helpers the portal UI needs (event-type catalog, template
// function list, and the stateless template dry-run).
func portalTarget(method, rest, consumer string) (string, bool) {
	switch {
	case rest == "portal-token", method == http.MethodPost && rest == "events":
		return "", false
	case method == http.MethodGet && (rest == "event-types" || strings.HasPrefix(rest, "event-types/")):
		return "/v1/" + rest, true
	case method == http.MethodGet && rest == "template-functions":
		return "/v1/template-functions", true
	case method == http.MethodPost && rest == "subscriptions:testTemplate":
		return "/v1/subscriptions:testTemplate", true
	default:
		return "/v1/consumers/" + consumer + "/" + rest, true
	}
}
