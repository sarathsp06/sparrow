package webhooks

import (
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
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

// canonicalHeaders returns h keyed by canonical header names (x-api-key
// becomes X-Api-Key), so a header is the same one however a request spells
// it. Two names that differ only in case are the same header, and refused.
func canonicalHeaders(field string, h map[string]string) (map[string]string, error) {
	out := canonicalNames(h)
	if len(out) == len(h) {
		return out, nil
	}
	names := slices.Sorted(maps.Keys(h))
	seen := make(map[string]string, len(names))
	for _, name := range names {
		key := http.CanonicalHeaderKey(name)
		if prev, ok := seen[key]; ok {
			return nil, svcerrors.Errorf(svcerrors.InvalidArgument,
				"%s: %q and %q are the same header (header names are case-insensitive); send it once", field, prev, name)
		}
		seen[key] = name
	}
	return out, nil
}

// canonicalNames returns h keyed by canonical header names. nil stays nil.
func canonicalNames(h map[string]string) map[string]string {
	if h == nil {
		return nil
	}
	out := make(map[string]string, len(h))
	for name, value := range h {
		out[http.CanonicalHeaderKey(name)] = value
	}
	return out
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
