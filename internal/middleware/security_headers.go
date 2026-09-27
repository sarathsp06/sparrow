package middleware

import (
	"net/http"
	"strings"
)

// buildCSP constructs a Content-Security-Policy value. When scriptHashes
// is non-empty, script-src uses those hashes instead of 'unsafe-inline';
// when empty, it falls back to 'unsafe-inline' so the server still works
// when the embedded UI was not built (development, API-only deploys).
func buildCSP(scriptHashes []string) string {
	scriptSrc := "'self' 'unsafe-inline'"
	if len(scriptHashes) > 0 {
		scriptSrc = "'self' " + strings.Join(scriptHashes, " ")
	}
	return "default-src 'self'" +
		"; script-src " + scriptSrc +
		"; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com" +
		"; font-src 'self' https://fonts.gstatic.com" +
		"; img-src 'self' data:" +
		"; connect-src 'self'"
}

// SecurityHeaders returns HTTP middleware that sets defensive security
// headers on every response. scriptHashes are CSP-formatted SHA-256
// hashes of the embedded SPA's inline <script> bodies (e.g.
// "'sha256-abc...'"); pass nil when the UI is not embedded.
//
// When hashes are provided, script-src lists them instead of
// 'unsafe-inline', so only the exact scripts the build produced are
// allowed to execute.
func SecurityHeaders(scriptHashes []string) func(http.Handler) http.Handler {
	csp := buildCSP(scriptHashes)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME-type sniffing. Without this, browsers may
			// interpret a JSON API response as HTML if it contains markup,
			// enabling reflected XSS.
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking by disallowing framing — except the
			// consumer portal, which is designed to be embedded in the
			// operator's own product via an iframe.
			portal := strings.HasPrefix(r.URL.Path, "/portal")
			if !portal {
				w.Header().Set("X-Frame-Options", "DENY")
			}

			// Limit the Referer header to same-origin only. This prevents
			// leaking internal URLs (which may contain consumer names or
			// webhook IDs) to external sites.
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Opt out of FLoC / Topics API tracking. Not critical for an
			// internal tool but costs nothing and is good hygiene.
			w.Header().Set("Permissions-Policy", "interest-cohort=()")

			// Restrict resource loading to same-origin (see buildCSP),
			// and mirror the framing policy in CSP (frame-ancestors
			// supersedes X-Frame-Options in modern browsers).
			//
			// Note: /docs (Huma's Scalar API reference) sets its own CSP
			// in the handler, which overwrites this header — so the
			// Scalar page is unaffected by these hashes.
			if portal {
				w.Header().Set("Content-Security-Policy", csp+"; frame-ancestors *")
			} else {
				w.Header().Set("Content-Security-Policy", csp+"; frame-ancestors 'none'")
			}

			next.ServeHTTP(w, r)
		})
	}
}
