// Package middleware provides HTTP middleware for the Sparrow server.
package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const (
	// APIKeyHeader is the HTTP header used to pass the API key.
	APIKeyHeader = "X-API-Key"
)

// APIKeyAuth holds the configuration for API key authentication.
// When APIKey is empty, all requests are allowed through (no-op mode).
type APIKeyAuth struct {
	// APIKey is the expected key. Empty means authentication is disabled.
	APIKey string

	// ExcludedPathPrefixes are HTTP path prefixes that bypass authentication
	// (e.g., "/health", "/ready").
	ExcludedPathPrefixes []string

	// Portal optionally verifies consumer-scoped portal bearer tokens
	// (Authorization: Bearer spt_v1....). A valid token grants access only
	// to that consumer's /v1/consumers/{consumer}/ routes plus a few
	// read-only helper endpoints — see portalAllowed.
	Portal *PortalTokens
}

// Enabled reports whether API key authentication is active.
func (a *APIKeyAuth) Enabled() bool {
	return a.APIKey != ""
}

// HTTPMiddleware returns an http.Handler that enforces API key authentication.
// When the API key is not configured (empty), requests pass through unchanged.
//
// The key must be provided via the X-API-Key header. Query parameters are not
// accepted because URLs are commonly logged by proxies, stored in browser
// history, and leaked via Referer headers.
//
// Excluded paths (health, ready, static UI files) are never checked.
func (a *APIKeyAuth) HTTPMiddleware(next http.Handler) http.Handler {
	if !a.Enabled() {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for excluded paths.
		for _, prefix := range a.ExcludedPathPrefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
		}

		if !a.validKey(a.keyFromHTTPRequest(r)) && !a.validPortalRequest(r) {
			http.Error(w, `{"error":"unauthorized","message":"missing or invalid API key"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *APIKeyAuth) keyFromHTTPRequest(r *http.Request) string {
	return r.Header.Get(APIKeyHeader)
}

// validPortalRequest reports whether the request carries a valid portal
// bearer token AND targets a path that token's consumer may access.
func (a *APIKeyAuth) validPortalRequest(r *http.Request) bool {
	if a.Portal == nil {
		return false
	}
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return false
	}
	consumer, err := a.Portal.Verify(strings.TrimSpace(token))
	if err != nil {
		return false
	}
	return portalAllowed(r.Method, r.URL.Path, consumer)
}

// portalAllowed is the whole authorization model for portal tokens: the API
// is already path-scoped per consumer, so a token for consumer C may hit
// anything under /v1/consumers/C/ except pushing events (event injection is
// the producer's job, not the receiving consumer's) and minting further
// tokens. A few global read-only/stateless helpers needed by the portal UI
// (event-type catalog, template helpers, template dry-run) are also allowed.
func portalAllowed(method, path, consumer string) bool {
	prefix := "/v1/consumers/" + consumer + "/"
	if strings.HasPrefix(path, prefix) {
		rest := path[len(prefix):]
		if rest == "portal-token" {
			return false
		}
		if method == http.MethodPost && rest == "events" {
			return false
		}
		return true
	}
	if method == http.MethodGet {
		if path == "/v1/event-types" || strings.HasPrefix(path, "/v1/event-types/") {
			return true
		}
		if path == "/v1/template-functions" {
			return true
		}
	}
	return method == http.MethodPost && path == "/v1/subscriptions:testTemplate"
}

// validKey performs a constant-time comparison to prevent timing attacks.
func (a *APIKeyAuth) validKey(provided string) bool {
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a.APIKey), []byte(provided)) == 1
}
