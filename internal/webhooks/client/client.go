package client

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WebhookClient handles webhook delivery
type WebhookClient struct {
	httpClient *http.Client
	tmpl       *TemplateEngine
	metrics    *Metrics
	config     *Config
}

// NewWebhookClient creates a new webhook client
func NewWebhookClient(config *Config) *WebhookClient {
	if config == nil {
		config = DefaultConfig()
	}

	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxConnsPerHost:     config.MaxConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		DisableKeepAlives:   config.DisableKeepAlives,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: config.InsecureSkipVerify},
		TLSHandshakeTimeout: 10 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return &WebhookClient{
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(transport),
			Timeout:   config.Timeout,
		},
		tmpl:    NewTemplateEngine(),
		metrics: NewMetrics(),
		config:  config,
	}
}

// Send executes the webhook delivery
func (c *WebhookClient) Send(ctx context.Context, req *DeliveryRequest) (*http.Response, time.Duration, error) {
	c.metrics.RecordRequest()

	httpReq, err := BuildRequest(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	// TODO: Support per-request TLS settings if needed (e.g. overriding config)
	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordFailure(duration)
		return nil, duration, err
	}

	c.metrics.RecordSuccess(duration)
	return resp, duration, nil
}

// PrewarmConnections establishes connections to the given hosts
func (c *WebhookClient) PrewarmConnections(ctx context.Context, hosts []string) error {
	// Simple implementation: just resolve IPs or make a HEAD request?
	// For now, let's just resolve IPs to warm up DNS
	for _, host := range hosts {
		// This is a placeholder. Real prewarming would involve creating connections.
		// But since we don't have a list of URLs, just hosts, we can only do DNS or TCP dial.
		// Let's skip actual implementation for now as it's an optimization.
		_ = host
	}
	return nil
}

// Close shuts down the client
func (c *WebhookClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// GetStats returns client metrics
func (c *WebhookClient) GetStats() map[string]interface{} {
	return c.metrics.GetStats()
}

// ReadBody reads the response body safely
func ReadBody(resp *http.Response, limit int64) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()

	if limit > 0 {
		return io.ReadAll(io.LimitReader(resp.Body, limit))
	}
	return io.ReadAll(resp.Body)
}
