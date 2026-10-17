package client

import (
	"net"
	"net/netip"
	"testing"
)

// restrictedV4 are IPv4 ranges ValidateIP must always reject, written
// independently of ssrf.go so the fuzzer checks it against a second opinion.
var restrictedV4 = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
}

// ipv6Embeddings returns every IPv6 form that carries v4 as its destination.
func ipv6Embeddings(v4 [4]byte) map[string]net.IP {
	inv := [4]byte{^v4[0], ^v4[1], ^v4[2], ^v4[3]}
	return map[string]net.IP{
		"mapped":     net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, v4[0], v4[1], v4[2], v4[3]},
		"compatible": net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, v4[0], v4[1], v4[2], v4[3]},
		"translated": net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0, 0, v4[0], v4[1], v4[2], v4[3]},
		"6to4":       net.IP{0x20, 0x02, v4[0], v4[1], v4[2], v4[3], 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		"nat64":      net.IP{0, 0x64, 0xff, 0x9b, 0, 0, 0, 0, 0, 0, 0, 0, v4[0], v4[1], v4[2], v4[3]},
		"teredo":     net.IP{0x20, 0x01, 0, 0, 0x41, 0x36, 0xe3, 0x78, 0x80, 0, 0x63, 0xbf, inv[0], inv[1], inv[2], inv[3]},
	}
}

// FuzzCheckIPEmbeddings checks that every IPv6 form embedding an IPv4
// address gets the same verdict as the IPv4 address itself, in the default
// policy and (for metadata endpoints) with AllowPrivate.
func FuzzCheckIPEmbeddings(f *testing.F) {
	for _, s := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "169.254.170.2",
		"100.100.100.200", "8.8.8.8", "0.0.0.1", "255.255.255.255", "192.168.1.1", "1.1.1.1"} {
		f.Add([]byte(net.ParseIP(s).To4()))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) != 4 {
			return
		}
		v4 := [4]byte(b)
		addr := netip.AddrFrom4(v4)
		for _, p := range restrictedV4 {
			if p.Contains(addr) && ValidateIP(net.IP(b)) == nil {
				t.Fatalf("ValidateIP(%s) allowed an address in %s", addr, p)
			}
		}
		for _, policy := range []NetworkPolicy{{}, {AllowPrivate: true}} {
			want := policy.CheckIP(net.IP(b)) == nil
			for form, ip := range ipv6Embeddings(v4) {
				if got := policy.CheckIP(ip) == nil; got != want {
					t.Fatalf("policy %+v: %s form %s of %s allowed=%v, IPv4 allowed=%v", policy, form, ip, addr, got, want)
				}
			}
		}
	})
}

// FuzzCheckIPParse feeds arbitrary strings through net.ParseIP and checks the
// verdict survives a round trip through the canonical text form, so the
// literal-IP path in CheckHost cannot be dodged by spelling an address
// differently.
func FuzzCheckIPParse(f *testing.F) {
	for _, s := range []string{"127.0.0.1", "::1", "::ffff:127.0.0.1", "::127.0.0.1", "::ffff:0:127.0.0.1",
		"2001:0:4136:e378:8000:63bf:80ff:fffe", "2002:7f00:1::", "64:ff9b::7f00:1", "fe80::1%eth0",
		"0:0:0:0:0:ffff:7f00:1", "[::1]", "0177.0.0.1", "2130706433", "8.8.8.8"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		ip := net.ParseIP(s)
		if ip == nil {
			return
		}
		for _, policy := range []NetworkPolicy{{}, {AllowPrivate: true}} {
			a := policy.CheckIP(ip) == nil
			b := policy.CheckIP(net.ParseIP(ip.String())) == nil
			if a != b {
				t.Fatalf("policy %+v: %q allowed=%v but canonical %q allowed=%v", policy, s, a, ip.String(), b)
			}
		}
	})
}
