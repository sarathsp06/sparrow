package ui

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// TestEmbeddedBuild checks the handler over the real embedded build: with a
// built frontend every unknown path gets index.html, without one (a plain
// go build that skipped the UI) it answers 404 with a hint.
func TestEmbeddedBuild(t *testing.T) {
	h := Handler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/webhooks/some-client-route", nil))

	if Available() {
		if rec.Code != http.StatusOK {
			t.Fatalf("built UI: GET of a client route = %d, want 200", rec.Code)
		}
		return
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("no built UI: GET = %d, want 404", rec.Code)
	}
	if InlineScriptHashes() != nil {
		t.Fatal("no built UI must yield no script hashes")
	}
}

func TestHandlerWithoutIndex(t *testing.T) {
	h := newHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), fstest.MapFS{"robots.txt": {Data: []byte("x")}})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing index.html: status %d, want 404", rec.Code)
	}
	if inlineScriptHashesFromFS(fstest.MapFS{}) != nil {
		t.Fatal("no index.html must yield no script hashes")
	}
}
