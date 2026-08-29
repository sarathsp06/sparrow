package webhooks

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// ValidateWebhookURL validates a webhook URL to prevent SSRF attacks.
// It ensures the URL:
//   - Uses http or https scheme only
//   - Does not point to loopback, private, or link-local addresses
//   - Does not target cloud metadata endpoints
//   - Has a valid, non-empty host
//
// All errors returned are *svcerrors.ServiceError with svcerrors.InvalidArgument,
// so they propagate through toGRPCError to the client as actionable messages.
func ValidateWebhookURL(rawURL string, allowPrivateNetworks bool) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return svcerrors.Wrapf(err, svcerrors.InvalidArgument, "invalid URL: %v", err)
	}

	// Only allow http and https schemes
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return svcerrors.Errorf(svcerrors.InvalidArgument, "invalid URL scheme %q: only http and https are allowed", parsed.Scheme)
	}

	// Ensure host is not empty
	host := parsed.Hostname()
	if host == "" {
		return svcerrors.Error(svcerrors.InvalidArgument, "URL must have a non-empty host")
	}

	// Skip network-level SSRF checks when private networks are allowed
	// (self-hosted deployments, integration tests with httptest.NewServer)
	if allowPrivateNetworks {
		return nil
	}

	// Check for IP addresses targeting internal networks
	ip := net.ParseIP(host)
	if ip != nil {
		if err := client.ValidateIP(ip); err != nil {
			return svcerrors.Wrapf(err, svcerrors.InvalidArgument, "%s", err.Error())
		}
	} else {
		// It's a hostname — block well-known internal hostnames
		lower := strings.ToLower(host)
		if lower == "localhost" ||
			lower == "metadata.google.internal" ||
			strings.HasSuffix(lower, ".internal") ||
			strings.HasSuffix(lower, ".local") {
			return svcerrors.Errorf(svcerrors.InvalidArgument, "URL host %q is not allowed: internal/reserved hostname", host)
		}

		// Resolve the hostname and validate all resulting IPs.
		// This prevents DNS-based SSRF where a public hostname resolves
		// to a private IP.
		ips, err := net.LookupIP(host)
		if err != nil {
			return svcerrors.Wrap(err, svcerrors.InvalidArgument, fmt.Sprintf("cannot resolve URL host %q", host))
		}
		for _, resolved := range ips {
			if err := client.ValidateIP(resolved); err != nil {
				return svcerrors.Wrap(err, svcerrors.InvalidArgument, fmt.Sprintf("URL host %q resolves to blocked address: %v", host, err))
			}
		}
	}

	return nil
}
