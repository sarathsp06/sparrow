package middleware

import "net/http"

// MaxBodyBytes returns middleware that caps the request body size using
// http.MaxBytesReader, preventing oversized bodies from exhausting memory.
//
// Reads past the limit return *http.MaxBytesError and close the connection;
// the resulting status depends on where the body is consumed. Huma enforces
// its own per-operation 1 MiB limit (413) before this one triggers, so keep
// the limit >= 1 MiB — a smaller limit would surface through Huma's body
// reader as a 500 instead of a 4xx.
func MaxBodyBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
