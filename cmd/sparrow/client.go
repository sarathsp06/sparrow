package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// apiClient is a minimal JSON client for the Sparrow REST API.
type apiClient struct {
	baseURL string
	apiKey  string
	hc      *http.Client
}

func newAPIClient(cfg config) *apiClient {
	return &apiClient{
		baseURL: strings.TrimRight(cfg.ServerURL, "/"),
		apiKey:  cfg.APIKey,
		hc:      &http.Client{Timeout: 30 * time.Second},
	}
}

// apiError is a non-2xx response.
type apiError struct {
	Status int
	Detail string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("server returned %d: %s", e.Status, e.Detail)
}

// do sends a JSON request and decodes a 2xx response into out (if non-nil).
// Non-2xx responses are returned as *apiError.
func (c *apiClient) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var problem struct {
			Detail string `json:"detail"`
		}
		detail := strings.TrimSpace(string(msg))
		if json.Unmarshal(msg, &problem) == nil && problem.Detail != "" {
			detail = problem.Detail
		}
		return &apiError{Status: resp.StatusCode, Detail: detail}
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil && err != io.EOF {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// ── API shapes (only the fields the CLI reads) ──────────────────────────────

type pushBody struct {
	Payload        map[string]any    `json:"payload"`
	Labels         map[string]string `json:"labels,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

type pushResult struct {
	EventID   string `json:"event_id"`
	Duplicate bool   `json:"duplicate"`
}

type webhookRequest struct {
	URL         string            `json:"url"`
	Events      []string          `json:"events"`
	Active      bool              `json:"active"`
	Description string            `json:"description,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

type webhookOut struct {
	WebhookID  string `json:"webhook_id"`
	URL        string `json:"url"`
	HTTPConfig struct {
		WebhookSecret string `json:"webhook_secret"`
	} `json:"http_config"`
}

type subscriptionItem struct {
	SubscriptionID string `json:"subscription_id"`
	EventName      string `json:"event_name"`
}

type subscriptionPatch struct {
	TransformEnabled  bool              `json:"transform_enabled"`
	TransformTemplate string            `json:"transform_template,omitempty"`
	LabelFilters      map[string]string `json:"label_filters,omitempty"`
}

type deliveryItem struct {
	DeliveryID   string `json:"delivery_id"`
	WebhookID    string `json:"webhook_id"`
	EventID      string `json:"event_id"`
	Status       string `json:"status"`
	ResponseCode int    `json:"response_code"`
	AttemptCount int    `json:"attempt_count"`
	MaxAttempts  int    `json:"max_attempts"`
	CreatedAt    string `json:"created_at"`
}

// ── Typed helpers ───────────────────────────────────────────────────────────

func (c *apiClient) createEventType(ctx context.Context, name, description string) error {
	return c.do(ctx, http.MethodPost, "/v1/event-types", map[string]any{
		"name":        name,
		"description": description,
		"active":      true,
	}, nil)
}

func (c *apiClient) pushEvent(ctx context.Context, namespace, event string, body pushBody) (pushResult, error) {
	var out pushResult
	path := "/v1/namespaces/" + url.PathEscape(namespace) + "/events?event=" + url.QueryEscape(event)
	err := c.do(ctx, http.MethodPost, path, body, &out)
	return out, err
}

func (c *apiClient) registerWebhook(ctx context.Context, namespace string, req webhookRequest) (webhookOut, error) {
	var out webhookOut
	err := c.do(ctx, http.MethodPost, "/v1/namespaces/"+url.PathEscape(namespace)+"/webhooks", req, &out)
	return out, err
}

func (c *apiClient) deleteWebhook(ctx context.Context, namespace, webhookID string) error {
	return c.do(ctx, http.MethodDelete, "/v1/namespaces/"+url.PathEscape(namespace)+"/webhooks/"+url.PathEscape(webhookID), nil, nil)
}

func (c *apiClient) listSubscriptions(ctx context.Context, namespace, webhookID string) ([]subscriptionItem, error) {
	var out struct {
		Items []subscriptionItem `json:"items"`
	}
	path := "/v1/namespaces/" + url.PathEscape(namespace) + "/subscriptions?webhook_id=" + url.QueryEscape(webhookID)
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out.Items, err
}

func (c *apiClient) patchSubscription(ctx context.Context, namespace, subscriptionID string, patch subscriptionPatch) error {
	path := "/v1/namespaces/" + url.PathEscape(namespace) + "/subscriptions/" + url.PathEscape(subscriptionID)
	return c.do(ctx, http.MethodPatch, path, patch, nil)
}

func (c *apiClient) listDeliveries(ctx context.Context, namespace, status string, limit int) ([]deliveryItem, error) {
	var out struct {
		Items []deliveryItem `json:"items"`
	}
	q := url.Values{"limit": {fmt.Sprint(limit)}}
	if status != "" {
		q.Set("status", status)
	}
	path := "/v1/namespaces/" + url.PathEscape(namespace) + "/deliveries?" + q.Encode()
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out.Items, err
}

func (c *apiClient) listWebhooks(ctx context.Context, namespace string) ([]webhookOut, error) {
	var out struct {
		Items []webhookOut `json:"items"`
	}
	err := c.do(ctx, http.MethodGet, "/v1/namespaces/"+url.PathEscape(namespace)+"/webhooks", nil, &out)
	return out.Items, err
}
