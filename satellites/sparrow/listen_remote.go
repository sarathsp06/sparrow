package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// listenWaitSeconds is how long one poll waits on the server (its maximum).
const listenWaitSeconds = 20

type listenSessionRequest struct {
	Events      []string `json:"events"`
	TTLSeconds  int      `json:"ttl_seconds,omitempty"`
	Description string   `json:"description,omitempty"`
}

type listenSessionOut struct {
	SessionID     string   `json:"session_id"`
	Events        []string `json:"events"`
	WebhookSecret string   `json:"webhook_secret"`
	ExpiresAt     string   `json:"expires_at"`
}

type listenDeliveryOut struct {
	DeliveryID string            `json:"delivery_id"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

type listenResponseIn struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    []byte            `json:"body,omitempty"`
}

func (c *apiClient) createListenSession(ctx context.Context, consumer string, req listenSessionRequest) (listenSessionOut, error) {
	var out listenSessionOut
	err := c.do(ctx, http.MethodPost, c.consumerPath(consumer, "listen-sessions"), req, &out)
	return out, err
}

func (c *apiClient) deleteListenSession(ctx context.Context, consumer, sessionID string) error {
	return c.do(ctx, http.MethodDelete, c.consumerPath(consumer, "listen-sessions/"+url.PathEscape(sessionID)), nil, nil)
}

func (c *apiClient) claimListenDeliveries(ctx context.Context, consumer, sessionID string, waitSeconds int) ([]listenDeliveryOut, error) {
	var out struct {
		Items []listenDeliveryOut `json:"items"`
	}
	path := c.consumerPath(consumer, "listen-sessions/"+url.PathEscape(sessionID)+fmt.Sprintf(":claim?wait_seconds=%d", waitSeconds))
	err := c.do(ctx, http.MethodPost, path, nil, &out)
	return out.Items, err
}

func (c *apiClient) respondListenDelivery(ctx context.Context, consumer, sessionID, deliveryID string, resp listenResponseIn) error {
	path := c.consumerPath(consumer, "listen-sessions/"+url.PathEscape(sessionID)+"/deliveries/"+url.PathEscape(deliveryID)+":respond")
	return c.do(ctx, http.MethodPost, path, resp, nil)
}

// runRemoteListen starts a listen session and serves its deliveries until
// ctx is cancelled, then deletes the session.
func runRemoteListen(ctx context.Context, out io.Writer, client *apiClient, consumer string, events listFlag, ttl time.Duration, forward string) error {
	session, err := client.createListenSession(ctx, consumer, listenSessionRequest{
		Events:      events,
		TTLSeconds:  int(ttl / time.Second),
		Description: listenDescription(),
	})
	if err != nil {
		return listenSessionError(err)
	}
	_, _ = fmt.Fprintf(out, "listen session %s started (events: %v, expires %s)\n", session.SessionID, []string(events), session.ExpiresAt)
	_, _ = fmt.Fprintf(out, "webhook secret: %s\n", session.WebhookSecret)
	if forward != "" {
		_, _ = fmt.Fprintf(out, "forwarding deliveries to %s\n", forward)
	}
	_, _ = fmt.Fprintln(out, "press Ctrl-C to stop and delete the session")

	serveErr := serveListenSession(ctx, out, client, consumer, session, forward)

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.deleteListenSession(cleanupCtx, consumer, session.SessionID); err != nil && !isAPIStatus(err, http.StatusNotFound) {
		return errors.Join(serveErr, fmt.Errorf("delete listen session %s: %w", session.SessionID, err))
	}
	_, _ = fmt.Fprintf(out, "\ndeleted listen session %s\n", session.SessionID)
	return serveErr
}

// serveListenSession polls for deliveries and answers each one. It returns
// nil on Ctrl-C and an error once the session is gone or expired.
func serveListenSession(ctx context.Context, out io.Writer, client *apiClient, consumer string, session listenSessionOut, forward string) error {
	var mu sync.Mutex // serializes printing of concurrently handled deliveries
	var wg sync.WaitGroup
	defer wg.Wait()
	backoff := time.Second
	for {
		items, err := client.claimListenDeliveries(ctx, consumer, session.SessionID, listenWaitSeconds)
		if ctx.Err() != nil {
			return nil
		}
		if isAPIStatus(err, http.StatusNotFound) || isAPIStatus(err, http.StatusConflict) {
			return fmt.Errorf("listen session ended: %w", err)
		}
		if err != nil {
			_, _ = fmt.Fprintf(out, "poll: %v (retrying in %s)\n", err, backoff)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, 15*time.Second)
			continue
		}
		backoff = time.Second
		for _, d := range items {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var buf bytes.Buffer
				resp := handleListenDelivery(ctx, &buf, d, session.WebhookSecret, forward)
				err := client.respondListenDelivery(ctx, consumer, session.SessionID, d.DeliveryID, resp)
				if err != nil && ctx.Err() == nil {
					_, _ = fmt.Fprintf(&buf, "report response: %v\n", err)
				}
				mu.Lock()
				out.Write(buf.Bytes()) //nolint:errcheck
				mu.Unlock()
			}()
		}
	}
}

// handleListenDelivery prints one delivery, then forwards it (or answers 200)
// if its signature verifies, and returns the response to report.
func handleListenDelivery(ctx context.Context, out io.Writer, d listenDeliveryOut, secret, forward string) listenResponseIn {
	req, err := http.NewRequestWithContext(ctx, d.Method, "/", bytes.NewReader(d.Body))
	if err != nil {
		return listenResponseIn{Status: http.StatusBadRequest, Body: []byte(err.Error())}
	}
	for k, v := range d.Headers {
		req.Header.Set(k, v)
	}
	if !printReceived(out, req, d.Body, secret) {
		return listenResponseIn{Status: http.StatusUnauthorized, Body: []byte("invalid or missing webhook signature")}
	}
	if forward == "" {
		return listenResponseIn{Status: http.StatusOK}
	}
	resp, err := forwardRequest(ctx, d.Method, forward, req.Header, d.Body)
	if err != nil {
		_, _ = fmt.Fprintf(out, "forward: %v\n", err)
		return listenResponseIn{Status: http.StatusBadGateway, Body: []byte("sparrow listen: " + err.Error())}
	}
	_, _ = fmt.Fprintf(out, "forward: %s -> %d\n", forward, resp.status)
	return listenResponseIn{Status: resp.status, Headers: resp.headers, Body: resp.body}
}

// listenSessionError explains the errors a user can act on.
func listenSessionError(err error) error {
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		return fmt.Errorf("start listen session: %w", err)
	}
	switch {
	case apiErr.Status == http.StatusNotFound:
		return fmt.Errorf("start listen session: this server has no listen sessions (upgrade it, or use --direct for a local Sparrow): %w", err)
	case apiErr.Status == http.StatusForbidden && strings.Contains(apiErr.Detail, "disabled"):
		return fmt.Errorf("start listen session: %w (or use --direct when Sparrow runs on this machine)", err)
	case apiErr.Status == http.StatusForbidden:
		return fmt.Errorf("start listen session: %w (with a consumer access token, add --portal)", err)
	}
	return fmt.Errorf("start listen session: %w", err)
}

func isAPIStatus(err error, status int) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.Status == status
}

// listenDescription labels the session's webhook with where it runs.
func listenDescription() string {
	host, _ := os.Hostname()
	return host
}
