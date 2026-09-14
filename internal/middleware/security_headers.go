package middleware

import "net/http"

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

		// Prevent clickjacking by disallowing framing entirely.
		// Sparrow has no legitimate use case for being embedded in
		// an iframe.
		w.Header().Set("X-Frame-Options", "DENY")

		// Limit the Referer header to same-origin only. This prevents
		// leaking internal URLs (which may contain consumer names or
		// webhook IDs) to external sites.
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Opt out of FLoC / Topics API tracking. Not critical for an
		// internal tool but costs nothing and is good hygiene.
		w.Header().Set("Permissions-Policy", "interest-cohort=()")

		// Restrict resource loading to same-origin (see csp above).
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}
