package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
)

func TestDocFetcher_ExtractsTextAndRespectsPolicy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/docs":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head><title>Hooks</title><style>p{}</style><script>x()</script></head>
<body><nav>Home Pricing</nav><h1>Webhook payload</h1><p>POST a JSON body:</p><pre>{"title": "...", "body": "..."}</pre><footer>© 2026</footer></body></html>`))
		case "/plain":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("Send   title and\n\n\n\nbody."))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	// httptest listens on loopback, so the default policy must refuse it
	// without connecting; AllowPrivate lets the test through.
	strict := NewDocFetcher(client.NetworkPolicy{})
	if _, err := strict(context.Background(), srv.URL+"/docs"); err == nil {
		t.Fatal("loopback docs_url should be refused by the default policy")
	}

	open := NewDocFetcher(client.NetworkPolicy{AllowPrivate: true})
	text, err := open(context.Background(), srv.URL+"/docs")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Webhook payload", "POST a JSON body:", `{"title": "...", "body": "..."}`} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	for _, drop := range []string{"x()", "p{}", "Home Pricing", "© 2026", "<h1>"} {
		if strings.Contains(text, drop) {
			t.Errorf("should have dropped %q, got:\n%s", drop, text)
		}
	}

	text, err = open(context.Background(), srv.URL+"/plain")
	if err != nil || text != "Send title and\n\nbody." {
		t.Fatalf("plain text: %q err=%v", text, err)
	}

	if _, err := open(context.Background(), srv.URL+"/missing"); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("404 should be reported: %v", err)
	}
	for _, bad := range []string{"ftp://example.com/x", "not a url", "https:///nohost"} {
		if _, err := open(context.Background(), bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
