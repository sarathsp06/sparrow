package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyHTTPMiddlewareAcceptsHeader(t *testing.T) {
	auth := &APIKeyAuth{APIKey: "secret"}
	called := false
	handler := auth.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/webhook.EventService/ListEvents", nil)
	req.Header.Set(APIKeyHeader, "secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestAPIKeyHTTPMiddlewareRejectsQueryParameter(t *testing.T) {
	auth := &APIKeyAuth{APIKey: "secret"}
	called := false
	handler := auth.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/webhook.EventService/ListEvents?api_key=secret", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("next handler was called")
	}
}

func TestAPIKeyHTTPMiddlewareBypassesExcludedPath(t *testing.T) {
	auth := &APIKeyAuth{
		APIKey:               "secret",
		ExcludedPathPrefixes: []string{"/health"},
	}
	called := false
	handler := auth.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}
