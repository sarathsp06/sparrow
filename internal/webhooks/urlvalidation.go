package webhooks

import (
	"net/url"
	"strings"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// ValidateWebhookURL validates a webhook URL to prevent SSRF attacks.
// It ensures the URL:
//   - Uses http or https scheme only
//   - Has a valid, non-empty host
//   - Does not point to a destination the network policy forbids (loopback,
//     private and link-local addresses by default; cloud metadata always,
//     unless explicitly allowed)
//
// The delivery dialer re-checks every resolved address at connect time, so
// this registration-time check is for early, actionable errors.
//
// All errors returned are *svcerrors.ServiceError with svcerrors.InvalidArgument,
// so they propagate to the client as actionable messages.
func ValidateWebhookURL(rawURL string, policy client.NetworkPolicy) error {
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

	if err := policy.CheckHost(host); err != nil {
		return svcerrors.Wrapf(err, svcerrors.InvalidArgument, "URL %s", err.Error())
	}
	return nil
}

// validateHeaders checks custom delivery headers (see client.ValidateHeaders)
// and reports problems as InvalidArgument.
func validateHeaders(field string, headers map[string]string) error {
	if err := client.ValidateHeaders(field, headers); err != nil {
		return svcerrors.Wrap(err, svcerrors.InvalidArgument, err.Error())
	}
	return nil
}

// stringHeaders keeps the string-valued entries of a JSON header map (the
// only ones delivered).
func stringHeaders(h map[string]any) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}
