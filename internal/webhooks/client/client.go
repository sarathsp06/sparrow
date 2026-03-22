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
			// SEC-002: Validate resolved IPs at connect time to prevent DNS
			// rebinding attacks. This closes the TOCTOU gap between URL
			// validation at webhook registration and actual delivery.
			Control: ssrfDialControl,
		}).DialContext,
	}

	return &WebhookClient{
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(transport),
			Timeout:   config.Timeout,
			// SEC-001: Validate redirect targets against SSRF blocklist.
			// Each redirect URL is checked for internal/private IPs and
			// restricted hostnames before following.
			CheckRedirect: ssrfSafeCheckRedirect,
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

// Close shuts down the client
func (c *WebhookClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// TransformPayload transforms the webhook payload using a template
func (c *WebhookClient) TransformPayload(tmplStr string, data WebhookTemplateContext) ([]byte, error) {
	return c.tmpl.TransformPayload(tmplStr, data)
}

// ReadBody reads the response body safely using a pooled buffer
func ReadBody(resp *http.Response, limit int64) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Use buffer from pool for reading
	buf := GetBuffer()
	defer PutBuffer(buf)

	var err error
	if limit > 0 {
		_, err = buf.ReadFrom(io.LimitReader(resp.Body, limit))
	} else {
		_, err = buf.ReadFrom(resp.Body)
	}

	if err != nil {
		return nil, err
	}

	// Copy to new slice since buffer goes back to pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}
