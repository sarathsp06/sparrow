package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersNoHashes(t *testing.T) {
	// When no hashes are provided (UI not built), script-src falls back
	// to 'unsafe-inline'.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeaders(nil)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "interest-cohort=()"},
		{"Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'"},
	}

	for _, tt := range tests {
		got := rec.Header().Get(tt.header)
		if got != tt.expected {
			t.Errorf("%s = %q, want %q", tt.header, got, tt.expected)
		}
	}
}

func TestSecurityHeadersWithHashes(t *testing.T) {
	// When hashes are provided, script-src uses hashes instead of
	// 'unsafe-inline'.
	hashes := []string{"'sha256-abc123'", "'sha256-def456'"}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeaders(hashes)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")

	// script-src must NOT contain 'unsafe-inline'
	for _, part := range strings.Split(csp, ";") {
		trimmed := strings.TrimSpace(part)
		if strings.HasPrefix(trimmed, "script-src") {
			if strings.Contains(trimmed, "'unsafe-inline'") {
				t.Errorf("script-src contains 'unsafe-inline' when hashes provided: %s", trimmed)
			}
		}
	}
	for _, h := range hashes {
		if !strings.Contains(csp, h) {
			t.Errorf("CSP missing hash %s: %s", h, csp)
		}
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("CSP missing frame-ancestors 'none': %s", csp)
	}
}

func TestSecurityHeadersPortalAllowsFraming(t *testing.T) {
	handler := SecurityHeaders(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/portal", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Frame-Options"); got != "" {
		t.Errorf("X-Frame-Options on /portal = %q, want unset", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors *") {
		t.Errorf("CSP on /portal = %q, want frame-ancestors *", got)
	}
}

func TestSecurityHeadersPassthrough(t *testing.T) {
	// Verify the middleware calls the next handler and doesn't block.
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("X-Custom", "test")
		w.WriteHeader(http.StatusCreated)
	})

	handler := SecurityHeaders(nil)(inner)

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if rec.Header().Get("X-Custom") != "test" {
		t.Error("inner handler's custom header was lost")
	}
	// Security headers should still be present.
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("security header missing on POST response")
	}
}

func TestBuildCSP(t *testing.T) {
	t.Run("no hashes falls back to unsafe-inline", func(t *testing.T) {
		csp := buildCSP(nil)
		if !strings.Contains(csp, "'unsafe-inline'") {
			t.Errorf("expected 'unsafe-inline' in script-src without hashes: %s", csp)
		}
	})

	t.Run("with hashes omits unsafe-inline", func(t *testing.T) {
		csp := buildCSP([]string{"'sha256-test123'"})
		for _, part := range strings.Split(csp, ";") {
			trimmed := strings.TrimSpace(part)
			if strings.HasPrefix(trimmed, "script-src") {
				if strings.Contains(trimmed, "'unsafe-inline'") {
					t.Errorf("script-src contains 'unsafe-inline' with hashes: %s", trimmed)
				}
				if !strings.Contains(trimmed, "'sha256-test123'") {
					t.Errorf("script-src missing hash: %s", trimmed)
				}
			}
			// style-src should still have 'unsafe-inline'
			if strings.HasPrefix(trimmed, "style-src") {
				if !strings.Contains(trimmed, "'unsafe-inline'") {
					t.Errorf("style-src should keep 'unsafe-inline': %s", trimmed)
				}
			}
		}
	})
}
