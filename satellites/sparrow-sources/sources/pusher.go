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
type Pusher interface {
	PushEvent(ctx context.Context, event string, payload json.RawMessage, labels map[string]string) error
}

// Client is a minimal Sparrow REST client for pushing events. It auto-creates
// the event type on the first 404 and retries the push once.
type Client struct {
	cfg  SparrowConfig
	http *http.Client
}

// NewClient builds a push client for the given Sparrow connection.
func NewClient(cfg SparrowConfig) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

// PushEvent records one event occurrence in Sparrow.
func (c *Client) PushEvent(ctx context.Context, event string, payload json.RawMessage, labels map[string]string) error {
	status, err := c.post(ctx, "/v1/namespaces/"+url.PathEscape(c.cfg.Namespace)+"/events?event="+url.QueryEscape(event),
		map[string]any{"payload": payload, "labels": labels})
	if err != nil {
		return err
	}
	if status == http.StatusNotFound { // unknown event type: create it, retry once
		if err := c.createEventType(ctx, event); err != nil {
			return err
		}
		status, err = c.post(ctx, "/v1/namespaces/"+url.PathEscape(c.cfg.Namespace)+"/events?event="+url.QueryEscape(event),
			map[string]any{"payload": payload, "labels": labels})
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+path, bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}
