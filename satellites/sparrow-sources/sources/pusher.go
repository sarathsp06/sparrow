package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Pusher pushes one event into Sparrow. Satisfied by *Client and by test fakes.
// A non-empty idempotencyKey makes the push idempotent: Sparrow records one
// event per key and consumer, and answers a repeat with the original event.
type Pusher interface {
	PushEvent(ctx context.Context, event string, payload json.RawMessage, labels map[string]string, idempotencyKey string) error
}

// Client is a minimal Sparrow REST client for pushing events. It auto-creates
// the event type on the first 404 and retries the push once.
type Client struct {
	cfg  SparrowConfig
	http *http.Client
}

// NewClient builds a push client for the given Sparrow connection. An empty
// Consumer means "default".
func NewClient(cfg SparrowConfig) *Client {
	if cfg.Consumer == "" {
		cfg.Consumer = "default"
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

// PushEvent records one event occurrence in Sparrow. With a non-empty
// idempotencyKey, a push that repeats an earlier key succeeds without
// creating a new event (Sparrow answers 201 with duplicate: true).
func (c *Client) PushEvent(ctx context.Context, event string, payload json.RawMessage, labels map[string]string, idempotencyKey string) error {
	body := map[string]any{"payload": payload, "labels": labels}
	if idempotencyKey != "" {
		body["idempotency_key"] = idempotencyKey
	}
	status, err := c.post(ctx, "/v1/consumers/"+url.PathEscape(c.cfg.Consumer)+"/events?event="+url.QueryEscape(event), body)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound { // unknown event type: create it, retry once
		if err := c.createEventType(ctx, event); err != nil {
			return err
		}
		status, err = c.post(ctx, "/v1/consumers/"+url.PathEscape(c.cfg.Consumer)+"/events?event="+url.QueryEscape(event), body)
		if err != nil {
			return err
		}
	}
	if status >= 300 {
		return fmt.Errorf("push event %s: status %d", event, status)
	}
	return nil
}

func (c *Client) createEventType(ctx context.Context, name string) error {
	status, err := c.post(ctx, "/v1/event-types", map[string]any{"name": name, "active": true})
	if err != nil {
		return err
	}
	if status >= 300 && status != http.StatusConflict {
		return fmt.Errorf("create event type %s: status %d", name, status)
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string, body any) (int, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	// The URL is the operator's configured Sparrow server, not request input.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+path, bytes.NewReader(b)) //nolint:gosec // see above
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	}
	resp, err := c.http.Do(req) //nolint:gosec // configured Sparrow URL
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}
