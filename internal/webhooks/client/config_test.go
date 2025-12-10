package client

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	// Verify default values
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", config.MaxIdleConns)
	}

	if config.MaxConnsPerHost != 10 {
		t.Errorf("Expected MaxConnsPerHost 10, got %d", config.MaxConnsPerHost)
	}

	if config.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected IdleConnTimeout 90s, got %v", config.IdleConnTimeout)
	}

	if config.DisableKeepAlives {
		t.Error("Expected DisableKeepAlives to be false")
	}

	if config.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be false")
	}
}

func TestCustomConfig(t *testing.T) {
	config := &Config{
		Timeout:            10 * time.Second,
		MaxIdleConns:       50,
		MaxConnsPerHost:    5,
		IdleConnTimeout:    60 * time.Second,
		DisableKeepAlives:  true,
		InsecureSkipVerify: true,
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", config.Timeout)
	}

	if config.MaxIdleConns != 50 {
		t.Errorf("Expected MaxIdleConns 50, got %d", config.MaxIdleConns)
	}

	if config.MaxConnsPerHost != 5 {
		t.Errorf("Expected MaxConnsPerHost 5, got %d", config.MaxConnsPerHost)
	}

	if config.IdleConnTimeout != 60*time.Second {
		t.Errorf("Expected IdleConnTimeout 60s, got %v", config.IdleConnTimeout)
	}

	if !config.DisableKeepAlives {
		t.Error("Expected DisableKeepAlives to be true")
	}

	if !config.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}
}
