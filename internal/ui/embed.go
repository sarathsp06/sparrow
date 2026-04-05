// Package ui provides an embedded SPA file server for the Sparrow web frontend.
//
// The static build output from web/build/ is embedded at compile time.
// When SPARROW_SERVE_UI=true, the Go binary serves the frontend on the
// same port as the Connect-RPC API. Chi router ensures API routes always
// take precedence; the UI handler is registered as the NotFound fallback.
//
// Build the frontend first:
//
//	cd web && npm run build:static
//	cd .. && go build ./cmd/server
package ui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
)

//go:embed all:dist
var embeddedFS embed.FS

// Config holds runtime configuration injected into the SPA HTML.
type Config struct {
	// APIKey is injected so the frontend can authenticate API requests.
	// Empty string means no authentication.
	APIKey string `json:"apiKey,omitempty"`
}

// Handler returns an http.Handler that serves the embedded SPA.
// It serves static files from the embedded filesystem, and falls back to
// index.html for any path that doesn't match a static file (SPA client-side routing).
//
// The config parameter is injected into index.html as a window.__SPARROW_CONFIG__
// global so the SPA can read runtime configuration without a rebuild.
func Handler(logger *slog.Logger, config *Config) http.Handler {
	// Strip the "dist" prefix from the embedded FS so files are served from root.
	staticFS, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		// This should never happen since "dist" is embedded at compile time.
		panic("ui: failed to create sub filesystem: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(staticFS))

	// Pre-render the config script tag to inject into index.html.
	configScript := buildConfigScript(config)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the requested file directly.
		path := strings.TrimPrefix(r.URL.Path, "/")

		// Check if the file exists in the embedded FS.
		if path != "" {
			if f, err := staticFS.Open(path); err == nil {
				_ = f.Close()
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
			logger.ErrorContext(r.Context(), "ui: index.html not found in embedded filesystem — was the frontend built?")
			http.Error(w, "UI not available. Build the frontend with: cd web && npm run build:static", http.StatusNotFound)
			return
		}

		// Inject runtime config into the HTML <head>.
		html := string(indexBytes)
		if configScript != "" {
			html = strings.Replace(html, "</head>", configScript+"</head>", 1)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte(html)) //nolint:errcheck
	})
}

// buildConfigScript returns a <script> tag that sets window.__SPARROW_CONFIG__,
// or an empty string if there is no config to inject.
func buildConfigScript(config *Config) string {
	if config == nil {
		return ""
	}
	data, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	// Only inject if there's something meaningful (not just "{}").
	if string(data) == "{}" {
		return ""
	}
	return fmt.Sprintf(`<script>window.__SPARROW_CONFIG__=%s;</script>`, data)
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
