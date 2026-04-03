// Package middleware provides HTTP and gRPC middleware for the Sparrow server.
package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// APIKeyHeader is the HTTP header used to pass the API key.
	APIKeyHeader = "X-API-Key"

	// apiKeyMetadataKey is the gRPC metadata key (lowercase per gRPC convention).
	apiKeyMetadataKey = "x-api-key"
)

// APIKeyAuth holds the configuration for API key authentication.
// When APIKey is empty, all requests are allowed through (no-op mode).
type APIKeyAuth struct {
	// APIKey is the expected key. Empty means authentication is disabled.
	APIKey string

	// ExcludedPathPrefixes are HTTP path prefixes that bypass authentication
	// (e.g., "/health", "/ready", static UI assets).
	ExcludedPathPrefixes []string
}

// Enabled reports whether API key authentication is active.
func (a *APIKeyAuth) Enabled() bool {
	return a.APIKey != ""
}

// HTTPMiddleware returns an http.Handler that enforces API key authentication.
// When the API key is not configured (empty), requests pass through unchanged.
//
// The key can be provided via:
//   - Header: X-API-Key: <key>
//   - Query parameter: ?api_key=<key> (useful for browser/curl convenience)
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

		// Extract key from header or query parameter.
		key := r.Header.Get(APIKeyHeader)
		if key == "" {
			key = r.URL.Query().Get("api_key")
		}

		if !a.validKey(key) {
			http.Error(w, `{"error":"unauthorized","message":"missing or invalid API key"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// UnaryServerInterceptor returns a gRPC unary interceptor that enforces API
// key authentication via the "x-api-key" metadata header.
// When the API key is not configured (empty), requests pass through unchanged.
func (a *APIKeyAuth) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !a.Enabled() {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		keys := md.Get(apiKeyMetadataKey)
		if len(keys) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing API key: set %s metadata", apiKeyMetadataKey)
		}

		if !a.validKey(keys[0]) {
			return nil, status.Errorf(codes.Unauthenticated, "invalid API key")
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream interceptor that enforces API
// key authentication via the "x-api-key" metadata header.
func (a *APIKeyAuth) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if !a.Enabled() {
			return handler(srv, ss)
		}

		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		keys := md.Get(apiKeyMetadataKey)
		if len(keys) == 0 {
			return status.Errorf(codes.Unauthenticated, "missing API key: set %s metadata", apiKeyMetadataKey)
		}

		if !a.validKey(keys[0]) {
			return status.Errorf(codes.Unauthenticated, "invalid API key")
		}

		return handler(srv, ss)
	}
}

// validKey performs a constant-time comparison to prevent timing attacks.
func (a *APIKeyAuth) validKey(provided string) bool {
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a.APIKey), []byte(provided)) == 1
}
