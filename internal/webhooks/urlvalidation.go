package webhooks

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
)

// ValidateWebhookURL validates a webhook URL to prevent SSRF attacks.
// It ensures the URL:
//   - Uses http or https scheme only
//   - Does not point to loopback, private, or link-local addresses
//   - Does not target cloud metadata endpoints
//   - Has a valid, non-empty host
func ValidateWebhookURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("invalid URL scheme %q: only http and https are allowed", parsed.Scheme)
	}

	// Ensure host is not empty
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("URL must have a non-empty host")
	}

	// Check for IP addresses targeting internal networks
	ip := net.ParseIP(host)
	if ip != nil {
		if err := client.ValidateIP(ip); err != nil {
			return err
		}
	} else {
		// It's a hostname — block well-known internal hostnames
		lower := strings.ToLower(host)
		if lower == "localhost" ||
			lower == "metadata.google.internal" ||
			strings.HasSuffix(lower, ".internal") ||
			strings.HasSuffix(lower, ".local") {
			return fmt.Errorf("URL host %q is not allowed: internal/reserved hostname", host)
		}

		// Resolve the hostname and validate all resulting IPs.
		// This prevents DNS-based SSRF where a public hostname resolves
		// to a private IP.
		ips, err := net.LookupIP(host)
		if err != nil {
			return fmt.Errorf("cannot resolve URL host %q: %w", host, err)
		}
		for _, resolved := range ips {
			if err := client.ValidateIP(resolved); err != nil {
				return fmt.Errorf("URL host %q resolves to blocked address: %w", host, err)
			}
		}
	}

	return nil
}
