package client

import (
	"net"
	"time"
)

// Config holds configuration for the webhook client
type Config struct {
	Timeout         time.Duration
	MaxIdleConns    int
	MaxConnsPerHost int
	IdleConnTimeout time.Duration

	// AllowPrivateNetworks disables SSRF protection, permitting webhooks
	// to target loopback and private-network addresses. Useful for
	// self-hosted deployments where webhook targets live on the same
	// network, and required for tests that use httptest.NewServer.
	AllowPrivateNetworks bool

	// AllowedNetworks are CIDRs deliveries may reach in addition to public
	// addresses (e.g. internal services on a VPN), without opening loopback,
	// cloud metadata, or the rest of the private address space.
	AllowedNetworks []*net.IPNet

	// MaxCapturedResponseBytes caps how much of a response body is stored
	// for webhooks with capture_response_body enabled (others store 1 KB).
	// 0 means DefaultMaxCapturedResponseBytes.
	MaxCapturedResponseBytes int64
}

// DefaultMaxCapturedResponseBytes is the full-capture storage limit (1 MiB).
const DefaultMaxCapturedResponseBytes = 1 << 20

// CapturedResponseLimit returns the effective full-capture storage limit.
func (c *Config) CapturedResponseLimit() int64 {
	if c == nil || c.MaxCapturedResponseBytes <= 0 {
		return DefaultMaxCapturedResponseBytes
	}
	return c.MaxCapturedResponseBytes
}

// Policy returns the delivery network policy for this configuration.
func (c *Config) Policy() NetworkPolicy {
	return NetworkPolicy{AllowPrivate: c.AllowPrivateNetworks, AllowedNetworks: c.AllowedNetworks}
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:         30 * time.Second,
		MaxIdleConns:    100,
		MaxConnsPerHost: 10,
		IdleConnTimeout: 90 * time.Second,
	}
}
