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
