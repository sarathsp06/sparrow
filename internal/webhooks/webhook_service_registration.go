package webhooks

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// setWebhookActive is the shared implementation of PauseWebhook and ResumeWebhook.
// It loads the webhook, checks whether a state transition is needed, and persists the change.
func (s *WebhookService) setWebhookActive(ctx context.Context, webhookID string, consumer string, active bool) error {
	if webhookID == "" {
		return svcerrors.Error(svcerrors.InvalidArgument, "webhook ID is required")
	}
	if consumer == "" {
		return svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := parseUUID(webhookID, "webhook ID")
	if err != nil {
		return err
	}

	webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, id, consumer)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get webhook", "error", err)
		return fmt.Errorf("failed to retrieve webhook: %w", err)
	}
	if webhook.Active == active {
		if active {
			return svcerrors.Error(svcerrors.FailedPrecondition, "webhook is already active")
		}
		return svcerrors.Error(svcerrors.FailedPrecondition, "webhook is already paused")
	}
	webhook.Active = active
	webhook.UpdatedAt = time.Now()
	if err := s.webhookRepo.UpdateWebhook(ctx, tenantID, webhook); err != nil {
		action := "pause"
		if active {
			action = "resume"
		}
		s.logger.ErrorContext(ctx, "Failed to "+action+" webhook", "error", err)
		return fmt.Errorf("failed to %s webhook: %w", action, err)
	}

	action := "paused"
	if active {
		action = "resumed"
	}
	s.logger.InfoContext(ctx, "Webhook "+action+" successfully", "webhook_id", webhookID)
	return nil
}

func (s *WebhookService) RegisterWebhook(ctx context.Context, consumer string, events []string, url string, headers map[string]string, timeout int, active bool, description string, secretHeaders map[string]string) (string, time.Time, error) {
	ctx, span := s.tracer.Start(ctx, "webhook.register",
		trace.WithAttributes(
			attribute.String("consumer", consumer),
			attribute.StringSlice("events", events),
			attribute.String("url", url),
		),
	)
	defer span.End()

	tenantID := tenant.DefaultTenantID

	s.logger.InfoContext(ctx, "Processing webhook registration request",
		"consumer", consumer,
		"events", events,
		"url", url,
	)

	if consumer == "" {
		return "", time.Time{}, svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}
	if url == "" {
		return "", time.Time{}, svcerrors.Error(svcerrors.InvalidArgument, "URL is required")
	}
	if err := ValidateWebhookURL(url, s.allowPrivateNetworks); err != nil {
		return "", time.Time{}, err
	}
	if len(events) > 0 {
		s.logger.InfoContext(ctx, "Validating event names", "events", events, "contains_empty", slices.Contains(events, ""))
		if slices.Contains(events, "") {
			s.logger.ErrorContext(ctx, "Event names validation failed", "events", events)
			return "", time.Time{}, svcerrors.Error(svcerrors.InvalidArgument, "event names cannot be empty")
		}
	}
	if timeout <= 0 {
		timeout = 30
	}
	registration := &store.WebhookRegistration{
		Consumer:      consumer,
		URL:           url,
		Headers:       headers,
		Timeout:       int(timeout),
		Active:        active,
		Description:   description,
		SignatureType: store.SignatureTypeHMAC,
		// HTTP config defaults (mirror DefaultWebhookHTTPConfig / DB column
		// defaults) — the insert writes every column, so zero values would
		// violate the timeout/backoff check constraints.
		MaxRetries:            3,
		RetryBackoffSeconds:   60,
		FollowRedirects:       true,
		VerifySSL:             true,
		RequestTimeoutSeconds: int(timeout),
		ExpectedStatusCodes:   pq.Int64Array{200, 201, 202, 204},
		UserAgent:             "Sparrow-Webhook/1.0",
		ContentType:           "application/json",
	}

	// Encrypt secret headers if provided
	if len(secretHeaders) > 0 {
		encrypted, err := s.EncryptSecretHeaders(secretHeaders)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to encrypt secret headers: %w", err)
		}
		registration.SecretHeaders = encrypted
	}

	// Build subscriptions slice for atomic creation
	var subscriptions []*store.EventSubscription
	for _, event := range events {
		subscriptions = append(subscriptions, &store.EventSubscription{
			EventName: event,
			Consumer:  consumer,
		})
	}

	// Atomically create webhook + all subscriptions in a single transaction
	if err := s.webhookRepo.RegisterWebhookWithSubscriptions(ctx, tenantID, registration, subscriptions); err != nil {
		s.logger.ErrorContext(ctx, "Failed to register webhook",
			"consumer", consumer,
			"events", events,
			"url", url,
			"error", err,
		)
		return "", time.Time{}, fmt.Errorf("failed to register webhook: %w", err)
	}

	// Always generate a Standard Webhook secret if not provided
	if registration.WebhookSecret == nil {
		secret, err := generateWebhookSecret()
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to generate webhook secret: %w", err)
		}
		encSecret, err := s.EncryptWebhookSecret(secret)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to encrypt webhook secret: %w", err)
		}
		registration.WebhookSecret = encSecret
		// Update the record with the secret
		if err := s.webhookRepo.UpdateWebhook(ctx, tenantID, registration); err != nil {
			s.logger.WarnContext(ctx, "Failed to update webhook with generated secret", "webhook_id", registration.ID, "error", err)
		}
	}

	if s.metrics != nil {
		s.metrics.WebhookRegistrations.Add(ctx, 1)
		s.metrics.ActiveWebhooks.Add(ctx, 1)
	}
	s.logger.InfoContext(ctx, "Webhook registered successfully",
		"webhook_id", registration.ID,
		"consumer", consumer,
		"events", events,
		"url", url,
	)
	return registration.ID.String(), registration.CreatedAt, nil
}

// CreateWebhook creates a webhook registration with HTTP configuration support
func (s *WebhookService) CreateWebhook(ctx context.Context, req WebhookRegistrationRequest) (*WebhookRegistration, error) {
	ctx, span := s.tracer.Start(ctx, "webhook.create",
		trace.WithAttributes(
			attribute.String("consumer", req.Consumer),
			attribute.StringSlice("events", req.Events),
			attribute.String("url", req.URL),
		),
	)
	defer span.End()

	s.logger.InfoContext(ctx, "Processing enhanced webhook creation request",
		"consumer", req.Consumer,
		"events", req.Events,
		"url", req.URL,
	)

	tenantID := tenant.DefaultTenantID

	if req.Consumer == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}

	// Validate webhook URL against SSRF
	if err := ValidateWebhookURL(req.URL, s.allowPrivateNetworks); err != nil {
		return nil, err
	}

	// Convert request to internal webhook registration
	webhookReg, err := req.ToWebhookRegistration()
	if err != nil {
		return nil, svcerrors.Wrapf(err, svcerrors.InvalidArgument, "invalid webhook configuration: %v", err)
	}

	// Generate ID if not provided
	if webhookReg.ID == "" {
		webhookReg.ID = uuid.New().String()
	}

	// Validate event names exist
	for _, event := range req.Events {
		if event == "" {
			return nil, svcerrors.Error(svcerrors.InvalidArgument, "empty event name not allowed")
		}
		// Check if event is registered
		events, _, err := s.webhookRepo.ListEventsPaginated(ctx, tenantID, false, 1000, 0)
		if err != nil {
			s.logger.WarnContext(ctx, "Failed to validate event names", "error", err)
		} else {
			if !slices.ContainsFunc(events, func(e *store.EventRegistration) bool {
				return e.Name == event
			}) {
				s.logger.WarnContext(ctx, "Event not registered", "event", event, "consumer", req.Consumer)
			}
		}
	}

	webhookID, err := parseUUID(webhookReg.ID, "webhook ID")
	if err != nil {
		return nil, err
	}

	// Convert internal webhook to store model for database operation
	storeWebhook := &store.WebhookRegistration{
		ID:                    webhookID,
		Consumer:              webhookReg.Consumer,
		URL:                   webhookReg.URL,
		Timeout:               webhookReg.HTTPConfig.RequestTimeoutSeconds,
		Active:                webhookReg.Active,
		Description:           webhookReg.Description,
		Health:                store.WebhookHealth(webhookReg.Health),
		MaxRetries:            DerefIntOr(webhookReg.HTTPConfig.MaxRetries, 3),
		RetryBackoffSeconds:   webhookReg.HTTPConfig.RetryBackoffSeconds,
		CaptureResponseBody:   DerefBoolOr(webhookReg.HTTPConfig.CaptureResponseBody, false),
		FollowRedirects:       DerefBoolOr(webhookReg.HTTPConfig.FollowRedirects, true),
		VerifySSL:             DerefBoolOr(webhookReg.HTTPConfig.VerifySSL, true),
		RequestTimeoutSeconds: webhookReg.HTTPConfig.RequestTimeoutSeconds,
		UserAgent:             webhookReg.HTTPConfig.UserAgent,
		ContentType:           webhookReg.HTTPConfig.ContentType,
		RateLimitRPS:          webhookReg.HTTPConfig.RateLimitRPS,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Set signature type, defaulting to HMAC
	sigType := store.SignatureType(webhookReg.SignatureType)
	if sigType != store.SignatureTypeEd25519 {
		sigType = store.SignatureTypeHMAC
	}
	storeWebhook.SignatureType = sigType

	// Convert headers to string map for store model
	headersMap := make(map[string]string)
	for k, v := range webhookReg.Headers {
		if str, ok := v.(string); ok {
			headersMap[k] = str
		}
	}
	storeWebhook.Headers = headersMap

	// Encrypt webhook secret if provided, or generate one if missing
	if webhookReg.HTTPConfig.WebhookSecret != "" {
		encSecret, err := s.EncryptWebhookSecret(webhookReg.HTTPConfig.WebhookSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt webhook secret: %w", err)
		}
		storeWebhook.WebhookSecret = encSecret
	} else {
		// Auto-generate a secret if missing to ensure all webhooks are signed
		secret, err := generateWebhookSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate webhook secret: %w", err)
		}
		encSecret, err := s.EncryptWebhookSecret(secret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt generated webhook secret: %w", err)
		}
		storeWebhook.WebhookSecret = encSecret
		webhookReg.HTTPConfig.WebhookSecret = secret // Return the plain secret in the response
	}

	// Generate Ed25519 keypair only when signature_type is "ed25519".
	// The private key is envelope-encrypted and stored; the public key is derived at runtime.
	if sigType == store.SignatureTypeEd25519 {
		encPrivKey, err := s.generateEncryptedEd25519Key()
		if err != nil {
			return nil, err
		}
		storeWebhook.Ed25519PrivateKey = encPrivKey
	}

	// Encrypt secret headers if provided
	if len(req.SecretHeaders) > 0 {
		encrypted, err := s.EncryptSecretHeaders(req.SecretHeaders)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret headers: %w", err)
		}
		storeWebhook.SecretHeaders = encrypted
	}

	// Convert expected status codes
	expectedCodes := make(pq.Int64Array, len(webhookReg.HTTPConfig.ExpectedStatusCodes))
	for i, code := range webhookReg.HTTPConfig.ExpectedStatusCodes {
		expectedCodes[i] = int64(code)
	}
	storeWebhook.ExpectedStatusCodes = expectedCodes

	// Build subscriptions slice for atomic creation
	var subscriptions []*store.EventSubscription
	for _, event := range req.Events {
		subscriptions = append(subscriptions, &store.EventSubscription{
			EventName: event,
			Consumer:  req.Consumer,
		})
	}

	// Atomically register the webhook and all subscriptions in a single transaction
	if err := s.webhookRepo.RegisterWebhookWithSubscriptions(ctx, tenantID, storeWebhook, subscriptions); err != nil {
		s.logger.ErrorContext(ctx, "Failed to register webhook",
			"consumer", req.Consumer,
			"events", req.Events,
			"url", req.URL,
			"error", err,
		)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to register webhook")
		return nil, fmt.Errorf("failed to register webhook: %w", err)
	}

	// Initialize rate limit state if rate limiting is configured
	if storeWebhook.RateLimitRPS != nil {
		if err := s.webhookRepo.UpsertRateLimitState(ctx, webhookID); err != nil {
			s.logger.WarnContext(ctx, "Failed to initialize rate limit state",
				"webhook_id", webhookReg.ID,
				"error", err,
			)
			// Non-fatal: the rate limit row will be created on first delivery slot acquisition
		}
	}

	// Update metrics
	if s.metrics != nil {
		s.metrics.WebhookRegistrations.Add(ctx, 1)
		s.metrics.ActiveWebhooks.Add(ctx, 1)
	}

	s.logger.InfoContext(ctx, "Enhanced webhook registered successfully",
		"webhook_id", webhookReg.ID,
		"consumer", req.Consumer,
		"events", req.Events,
		"url", req.URL,
		"http_config_provided", req.HTTPConfig != nil,
	)

	span.SetStatus(otelcodes.Ok, "webhook created successfully")

	// Attach encrypted Ed25519 key and the actual persisted timestamps so the
	// handler returns what was stored, not zero-value placeholders (F-009).
	webhookReg.Ed25519EncryptedPrivateKey = storeWebhook.Ed25519PrivateKey
	webhookReg.SignatureType = string(storeWebhook.SignatureType)
	webhookReg.CreatedAt = storeWebhook.CreatedAt
	webhookReg.UpdatedAt = storeWebhook.UpdatedAt

	return webhookReg, nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookService) UnregisterWebhook(ctx context.Context, webhookID string, consumer string) error {
	s.logger.InfoContext(ctx, "Processing webhook un registration request",
		"webhook_id", webhookID,
		"consumer", consumer,
	)
	if webhookID == "" {
		return svcerrors.Error(svcerrors.InvalidArgument, "webhook_id is required")
	}
	if consumer == "" {
		return svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := parseUUID(webhookID, "webhook ID")
	if err != nil {
		return err
	}

	// Check if webhook exists in consumer
	_, err = s.webhookRepo.GetWebhookByID(ctx, tenantID, id, consumer)
	if err != nil {
		return fmt.Errorf("failed to retrieve webhook: %w", err)
	}

	if err := s.webhookRepo.UnregisterWebhook(ctx, tenantID, id); err != nil {
		s.logger.ErrorContext(ctx, "Failed to unregister webhook",
			"webhook_id", webhookID,
			"error", err,
		)
		return fmt.Errorf("failed to unregister webhook: %w", err)
	}
	s.logger.InfoContext(ctx, "Webhook unregistered successfully",
		"webhook_id", webhookID,
	)
	return nil
}

// ListWebhooks lists all registered webhooks with optional consumer and other filters.
// When consumer is empty, returns webhooks across all consumers.
// When health is non-empty, only webhooks with that health status are returned.
func (s *WebhookService) ListWebhooks(ctx context.Context, consumer string, webhookID string, event string, activeOnly bool, health string, limit, offset int32) ([]*store.WebhookRegistration, int32, error) {
	s.logger.InfoContext(ctx, "Processing list webhooks request",
		"consumer", consumer,
		"webhook_id", webhookID,
		"event", event,
		"active_only", activeOnly,
		"limit", limit,
		"offset", offset,
	)

	tenantID := tenant.DefaultTenantID

	l, o := normalizePagination(int(limit), int(offset))
	limit, offset = int32(l), int32(o)

	if webhookID != "" {
		id, err := parseUUID(webhookID, "webhook ID")
		if err != nil {
			return nil, 0, err
		}

		reg, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, id, consumer)
		if err != nil {
			if storage.IsNotFound(err) {
				return []*store.WebhookRegistration{}, 0, nil
			}
			return nil, 0, fmt.Errorf("failed to retrieve webhook: %w", err)
		}
		if activeOnly && !reg.Active {
			return []*store.WebhookRegistration{}, 0, nil
		}
		if health != "" && string(reg.Health) != health {
			return []*store.WebhookRegistration{}, 0, nil
		}
		return []*store.WebhookRegistration{reg}, 1, nil
	}

	registrations, totalCount, err := s.webhookRepo.ListWebhooksPaginated(ctx, tenantID, consumer, event, activeOnly, store.WebhookHealth(health), int(limit), int(offset))
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list webhooks",
			"consumer", consumer,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to list webhooks: %w", err)
	}

	s.logger.InfoContext(ctx, "Listed webhooks successfully",
		"consumer", consumer,
		"count", len(registrations),
		"total", totalCount,
	)
	return registrations, int32(totalCount), nil
}

// PauseWebhook temporarily disables webhook deliveries
func (s *WebhookService) PauseWebhook(ctx context.Context, webhookID string, consumer string, reason string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.PauseWebhook")
	defer span.End()

	s.logger.InfoContext(ctx, "Pausing webhook", "webhook_id", webhookID, "consumer", consumer, "reason", reason)
	return s.setWebhookActive(ctx, webhookID, consumer, false)
}

// ResumeWebhook re-enables webhook deliveries
func (s *WebhookService) ResumeWebhook(ctx context.Context, webhookID string, consumer string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ResumeWebhook")
	defer span.End()

	s.logger.InfoContext(ctx, "Resuming webhook", "webhook_id", webhookID, "consumer", consumer)
	return s.setWebhookActive(ctx, webhookID, consumer, true)
}

// UpdateWebhookConfig updates webhook configuration.
// When updateMask is non-empty, only the listed field paths are applied.
// When updateMask is empty, falls back to legacy behavior (all non-zero fields applied).
//
// Supported mask paths:
//
//	"url", "active", "description", "events", "headers", "secret_headers",
//	"signature_type", "http_config", "http_config.webhook_secret",
//	"http_config.rate_limit_rps"
func (s *WebhookService) UpdateWebhookConfig(ctx context.Context, webhookID string, consumer string, events []string, url string, headers map[string]string, active bool, description string, httpConfig *HTTPConfigUpdate, secretHeaders map[string]string, signatureType string, updateMask []string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.UpdateWebhookConfig")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing update webhook config request",
		"webhook_id", webhookID,
		"consumer", consumer,
		"update_mask", updateMask)

	if webhookID == "" {
		return svcerrors.Error(svcerrors.InvalidArgument, "webhook ID is required")
	}
	if consumer == "" {
		return svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
	}

	tenantID := tenant.DefaultTenantID

	webhookUUID, err := parseUUID(webhookID, "webhook ID")
	if err != nil {
		return err
	}

	webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, webhookUUID, consumer)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get webhook", "error", err)
		return fmt.Errorf("failed to retrieve webhook: %w", err)
	}

	// Build a set for O(1) lookup. When empty, all non-zero fields are applied (legacy).
	mask := make(map[string]bool, len(updateMask))
	for _, p := range updateMask {
		mask[p] = true
	}
	useMask := len(mask) > 0

	shouldUpdate := func(field string) bool {
		if !useMask {
			return true // legacy: apply everything
		}
		return mask[field]
	}

	// Build new subscription list (pure logic, no DB calls)
	var newSubs []*store.EventSubscription
	replaceEvents := shouldUpdate("events") && len(events) > 0
	if replaceEvents {
		for _, event := range events {
			newSubs = append(newSubs, &store.EventSubscription{
				EventName: event,
			})
		}
	}

	// Apply config mutations to the in-memory webhook (pure logic, no DB calls)
	if shouldUpdate("url") && url != "" {
		normalizedURL := strings.TrimSpace(url)
		if normalizedURL == "" {
			return svcerrors.Error(svcerrors.InvalidArgument, "URL is required")
		}
		if err := ValidateWebhookURL(normalizedURL, s.allowPrivateNetworks); err != nil {
			return err
		}
		webhook.URL = normalizedURL
	}
	if shouldUpdate("headers") && headers != nil {
		webhook.Headers = headers
	}
	if shouldUpdate("active") {
		webhook.Active = active
	}
	if shouldUpdate("description") && description != "" {
		webhook.Description = description
	}
	// Apply HTTP config updates if provided
	if shouldUpdate("http_config") && httpConfig != nil {
		if httpConfig.MaxRetries != nil {
			webhook.MaxRetries = *httpConfig.MaxRetries
		}
		if httpConfig.RetryBackoffSeconds > 0 {
			webhook.RetryBackoffSeconds = httpConfig.RetryBackoffSeconds
		}
		if httpConfig.RequestTimeoutSeconds > 0 {
			webhook.RequestTimeoutSeconds = httpConfig.RequestTimeoutSeconds
		}
		if len(httpConfig.ExpectedStatusCodes) > 0 {
			int64Codes := make([]int64, len(httpConfig.ExpectedStatusCodes))
			for i, c := range httpConfig.ExpectedStatusCodes {
				int64Codes[i] = int64(c)
			}
			webhook.ExpectedStatusCodes = int64Codes
		}
		// Only update the webhook secret if explicitly requested via mask.
		// Without mask (legacy mode), non-empty secret is applied.
		// With mask, "http_config.webhook_secret" must be in the mask.
		updateSecret := false
		if useMask {
			updateSecret = mask["http_config.webhook_secret"]
		} else {
			updateSecret = httpConfig.WebhookSecret != ""
		}
		if updateSecret && httpConfig.WebhookSecret != "" {
			encSecret, err := s.EncryptWebhookSecret(httpConfig.WebhookSecret)
			if err != nil {
				return fmt.Errorf("failed to encrypt webhook secret: %w", err)
			}
			webhook.WebhookSecret = encSecret
		}
		if httpConfig.UserAgent != "" {
			webhook.UserAgent = httpConfig.UserAgent
		}
		if httpConfig.ContentType != "" {
			webhook.ContentType = httpConfig.ContentType
		}
		if httpConfig.CaptureResponseBody != nil {
			webhook.CaptureResponseBody = *httpConfig.CaptureResponseBody
		}
		if httpConfig.FollowRedirects != nil {
			webhook.FollowRedirects = *httpConfig.FollowRedirects
		}
		if httpConfig.VerifySSL != nil {
			webhook.VerifySSL = *httpConfig.VerifySSL
		}
		// RateLimitRPS: pointer field — nil means "not provided" (keep existing),
		// non-nil overrides. With mask, "http_config.rate_limit_rps" must be in the mask.
		updateRateLimit := false
		if useMask {
			updateRateLimit = mask["http_config.rate_limit_rps"]
		} else {
			updateRateLimit = httpConfig.RateLimitRPS != nil
		}
		if updateRateLimit {
			webhook.RateLimitRPS = httpConfig.RateLimitRPS
		}
	}
	// Encrypt and set secret headers if in mask (or legacy non-empty)
	if shouldUpdate("secret_headers") && len(secretHeaders) > 0 {
		encrypted, err := s.EncryptSecretHeaders(secretHeaders)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret headers: %w", err)
		}
		webhook.SecretHeaders = encrypted
	}
	// Update signature_type if in mask (or legacy non-empty)
	if shouldUpdate("signature_type") && signatureType != "" {
		newSigType := store.SignatureType(signatureType)
		if newSigType != store.SignatureTypeHMAC && newSigType != store.SignatureTypeEd25519 {
			return svcerrors.Errorf(svcerrors.InvalidArgument, "invalid signature_type: %q (must be \"hmac\" or \"ed25519\")", signatureType)
		}
		oldSigType := webhook.SignatureType
		webhook.SignatureType = newSigType
		// Generate Ed25519 keypair when switching to ed25519
		if newSigType == store.SignatureTypeEd25519 && oldSigType != store.SignatureTypeEd25519 {
			encPrivKey, err := s.generateEncryptedEd25519Key()
			if err != nil {
				return err
			}
			webhook.Ed25519PrivateKey = encPrivKey
		}
		// Clear Ed25519 key when switching away from ed25519
		if newSigType == store.SignatureTypeHMAC && oldSigType == store.SignatureTypeEd25519 {
			webhook.Ed25519PrivateKey = nil
		}
	}

	// Persist subscription replacement + webhook update atomically.
	err = s.webhookRepo.RunInTransaction(func(txRepo store.RepositoryInterface) error {
		if replaceEvents {
			if err := txRepo.ReplaceWebhookSubscriptions(ctx, tenantID, webhookUUID, consumer, newSubs); err != nil {
				return fmt.Errorf("failed to update webhook subscriptions: %w", err)
			}
		}
		return txRepo.UpdateWebhook(ctx, tenantID, webhook)
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update webhook config",
			"webhook_id", webhookID,
			"error", err)
		return fmt.Errorf("failed to update webhook configuration: %w", err)
	}

	// Manage rate limit state: upsert when rate limit is set, delete when cleared
	if webhook.RateLimitRPS != nil {
		if err := s.webhookRepo.UpsertRateLimitState(ctx, webhookUUID); err != nil {
			s.logger.WarnContext(ctx, "Failed to upsert rate limit state",
				"webhook_id", webhookID,
				"error", err,
			)
		}
	} else {
		// Rate limit removed (or was never set) — clean up any existing state
		if err := s.webhookRepo.DeleteRateLimitState(ctx, webhookUUID); err != nil {
			s.logger.WarnContext(ctx, "Failed to delete rate limit state",
				"webhook_id", webhookID,
				"error", err,
			)
		}
	}

	s.logger.InfoContext(ctx, "Webhook configuration updated successfully",
		"webhook_id", webhookID)
	return nil
}
