package middleware

import (
	"net"
	"net/http"
	"net/url"
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
	// CORSAllowAll allows every origin (development default with an API key).
	CORSAllowAll
	// CORSLoopbackOnly allows only localhost origins (development default
	// without an API key).
	CORSLoopbackOnly
)

// NormalizeOrigins trims whitespace and trailing slashes and drops empty
// entries. Browsers send Origin without a trailing slash, so
// "https://ui.example.com/" in SPARROW_CORS_ALLOWED_ORIGINS would otherwise never match.
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
//   - origins empty, auth on: all origins, for local development (e.g. the
//     vite dev server on :5173 talking to :8080). A page still needs the key.
//   - origins empty, auth off: localhost origins only. Without a key every
//     caller is root, so allowing all origins would let any website a
//     browser on the network visits drive the API.
//
// The UI authenticates with the X-API-Key header (admin) or an Authorization
// bearer token (portal), never cookies, so credentials are not allowed.
func CORS(origins []string, production, authEnabled bool) (func(http.Handler) http.Handler, CORSMode) {
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
	case authEnabled:
		return cors.AllowAll().Handler, CORSAllowAll
	default:
		return cors.New(cors.Options{
			AllowOriginFunc: isLoopbackOrigin,
			AllowedMethods:  []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
			AllowedHeaders:  []string{"Authorization", "Content-Type", APIKeyHeader},
			MaxAge:          300,
		}).Handler, CORSLoopbackOnly
	}
}

// isLoopbackOrigin reports whether origin is http(s) on localhost or a
// loopback address, at any port.
func isLoopbackOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
