package client

import (
	"net"
	"strings"
	"testing"
	"time"
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
		{"::ffff:127.0.0.1", true},  // IPv6-mapped loopback
		{"0.1.2.3", true},           // "this network" 0.0.0.0/8
		{"2002:0a00:0001::1", true}, // 6to4 embedding 10.0.0.1
		{"2002:0808:0808::1", false},
		{"64:ff9b::7f00:1", true}, // NAT64 embedding 127.0.0.1
		{"64:ff9b::a9fe:a9fe", true},
		{"64:ff9b::808:808", false}, // NAT64 embedding 8.8.8.8
		{"64:ff9b:1::1", true},      // local-use NAT64
		{"fc00::1", true},           // ULA
		{"fe80::1", true},           // link-local
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

func TestNetworkPolicy(t *testing.T) {
	vpn, err := ParseNetworks([]string{"10.20.0.0/16", "fd12::/48"})
	if err != nil {
		t.Fatal(err)
	}
	metaAllowed, _ := ParseNetworks([]string{"169.254.169.254"})
	tests := []struct {
		name    string
		policy  NetworkPolicy
		ip      string
		allowed bool
	}{
		{"default public", NetworkPolicy{}, "8.8.8.8", true},
		{"default private", NetworkPolicy{}, "10.20.1.1", false},
		{"allowlist hit", NetworkPolicy{AllowedNetworks: vpn}, "10.20.1.1", true},
		{"allowlist hit v6", NetworkPolicy{AllowedNetworks: vpn}, "fd12::5", true},
		{"allowlist miss", NetworkPolicy{AllowedNetworks: vpn}, "10.30.1.1", false},
		{"allowlist keeps loopback blocked", NetworkPolicy{AllowedNetworks: vpn}, "127.0.0.1", false},
		{"allowlist keeps public", NetworkPolicy{AllowedNetworks: vpn}, "8.8.8.8", true},
		{"allow-private loopback", NetworkPolicy{AllowPrivate: true}, "127.0.0.1", true},
		{"allow-private private", NetworkPolicy{AllowPrivate: true}, "192.168.1.5", true},
		{"allow-private blocks metadata", NetworkPolicy{AllowPrivate: true}, "169.254.169.254", false},
		{"allow-private blocks mapped metadata", NetworkPolicy{AllowPrivate: true}, "::ffff:169.254.169.254", false},
		{"allow-private blocks NAT64 metadata", NetworkPolicy{AllowPrivate: true}, "64:ff9b::a9fe:a9fe", false},
		{"allow-private blocks ECS creds", NetworkPolicy{AllowPrivate: true}, "169.254.170.2", false},
		{"allow-private blocks IMDS v6", NetworkPolicy{AllowPrivate: true}, "fd00:ec2::254", false},
		{"allow-private other link-local", NetworkPolicy{AllowPrivate: true}, "169.254.10.10", true},
		{"metadata explicitly allowed", NetworkPolicy{AllowedNetworks: metaAllowed}, "169.254.169.254", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.CheckIP(net.ParseIP(tt.ip))
			if tt.allowed && err != nil {
				t.Fatalf("%s: expected allowed, got %v", tt.ip, err)
			}
			if !tt.allowed && err == nil {
				t.Fatalf("%s: expected blocked", tt.ip)
			}
		})
	}
}

func TestNetworkPolicyHostnames(t *testing.T) {
	if err := (NetworkPolicy{}).CheckHostname("svc.internal"); err == nil {
		t.Fatal("default policy should block .internal names")
	}
	vpn, _ := ParseNetworks([]string{"10.0.0.0/8"})
	if err := (NetworkPolicy{AllowedNetworks: vpn}).CheckHostname("svc.internal"); err != nil {
		t.Fatalf("allowlist policy should defer .internal names to address checks: %v", err)
	}
	if err := (NetworkPolicy{AllowPrivate: true}).CheckHost("does-not-resolve.invalid"); err != nil {
		t.Fatalf("allow-private should not require resolution at registration: %v", err)
	}
	if err := (NetworkPolicy{AllowPrivate: true}).CheckHost("169.254.169.254"); err == nil {
		t.Fatal("allow-private must still block a literal metadata IP")
	}
}

func TestParseNetworks(t *testing.T) {
	nets, err := ParseNetworks([]string{" 10.0.0.0/8 ", "192.168.1.7", "fd00::1", ""})
	if err != nil || len(nets) != 3 {
		t.Fatalf("ParseNetworks = %v, %v", nets, err)
	}
	if !nets[1].Contains(net.ParseIP("192.168.1.7")) || nets[1].Contains(net.ParseIP("192.168.1.8")) {
		t.Fatal("bare IPv4 should be a /32")
	}
	if _, err := ParseNetworks([]string{"not-a-cidr"}); err == nil {
		t.Fatal("expected error for invalid entry")
	}
}

func TestDialControlBlocksMetadataWithAllowPrivate(t *testing.T) {
	c := NewWebhookClient(&Config{Timeout: time.Second, AllowPrivateNetworks: true})
	_, err := c.httpClient.Get("http://169.254.169.254/latest/meta-data/")
	if err == nil || !strings.Contains(err.Error(), "cloud metadata") {
		t.Fatalf("expected metadata dial to be refused, got %v", err)
	}
}
