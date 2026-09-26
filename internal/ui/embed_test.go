package ui

import (
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

func serve(t *testing.T, cfg *Config, path string) (*http.Response, string) {
	t.Helper()
	h := newHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), cfg, testFS)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	res := rec.Result()
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

func TestHandlerInjectsAPIKeyForAdminPages(t *testing.T) {
	for _, path := range []string{"/", "/webhooks", "/events/instances/abc"} {
		_, body := serve(t, &Config{APIKey: "secret"}, path)
		want := `<script>window.__SPARROW_CONFIG__={"apiKey":"secret"};</script></head>`
		if !strings.Contains(body, want) {
			t.Fatalf("%s: body %q does not contain %q", path, body, want)
		}
		// The injected script must come after /config.js so it wins.
		if strings.Index(body, "/config.js") > strings.Index(body, "__SPARROW_CONFIG__={") {
			t.Fatalf("%s: injected config precedes config.js", path)
		}
	}
}

func TestHandlerNeverInjectsAPIKeyIntoPortal(t *testing.T) {
	for _, path := range []string{"/portal", "/portal/", "/portal/webhooks"} {
		_, body := serve(t, &Config{APIKey: "secret"}, path)
		if strings.Contains(body, "secret") {
			t.Fatalf("%s: portal page leaked the admin API key: %q", path, body)
		}
	}
}

func TestHandlerNoInjectionWithoutKey(t *testing.T) {
	_, body := serve(t, &Config{}, "/webhooks")
	if body != indexHTML {
		t.Fatalf("body = %q, want untouched index.html", body)
	}
}

func TestHandlerEscapesInjectedKey(t *testing.T) {
	_, body := serve(t, &Config{APIKey: `</script><script>alert(1)</script>`}, "/")
	if strings.Contains(body, "<script>alert(1)") {
		t.Fatalf("API key was not escaped: %q", body)
	}
}

func TestHandlerServesStaticConfigAndAssets(t *testing.T) {
	res, body := serve(t, &Config{APIKey: "secret"}, "/config.js")
	if res.StatusCode != http.StatusOK || strings.Contains(body, "secret") {
		t.Fatalf("/config.js: status %d body %q", res.StatusCode, body)
	}
	if cc := res.Header.Get("Cache-Control"); strings.Contains(cc, "immutable") {
		t.Fatalf("/config.js must not be cached as immutable (operators edit it), got %q", cc)
	}

	res, _ = serve(t, nil, "/_app/immutable/entry.js")
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("hashed asset Cache-Control = %q, want immutable", cc)
	}
}
