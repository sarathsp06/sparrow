package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimit returns middleware enforcing a token-bucket limit of rps
// requests per second (burst = rps) per client. Clients are identified by
// API key when present, else by remote IP. rps <= 0 disables limiting.
//
// State is in-memory and process-local — correct fit for Sparrow's
// single-container Docker Compose deployment model (see AGENTS.md).
func RateLimit(rps int) func(http.Handler) http.Handler {
	if rps <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	var (
		mu       sync.Mutex
		limiters = make(map[string]*rate.Limiter)
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(APIKeyHeader)
			if key == "" {
				if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
					key = host
				} else {
					key = r.RemoteAddr
				}
			}

			mu.Lock()
			// ponytail: unbounded key map reset at 10k entries; switch to LRU
			// eviction if churn from many distinct client IPs matters.
			if len(limiters) > 10000 {
				limiters = make(map[string]*rate.Limiter)
			}
			lim, ok := limiters[key]
			if !ok {
				lim = rate.NewLimiter(rate.Limit(rps), rps)
				limiters[key] = lim
			}
			mu.Unlock()

			if !lim.Allow() {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate_limited","message":"API rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
