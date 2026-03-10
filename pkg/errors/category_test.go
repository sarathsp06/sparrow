package errors

import (
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected ErrorCategory
	}{
		{"200 OK", 200, CategorySuccess},
		{"201 Created", 201, CategorySuccess},
		{"204 No Content", 204, CategorySuccess},
		{"299 edge", 299, CategorySuccess},
		{"400 Bad Request", 400, CategoryClientError},
		{"401 Unauthorized", 401, CategoryClientError},
		{"403 Forbidden", 403, CategoryClientError},
		{"404 Not Found", 404, CategoryClientError},
		{"410 Gone", 410, CategoryClientError},
		{"422 Unprocessable", 422, CategoryClientError},
		{"429 Too Many Requests", 429, CategoryClientError},
		{"499 edge", 499, CategoryClientError},
		{"500 Internal Server Error", 500, CategoryServerError},
		{"502 Bad Gateway", 502, CategoryServerError},
		{"503 Service Unavailable", 503, CategoryServerError},
		{"504 Gateway Timeout", 504, CategoryServerError},
		{"599 edge", 599, CategoryServerError},
		{"100 Continue", 100, CategoryUnknown},
		{"301 Redirect", 301, CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyHTTPStatus(tt.status)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestClassifyError_Nil(t *testing.T) {
	assert.Equal(t, CategorySuccess, ClassifyError(nil))
}

func TestClassifyError_Timeout(t *testing.T) {
	// url.Error with Timeout
	urlErr := &url.Error{
		Op:  "Get",
		URL: "https://example.com",
		Err: &timeoutError{},
	}
	assert.Equal(t, CategoryTimeout, ClassifyError(urlErr))

	// net.Error with Timeout (not wrapped in url.Error)
	assert.Equal(t, CategoryTimeout, ClassifyError(&timeoutError{}))
}

func TestClassifyError_DNS(t *testing.T) {
	dnsErr := &net.DNSError{
		Err:  "no such host",
		Name: "bad.example.com",
	}
	assert.Equal(t, CategoryDNSError, ClassifyError(dnsErr))

	// DNS error wrapped in url.Error
	urlErr := &url.Error{
		Op:  "Get",
		URL: "https://bad.example.com",
		Err: dnsErr,
	}
	assert.Equal(t, CategoryDNSError, ClassifyError(urlErr))

	// DNS error wrapped in net.OpError
	opErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: dnsErr,
	}
	assert.Equal(t, CategoryDNSError, ClassifyError(opErr))
}

func TestClassifyError_TLS(t *testing.T) {
	// x509 certificate errors
	certErr := x509.CertificateInvalidError{
		Reason: x509.Expired,
	}
	assert.Equal(t, CategoryTLSError, ClassifyError(certErr))

	hostnameErr := x509.HostnameError{
		Host: "example.com",
	}
	assert.Equal(t, CategoryTLSError, ClassifyError(hostnameErr))

	unknownAuthErr := x509.UnknownAuthorityError{}
	assert.Equal(t, CategoryTLSError, ClassifyError(unknownAuthErr))

	// TLS error by message fallback
	tlsMessageErr := fmt.Errorf("tls: handshake failure")
	assert.Equal(t, CategoryTLSError, ClassifyError(tlsMessageErr))
}

func TestClassifyError_ConnectionRefused(t *testing.T) {
	// syscall.ECONNREFUSED via os.SyscallError
	sysErr := &os.SyscallError{
		Syscall: "connect",
		Err:     syscall.ECONNREFUSED,
	}
	opErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: sysErr,
	}
	assert.Equal(t, CategoryConnectionRefused, ClassifyError(opErr))

	// connection refused by message
	msgErr := fmt.Errorf("connection refused")
	assert.Equal(t, CategoryConnectionRefused, ClassifyError(msgErr))
}

func TestClassifyError_NetworkError(t *testing.T) {
	// ECONNRESET
	sysErr := &os.SyscallError{
		Syscall: "read",
		Err:     syscall.ECONNRESET,
	}
	opErr := &net.OpError{
		Op:  "read",
		Net: "tcp",
		Err: sysErr,
	}
	assert.Equal(t, CategoryNetworkError, ClassifyError(opErr))

	// EPIPE (broken pipe)
	sysErrPipe := &os.SyscallError{
		Syscall: "write",
		Err:     syscall.EPIPE,
	}
	opErrPipe := &net.OpError{
		Op:  "write",
		Net: "tcp",
		Err: sysErrPipe,
	}
	assert.Equal(t, CategoryNetworkError, ClassifyError(opErrPipe))

	// Host unreachable
	sysErrUnreach := &os.SyscallError{
		Syscall: "connect",
		Err:     syscall.EHOSTUNREACH,
	}
	opErrUnreach := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: sysErrUnreach,
	}
	assert.Equal(t, CategoryNetworkError, ClassifyError(opErrUnreach))

	// By message
	assert.Equal(t, CategoryNetworkError, ClassifyError(fmt.Errorf("connection reset by peer")))
	assert.Equal(t, CategoryNetworkError, ClassifyError(fmt.Errorf("broken pipe")))
}

func TestClassifyError_Unknown(t *testing.T) {
	err := fmt.Errorf("some weird error we don't recognize")
	assert.Equal(t, CategoryUnknown, ClassifyError(err))
}

func TestClassifyError_UrlErrorUnwrap(t *testing.T) {
	// Non-timeout url.Error should unwrap and continue classifying
	inner := &net.DNSError{
		Err:  "no such host",
		Name: "example.com",
	}
	urlErr := &url.Error{
		Op:  "Get",
		URL: "https://example.com",
		Err: inner,
	}
	assert.Equal(t, CategoryDNSError, ClassifyError(urlErr))
}

func TestClassifyError_SyscallErrno_Direct(t *testing.T) {
	// Direct syscall.Errno (not wrapped in os.SyscallError)
	opErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: syscall.ECONNREFUSED,
	}
	assert.Equal(t, CategoryConnectionRefused, ClassifyError(opErr))
}

func TestClassifyError_SyscallTimeout(t *testing.T) {
	sysErr := &os.SyscallError{
		Syscall: "connect",
		Err:     syscall.ETIMEDOUT,
	}
	opErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: sysErr,
	}
	assert.Equal(t, CategoryTimeout, ClassifyError(opErr))
}

func TestIsRetryableCategory(t *testing.T) {
	tests := []struct {
		category  ErrorCategory
		retryable bool
	}{
		{CategorySuccess, false},
		{CategoryClientError, false},
		{CategoryServerError, true},
		{CategoryTimeout, true},
		{CategoryDNSError, false},
		{CategoryTLSError, false},
		{CategoryConnectionRefused, true},
		{CategoryNetworkError, true},
		{CategoryUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			assert.Equal(t, tt.retryable, IsRetryableCategory(tt.category))
		})
	}
}

func TestClassifyByMessage(t *testing.T) {
	tests := []struct {
		message  string
		expected ErrorCategory
	}{
		{"i/o timeout", CategoryTimeout},
		{"connection refused", CategoryConnectionRefused},
		{"no such host found", CategoryDNSError},
		{"dns lookup failed", CategoryDNSError},
		{"tls: protocol version not supported", CategoryTLSError},
		{"x509: certificate has expired", CategoryTLSError},
		{"certificate signed by unknown authority", CategoryTLSError},
		{"connection reset by peer", CategoryNetworkError},
		{"broken pipe", CategoryNetworkError},
		{"host is unreachable", CategoryNetworkError},
		{"network is unreachable", CategoryNetworkError},
		{"something completely unknown", CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			got := classifyByMessage(tt.message)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// timeoutError is a test helper that implements net.Error with Timeout() = true
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
