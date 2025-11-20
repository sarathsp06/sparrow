package client

import "time"

// Config holds configuration for the webhook client
type Config struct {
	Timeout            time.Duration
	MaxIdleConns       int
	MaxConnsPerHost    int
	IdleConnTimeout    time.Duration
	DisableKeepAlives  bool
	InsecureSkipVerify bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:            30 * time.Second,
		MaxIdleConns:       100,
		MaxConnsPerHost:    10,
		IdleConnTimeout:    90 * time.Second,
		DisableKeepAlives:  false,
		InsecureSkipVerify: false,
	}
}
