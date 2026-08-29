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

		if !a.validKey(a.keyFromHTTPRequest(r)) {
			http.Error(w, `{"error":"unauthorized","message":"missing or invalid API key"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *APIKeyAuth) keyFromHTTPRequest(r *http.Request) string {
	return r.Header.Get(APIKeyHeader)
}

// validKey performs a constant-time comparison to prevent timing attacks.
func (a *APIKeyAuth) validKey(provided string) bool {
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a.APIKey), []byte(provided)) == 1
}
