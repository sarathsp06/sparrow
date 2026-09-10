package client

import (
	"context"
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
	config     *Config
}

// NewWebhookClient creates a new webhook client
func NewWebhookClient(config *Config) *WebhookClient {
	if config == nil {
		config = DefaultConfig()
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	if !config.AllowPrivateNetworks {
		// SEC-002: Validate resolved IPs at connect time to prevent DNS
		// rebinding attacks. This closes the TOCTOU gap between URL
		// validation at webhook registration and actual delivery.
		dialer.Control = ssrfDialControl
	}

	var checkRedirect func(req *http.Request, via []*http.Request) error
	if !config.AllowPrivateNetworks {
		// SEC-001: Validate redirect targets against SSRF blocklist.
		// Each redirect URL is checked for internal/private IPs and
		// restricted hostnames before following.
		checkRedirect = ssrfSafeCheckRedirect
	}

	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxConnsPerHost:     config.MaxConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
		DialContext:         dialer.DialContext,
	}

	return &WebhookClient{
		httpClient: &http.Client{
			Transport:     otelhttp.NewTransport(transport),
			Timeout:       config.Timeout,
			CheckRedirect: checkRedirect,
		},
		tmpl:   NewTemplateEngine(),
		config: config,
	}
}

// Send executes the webhook delivery
func (c *WebhookClient) Send(ctx context.Context, req *DeliveryRequest) (*http.Response, time.Duration, error) {
	httpReq, err := BuildRequest(ctx, req)

	// Return the pooled header map now that BuildRequest has copied the
	// values into http.Header. Holding the map longer would be a leak
	// since nothing else reads DeliveryRequest.Headers after this point.
	if req.Headers != nil {
		PutHeaderMap(req.Headers)
		req.Headers = nil
	}

	if err != nil {
		return nil, 0, err
	}

	// Apply per-webhook request timeout if set, overriding the global client timeout.
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
		httpReq = httpReq.WithContext(ctx)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		return nil, duration, err
	}

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

// ReadBody reads up to limit bytes of the response body using a pooled buffer,
// then drains a bounded remainder so the keep-alive connection can be reused.
// The caller is responsible for closing resp.Body.
func ReadBody(resp *http.Response, limit int64) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}

	// Use buffer from pool for reading
	buf := GetBuffer()
	defer PutBuffer(buf)

	var err error
	if limit > 0 {
		_, err = buf.ReadFrom(io.LimitReader(resp.Body, limit))
	} else {
		_, err = buf.ReadFrom(resp.Body)
	}

	// Drain a bounded remainder so the transport can reuse the connection.
	// Unbounded drain would let a hostile server stream forever.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

	if err != nil {
		return nil, err
	}

	// Copy to new slice since buffer goes back to pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}
