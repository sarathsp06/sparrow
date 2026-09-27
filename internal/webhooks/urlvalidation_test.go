package webhooks

import (
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
