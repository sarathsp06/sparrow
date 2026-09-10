package client

import (
	"net"
	"testing"
)

func TestValidateIP(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"8.8.8.8", false},
		{"2600::1", false},
		{"127.0.0.1", true},         // loopback
		{"10.1.2.3", true},          // private
		{"169.254.169.254", true},   // link-local / cloud metadata
		{"100.64.0.1", true},        // CGNAT (RFC 6598)
		{"192.0.0.10", true},        // IETF protocol assignments (RFC 6890)
		{"198.18.5.5", true},        // benchmarking (RFC 2544)
		{"240.1.2.3", true},         // reserved class E (RFC 1112)
		{"::ffff:100.64.0.1", true}, // IPv6-mapped CGNAT
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tt.ip)
			}
			err := ValidateIP(ip)
			if tt.blocked && err == nil {
				t.Errorf("expected %s to be blocked", tt.ip)
			}
			if !tt.blocked && err != nil {
				t.Errorf("expected %s to be allowed, got %v", tt.ip, err)
			}
		})
	}
}
