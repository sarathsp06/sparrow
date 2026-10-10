package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sarathsp06/sparrow/pkg/signature"
)

type startListenIn struct {
	Events    []string `json:"events" jsonschema:"event types to subscribe the temporary webhook to"`
	Port      int      `json:"port,omitempty" jsonschema:"local port (default: random free port)"`
	PublicURL string   `json:"public_url,omitempty" jsonschema:"URL the Sparrow server delivers to (default http://host.docker.internal:<port>, for a Sparrow running in Docker on this machine)"`
}

type receivedDelivery struct {
	ReceivedAt string            `json:"received_at"`
	EventName  string            `json:"event_name,omitempty"`
	EventID    string            `json:"event_id,omitempty"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
}

type listenStatus struct {
	Running   bool               `json:"running"`
	WebhookID string             `json:"webhook_id,omitempty"`
	URL       string             `json:"url,omitempty"`
	LocalAddr string             `json:"local_addr,omitempty"`
	Events    []string           `json:"events,omitempty"`
	Received  []receivedDelivery `json:"received"`
}

// mcpListener is the MCP counterpart of `sparrow listen`: a local receiver
// behind a temporary webhook, holding what it received for the agent to read.
type mcpListener struct {
	client   *apiClient
	consumer string

	mu       sync.Mutex
	srv      *http.Server
	status   listenStatus
	received []receivedDelivery
}

var errListenerRunning = errors.New("a listener is already running; call stop_listen first")

func (l *mcpListener) start(ctx context.Context, in startListenIn) (listenStatus, error) {
	if len(in.Events) == 0 {
		return listenStatus{}, errors.New("events is required")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.srv != nil {
		return listenStatus{}, errListenerRunning
	}

	ln, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(in.Port)))
	if err != nil {
		return listenStatus{}, err
	}
	localPort := ln.Addr().(*net.TCPAddr).Port
	registerURL := in.PublicURL
	if registerURL == "" {
		registerURL = fmt.Sprintf("http://host.docker.internal:%d", localPort)
	}

	hook, err := l.client.registerWebhook(ctx, l.consumer, webhookRequest{
		URL: registerURL, Events: in.Events, Active: true, Description: "sparrow mcp start_listen (temporary)",
	})
	if err != nil {
		ln.Close() //nolint:errcheck
		return listenStatus{}, fmt.Errorf("register webhook: %w", err)
	}

	l.received = nil
	l.srv = &http.Server{Handler: l.handler(hook.HTTPConfig.WebhookSecret), ReadHeaderTimeout: 10 * time.Second}
	go l.srv.Serve(ln) //nolint:errcheck
	l.status = listenStatus{Running: true, WebhookID: hook.WebhookID, URL: registerURL, LocalAddr: ln.Addr().String(), Events: in.Events}
	return l.snapshot(), nil
}

// handler verifies each delivery against the temporary webhook's secret and
// records it; unsigned requests get 401 and are never recorded, since the
// port may be reachable by others on the network.
func (l *mcpListener) handler(secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, maxListenBody))
		if secret != "" {
			if err := signature.VerifyHMAC(body, r.Header, secret); err != nil {
				http.Error(w, "invalid or missing webhook signature", http.StatusUnauthorized)
				return
			}
		}
		d := receivedDelivery{ReceivedAt: time.Now().UTC().Format(time.RFC3339), Headers: map[string]string{}, Body: body}
		for k := range r.Header {
			d.Headers[k] = r.Header.Get(k)
		}
		if !json.Valid(body) {
			d.Body, _ = json.Marshal(string(body))
		}
		var env struct {
			EventName string `json:"event_name"`
			EventID   string `json:"event_id"`
		}
		_ = json.Unmarshal(body, &env)
		d.EventName, d.EventID = env.EventName, env.EventID

		l.mu.Lock()
		l.received = append(l.received, d)
		l.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
}

func (l *mcpListener) current() (listenStatus, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.snapshot(), nil
}

// stop shuts the receiver down and deletes the temporary webhook. Calling it
// with no listener running is a no-op, so shutdown paths can call it blindly.
func (l *mcpListener) stop(ctx context.Context) (listenStatus, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.srv == nil {
		return l.snapshot(), nil
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	l.srv.Shutdown(shutdownCtx) //nolint:errcheck
	l.srv = nil
	err := l.client.deleteWebhook(shutdownCtx, l.consumer, l.status.WebhookID)
	l.status.Running = false
	out := l.snapshot()
	if err != nil {
		return out, fmt.Errorf("delete temporary webhook %s: %w", l.status.WebhookID, err)
	}
	return out, nil
}

// snapshot copies the status under l.mu so callers can release the lock.
func (l *mcpListener) snapshot() listenStatus {
	s := l.status
	s.Received = append([]receivedDelivery(nil), l.received...)
	if s.Received == nil {
		s.Received = []receivedDelivery{}
	}
	return s
}
