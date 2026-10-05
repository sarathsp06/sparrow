package webhooks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// ListenPolicy controls listen sessions (`sparrow listen` against a server
// that cannot reach the developer's machine). They are off unless Enabled:
// a session sends real event payloads to whoever runs the CLI.
type ListenPolicy struct {
	Enabled bool
	// MaxTTL caps (and is the default for) a session's lifetime.
	MaxTTL time.Duration
	// MaxPerConsumer caps a consumer's concurrent sessions.
	MaxPerConsumer int
}

// WithListenSessions enables listen sessions under policy.
func WithListenSessions(policy ListenPolicy) WebhookServiceOption {
	return func(s *WebhookService) {
		s.listen = policy
	}
}

const (
	// MaxListenWait caps one long-poll for deliveries; it stays well inside
	// the server's 30s write timeout.
	MaxListenWait = 20 * time.Second
	// MaxListenClaim is the most deliveries one poll returns.
	MaxListenClaim = 10
	// listenClaimPoll is how often a waiting poll re-checks for deliveries.
	listenClaimPoll = 250 * time.Millisecond

	// A session delivers to a terminal: answer fast, retry soon.
	listenRequestTimeoutSeconds = 10
	listenMaxRetries            = 2
	listenRetryBackoffSeconds   = 5
)

// ListenSessionRequest creates a listen session.
type ListenSessionRequest struct {
	Consumer    string
	Events      []string
	TTL         time.Duration // 0 = the policy's MaxTTL
	Description string
}

// ListenSession is a created session. Secret is returned only at creation.
type ListenSession struct {
	SessionID string
	Consumer  string
	Events    []string
	Secret    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// ListenManager runs listen sessions: a temporary webhook that the delivery
// worker hands to a polling CLI instead of dialing.
type ListenManager interface {
	CreateListenSession(ctx context.Context, req ListenSessionRequest) (*ListenSession, error)
	DeleteListenSession(ctx context.Context, consumer, sessionID string) error
	// ClaimListenDeliveries waits up to wait for deliveries to the session and
	// claims them. An empty result means none arrived in time.
	ClaimListenDeliveries(ctx context.Context, consumer, sessionID string, wait time.Duration) ([]*store.ListenDelivery, error)
	// RespondListenDelivery reports the local app's response to a claimed
	// delivery; it becomes that delivery attempt's outcome.
	RespondListenDelivery(ctx context.Context, consumer, sessionID, deliveryID string, status int, headers map[string]string, body []byte) error
}

func (s *WebhookService) listenEnabled() error {
	if !s.listen.Enabled {
		return svcerrors.Error(svcerrors.PermissionDenied, "listen sessions are disabled on this server (set SPARROW_LISTEN_ENABLED=true)")
	}
	return nil
}

// CreateListenSession registers a sparrow-cli:// webhook subscribed to
// req.Events, plus its session row.
func (s *WebhookService) CreateListenSession(ctx context.Context, req ListenSessionRequest) (*ListenSession, error) {
	if err := s.listenEnabled(); err != nil {
		return nil, err
	}
	if req.Consumer == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}
	if len(req.Events) == 0 {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "at least one event is required")
	}
	ttl := req.TTL
	if ttl <= 0 || ttl > s.listen.MaxTTL {
		ttl = s.listen.MaxTTL
	}
	tenantID := tenant.DefaultTenantID

	if s.listen.MaxPerConsumer > 0 {
		n, err := s.webhookRepo.CountListenSessions(ctx, tenantID, req.Consumer)
		if err != nil {
			return nil, fmt.Errorf("count listen sessions: %w", err)
		}
		if n >= s.listen.MaxPerConsumer {
			return nil, svcerrors.Errorf(svcerrors.ResourceExhausted, "consumer %q already has %d listen sessions (the maximum); stop one first", req.Consumer, n)
		}
	}

	description := "sparrow listen"
	if d := strings.TrimSpace(req.Description); d != "" {
		description += ": " + d
	}
	id := uuid.New()
	active := true
	maxRetries := listenMaxRetries
	hook, err := s.createWebhook(ctx, WebhookRegistrationRequest{
		ID:          id.String(),
		Consumer:    req.Consumer,
		Events:      req.Events,
		URL:         client.ListenURL(id),
		Active:      &active,
		Description: description,
		HTTPConfig: &WebhookHTTPConfig{
			MaxRetries:            &maxRetries,
			RetryBackoffSeconds:   listenRetryBackoffSeconds,
			RequestTimeoutSeconds: listenRequestTimeoutSeconds,
		},
	}, false)
	if err != nil {
		return nil, err
	}

	session := &store.ListenSession{
		WebhookID: id,
		TenantID:  tenantID,
		Consumer:  req.Consumer,
		ExpiresAt: time.Now().Add(ttl),
	}
	if err := s.webhookRepo.CreateListenSession(ctx, session); err != nil {
		// Without its session row the webhook would only collect failures.
		_ = s.webhookRepo.UnregisterWebhook(context.WithoutCancel(ctx), tenantID, id)
		return nil, fmt.Errorf("create listen session: %w", err)
	}
	s.logger.InfoContext(ctx, "Listen session started", "session_id", id, "consumer", req.Consumer, "events", req.Events, "expires_at", session.ExpiresAt)

	return &ListenSession{
		SessionID: id.String(),
		Consumer:  req.Consumer,
		Events:    req.Events,
		Secret:    hook.HTTPConfig.WebhookSecret,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

// getListenSession resolves a session of consumer, or NotFound.
func (s *WebhookService) getListenSession(ctx context.Context, consumer, sessionID string) (*store.ListenSession, error) {
	if err := s.listenEnabled(); err != nil {
		return nil, err
	}
	id, err := parseUUID(sessionID, "session ID")
	if err != nil {
		return nil, err
	}
	session, err := s.webhookRepo.GetListenSession(ctx, tenant.DefaultTenantID, id)
	if storage.IsNotFound(err) || (err == nil && session.Consumer != consumer) {
		return nil, svcerrors.Error(svcerrors.NotFound, "listen session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get listen session: %w", err)
	}
	return session, nil
}

// DeleteListenSession deletes the session's webhook (and with it the
// session, its subscriptions and deliveries).
func (s *WebhookService) DeleteListenSession(ctx context.Context, consumer, sessionID string) error {
	session, err := s.getListenSession(ctx, consumer, sessionID)
	if err != nil {
		return err
	}
	if err := s.webhookRepo.UnregisterWebhook(ctx, session.TenantID, session.WebhookID); err != nil {
		return fmt.Errorf("delete listen session: %w", err)
	}
	s.logger.InfoContext(ctx, "Listen session stopped", "session_id", session.WebhookID, "consumer", consumer)
	return nil
}

// ClaimListenDeliveries long-polls for the session's deliveries. Every call
// marks the CLI as connected.
func (s *WebhookService) ClaimListenDeliveries(ctx context.Context, consumer, sessionID string, wait time.Duration) ([]*store.ListenDelivery, error) {
	session, err := s.getListenSession(ctx, consumer, sessionID)
	if err != nil {
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, svcerrors.Error(svcerrors.FailedPrecondition, "listen session expired; start a new one")
	}
	if err := s.webhookRepo.TouchListenSession(ctx, session.TenantID, session.WebhookID); err != nil {
		return nil, fmt.Errorf("touch listen session: %w", err)
	}
	wait = min(max(wait, 0), MaxListenWait)
	deadline := time.Now().Add(wait)
	for {
		claimed, err := s.webhookRepo.ClaimListenDeliveries(ctx, session.WebhookID, MaxListenClaim)
		if err != nil {
			return nil, fmt.Errorf("claim listen deliveries: %w", err)
		}
		if len(claimed) > 0 || !time.Now().Before(deadline) {
			return claimed, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(listenClaimPoll):
		}
	}
}

// RespondListenDelivery records the CLI's response to a claimed delivery.
func (s *WebhookService) RespondListenDelivery(ctx context.Context, consumer, sessionID, deliveryID string, status int, headers map[string]string, body []byte) error {
	session, err := s.getListenSession(ctx, consumer, sessionID)
	if err != nil {
		return err
	}
	id, err := parseUUID(deliveryID, "delivery ID")
	if err != nil {
		return err
	}
	if status < 100 || status > 599 {
		return svcerrors.Errorf(svcerrors.InvalidArgument, "status %d is not an HTTP status code", status)
	}
	err = s.webhookRepo.RespondListenDelivery(ctx, session.WebhookID, id, status, headers, body)
	if storage.IsNotFound(err) {
		return svcerrors.Error(svcerrors.NotFound, "no claimed delivery waiting for a response (it may have timed out)")
	}
	return err
}
