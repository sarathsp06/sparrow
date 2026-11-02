package webhooks

import (
	"strings"
	"testing"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
)

func TestValidateWebhookURLPolicies(t *testing.T) {
	vpn, err := client.ParseNetworks([]string{"10.20.0.0/16"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		url    string
		policy client.NetworkPolicy
		ok     bool
	}{
		{"bad scheme", "ftp://example.com", client.NetworkPolicy{}, false},
		{"default blocks private", "http://10.20.1.1/hook", client.NetworkPolicy{}, false},
		{"default blocks localhost", "http://localhost:9000", client.NetworkPolicy{}, false},
		{"allowlist permits VPN", "http://10.20.1.1/hook", client.NetworkPolicy{AllowedNetworks: vpn}, true},
		{"allowlist blocks other private", "http://10.21.1.1/hook", client.NetworkPolicy{AllowedNetworks: vpn}, false},
		{"allow-private permits loopback", "http://127.0.0.1:9000", client.NetworkPolicy{AllowPrivate: true}, true},
		{"allow-private permits unresolved name", "http://receiver.test:9000", client.NetworkPolicy{AllowPrivate: true}, true},
		{"allow-private blocks metadata", "http://169.254.169.254/latest", client.NetworkPolicy{AllowPrivate: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url, tt.policy)
			if tt.ok != (err == nil) {
				t.Fatalf("ValidateWebhookURL(%q) err = %v, want ok=%v", tt.url, err, tt.ok)
			}
		})
	}
}

func TestCanonicalHeaders(t *testing.T) {
	got, err := canonicalHeaders("secret_headers", map[string]string{"x-api-key": "k", "AUTHORIZATION": ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["X-Api-Key"] != "k" || got["Authorization"] != "" {
		t.Errorf("canonicalHeaders = %v, want X-Api-Key and Authorization", got)
	}
	if got, _ := canonicalHeaders("secret_headers", nil); got != nil {
		t.Errorf("nil stays nil, got %v", got)
	}

	_, err = canonicalHeaders("secret_headers", map[string]string{"x-api-key": "a", "X-Api-Key": "b"})
	if err == nil || !strings.Contains(err.Error(), `"X-Api-Key" and "x-api-key" are the same header`) {
		t.Errorf("duplicate by case: err = %v", err)
	}
}

// Both origin checks (webhook URL moves and redirects) share one rule.
func TestSameOriginStrings(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"https://a.com/x", "https://a.com:443/y", true},
		{"https://A.com/", "https://a.com/", true},
		{"https://a.com/", "https://a.com:8443/", false},
		{"https://a.com/", "http://a.com/", false},
		{"https://a.com/", "://bad", false},
	} {
		if got := sameOrigin(c.a, c.b); got != c.want {
			t.Errorf("sameOrigin(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
