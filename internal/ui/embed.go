// Package ui provides an embedded SPA file server for the Sparrow web frontend.
//
// The static build output from web/build/ is embedded at compile time.
// When SPARROW_SERVE_UI=true, the Go binary serves the frontend on the same
// port as the Connect-RPC API, eliminating the need for a separate frontend
// deployment in self-hosted scenarios.
//
// Build the frontend first:
//
//	cd web && npm run build:static
//	cd .. && go build ./cmd/server
package ui

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
)

//go:embed all:dist
var embeddedFS embed.FS

// Handler returns an http.Handler that serves the embedded SPA.
// It serves static files from the embedded filesystem, and falls back to
// index.html for any path that doesn't match a static file (SPA client-side routing).
//
// The apiPrefixes parameter specifies URL path prefixes that should NOT be
// handled by the UI (e.g., "/sparrow.", "/health", "/ready"). These are left
// for the API handlers registered on the same mux.
func Handler(logger *slog.Logger, apiPrefixes []string) http.Handler {
	// Strip the "dist" prefix from the embedded FS so files are served from root.
	staticFS, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		// This should never happen since "dist" is embedded at compile time.
		panic("ui: failed to create sub filesystem: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't serve UI for API routes — let them 404 naturally from the mux.
		for _, prefix := range apiPrefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				http.NotFound(w, r)
				return
			}
		}

		// Try to serve the requested file directly.
		path := strings.TrimPrefix(r.URL.Path, "/")

		// Check if the file exists in the embedded FS.
		if path != "" {
			if f, err := staticFS.Open(path); err == nil {
				f.Close()
				// File exists — serve it with proper caching.
				// Immutable assets (hashed filenames) get long cache.
				if strings.HasPrefix(path, "_app/immutable/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// File doesn't exist — serve index.html for SPA client-side routing.
		indexBytes, err := fs.ReadFile(staticFS, "index.html")
		if err != nil {
			logger.Error("ui: index.html not found in embedded filesystem — was the frontend built?")
			http.Error(w, "UI not available. Build the frontend with: cd web && npm run build:static", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(indexBytes) //nolint:errcheck
	})
}

// Available reports whether the embedded UI contains a built frontend.
// Returns false if only the placeholder .gitkeep exists.
func Available() bool {
	entries, err := fs.ReadDir(embeddedFS, "dist")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() == "index.html" {
			return true
		}
	}
	return false
}
