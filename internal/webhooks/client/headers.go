package client

import (
	"fmt"
	"net/http"
	"strings"
)

// reservedHeaders are headers a webhook, subscription or secret header may
// not set. They control HTTP framing and connection handling, which Go's
// client owns: setting them either has no effect (Host, Content-Length) or
// corrupts the request, so rejecting them only turns a silent misconfig into
// a clear error. Content-Type, User-Agent and Authorization stay settable.
var reservedHeaders = map[string]string{
	"Host":              "the Host header always matches the webhook URL",
	"Content-Length":    "the length is computed from the payload",
	"Transfer-Encoding": "framing is managed by the HTTP client",
	"Connection":        "connection handling is managed by the HTTP client",
	"Keep-Alive":        "connection handling is managed by the HTTP client",
	"Proxy-Connection":  "connection handling is managed by the HTTP client",
	"Te":                "framing is managed by the HTTP client",
	"Trailer":           "framing is managed by the HTTP client",
	"Upgrade":           "protocol upgrades are not supported for deliveries",
}

// maxHeaderValueBytes bounds a single custom header value.
const maxHeaderValueBytes = 8 << 10

// ValidateHeaders checks custom delivery headers: names must be valid HTTP
// tokens, values must not contain CR, LF or other control characters (header
// injection), and reserved framing headers are refused. field names the
// error (e.g. "headers", "secret_headers").
func ValidateHeaders(field string, headers map[string]string) error {
	for name, value := range headers {
		if !validHeaderName(name) {
			return fmt.Errorf("%s: invalid header name %q", field, name)
		}
		if why, reserved := reservedHeaders[http.CanonicalHeaderKey(name)]; reserved {
			return fmt.Errorf("%s: header %q cannot be set: %s", field, name, why)
		}
		if len(value) > maxHeaderValueBytes {
			return fmt.Errorf("%s: header %q value exceeds %d bytes", field, name, maxHeaderValueBytes)
		}
		if !validHeaderValue(value) {
			return fmt.Errorf("%s: header %q value contains control characters (CR/LF are not allowed)", field, name)
		}
	}
	return nil
}

// isReservedHeader reports whether name is a reserved framing header.
func isReservedHeader(name string) bool {
	_, ok := reservedHeaders[http.CanonicalHeaderKey(name)]
	return ok
}

// validHeaderName reports whether name is an RFC 9110 token.
func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0:
		default:
			return false
		}
	}
	return true
}

// validHeaderValue rejects control characters other than horizontal tab.
func validHeaderValue(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c < 0x20 && c != '\t') || c == 0x7f {
			return false
		}
	}
	return true
}
