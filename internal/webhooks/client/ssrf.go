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

// ValidateIP checks whether an IP address is safe for outbound webhook delivery.
// It blocks loopback, private, link-local, multicast, unspecified addresses,
// cloud metadata endpoints, and IPv6-mapped IPv4 private addresses.
func ValidateIP(ip net.IP) error {
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

	// Block AWS/GCP/Azure metadata endpoint: 169.254.169.254
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return fmt.Errorf("cloud metadata endpoint address is not allowed")
	}

	// Block IPv6-mapped IPv4 private addresses
	if ip4 := ip.To4(); ip4 != nil {
		if ip4.IsLoopback() || ip4.IsPrivate() || ip4.IsLinkLocalUnicast() {
			return fmt.Errorf("address maps to a restricted IPv4 range")
		}
	}

	return nil
}

// validateRedirectURL checks that an HTTP redirect target is safe against SSRF.
// It validates scheme, hostname, and resolved IP addresses.
func validateRedirectURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid redirect URL: %w", err)
	}

	// Only allow http and https schemes
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("redirect to disallowed scheme %q", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("redirect URL must have a non-empty host")
	}

	// Block well-known internal hostnames
	lower := strings.ToLower(host)
	if lower == "localhost" ||
		lower == "metadata.google.internal" ||
		strings.HasSuffix(lower, ".internal") ||
		strings.HasSuffix(lower, ".local") {
		return fmt.Errorf("redirect to internal/reserved hostname %q is not allowed", host)
	}

	// If the host is a literal IP, validate it directly
	if ip := net.ParseIP(host); ip != nil {
		return ValidateIP(ip)
	}

	// For hostnames, resolve and validate all IPs
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve redirect host %q: %w", host, err)
	}
	for _, ip := range ips {
		if err := ValidateIP(ip); err != nil {
			return fmt.Errorf("redirect host %q resolves to blocked address: %w", host, err)
		}
	}

	return nil
}

// ssrfSafeCheckRedirect is an http.Client CheckRedirect function that
// prevents SSRF via HTTP redirects. Each redirect target is validated
// against the SSRF blocklist before following.
func ssrfSafeCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	return validateRedirectURL(req.URL.String())
}

// ssrfDialControl returns a net.Dialer Control function that validates
// resolved IP addresses at connect time, preventing DNS rebinding attacks.
// This is called after DNS resolution but before the TCP connection is
// established, closing the TOCTOU gap between URL validation at registration
// time and actual delivery.
func ssrfDialControl(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("non-IP address %q in dial", host)
	}
	return ValidateIP(ip)
}
