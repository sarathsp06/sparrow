package queue

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

const (
	// ListenOfflineAfter is how long after its last poll a listen session
	// counts as disconnected. The CLI long-polls for at most 20s, so a
	// connected CLI is always seen more often than this.
	ListenOfflineAfter = 45 * time.Second
	// listenDefaultTimeout bounds the wait for the CLI when the webhook has
	// no request timeout, and listenMaxTimeout caps it so a delivery never
	// holds a worker slot past River's job timeout.
	listenDefaultTimeout = 10 * time.Second
	listenMaxTimeout     = 30 * time.Second
	listenPollInterval   = 200 * time.Millisecond
)

// Error messages start with the words error classification keys on, so the
// attempt is recorded as connection_refused / timeout and retried like any
// unreachable receiver.
var errListenOffline = errors.New("connection refused: listen session is not connected (sparrow listen is not running)")

// listenSender delivers to a listen session (sparrow-cli:// webhook): it
// queues the prepared, signed request for the CLI and waits for the response
// the CLI reports, which then goes through the same success/failure handling
// as an HTTP response.
type listenSender struct {
	repo store.ListenRepository
	poll time.Duration
}

func newListenSender(repo store.ListenRepository) *listenSender {
	return &listenSender{repo: repo, poll: listenPollInterval}
}

// Send queues req for the session and waits for the CLI's response.
func (s *listenSender) Send(ctx context.Context, tenantID uuid.UUID, req *client.DeliveryRequest) (*http.Response, time.Duration, error) {
	session, err := s.repo.GetListenSession(ctx, tenantID, req.WebhookID)
	if storage.IsNotFound(err) {
		return nil, 0, errListenOffline
	}
	if err != nil {
		return nil, 0, fmt.Errorf("get listen session: %w", err)
	}
	if now := time.Now(); now.After(session.ExpiresAt) || now.Sub(session.LastSeenAt) > ListenOfflineAfter {
		return nil, 0, errListenOffline
	}

	// BuildRequest sets the signature and Sparrow headers exactly as an HTTP
	// delivery would, so the CLI forwards byte-for-byte what a real receiver
	// gets.
	httpReq, err := client.BuildRequest(ctx, req)
	if req.Headers != nil {
		client.PutHeaderMap(req.Headers)
		req.Headers = nil
	}
	if err != nil {
		return nil, 0, err
	}
	headers := make(map[string]string, len(httpReq.Header))
	for k := range httpReq.Header {
		headers[k] = httpReq.Header.Get(k)
	}
	deliveryID, err := uuid.Parse(req.DeliveryID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid delivery ID %q: %w", req.DeliveryID, err)
	}

	start := time.Now()
	if err := s.repo.QueueListenDelivery(ctx, &store.ListenDelivery{
		DeliveryID: deliveryID,
		WebhookID:  req.WebhookID,
		Method:     httpReq.Method,
		Headers:    headers,
		Body:       req.Payload,
	}); err != nil {
		return nil, 0, fmt.Errorf("queue listen delivery: %w", err)
	}
	// The attempt is over whichever way this returns; a late response from
	// the CLI then finds nothing to answer.
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = s.repo.DeleteListenDelivery(cleanupCtx, deliveryID)
	}()

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = listenDefaultTimeout
	}
	timeout = min(timeout, listenMaxTimeout)
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(s.poll)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, time.Since(start), ctx.Err()
		case <-deadline.C:
			return nil, time.Since(start), fmt.Errorf("timeout: sparrow listen did not answer within %s", timeout)
		case <-tick.C:
		}
		d, err := s.repo.GetListenDelivery(ctx, deliveryID)
		if storage.IsNotFound(err) {
			// The session was deleted (Ctrl-C) while the attempt waited.
			return nil, time.Since(start), errListenOffline
		}
		if err != nil {
			return nil, time.Since(start), fmt.Errorf("read listen delivery: %w", err)
		}
		if d.RespondedAt == nil || d.ResponseStatus == nil {
			continue
		}
		return listenResponse(*d.ResponseStatus, d.ResponseHeaders, d.ResponseBody), time.Since(start), nil
	}
}

// listenResponse turns the CLI's reported response into an *http.Response.
func listenResponse(status int, headers map[string]string, body []byte) *http.Response {
	h := make(http.Header, len(headers))
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     h,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}
