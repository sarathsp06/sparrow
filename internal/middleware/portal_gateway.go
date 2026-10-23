package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/internal/accessauth"
	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/httpauth"
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

// PortalOptions are the operator's switches for what portal tokens may do.
type PortalOptions struct {
	// Retry lets portal tokens retry their own deliveries
	// (SPARROW_PORTAL_RETRY). Without it a consumer still sees every
	// delivery but cannot send one again.
	Retry bool
}

// Why a portal call is refused, as the 403 message says it.
const (
	refusedForPortal = "not permitted for portal tokens"
	refusedRetry     = "retrying deliveries is turned off for portal tokens on this server"
)

// PortalGateway serves the portal API under one static prefix (/portal/api/).
// It verifies the consumer-scoped bearer token, maps the portal-relative path
// to its real /v1 path, and re-dispatches into next (the main router) so the
// existing handlers run unchanged. The consumer never appears in the public
// URL, which collapses the operator's proxy allowlist to a single prefix and
// makes cross-consumer access structurally impossible.
func PortalGateway(verify PortalVerifier, opts PortalOptions, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "missing portal token")
			return
		}
		consumer, err := verify(r.Context(), strings.TrimSpace(token))
		if err != nil {
			writePortalAuthError(w, err)
			return
		}
		// The consumer becomes one path segment of the /v1 target, so it
		// must be one literal segment whoever minted the token.
		if accessauth.ValidateConsumer(consumer) != nil {
			writeJSONError(w, http.StatusForbidden, "forbidden", refusedForPortal)
			return
		}
		rest := strings.TrimPrefix(r.URL.Path, portalAPIPrefix)
		target, refused := portalTarget(r.Method, rest, r.URL.Query(), consumer, opts)
		if refused != "" {
			writeJSONError(w, http.StatusForbidden, "forbidden", refused)
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
		if target == "/v1/event-types" {
			// The catalog lists only what this consumer can subscribe to, so a
			// portal never offers Sparrow's own sparrow.* events.
			q := r2.URL.Query()
			q.Set("consumer", consumer)
			r2.URL.RawQuery = q.Encode()
		}
		r2.RequestURI = ""
		next.ServeHTTP(w, r2)
	})
}

// portalTarget is the entire portal authorization model, in one place: it
// maps a portal call to its real /v1 path, or says why it is refused.
//
// A token for consumer C may reach anything under /v1/consumers/C/ plus the
// read-only global helpers the portal UI needs (event-type catalog, template
// function list, server capabilities, and the stateless template dry-run),
// except:
//   - pushing or re-pushing events, and the snapshot that feeds a re-push
//     (the producer's job: re-push fans events out again, in bulk, to the
//     current subscriptions);
//   - retrying deliveries, and the snapshot that feeds a batch retry, unless
//     opts.Retry is set.
func portalTarget(method, rest string, query url.Values, consumer string, opts PortalOptions) (target, refused string) {
	if !cleanPortalPath(rest) {
		return "", refusedForPortal
	}
	switch {
	case method == http.MethodPost && (rest == "events" || rest == "events:rePush"),
		rest == "repush-jobs" || strings.HasPrefix(rest, "repush-jobs/"),
		method == http.MethodGet && rest == "events" && query.Has("prepare_repush"):
		return "", refusedForPortal
	case !opts.Retry && sendsRetry(method, rest, query):
		return "", refusedRetry
	case method == http.MethodGet && (rest == "event-types" || strings.HasPrefix(rest, "event-types/")):
		return "/v1/" + rest, ""
	case method == http.MethodGet && rest == "template-functions":
		return "/v1/template-functions", ""
	case method == http.MethodGet && rest == "capabilities":
		return "/v1/capabilities", ""
	case method == http.MethodPost && rest == "subscriptions:testTemplate":
		return "/v1/subscriptions:testTemplate", ""
	default:
		return "/v1/consumers/" + consumer + "/" + rest, ""
	}
}

// sendsRetry reports whether a portal call sends deliveries again: one
// (deliveries/{id}:retry), by filter (deliveries:retry), by id list
// (deliveries:retryBatch), or the list call that snapshots that id list
// (GET deliveries?prepare_retry). Reading or cancelling a retry job is not.
func sendsRetry(method, rest string, query url.Values) bool {
	switch method {
	case http.MethodGet:
		return rest == "deliveries" && query.Has("prepare_retry")
	case http.MethodPost:
		return rest == "deliveries:retry" || rest == "deliveries:retryBatch" ||
			(strings.HasPrefix(rest, "deliveries/") && strings.HasSuffix(rest, ":retry"))
	}
	return false
}

// cleanPortalPath reports whether rest is a plain relative path: no "." or
// ".." segments, empty segments, backslashes, or control characters. The
// router matches paths literally today, but the consumer pin must never
// depend on that — a path-cleaning layer added later would otherwise turn
// "../other/webhooks" into cross-consumer access. No portal API path
// legitimately contains any of these.
func cleanPortalPath(rest string) bool {
	if rest == "" || strings.ContainsAny(rest, "\\\x00") {
		return false
	}
	for _, seg := range strings.Split(strings.TrimSuffix(rest, "/"), "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
		for _, r := range seg {
			if r < 0x20 || r == 0x7f {
				return false
			}
		}
	}
	return true
}

// writePortalAuthError maps verifier failures: a store outage is 503 (the
// portal keeps its token and retries), a full-access token is 403, and an
// expired or revoked access token keeps its reason so the portal can say so.
func writePortalAuthError(w http.ResponseWriter, err error) {
	var authErr *access.AuthError
	switch {
	case errors.Is(err, ErrPortalFullAccessToken):
		writeJSONError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, access.ErrUnavailable), errors.As(err, &authErr):
		httpauth.WriteError(w, err)
	default:
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired portal token")
	}
}
