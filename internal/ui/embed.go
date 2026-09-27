// Package ui provides an embedded SPA file server for the Sparrow web frontend.
//
// The static build output from web/build/ is embedded at compile time.
// When SPARROW_SERVE_UI=true, the Go binary serves the frontend on the
// same port as the REST API. Chi router ensures API routes always
// take precedence; the UI handler is registered as the NotFound fallback.
//
// Build the frontend first:
//
//	cd web && npm run build
//	cd .. && go build ./cmd/server
package ui

import (
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"io/fs"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

//go:embed all:dist
var embeddedFS embed.FS

// Handler returns an http.Handler that serves the embedded SPA.
// It serves static files from the embedded filesystem, and falls back to
// index.html for any path that doesn't match a static file (SPA client-side routing).
//
// Pages are served exactly as built: no credential is ever written into them.
// The UI signs in with the API key, an access token, or an invite link.
func Handler(logger *slog.Logger) http.Handler {
	// Strip the "dist" prefix from the embedded FS so files are served from root.
	staticFS, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		// This should never happen since "dist" is embedded at compile time.
		panic("ui: failed to create sub filesystem: " + err.Error())
	}
	return newHandler(logger, staticFS)
}

// newHandler serves the SPA from staticFS (the build output root).
func newHandler(logger *slog.Logger, staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))

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
			http.Error(w, "UI not available. Build the frontend with: cd web && npm run build", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(indexBytes) //nolint:errcheck
	})
}

// scriptRe matches all <script ...>...</script> blocks including empty-body
// ones (external scripts with src="..."). The opening-tag portion is captured
// separately so we can check for a src attribute without re-parsing.
var scriptRe = regexp.MustCompile(`(?s)(<script(?:\s[^>]*)?>)(.*?)</script>`)
var srcAttrRe = regexp.MustCompile(`\bsrc\s*=`)

// InlineScriptHashes reads index.html from the embedded build output,
// extracts the bodies of all inline <script> tags (those without a src
// attribute), and returns their SHA-256 hashes in CSP format:
// 'sha256-<base64>'.
//
// These hashes let the Content-Security-Policy use script hashes instead
// of 'unsafe-inline', so only the exact scripts the build produced are
// allowed to execute. Returns nil when no built frontend is embedded.
func InlineScriptHashes() []string {
	data, err := fs.ReadFile(embeddedFS, "dist/index.html")
	if err != nil {
		return nil
	}
	return computeInlineScriptHashes(data)
}

// inlineScriptHashesFromFS is the testable variant that accepts an arbitrary FS.
func inlineScriptHashesFromFS(staticFS fs.FS) []string {
	data, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		return nil
	}
	return computeInlineScriptHashes(data)
}

// computeInlineScriptHashes extracts inline script bodies from raw HTML
// and returns their CSP sha256 hashes.
func computeInlineScriptHashes(html []byte) []string {
	matches := scriptRe.FindAllSubmatch(html, -1)
	var hashes []string
	for _, m := range matches {
		openTag := m[1] // e.g. `<script>` or `<script src="...">`
		body := m[2]
		// Skip scripts with a src attribute — those are external.
		if srcAttrRe.Match(openTag) {
			continue
		}
		// Skip empty inline scripts (nothing to hash).
		if len(body) == 0 {
			continue
		}
		h := sha256.Sum256(body)
		hashes = append(hashes, "'sha256-"+base64.StdEncoding.EncodeToString(h[:])+"'")
	}
	return hashes
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
