package errors

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/url"
	"os"
	"slices"
	"strings"
	"syscall"
)

// ErrorCategory classifies delivery errors for health metrics and retry decisions.
type ErrorCategory string

const (
	// CategorySuccess indicates no error occurred.
	CategorySuccess ErrorCategory = "success"

	// CategoryClientError indicates a 4xx HTTP response.
	// These are permanent failures that should never be retried
	// (bad request, unauthorized, forbidden, not found, etc.).
	CategoryClientError ErrorCategory = "client_error"

	// CategoryServerError indicates a 5xx HTTP response.
	// These are temporary server-side failures that should be retried.
	CategoryServerError ErrorCategory = "server_error"

	// CategoryTimeout indicates a request or connection timeout.
	// Retryable - the server may recover.
	CategoryTimeout ErrorCategory = "timeout"

	// CategoryDNSError indicates the domain could not be resolved.
	// Typically persistent - suggests misconfigured webhook URL.
	CategoryDNSError ErrorCategory = "dns_error"

	// CategoryTLSError indicates a TLS/SSL handshake failure.
	// Usually persistent - certificate issues, protocol mismatch, etc.
	CategoryTLSError ErrorCategory = "tls_error"

	// CategoryConnectionRefused indicates the target actively refused the connection.
	// May be temporary (service restarting) or permanent (wrong port).
	CategoryConnectionRefused ErrorCategory = "connection_refused"

	// CategoryNetworkError covers other network-level errors:
	// connection reset, host unreachable, broken pipe, etc.
	// Typically retryable.
	CategoryNetworkError ErrorCategory = "network_error"

	// CategoryUnknown is used when the error cannot be classified.
	CategoryUnknown ErrorCategory = "unknown"
)

// IsRetryableCategory returns whether a given error category is worth retrying.
// Client errors (4xx) are never retried. DNS and TLS errors are generally
// not retried since they indicate configuration problems.
func IsRetryableCategory(cat ErrorCategory) bool {
	switch cat {
	case CategoryServerError, CategoryTimeout, CategoryConnectionRefused, CategoryNetworkError:
		return true
	case CategoryClientError, CategoryDNSError, CategoryTLSError:
		return false
	default:
		return false
	}
}

// ClassifyHTTPStatus categorizes an HTTP response status code.
func ClassifyHTTPStatus(statusCode int) ErrorCategory {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return CategorySuccess
	case statusCode >= 400 && statusCode < 500:
		return CategoryClientError
	case statusCode >= 500:
		return CategoryServerError
	default:
		// 1xx, 3xx treated as unknown (shouldn't normally reach here)
		return CategoryUnknown
	}
}

// ClassifyError inspects a Go error from an HTTP request and returns the
// appropriate error category. It unwraps url.Error, net.OpError, and other
// standard library error types to determine the root cause.
func ClassifyError(err error) ErrorCategory {
	if err == nil {
		return CategorySuccess
	}

	// Unwrap *url.Error (returned by http.Client.Do)
	if urlErr, ok := err.(*url.Error); ok {
		if urlErr.Timeout() {
			return CategoryTimeout
		}
		err = urlErr.Err
	}

	// Check for timeout at any level
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return CategoryTimeout
	}

	// Check for TLS errors
	if isTLSError(err) {
		return CategoryTLSError
	}

	// Check for DNS errors
	if isDNSError(err) {
		return CategoryDNSError
	}

	// Unwrap *net.OpError
	if opErr, ok := err.(*net.OpError); ok {
		err = opErr.Err
	}

	// Check for syscall-level errors
	if sysErr, ok := err.(*os.SyscallError); ok {
		return classifySyscallError(sysErr.Err)
	}

	// Direct syscall.Errno check
	if errno, ok := err.(syscall.Errno); ok {
		return classifySyscallError(errno)
	}

	// Check error message as last resort
	return classifyByMessage(err.Error())
}

// isDNSError checks whether the error is DNS-related.
func isDNSError(err error) bool {
	if dnsErr, ok := err.(*net.DNSError); ok {
		_ = dnsErr
		return true
	}
	// Also check nested in *net.OpError
	if opErr, ok := err.(*net.OpError); ok {
		if _, ok := opErr.Err.(*net.DNSError); ok {
			return true
		}
	}
	return false
}

// isTLSError checks whether the error is TLS/SSL related.
func isTLSError(err error) bool {
	switch err.(type) {
	case tls.RecordHeaderError:
		return true
	case x509.CertificateInvalidError:
		return true
	case x509.HostnameError:
		return true
	case x509.UnknownAuthorityError:
		return true
	case x509.SystemRootsError:
		return true
	case *tls.CertificateVerificationError:
		return true
	}

	// Check wrapped errors
	if opErr, ok := err.(*net.OpError); ok {
		return isTLSError(opErr.Err)
	}

	// Fallback: check error string for common TLS patterns
	msg := err.Error()
	tlsPatterns := []string{
		"tls:", "certificate", "x509:", "handshake",
		"ssl", "CERTIFICATE_VERIFY_FAILED",
	}
	lower := strings.ToLower(msg)
	if slices.ContainsFunc(tlsPatterns, func(p string) bool {
		return strings.Contains(lower, strings.ToLower(p))
	}) {
		return true
	}

	return false
}

// classifySyscallError maps syscall errors to error categories.
func classifySyscallError(err error) ErrorCategory {
	if errno, ok := err.(syscall.Errno); ok {
		switch errno {
		case syscall.ECONNREFUSED:
			return CategoryConnectionRefused
		case syscall.ECONNRESET, syscall.ECONNABORTED, syscall.EPIPE:
			return CategoryNetworkError
		case syscall.EHOSTUNREACH, syscall.ENETUNREACH, syscall.ENETDOWN:
			return CategoryNetworkError
		case syscall.ETIMEDOUT:
			return CategoryTimeout
		}
	}
	return CategoryNetworkError
}

// classifyByMessage is a last-resort classifier that inspects the error string.
func classifyByMessage(msg string) ErrorCategory {
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "timeout"):
		return CategoryTimeout
	case strings.Contains(lower, "connection refused"):
		return CategoryConnectionRefused
	case strings.Contains(lower, "no such host"), strings.Contains(lower, "dns"):
		return CategoryDNSError
	case strings.Contains(lower, "tls"), strings.Contains(lower, "certificate"), strings.Contains(lower, "x509"):
		return CategoryTLSError
	case strings.Contains(lower, "connection reset"),
		strings.Contains(lower, "broken pipe"),
		strings.Contains(lower, "host is unreachable"),
		strings.Contains(lower, "network is unreachable"):
		return CategoryNetworkError
	default:
		return CategoryUnknown
	}
}
