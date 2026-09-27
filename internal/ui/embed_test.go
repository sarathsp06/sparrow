package ui

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

const indexHTML = `<!doctype html><html><head><script src="/config.js"></script></head><body></body></html>`

var testFS = fstest.MapFS{
	"index.html":              {Data: []byte(indexHTML)},
	"config.js":               {Data: []byte(`window.__SPARROW_CONFIG__ = window.__SPARROW_CONFIG__ || {};`)},
	"_app/immutable/entry.js": {Data: []byte(`console.log("app")`)},
}

func serve(t *testing.T, path string) (*http.Response, string) {
	t.Helper()
	h := newHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), testFS)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	res := rec.Result()
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

// The server never writes a credential (or any config) into served pages:
// admin and portal pages alike are the built index.html, byte for byte.
func TestHandlerServesIndexUntouched(t *testing.T) {
	for _, path := range []string{"/", "/webhooks", "/events/instances/abc", "/portal", "/portal/webhooks"} {
		_, body := serve(t, path)
		if body != indexHTML {
			t.Fatalf("%s: body = %q, want untouched index.html", path, body)
		}
	}
}

func TestHandlerServesStaticConfigAndAssets(t *testing.T) {
	res, body := serve(t, "/config.js")
	if res.StatusCode != http.StatusOK || strings.Contains(body, "apiKey") {
		t.Fatalf("/config.js: status %d body %q", res.StatusCode, body)
	}
	if cc := res.Header.Get("Cache-Control"); strings.Contains(cc, "immutable") {
		t.Fatalf("/config.js must not be cached as immutable (operators edit it), got %q", cc)
	}

	res, _ = serve(t, "/_app/immutable/entry.js")
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("hashed asset Cache-Control = %q, want immutable", cc)
	}
}

func TestComputeInlineScriptHashes(t *testing.T) {
	t.Run("no inline scripts", func(t *testing.T) {
		html := []byte(`<!doctype html><html><head><script src="/app.js"></script></head><body></body></html>`)
		hashes := computeInlineScriptHashes(html)
		if len(hashes) != 0 {
			t.Errorf("expected no hashes for external-only scripts, got %v", hashes)
		}
	})

	t.Run("one inline script", func(t *testing.T) {
		body := `console.log("hello")`
		html := []byte(`<html><body><script>` + body + `</script></body></html>`)
		hashes := computeInlineScriptHashes(html)
		if len(hashes) != 1 {
			t.Fatalf("expected 1 hash, got %d: %v", len(hashes), hashes)
		}
		h := sha256.Sum256([]byte(body))
		want := "'sha256-" + base64.StdEncoding.EncodeToString(h[:]) + "'"
		if hashes[0] != want {
			t.Errorf("hash = %s, want %s", hashes[0], want)
		}
	})

	t.Run("mixed inline and external", func(t *testing.T) {
		html := []byte(`<html><head><script src="/config.js"></script></head><body><script>var x=1;</script></body></html>`)
		hashes := computeInlineScriptHashes(html)
		if len(hashes) != 1 {
			t.Fatalf("expected 1 hash (inline only), got %d: %v", len(hashes), hashes)
		}
	})

	t.Run("sveltekit bootstrap script", func(t *testing.T) {
		// Simulate a SvelteKit-style inline bootstrap script.
		html := []byte(`<!doctype html><html><head><script src="/config.js"></script></head><body>
<div style="display: contents">
	<script>
		{
			__sveltekit_test = { base: "" };
			import("/_app/entry/start.js").then(([kit, app]) => {
				kit.start(app, document.currentScript.parentElement);
			});
		}
	</script>
</div></body></html>`)
		hashes := computeInlineScriptHashes(html)
		if len(hashes) != 1 {
			t.Fatalf("expected 1 hash, got %d: %v", len(hashes), hashes)
		}
		if !strings.HasPrefix(hashes[0], "'sha256-") {
			t.Errorf("hash format wrong: %s", hashes[0])
		}
	})
}

func TestInlineScriptHashesFromFS(t *testing.T) {
	t.Run("no index.html", func(t *testing.T) {
		fs := fstest.MapFS{}
		hashes := inlineScriptHashesFromFS(fs)
		if hashes != nil {
			t.Errorf("expected nil for missing index.html, got %v", hashes)
		}
	})

	t.Run("index with no inline scripts", func(t *testing.T) {
		fs := fstest.MapFS{
			"index.html": {Data: []byte(indexHTML)}, // only has <script src=...>
		}
		hashes := inlineScriptHashesFromFS(fs)
		if len(hashes) != 0 {
			t.Errorf("expected no hashes, got %v", hashes)
		}
	})
}
