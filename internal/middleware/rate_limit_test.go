package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func doRateLimited(h http.Handler, apiKey, remoteAddr string) int {
	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks", nil)
	req.RemoteAddr = remoteAddr
	if apiKey != "" {
		req.Header.Set(APIKeyHeader, apiKey)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestRateLimit(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("throttles after burst and sets Retry-After", func(t *testing.T) {
		h := RateLimit(2)(ok)
		for i := range 2 {
			if code := doRateLimited(h, "key-a", "1.2.3.4:100"); code != http.StatusOK {
				t.Fatalf("request %d: got %d, want 200", i, code)
			}
		}
		req := httptest.NewRequest(http.MethodGet, "/v1/webhooks", nil)
		req.Header.Set(APIKeyHeader, "key-a")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want 429", rec.Code)
		}
		if rec.Header().Get("Retry-After") == "" {
			t.Fatal("missing Retry-After header")
		}
	})

	t.Run("clients are limited independently", func(t *testing.T) {
		h := RateLimit(1)(ok)
		if code := doRateLimited(h, "", "10.0.0.1:5"); code != http.StatusOK {
			t.Fatalf("ip1 first: got %d, want 200", code)
		}
		if code := doRateLimited(h, "", "10.0.0.1:5"); code != http.StatusTooManyRequests {
			t.Fatalf("ip1 second: got %d, want 429", code)
		}
		if code := doRateLimited(h, "", "10.0.0.2:5"); code != http.StatusOK {
			t.Fatalf("ip2 (fresh bucket): got %d, want 200", code)
		}
	})

	t.Run("disabled when rps is zero", func(t *testing.T) {
		h := RateLimit(0)(ok)
		for i := range 50 {
			if code := doRateLimited(h, "", "10.0.0.3:5"); code != http.StatusOK {
				t.Fatalf("request %d: got %d, want 200", i, code)
			}
		}
	})
}
