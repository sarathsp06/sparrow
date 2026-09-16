package middleware

import (
	"net/http"
	"strings"
)

// csp is the Content-Security-Policy for the embedded Svelte SPA. The UI
// injects an inline <script> for runtime config and Svelte emits inline
// styles, so 'unsafe-inline' is required for both. Everything else is
// same-origin; images additionally allow data: URIs for inline icons.
const csp = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'"

// SecurityHeaders is HTTP middleware that sets defensive security headers
// on every response. These headers provide defense-in-depth against common
// web attacks (clickjacking, MIME sniffing, XSS, information leakage).
func SecurityHeaders(next http.Handler) http.Handler {
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

		// Restrict resource loading to same-origin (see csp above), and
		// mirror the framing policy in CSP (frame-ancestors supersedes
		// X-Frame-Options in modern browsers).
		if portal {
			w.Header().Set("Content-Security-Policy", csp+"; frame-ancestors *")
		} else {
			w.Header().Set("Content-Security-Policy", csp+"; frame-ancestors 'none'")
		}

		next.ServeHTTP(w, r)
	})
}
