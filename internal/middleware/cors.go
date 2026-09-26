package middleware

import (
	"net/http"
	"strings"

	"github.com/rs/cors"
)

// CORSMode describes which cross-origin policy CORS selected, for startup logs.
type CORSMode int

const (
	// CORSAllowList allows only the configured origins.
	CORSAllowList CORSMode = iota
	// CORSBlockAll rejects every cross-origin request (production default).
	CORSBlockAll
	// CORSAllowAll allows every origin (development default).
	CORSAllowAll
)

// NormalizeOrigins trims whitespace and trailing slashes and drops empty
// entries. Browsers send Origin without a trailing slash, so
// "https://ui.example.com/" in CORS_ALLOWED_ORIGINS would otherwise never match.
func NormalizeOrigins(origins []string) []string {
	var out []string
	for _, o := range origins {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o != "" {
			out = append(out, o)
		}
	}
	return out
}

// CORS builds the cross-origin policy for the HTTP server.
//
//   - origins set: only those origins may call the API (a separately hosted UI
//     must be listed here).
//   - origins empty, production: no cross-origin access. The embedded UI is
//     same-origin and does not need CORS.
//   - origins empty, otherwise: all origins, for local development (e.g. the
//     vite dev server on :5173 talking to :8080).
//
// The UI authenticates with the X-API-Key header (admin) or an Authorization
// bearer token (portal), never cookies, so credentials are not allowed.
func CORS(origins []string, production bool) (func(http.Handler) http.Handler, CORSMode) {
	origins = NormalizeOrigins(origins)
	switch {
	case len(origins) > 0:
		return cors.New(cors.Options{
			AllowedOrigins: origins,
			AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
			AllowedHeaders: []string{"Authorization", "Content-Type", APIKeyHeader},
			MaxAge:         300,
		}).Handler, CORSAllowList
	case production:
		// An empty AllowedOrigins list means "allow all" in rs/cors, so deny
		// explicitly with an origin func.
		return cors.New(cors.Options{
			AllowOriginFunc: func(string) bool { return false },
		}).Handler, CORSBlockAll
	default:
		return cors.AllowAll().Handler, CORSAllowAll
	}
}
