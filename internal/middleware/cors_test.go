package middleware

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const uiOrigin = "https://ui.example.com"

// corsChain mirrors cmd/server's ordering: CORS runs before API key auth, so a
// preflight is answered without needing the key.
func corsChain(origins []string, production bool, apiKey string) (http.Handler, CORSMode) {
	cors, mode := CORS(origins, production)
	auth := &APIKeyAuth{APIKey: apiKey}
	api := auth.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	return cors(api), mode
}

func preflight(h http.Handler, origin, headers string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodOptions, "/v1/webhooks", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", headers)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func get(h http.Handler, origin, apiKey string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks", nil)
	req.Header.Set("Origin", origin)
	if apiKey != "" {
		req.Header.Set(APIKeyHeader, apiKey)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCORSModes(t *testing.T) {
	cases := []struct {
		name       string
		origins    []string
		production bool
		want       CORSMode
	}{
		{"allow list", []string{uiOrigin}, true, CORSAllowList},
		{"allow list in dev", []string{uiOrigin}, false, CORSAllowList},
		{"production default", nil, true, CORSBlockAll},
		{"production blank entries", []string{"", " "}, true, CORSBlockAll},
		{"dev default", nil, false, CORSAllowAll},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, got := CORS(tc.origins, tc.production); got != tc.want {
				t.Fatalf("mode = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCORSProductionDefaultBlocksCrossOrigin(t *testing.T) {
	// Regression: rs/cors treats an empty AllowedOrigins as "*", so the
	// production lockdown used to allow every origin.
	h, _ := corsChain(nil, true, "")

	if got := preflight(h, "https://evil.example.com", "x-api-key").Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("preflight Access-Control-Allow-Origin = %q, want none", got)
	}
	if got := get(h, "https://evil.example.com", "").Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("GET Access-Control-Allow-Origin = %q, want none", got)
	}
}

func TestCORSAllowListSplitDeploymentWithAPIKey(t *testing.T) {
	// UI on its own origin, server in production with an API key: the
	// preflight must succeed without the key, the real request needs it.
	h, _ := corsChain([]string{uiOrigin + "/"}, true, "secret")

	// Browsers send the requested headers lowercased and sorted.
	rec := preflight(h, uiOrigin, "content-type,x-api-key")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204 (must not hit API key auth)", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != uiOrigin {
		t.Fatalf("preflight Access-Control-Allow-Origin = %q, want %q", got, uiOrigin)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("preflight did not allow the X-API-Key header")
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want none (auth is header based)", got)
	}

	rec = get(h, uiOrigin, "secret")
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != uiOrigin {
		t.Fatalf("GET with key: status %d, ACAO %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}

	// A 401 must still carry CORS headers so the browser lets the UI read it
	// and show the API key prompt instead of an opaque network error.
	rec = get(h, uiOrigin, "")
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("Access-Control-Allow-Origin") != uiOrigin {
		t.Fatalf("GET without key: status %d, ACAO %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if got := preflight(h, "https://other.example.com", "x-api-key").Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unlisted origin got Access-Control-Allow-Origin = %q", got)
	}
}

func TestCORSDevAllowsAnyOriginWithAPIKeyHeader(t *testing.T) {
	// `make run` + `make run-web`: vite on :5173 calling :8080.
	h, _ := corsChain(nil, false, "secret")

	rec := preflight(h, "http://localhost:5173", "x-api-key")
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("preflight: status %d, ACAO %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if got := get(h, "http://localhost:5173", "secret").Code; got != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", got)
	}
}

func TestNormalizeOrigins(t *testing.T) {
	got := NormalizeOrigins([]string{" https://a.example.com/ ", "", "https://b.example.com//", "  "})
	want := []string{"https://a.example.com", "https://b.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeOrigins = %v, want %v", got, want)
	}
}
