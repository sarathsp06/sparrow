package client

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
)

// maxRedirects is the maximum number of HTTP redirects allowed per request.
const maxRedirects = 10

func mustCIDRs(cidrs ...string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		nets = append(nets, n)
	}
	return nets
}

// blockedPrefixes are reserved/special-use ranges not caught by the net.IP
// classification helpers in ValidateIP.
var blockedPrefixes = mustCIDRs(
	"0.0.0.0/8",      // RFC 1122 "this network" (0.x.y.z reaches the local host on Linux)
	"100.64.0.0/10",  // RFC 6598 carrier-grade NAT
	"192.0.0.0/24",   // RFC 6890 IETF protocol assignments
	"198.18.0.0/15",  // RFC 2544 benchmarking
	"240.0.0.0/4",    // RFC 1112 reserved (includes 255.255.255.255 broadcast)
	"64:ff9b:1::/48", // RFC 8215 local-use NAT64 (translates to internal IPv4)
)

// metadataPrefixes are cloud instance-metadata endpoints. They hand out
// credentials, so they stay blocked even with AllowPrivateNetworks — only an
// explicit AllowedNetworks entry can reach them.
var metadataPrefixes = mustCIDRs(
	"169.254.169.254/32", // AWS, GCP, Azure, OpenStack, DigitalOcean, ...
	"169.254.170.2/32",   // AWS ECS task metadata / credentials
	"169.254.170.23/32",  // AWS EKS Pod Identity agent
	"100.100.100.200/32", // Alibaba Cloud
	"fd00:ec2::254/128",  // AWS IMDS over IPv6
	"fd00:ec2::23/128",   // AWS EKS Pod Identity agent over IPv6
)

// Transition prefixes that embed an IPv4 address: the embedded address is
// validated like any other, so public 6to4/NAT64 targets still work.
var (
	sixToFour = mustCIDRs("2002::/16")[0]    // RFC 3056: 2002:AABB:CCDD::/48 embeds A.B.C.D
	nat64     = mustCIDRs("64:ff9b::/96")[0] // RFC 6052: well-known prefix, IPv4 in the last 32 bits
)

// ValidateIP checks whether an IP address is safe for outbound webhook delivery.
// It blocks loopback, private, link-local (which covers the 169.254.169.254
// cloud metadata endpoint), multicast, unspecified addresses, reserved
// special-use ranges, and IPv6 forms that embed a restricted IPv4 address
// (IPv4-mapped, 6to4, NAT64).
func ValidateIP(ip net.IP) error {
	// Canonicalize IPv4-mapped IPv6 (::ffff:a.b.c.d) so every IPv4 rule applies.
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	if ip.IsLoopback() {
		return fmt.Errorf("loopback addresses are not allowed")
	}
	if ip.IsPrivate() {
		return fmt.Errorf("private network addresses are not allowed")
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("link-local addresses are not allowed")
	}
	if ip.IsUnspecified() {
		return fmt.Errorf("unspecified address (0.0.0.0) is not allowed")
	}
	if ip.IsMulticast() {
		return fmt.Errorf("multicast addresses are not allowed")
	}

	// Block reserved/special-use ranges (CGNAT, benchmarking, class E, ...).
	for _, n := range blockedPrefixes {
		if n.Contains(ip) {
			return fmt.Errorf("address in reserved range %s is not allowed", n)
		}
	}

	// IPv6 transition addresses carry an IPv4 destination: validate it.
	if embedded := embeddedIPv4(ip); embedded != nil {
		if err := ValidateIP(embedded); err != nil {
			return fmt.Errorf("address embeds a restricted IPv4 address %s: %w", embedded, err)
		}
	}

	return nil
}

// embeddedIPv4 returns the IPv4 address carried by a 6to4 or NAT64 address,
// or nil for any other address.
func embeddedIPv4(ip net.IP) net.IP {
	ip16 := ip.To16()
	if ip16 == nil || ip.To4() != nil {
		return nil
	}
	switch {
	case sixToFour.Contains(ip16):
		return net.IPv4(ip16[2], ip16[3], ip16[4], ip16[5]).To4()
	case nat64.Contains(ip16):
		return net.IPv4(ip16[12], ip16[13], ip16[14], ip16[15]).To4()
	}
	return nil
}

// isMetadataIP reports whether ip is a cloud instance-metadata endpoint,
// including its IPv4-mapped, 6to4 and NAT64 forms.
func isMetadataIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	for _, n := range metadataPrefixes {
		if n.Contains(ip) {
			return true
		}
	}
	if embedded := embeddedIPv4(ip); embedded != nil {
		return isMetadataIP(embedded)
	}
	return false
}

// NetworkPolicy decides which destinations webhook deliveries may reach.
//
//   - Default (zero value): only public addresses (see ValidateIP).
//   - AllowedNetworks: these CIDRs are reachable in addition to public
//     addresses — the precise way to deliver to internal services (e.g. a
//     VPN's 10.20.0.0/16) without opening everything else.
//   - AllowPrivate: every address is reachable (loopback, private, ...) —
//     for local development and tests.
//
// Cloud metadata endpoints stay blocked in every mode unless an
// AllowedNetworks entry explicitly covers them.
type NetworkPolicy struct {
	AllowPrivate    bool
	AllowedNetworks []*net.IPNet
}

// CheckIP returns an error if ip is not a permitted delivery destination.
func (p NetworkPolicy) CheckIP(ip net.IP) error {
	for _, n := range p.AllowedNetworks {
		if n.Contains(ip) {
			return nil
		}
	}
	if isMetadataIP(ip) {
		return fmt.Errorf("cloud metadata address %s is not allowed (list it in SPARROW_ALLOWED_NETWORKS to permit it)", ip)
	}
	if p.AllowPrivate {
		return nil
	}
	return ValidateIP(ip)
}

// CheckHostname rejects well-known internal hostnames before resolution.
// Name-based blocking only applies in the default mode: once internal
// destinations are allowed, internal names (e.g. svc.internal) are legitimate
// and the resolved addresses are checked instead.
func (p NetworkPolicy) CheckHostname(host string) error {
	if p.AllowPrivate || len(p.AllowedNetworks) > 0 {
		return nil
	}
	lower := strings.ToLower(strings.TrimSuffix(host, "."))
	if lower == "localhost" ||
		strings.HasSuffix(lower, ".localhost") ||
		lower == "metadata.google.internal" ||
		strings.HasSuffix(lower, ".internal") ||
		strings.HasSuffix(lower, ".local") {
		return fmt.Errorf("internal/reserved hostname %q is not allowed", host)
	}
	return nil
}

// CheckHost validates a URL host: a literal IP directly, a hostname by name
// and by every address it resolves to.
func (p NetworkPolicy) CheckHost(host string) error {
	if ip := net.ParseIP(host); ip != nil {
		return p.CheckIP(ip)
	}
	if err := p.CheckHostname(host); err != nil {
		return err
	}
	if p.AllowPrivate {
		// Every resolved address is permitted except metadata, which the
		// dialer rejects at connect time; don't require the name to resolve
		// yet (matches the pre-policy behaviour of this mode).
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve host %q: %w", host, err)
	}
	for _, ip := range ips {
		if err := p.CheckIP(ip); err != nil {
			return fmt.Errorf("host %q resolves to blocked address: %w", host, err)
		}
	}
	return nil
}

// checkRedirect is an http.Client CheckRedirect function that bounds the
// redirect chain, restricts schemes to http/https, and validates each target
// against the policy before following (the dialer re-checks at connect time).
func (p NetworkPolicy) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	return p.validateRedirectURL(req.URL)
}

func (p NetworkPolicy) validateRedirectURL(u *url.URL) error {
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("redirect to disallowed scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("redirect URL must have a non-empty host")
	}
	if err := p.CheckHost(host); err != nil {
		return fmt.Errorf("redirect: %w", err)
	}
	return nil
}

// dialControl returns a net.Dialer Control function that validates resolved
// IP addresses at connect time, preventing DNS rebinding attacks. It runs
// after DNS resolution but before the TCP connection is established, closing
// the TOCTOU gap between URL validation at registration and actual delivery.
func (p NetworkPolicy) dialControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("non-IP address %q in dial", host)
	}
	return p.CheckIP(ip)
}

// ParseNetworks parses CIDRs or bare IP addresses (treated as /32 or /128).
func ParseNetworks(entries []string) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.Contains(e, "/") {
			ip := net.ParseIP(e)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP or CIDR %q", e)
			}
			bits := 128
			if ip.To4() != nil {
				ip, bits = ip.To4(), 32
			}
			nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		_, n, err := net.ParseCIDR(e)
		if err != nil {
			return nil, fmt.Errorf("invalid IP or CIDR %q: %w", e, err)
		}
		nets = append(nets, n)
	}
	return nets, nil
}
