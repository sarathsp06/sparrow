package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// DeliveryClient handles webhook delivery with configurable HTTP settings
type DeliveryClient struct {
	httpClient *http.Client
	config     *ClientConfig
}

// ClientConfig contains configuration for the delivery client
type ClientConfig struct {
	DefaultTimeout    time.Duration
	MaxIdleConns      int
	MaxConnsPerHost   int
	IdleConnTimeout   time.Duration
	DisableKeepAlives bool
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		DefaultTimeout:    30 * time.Second,
		MaxIdleConns:      50,
		MaxConnsPerHost:   2,
		IdleConnTimeout:   15 * time.Second,
		DisableKeepAlives: false,
	}
}

// NewDeliveryClient creates a new webhook delivery client
func NewDeliveryClient(config *ClientConfig) *DeliveryClient {
	if config == nil {
		config = DefaultClientConfig()
	}

	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxConnsPerHost:     config.MaxConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		DisableKeepAlives:   config.DisableKeepAlives,
		TLSHandshakeTimeout: 15 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Transport: otelhttp.NewTransport(transport),
		Timeout:   config.DefaultTimeout,
	}

	return &DeliveryClient{
		httpClient: client,
		config:     config,
	}
}

// WebhookDeliveryRequest contains all information needed for webhook delivery
type WebhookDeliveryRequest struct {
	Webhook     *WebhookRegistration `json:"webhook"`
	EventID     string               `json:"event_id"`
	DeliveryID  string               `json:"delivery_id"`
	Payload     interface{}          `json:"payload"`
	AttemptNum  int                  `json:"attempt_num"`
	MaxAttempts int                  `json:"max_attempts"`
}

// WebhookDeliveryResponse contains the result of a webhook delivery attempt
type WebhookDeliveryResponse struct {
	DeliveryID      string        `json:"delivery_id"`
	Success         bool          `json:"success"`
	StatusCode      int           `json:"status_code"`
	ResponseBody    string        `json:"response_body,omitempty"`
	ResponseHeaders http.Header   `json:"response_headers,omitempty"`
	Duration        time.Duration `json:"duration"`
	Error           string        `json:"error,omitempty"`
	ShouldRetry     bool          `json:"should_retry"`
	NextRetryAfter  time.Duration `json:"next_retry_after,omitempty"`
	SignatureValid  *bool         `json:"signature_valid,omitempty"`
}

// DeliverWebhook delivers a webhook according to its configuration
func (dc *DeliveryClient) DeliverWebhook(ctx context.Context, req *WebhookDeliveryRequest) *WebhookDeliveryResponse {
	startTime := time.Now()

	response := &WebhookDeliveryResponse{
		DeliveryID:  req.DeliveryID,
		Success:     false,
		ShouldRetry: false,
	}

	// Validate webhook configuration
	if err := req.Webhook.HTTPConfig.ValidateConfig(); err != nil {
		response.Error = fmt.Sprintf("invalid webhook config: %v", err)
		response.Duration = time.Since(startTime)
		return response
	}

	// Create HTTP client with webhook-specific configuration
	client := dc.createConfiguredClient(req.Webhook.HTTPConfig)

	// Create request context with timeout
	requestTimeout := req.Webhook.HTTPConfig.GetRequestTimeout()
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	// Prepare request body
	var requestBody []byte
	var err error
	if req.Payload != nil {
		requestBody, err = json.Marshal(req.Payload)
		if err != nil {
			response.Error = fmt.Sprintf("failed to marshal payload: %v", err)
			response.Duration = time.Since(startTime)
			return response
		}
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.Webhook.URL, bytes.NewReader(requestBody))
	if err != nil {
		response.Error = fmt.Sprintf("failed to create request: %v", err)
		response.Duration = time.Since(startTime)
		return response
	}

	// Set headers
	dc.setRequestHeaders(httpReq, req, requestBody)

	// Perform request
	resp, err := client.Do(httpReq)
	response.Duration = time.Since(startTime)

	if err != nil {
		response.Error = err.Error()
		response.ShouldRetry = dc.shouldRetryError(err, req.AttemptNum, req.Webhook.HTTPConfig.MaxRetries)
		if response.ShouldRetry {
			response.NextRetryAfter = dc.calculateRetryDelay(req.AttemptNum, req.Webhook.HTTPConfig.GetRetryBackoff())
		}
		return response
	}
	defer resp.Body.Close()

	// Process response
	response.StatusCode = resp.StatusCode
	response.ResponseHeaders = resp.Header
	response.Success = req.Webhook.HTTPConfig.IsSuccessStatusCode(resp.StatusCode)

	// Read response body if configured to do so
	if req.Webhook.HTTPConfig.CaptureResponseBody {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			response.Error = fmt.Sprintf("failed to read response body: %v", err)
		} else {
			response.ResponseBody = string(bodyBytes)
		}
	} else {
		// Just read and discard a small portion for logging
		bodyBytes := make([]byte, 512)
		n, _ := io.ReadFull(resp.Body, bodyBytes)
		if n > 0 {
			response.ResponseBody = string(bodyBytes[:n]) + "..."
		}
	}

	// Determine if we should retry on failure
	if !response.Success {
		response.ShouldRetry = dc.shouldRetryStatusCode(resp.StatusCode, req.AttemptNum, req.Webhook.HTTPConfig.MaxRetries)
		if response.ShouldRetry {
			response.NextRetryAfter = dc.calculateRetryDelay(req.AttemptNum, req.Webhook.HTTPConfig.GetRetryBackoff())
		}
	}

	// Validate webhook signature if secret is configured
	if req.Webhook.HTTPConfig.WebhookSecret != "" && response.Success {
		signatureValid := dc.validateWebhookSignature(req.Webhook.HTTPConfig.WebhookSecret, requestBody, resp.Header)
		response.SignatureValid = &signatureValid
	}

	return response
}

// createConfiguredClient creates an HTTP client with webhook-specific configuration
func (dc *DeliveryClient) createConfiguredClient(config WebhookHTTPConfig) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        dc.config.MaxIdleConns,
		MaxConnsPerHost:     dc.config.MaxConnsPerHost,
		IdleConnTimeout:     dc.config.IdleConnTimeout,
		DisableKeepAlives:   dc.config.DisableKeepAlives,
		TLSHandshakeTimeout: 15 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	// Configure SSL verification
	if !config.VerifySSL {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	// Configure redirect policy
	client := &http.Client{
		Transport: otelhttp.NewTransport(transport),
		Timeout:   config.GetRequestTimeout(),
	}

	if !config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client
}

// setRequestHeaders sets the appropriate headers for the webhook request
func (dc *DeliveryClient) setRequestHeaders(req *http.Request, deliveryReq *WebhookDeliveryRequest, body []byte) {
	webhook := deliveryReq.Webhook

	// Set content type
	req.Header.Set("Content-Type", webhook.HTTPConfig.ContentType)

	// Set user agent
	req.Header.Set("User-Agent", webhook.HTTPConfig.UserAgent)

	// Set custom headers from webhook configuration
	if webhook.Headers != nil {
		for key, value := range webhook.Headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// Set webhook-specific headers
	req.Header.Set("X-Webhook-Event-ID", deliveryReq.EventID)
	req.Header.Set("X-Webhook-Delivery-ID", deliveryReq.DeliveryID)
	req.Header.Set("X-Webhook-Attempt", strconv.Itoa(deliveryReq.AttemptNum))
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))

	// Set webhook signature if secret is configured
	if webhook.HTTPConfig.WebhookSecret != "" {
		signature := dc.generateWebhookSignature(webhook.HTTPConfig.WebhookSecret, body)
		req.Header.Set("X-Webhook-Signature-256", signature)
	}
}

// generateWebhookSignature generates HMAC-SHA256 signature for webhook payload
func (dc *DeliveryClient) generateWebhookSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// GenerateTestSignature exposes signature generation for testing
func (dc *DeliveryClient) GenerateTestSignature(secret string, payload []byte) string {
	return dc.generateWebhookSignature(secret, payload)
}

// validateWebhookSignature validates webhook response signature (if the webhook endpoint returns one)
func (dc *DeliveryClient) validateWebhookSignature(secret string, payload []byte, headers http.Header) bool {
	// This would be used if the webhook endpoint returns a signature for verification
	// Implementation depends on the specific webhook provider's signature scheme
	return true // Placeholder - implement based on your webhook provider's requirements
}

// shouldRetryError determines if an error should trigger a retry
func (dc *DeliveryClient) shouldRetryError(err error, attemptNum, maxRetries int) bool {
	if attemptNum >= maxRetries {
		return false
	}

	// Check for network errors that should be retried
	errorStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"timeout", "connection reset", "connection refused", "temporary failure",
		"no such host", "network is unreachable", "broken pipe",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errorStr, retryable) {
			return true
		}
	}

	return false
}

// shouldRetryStatusCode determines if a status code should trigger a retry
func (dc *DeliveryClient) shouldRetryStatusCode(statusCode, attemptNum, maxRetries int) bool {
	if attemptNum >= maxRetries {
		return false
	}

	// Retry on server errors (5xx) and specific client errors
	return statusCode >= 500 || statusCode == 429 || statusCode == 408
}

// calculateRetryDelay calculates the delay before the next retry attempt
func (dc *DeliveryClient) calculateRetryDelay(attemptNum int, baseDelay time.Duration) time.Duration {
	// Exponential backoff with jitter
	delay := baseDelay
	for i := 1; i < attemptNum; i++ {
		delay *= 2
	}

	// Add jitter (±25%)
	jitter := time.Duration(float64(delay) * 0.25)
	delay += jitter

	// Cap at maximum delay
	maxDelay := 30 * time.Minute
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// ValidateWebhookURL validates that a webhook URL is valid and reachable
func (dc *DeliveryClient) ValidateWebhookURL(ctx context.Context, url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	resp, err := dc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("URL not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("URL returned error status: %d", resp.StatusCode)
	}

	return nil
}
