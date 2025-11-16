// Package client provides an optimized HTTP client for webhook deliveries
// with memory pooling, DNS caching, and connection reuse optimizations
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WebhookClient provides optimized HTTP client for webhook deliveries
type WebhookClient struct {
	httpClient   *http.Client
	bufferPool   *sync.Pool
	responsePool *sync.Pool
	metrics      *Metrics
	mu           sync.RWMutex
	dnsCache     map[string][]net.IP
	dnsTTL       time.Duration
	dnsExpiry    map[string]time.Time
}

// WebhookRequest represents a webhook request
type WebhookRequest struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Payload any               `json:"payload"`
	Timeout time.Duration     `json:"timeout"`
}

// WebhookResponse represents a webhook response
type WebhookResponse struct {
	StatusCode int           `json:"status_code"`
	Status     string        `json:"status"`
	Body       string        `json:"body"`
	Duration   time.Duration `json:"duration"`
	Error      error         `json:"error,omitempty"`
}

// Config holds configuration for the webhook client
type Config struct {
	// Connection pooling
	MaxIdleConns        int
	MaxConnsPerHost     int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration

	// TLS configuration
	TLSHandshakeTimeout time.Duration
	InsecureSkipVerify  bool
	TLSSessionCacheSize int           // Size of TLS session cache (0 = disabled)
	TLSSessionTimeout   time.Duration // How long to keep TLS sessions

	// DNS caching
	DNSCacheTTL time.Duration

	// Timeouts
	DialTimeout           time.Duration
	KeepAlive             time.Duration
	ExpectContinueTimeout time.Duration
	ResponseHeaderTimeout time.Duration

	// Buffer pool settings
	InitialBufferSize int
	MaxBufferSize     int
}

// DefaultConfig returns a default configuration optimized for webhook deliveries
func DefaultConfig() *Config {
	return &Config{
		// Connection pooling - optimized for diverse webhook endpoints
		// Since webhooks go to many different URLs (1000+), connection reuse is unlikely:
		// - Low total idle connections since each delivery is typically to a unique URL
		// - Minimal per-host pooling since we rarely hit the same endpoint twice
		// - Very short idle timeout since connections won't be reused
		MaxIdleConns:        50,               // Low - connections rarely reused across different URLs
		MaxConnsPerHost:     2,                // Minimal since we hit many different hosts
		MaxIdleConnsPerHost: 1,                // No pooling - each webhook URL is typically unique
		IdleConnTimeout:     15 * time.Second, // Very short - connections won't be reused

		// TLS settings - optimized for quick handshakes
		TLSHandshakeTimeout: 15 * time.Second, // Longer to handle slower webhook endpoints
		InsecureSkipVerify:  false,
		TLSSessionCacheSize: 100,             // Cache 100 TLS sessions for potential reuse
		TLSSessionTimeout:   5 * time.Minute, // Keep sessions for 5 minutes

		// DNS caching - valuable for webhook services that might retry
		DNSCacheTTL: 2 * time.Minute, // Shorter TTL for dynamic webhook endpoints

		// Timeouts - balanced for webhook reliability vs throughput
		DialTimeout:           8 * time.Second,  // Longer for potentially slower webhook servers
		KeepAlive:             15 * time.Second, // Shorter since connections rarely reused
		ExpectContinueTimeout: 2 * time.Second,  // Longer for webhook servers that might be slower
		ResponseHeaderTimeout: 15 * time.Second, // Longer to handle slow webhook responses

		// Buffer pool - optimized for typical webhook payload sizes
		InitialBufferSize: 2048,      // 2KB - larger for typical webhook payloads
		MaxBufferSize:     1024 * 32, // 32KB - smaller max since webhooks are usually small
	}
}

// HighThroughputConfig returns a configuration optimized for maximum webhook delivery throughput
// Use this when you care more about delivery speed than connection reuse
func HighThroughputConfig() *Config {
	return &Config{
		// Aggressive settings for high-volume webhook delivery to unique URLs
		MaxIdleConns:        20,              // Very low - no reuse expected for unique URLs
		MaxConnsPerHost:     1,               // One connection per host - treat each delivery independently
		MaxIdleConnsPerHost: 0,               // Zero pooling - every delivery is to a unique URL
		IdleConnTimeout:     5 * time.Second, // Extremely short - connections are single-use

		// Fast TLS settings
		TLSHandshakeTimeout: 10 * time.Second,
		InsecureSkipVerify:  false,
		TLSSessionCacheSize: 200,             // Larger cache for high throughput
		TLSSessionTimeout:   3 * time.Minute, // Shorter timeout for faster turnover

		// Minimal DNS caching for dynamic environments
		DNSCacheTTL: 30 * time.Second, // Very short for dynamic webhook endpoints

		// Aggressive timeouts for fast failure
		DialTimeout:           5 * time.Second,  // Fail fast on connection issues
		KeepAlive:             5 * time.Second,  // Minimal keep-alive
		ExpectContinueTimeout: 1 * time.Second,  // Fast 100-continue handling
		ResponseHeaderTimeout: 10 * time.Second, // Fail fast on slow responses

		// Optimized buffer sizes for speed
		InitialBufferSize: 1024,      // 1KB - minimize allocation overhead
		MaxBufferSize:     1024 * 16, // 16KB - smaller for speed
	}
}

// NoPoolingConfig returns a configuration that disables connection pooling entirely
// Use this when webhook URLs are guaranteed to be unique (e.g., using UUID endpoints)
// This minimizes memory usage and eliminates connection pool overhead
func NoPoolingConfig() *Config {
	return &Config{
		// No connection pooling - every request gets a fresh connection
		MaxIdleConns:        1,               // Minimal - just enough to prevent errors
		MaxConnsPerHost:     1,               // One connection per request
		MaxIdleConnsPerHost: 0,               // No idle connections
		IdleConnTimeout:     1 * time.Second, // Immediate cleanup

		// Fast TLS settings for fresh connections
		TLSHandshakeTimeout: 8 * time.Second,
		InsecureSkipVerify:  false,
		TLSSessionCacheSize: 10,              // Small cache even for unique URLs (CDN/load balancer reuse)
		TLSSessionTimeout:   2 * time.Minute, // Short timeout

		// Minimal DNS caching since URLs are unique
		DNSCacheTTL: 1 * time.Minute, // Short since hosts might be unique too

		// Fast timeouts for single-use connections
		DialTimeout:           3 * time.Second, // Fast failure for bad endpoints
		KeepAlive:             0,               // Disable keep-alive entirely
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,

		// Minimal buffer sizes for unique deliveries
		InitialBufferSize: 512,      // 512B - minimal allocation
		MaxBufferSize:     1024 * 8, // 8KB - small max size
	}
}

// NewWebhookClient creates a new optimized webhook client
func NewWebhookClient(config *Config) *WebhookClient {
	if config == nil {
		config = DefaultConfig()
	}

	// Custom dialer with DNS caching
	client := &WebhookClient{
		dnsCache:  make(map[string][]net.IP),
		dnsExpiry: make(map[string]time.Time),
		dnsTTL:    config.DNSCacheTTL,
		metrics:   NewMetrics(),
	}

	// Create custom dialer
	dialer := &net.Dialer{
		Timeout:   config.DialTimeout,
		KeepAlive: config.KeepAlive,
		Resolver: &net.Resolver{
			PreferGo: true,
		},
	}

	// Wrap dialer with DNS caching
	cachedDialer := client.createCachedDialer(dialer)

	// Create transport with webhook-optimized settings
	transport := &http.Transport{
		DialContext:           cachedDialer,
		MaxIdleConns:          config.MaxIdleConns,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,

		// Webhook-specific optimizations:
		// Disable keep-alives if we're not pooling connections (unique URLs)
		DisableKeepAlives:  config.MaxIdleConns <= 1 || config.KeepAlive == 0,
		DisableCompression: true,  // Disable compression for webhooks (usually small payloads)
		ForceAttemptHTTP2:  false, // Disable HTTP/2 - many webhook endpoints don't support it well

		// Optimize for many different hosts
		WriteBufferSize: 4096, // Smaller write buffer for many connections
		ReadBufferSize:  4096, // Smaller read buffer for many connections
	}

	// Configure TLS with session caching
	var tlsConfig *tls.Config
	if config.InsecureSkipVerify {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	} else {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	// Enable TLS session caching if configured
	if config.TLSSessionCacheSize > 0 {
		// Create LRU cache for TLS sessions
		tlsConfig.ClientSessionCache = tls.NewLRUClientSessionCache(config.TLSSessionCacheSize)

		// Note: TLS session timeout is handled by the TLS library itself
		// The cache size controls how many sessions we keep in memory
	}

	transport.TLSClientConfig = tlsConfig

	transport.TLSClientConfig = tlsConfig

	// Create HTTP client with OpenTelemetry instrumentation
	client.httpClient = &http.Client{
		Transport: otelhttp.NewTransport(transport),
		// No default timeout - will be set per request
	}

	// Initialize buffer pools
	client.bufferPool = &sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, config.InitialBufferSize))
		},
	}

	client.responsePool = &sync.Pool{
		New: func() any {
			return &WebhookResponse{}
		},
	}

	return client
}

// createCachedDialer creates a new dialer with DNS caching
// It checks the DNS cache for the given host and tries each cached IP
// If no cached IPs are found, it resolves the host and caches the result
// The function returns a func that takes a context, network, and address and returns a connection and error
func (wc *WebhookClient) createCachedDialer(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		// Check DNS cache
		if ips := wc.getCachedIPs(host); len(ips) > 0 {
			wc.metrics.RecordCacheHit()
			// Try each cached IP
			for _, ip := range ips {
				addr := net.JoinHostPort(ip.String(), port)
				conn, err := dialer.DialContext(ctx, network, addr)
				if err == nil {
					wc.metrics.RecordConnectionReused()
					return conn, nil
				}
			}
		}

		// Cache miss or all cached IPs failed - resolve and cache
		wc.metrics.RecordCacheMiss()
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}

		// Cache the IPs
		wc.cacheIPs(host, ips)

		// Try connecting to the resolved IPs
		for _, ip := range ips {
			addr := net.JoinHostPort(ip.IP.String(), port)
			conn, err := dialer.DialContext(ctx, network, addr)
			if err == nil {
				wc.metrics.RecordConnectionCreated()
				return conn, nil
			}
		}

		return nil, fmt.Errorf("failed to connect to any resolved IP for %s", host)
	}
}

// getCachedIPs retrieves cached IPs for a host
func (wc *WebhookClient) getCachedIPs(host string) []net.IP {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	// Check if cache entry exists and is not expired
	if expiry, exists := wc.dnsExpiry[host]; exists {
		if time.Now().Before(expiry) {
			return wc.dnsCache[host]
		}
		// Cache expired - clean up
		delete(wc.dnsCache, host)
		delete(wc.dnsExpiry, host)
	}

	return nil
}

// cacheIPs caches the resolved IPs for a host with the given TTL
// It updates the DNS cache and expiry map
func (wc *WebhookClient) cacheIPs(host string, ipAddrs []net.IPAddr) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	ips := make([]net.IP, len(ipAddrs))
	for i, ipAddr := range ipAddrs {
		ips[i] = ipAddr.IP
	}

	wc.dnsCache[host] = ips
	wc.dnsExpiry[host] = time.Now().Add(wc.dnsTTL)
}

// SendWebhook sends a webhook request with optimizations
func (wc *WebhookClient) SendWebhook(ctx context.Context, request *WebhookRequest) *WebhookResponse {
	// Record metrics
	wc.metrics.RecordRequest()

	// Get response object from pool
	response := wc.responsePool.Get().(*WebhookResponse)
	defer wc.responsePool.Put(response)

	// Reset response object
	*response = WebhookResponse{}

	// Get buffer from pool for payload
	buffer := wc.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buffer.Reset()
		wc.bufferPool.Put(buffer)
	}()

	// Marshal payload to buffer
	if err := json.NewEncoder(buffer).Encode(request.Payload); err != nil {
		response.Error = fmt.Errorf("failed to marshal payload: %w", err)
		wc.metrics.RecordFailure(0)
		return response
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, request.URL, buffer)
	if err != nil {
		response.Error = fmt.Errorf("failed to create request: %w", err)
		wc.metrics.RecordFailure(0)
		return response
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Sparrow-Webhook-Client/1.0")

	// Add custom headers
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	// Create client with timeout for this request
	client := wc.httpClient
	if request.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, request.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// Send request and measure time
	startTime := time.Now()
	resp, err := client.Do(req)
	response.Duration = time.Since(startTime)

	if err != nil {
		response.Error = fmt.Errorf("failed to send request: %w", err)
		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			wc.metrics.RecordTimeout()
		} else {
			wc.metrics.RecordFailure(response.Duration)
		}
		return response
	}
	defer resp.Body.Close()

	// Set response metadata
	response.StatusCode = resp.StatusCode
	response.Status = resp.Status

	// For webhook deliveries, we primarily care about the status code
	// Read response body with minimal size limit and timeout to avoid blocking
	// Most webhook endpoints return small responses anyway
	bodyBytes := make([]byte, 512) // Only read first 512 bytes
	n, err := io.ReadFull(resp.Body, bodyBytes)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		// If we can't read the body quickly, just note the status
		response.Body = fmt.Sprintf("Status: %s (body read error)", resp.Status)
	} else {
		response.Body = string(bodyBytes[:n])
	}

	// Record success/failure metrics based on status code
	// For webhooks, 2xx and 3xx are generally considered success
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		wc.metrics.RecordSuccess(response.Duration)
	} else {
		wc.metrics.RecordFailure(response.Duration)
	}

	return response
}

// SendWebhookFireAndForget sends a webhook request optimized for maximum throughput
// This method only waits for the HTTP status code and immediately returns, making it
// ideal for high-volume webhook delivery where you care about delivery confirmation
// but not the response content
func (wc *WebhookClient) SendWebhookFireAndForget(ctx context.Context, request *WebhookRequest) *WebhookResponse {
	// Record metrics
	wc.metrics.RecordRequest()

	// Get response object from pool
	response := wc.responsePool.Get().(*WebhookResponse)
	defer wc.responsePool.Put(response)

	// Reset response object
	*response = WebhookResponse{}

	// Get buffer from pool for payload
	buffer := wc.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buffer.Reset()
		wc.bufferPool.Put(buffer)
	}()

	// Marshal payload to buffer
	if err := json.NewEncoder(buffer).Encode(request.Payload); err != nil {
		response.Error = fmt.Errorf("failed to marshal payload: %w", err)
		wc.metrics.RecordFailure(0)
		return response
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, request.URL, buffer)
	if err != nil {
		response.Error = fmt.Errorf("failed to create request: %w", err)
		wc.metrics.RecordFailure(0)
		return response
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Sparrow-Webhook-Client/1.0")
	req.Header.Set("Connection", "close") // Force connection close for fire-and-forget

	// Add custom headers
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	// Create client with timeout for this request
	client := wc.httpClient
	if request.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, request.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// Send request and measure time
	startTime := time.Now()
	resp, err := client.Do(req)
	response.Duration = time.Since(startTime)

	if err != nil {
		response.Error = fmt.Errorf("failed to send request: %w", err)
		if ctx.Err() == context.DeadlineExceeded {
			wc.metrics.RecordTimeout()
		} else {
			wc.metrics.RecordFailure(response.Duration)
		}
		return response
	}

	// For fire-and-forget, we only care about the status code
	response.StatusCode = resp.StatusCode
	response.Status = resp.Status
	response.Body = "fire-and-forget: not read"
	// Close the response body immediately without reading
	resp.Body.Close()

	// Record success/failure metrics based on status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		wc.metrics.RecordSuccess(response.Duration)
	} else {
		wc.metrics.RecordFailure(response.Duration)
	}

	return response
}

// Close cleanly shuts down the client
func (wc *WebhookClient) Close() error {
	// For proper connection cleanup, we'll need to recreate the client
	// This is a limitation with wrapped transports

	// Clear DNS cache
	wc.mu.Lock()
	wc.dnsCache = make(map[string][]net.IP)
	wc.dnsExpiry = make(map[string]time.Time)
	wc.mu.Unlock()

	return nil
}

// GetStats returns client statistics
func (wc *WebhookClient) GetStats() map[string]any {
	stats := wc.metrics.GetStats()

	wc.mu.RLock()
	stats["dns_cache_entries"] = len(wc.dnsCache)
	wc.mu.RUnlock()

	return stats
}

// ClearDNSCache clears the DNS cache
func (wc *WebhookClient) ClearDNSCache() {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	wc.dnsCache = make(map[string][]net.IP)
	wc.dnsExpiry = make(map[string]time.Time)
}

// PrewarmConnections pre-establishes connections to frequently used hosts
func (wc *WebhookClient) PrewarmConnections(ctx context.Context, hosts []string) error {
	for _, host := range hosts {
		// Create a dummy request to establish connection
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, fmt.Sprintf("https://%s", host), nil)
		if err != nil {
			continue
		}

		// Send request but ignore response (connection will be pooled)
		resp, err := wc.httpClient.Do(req)
		if err == nil && resp != nil {
			resp.Body.Close()
		}
	}

	return nil
}
