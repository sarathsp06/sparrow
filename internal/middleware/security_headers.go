package middleware

import "net/http"

// SecurityHeaders is HTTP middleware that sets defensive security headers
// on every response. These headers provide defense-in-depth against common
// web attacks (clickjacking, MIME sniffing, information leakage).
//
// CSP is intentionally omitted here because the embedded UI injects an
// inline <script> for runtime config. A nonce-based CSP would require
// coordination with the UI handler; this can be added later if Sparrow
// is exposed beyond an internal network.
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
		// leaking internal URLs (which may contain namespace names or
		// webhook IDs) to external sites.
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Opt out of FLoC / Topics API tracking. Not critical for an
		// internal tool but costs nothing and is good hygiene.
		w.Header().Set("Permissions-Policy", "interest-cohort=()")

		next.ServeHTTP(w, r)
	})
}
