package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	jsonschema "github.com/kaptinlin/jsonschema"
	"github.com/lib/pq"
	"github.com/sarathsp06/schemagen"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"google.golang.org/grpc/codes"

	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

type WebhookService struct {
	jobInserter          queue.JobInserter
	webhookRepo          store.RepositoryInterface
	crypto               *crypto.Service
	logger               *slog.Logger
	tracer               trace.Tracer
	metrics              *observability.SparrowMetrics
	allowPrivateNetworks bool
}

//go:generate gowrap gen -i WebhookServiceInterface -t ../../templates/opentelemetry.tmpl -o WebhookServiceInterface_otel.go
type WebhookServiceInterface interface {
	// Webhook Management
	RegisterWebhook(ctx context.Context, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string, secretHeaders map[string]string) (string, time.Time, error)
	CreateWebhook(ctx context.Context, req WebhookRegistrationRequest) (*WebhookRegistration, error)
	UnregisterWebhook(ctx context.Context, webhookID string, namespace string) error
	ListWebhooks(ctx context.Context, namespace string, webhookID string, event string, activeOnly bool, limit, offset int32) ([]*store.WebhookRegistration, int32, error)
	UpdateWebhookConfig(ctx context.Context, webhookID string, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string, httpConfig *HTTPConfigUpdate, secretHeaders map[string]string, updateMask []string) error
	PauseWebhook(ctx context.Context, webhookID string, namespace string, reason string) error
	ResumeWebhook(ctx context.Context, webhookID string, namespace string) error
	GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStatsData, error)

	// Event Management
	RegisterEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) (string, time.Time, error)
	ListEvents(ctx context.Context, activeOnly bool, limit, offset int32) ([]*store.EventRegistration, int32, error)
	UpdateEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) error
	DeleteEvent(ctx context.Context, name string) error
	GetEvent(ctx context.Context, name string) (*store.EventRegistration, error)
	PushEvent(ctx context.Context, namespace string, event string, payload map[string]any, ttlSeconds int64, metadata map[string]string, labels map[string]string) (string, []string, error)
	RePushEvent(ctx context.Context, eventID string) (string, []string, error)
	GetEventRecord(ctx context.Context, eventID string) (*store.EventRecord, int32, int32, int32, int32, error)
	ListEventReports(ctx context.Context, filter store.EventReportFilter) ([]*store.EventReportWithStats, int32, string, error)

	// Subscription Management
	CreateSubscription(ctx context.Context, webhookID, eventName, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string) (string, time.Time, error)
	GetSubscription(ctx context.Context, subscriptionID string, namespace string) (*store.EventSubscription, error)
	ListSubscriptions(ctx context.Context, namespace string, webhookID string, eventName string, limit, offset int32) ([]*store.EventSubscription, int32, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string) error
	DeleteSubscription(ctx context.Context, subscriptionID string, namespace string) error
	TestSubscriptionTemplate(ctx context.Context, eventName, transformTemplate, namespace string) (string, error)

	// Delivery Management
	GetDeliveryStatus(ctx context.Context, deliveryID string, namespace string) (*store.WebhookDelivery, error)
	GetDeliveryAttempts(ctx context.Context, deliveryID string) ([]*store.WebhookHealthEvent, error)
	ListDeliveries(ctx context.Context, filter store.DeliveryFilter) ([]*store.WebhookDelivery, int32, string, error)
	RetryDelivery(ctx context.Context, namespace string, deliveryID string, webhookID string, force bool) ([]string, int32, error)

	// Health Management
	GetWebhookHealth(ctx context.Context, webhookID string, namespace string) (*WebhookHealthData, error)
	ListWebhooksByHealth(ctx context.Context, health store.WebhookHealth, limit, offset int32) ([]*store.WebhookRegistration, int32, error)
	GetHealthSummary(ctx context.Context) (*HealthSummaryData, error)

	// Batch Operations
	RePushEvents(ctx context.Context, repushID string) error
	GetRepushStatus(ctx context.Context, repushID string) (*store.BatchJob, error)
	CancelRepush(ctx context.Context, repushID string) error
	RetryDeliveries(ctx context.Context, retryID string) error
	GetRetryStatus(ctx context.Context, retryID string) (*store.BatchJob, error)
	CancelRetry(ctx context.Context, retryID string) error

	// Metadata
	GetTemplateFunctions() []TemplateFunctionInfo

	// Repository access
	GetWebhookRepo() store.RepositoryInterface

	// Crypto
	DecryptSecretHeaders(encrypted []byte) (map[string]string, error)
	DecryptWebhookSecret(encrypted []byte) (string, error)
	GetCrypto() *crypto.Service
}

type TemplateFunctionInfo struct {
	Name        string
	Description string
}

var _ WebhookServiceInterface = (*WebhookService)(nil)

// NewWebhookService creates a new WebhookService instance
// WebhookServiceOption configures a WebhookService.
type WebhookServiceOption func(*WebhookService)

// WithAllowPrivateNetworks disables SSRF protection for webhook URL validation,
// permitting loopback and private-network addresses. Useful for self-hosted
// deployments where webhook targets live on the same network, and required for
// integration tests that use httptest.NewServer.
func WithAllowPrivateNetworks(allow bool) WebhookServiceOption {
	return func(s *WebhookService) {
		s.allowPrivateNetworks = allow
	}
}

func NewWebhookService(queueManager queue.JobInserter, webhookRepo store.RepositoryInterface, cryptoSvc *crypto.Service, opts ...WebhookServiceOption) *WebhookService {
	metrics, err := observability.NewSparrowMetrics()
	if err != nil {
		// Log error but continue without metrics
		log := logger.NewLogger("webhook-service")
		log.Error("Failed to initialize metrics", "error", err)
	}

	svc := &WebhookService{
		jobInserter: queueManager,
		webhookRepo: webhookRepo,
		crypto:      cryptoSvc,
		logger:      logger.NewLogger("webhook-service"),
		tracer:      observability.GetTracer("sparrow.service.webhook"),
		metrics:     metrics,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// EncryptSecretHeaders encrypts a plaintext secret headers map to bytes for storage.
// Returns nil if the map is empty or nil, or if encryption is not configured.
func (s *WebhookService) EncryptSecretHeaders(headers map[string]string) ([]byte, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, svcerrors.FailedPrecondition("encryption is required for secret headers but SPARROW_ENCRYPTION_KEY is not configured")
	}
	return s.crypto.EncryptJSON(headers)
}

// DecryptSecretHeaders decrypts encrypted secret headers bytes back to a plaintext map.
// Returns nil map if the encrypted data is nil/empty or if encryption is not configured.
func (s *WebhookService) DecryptSecretHeaders(encrypted []byte) (map[string]string, error) {
	if len(encrypted) == 0 {
		return nil, nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, svcerrors.FailedPrecondition("encryption key not configured; cannot decrypt secret headers")
	}
	var headers map[string]string
	if err := s.crypto.DecryptJSON(encrypted, &headers); err != nil {
		return nil, fmt.Errorf("failed to decrypt secret headers: %w", err)
	}
	return headers, nil
}

// EncryptWebhookSecret encrypts a plaintext webhook secret string to bytes for storage.
// Returns nil if the secret is empty, or if encryption is not configured.
func (s *WebhookService) EncryptWebhookSecret(secret string) ([]byte, error) {
	if secret == "" {
		return nil, nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return nil, svcerrors.FailedPrecondition("encryption is required for webhook secrets but SPARROW_ENCRYPTION_KEY is not configured")
	}
	return s.crypto.EncryptString(secret)
}

// DecryptWebhookSecret decrypts encrypted webhook secret bytes back to a plaintext string.
// Returns "" if the encrypted data is nil/empty or if encryption is not configured.
func (s *WebhookService) DecryptWebhookSecret(encrypted []byte) (string, error) {
	if len(encrypted) == 0 {
		return "", nil
	}
	if s.crypto == nil || !s.crypto.Enabled() {
		return "", svcerrors.FailedPrecondition("encryption key not configured; cannot decrypt webhook secret")
	}
	return s.crypto.DecryptString(encrypted)
}

// GetCrypto returns the crypto service for use by workers and handlers
func (s *WebhookService) GetCrypto() *crypto.Service {
	return s.crypto
}

// GetWebhookRepo returns the repository interface for direct access
func (s *WebhookService) GetWebhookRepo() store.RepositoryInterface {
	return s.webhookRepo
}

// Label validation constraints
const (
	maxLabelsPerMap  = 20  // max key-value pairs per labels/labelFilters map
	maxLabelKeyLen   = 64  // max characters per label key
	maxLabelValueLen = 256 // max characters per label value
)

// labelKeyPattern restricts label keys to alphanumeric, dot, underscore, and hyphen.
var labelKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// validateLabels checks that a labels/label_filters map meets size and format constraints.
func validateLabels(m map[string]string, fieldName string) error {
	if len(m) > maxLabelsPerMap {
		return fmt.Errorf("%s: too many entries (%d), maximum is %d", fieldName, len(m), maxLabelsPerMap)
	}
	for k, v := range m {
		if k == "" {
			return fmt.Errorf("%s: key must not be empty", fieldName)
		}
		if len(k) > maxLabelKeyLen {
			return fmt.Errorf("%s: key %q exceeds maximum length of %d characters", fieldName, k, maxLabelKeyLen)
		}
		if !labelKeyPattern.MatchString(k) {
			return fmt.Errorf("%s: key %q contains invalid characters (allowed: alphanumeric, '.', '_', '-')", fieldName, k)
		}
		if len(v) > maxLabelValueLen {
			return fmt.Errorf("%s: value for key %q exceeds maximum length of %d characters", fieldName, k, maxLabelValueLen)
		}
	}
	return nil
}

func (s *WebhookService) RegisterWebhook(ctx context.Context, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string, secretHeaders map[string]string) (string, time.Time, error) {
	ctx, span := s.tracer.Start(ctx, "webhook.register",
		trace.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.StringSlice("events", events),
			attribute.String("url", url),
		),
	)
	defer span.End()

	tenantID := tenant.DefaultTenantID

	s.logger.InfoContext(ctx, "Processing webhook registration request",
		"namespace", namespace,
		"events", events,
		"url", url,
	)

	if namespace == "" {
		return "", time.Time{}, fmt.Errorf("namespace is required")
	}
	if url == "" {
		return "", time.Time{}, fmt.Errorf("URL is required")
	}
	if err := ValidateWebhookURL(url, s.allowPrivateNetworks); err != nil {
		return "", time.Time{}, err
	}
	if len(events) > 0 {
		s.logger.InfoContext(ctx, "Validating event names", "events", events, "contains_empty", slices.Contains(events, ""))
		if slices.Contains(events, "") {
			s.logger.ErrorContext(ctx, "Event names validation failed", "events", events)
			return "", time.Time{}, fmt.Errorf("event names cannot be empty")
		}
	}
	if timeout <= 0 {
		timeout = 30
	}
	registration := &store.WebhookRegistration{
		Namespace:   namespace,
		URL:         url,
		Headers:     headers,
		Timeout:     int(timeout),
		Active:      active,
		Description: description,
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
			Namespace: namespace,
		})
	}

	// Atomically create webhook + all subscriptions in a single transaction
	if err := s.webhookRepo.RegisterWebhookWithSubscriptions(ctx, tenantID, registration, subscriptions); err != nil {
		s.logger.ErrorContext(ctx, "Failed to register webhook",
			"namespace", namespace,
			"events", events,
			"url", url,
			"error", err,
		)
		return "", time.Time{}, fmt.Errorf("failed to register webhook: %w", err)
	}

	if s.metrics != nil {
		s.metrics.WebhookRegistrations.Add(ctx, 1)
		s.metrics.ActiveWebhooks.Add(ctx, 1)
	}
	s.logger.InfoContext(ctx, "Webhook registered successfully",
		"webhook_id", registration.ID,
		"namespace", namespace,
		"events", events,
		"url", url,
	)
	return registration.ID.String(), registration.CreatedAt, nil
}

// CreateWebhook creates a webhook registration with HTTP configuration support
func (s *WebhookService) CreateWebhook(ctx context.Context, req WebhookRegistrationRequest) (*WebhookRegistration, error) {
	ctx, span := s.tracer.Start(ctx, "webhook.create",
		trace.WithAttributes(
			attribute.String("namespace", req.Namespace),
			attribute.StringSlice("events", req.Events),
			attribute.String("url", req.URL),
		),
	)
	defer span.End()

	s.logger.InfoContext(ctx, "Processing enhanced webhook creation request",
		"namespace", req.Namespace,
		"events", req.Events,
		"url", req.URL,
	)

	tenantID := tenant.DefaultTenantID

	if req.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	// Validate webhook URL against SSRF
	if err := ValidateWebhookURL(req.URL, s.allowPrivateNetworks); err != nil {
		return nil, err
	}

	// Convert request to internal webhook registration
	webhookReg, err := req.ToWebhookRegistration()
	if err != nil {
		return nil, svcerrors.Wrapf(err, codes.InvalidArgument, "invalid webhook configuration: %v", err)
	}

	// Generate ID if not provided
	if webhookReg.ID == "" {
		webhookReg.ID = uuid.New().String()
	}

	// Validate event names exist
	for _, event := range req.Events {
		if event == "" {
			return nil, fmt.Errorf("empty event name not allowed")
		}
		// Check if event is registered
		events, _, err := s.webhookRepo.ListEventsPaginated(ctx, tenantID, false, 1000, 0)
		if err != nil {
			s.logger.WarnContext(ctx, "Failed to validate event names", "error", err)
		} else {
			if !slices.ContainsFunc(events, func(e *store.EventRegistration) bool {
				return e.Name == event
			}) {
				s.logger.WarnContext(ctx, "Event not registered", "event", event, "namespace", req.Namespace)
			}
		}
	}

	webhookID, err := uuid.Parse(webhookReg.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook ID: %w", err)
	}

	// Convert internal webhook to store model for database operation
	storeWebhook := &store.WebhookRegistration{
		ID:                    webhookID,
		Namespace:             webhookReg.Namespace,
		URL:                   webhookReg.URL,
		Timeout:               webhookReg.HTTPConfig.RequestTimeoutSeconds,
		Active:                webhookReg.Active,
		Description:           webhookReg.Description,
		Health:                store.WebhookHealth(webhookReg.Health),
		MaxRetries:            webhookReg.HTTPConfig.MaxRetries,
		RetryBackoffSeconds:   webhookReg.HTTPConfig.RetryBackoffSeconds,
		CaptureResponseBody:   webhookReg.HTTPConfig.CaptureResponseBody,
		FollowRedirects:       webhookReg.HTTPConfig.FollowRedirects,
		VerifySSL:             webhookReg.HTTPConfig.VerifySSL,
		RequestTimeoutSeconds: webhookReg.HTTPConfig.RequestTimeoutSeconds,
		UserAgent:             webhookReg.HTTPConfig.UserAgent,
		ContentType:           webhookReg.HTTPConfig.ContentType,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Convert headers to string map for store model
	headersMap := make(map[string]string)
	for k, v := range webhookReg.Headers {
		if str, ok := v.(string); ok {
			headersMap[k] = str
		}
	}
	storeWebhook.Headers = headersMap

	// Encrypt webhook secret if provided
	if webhookReg.HTTPConfig.WebhookSecret != "" {
		encSecret, err := s.EncryptWebhookSecret(webhookReg.HTTPConfig.WebhookSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt webhook secret: %w", err)
		}
		storeWebhook.WebhookSecret = encSecret
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
			Namespace: req.Namespace,
		})
	}

	// Atomically register the webhook and all subscriptions in a single transaction
	if err := s.webhookRepo.RegisterWebhookWithSubscriptions(ctx, tenantID, storeWebhook, subscriptions); err != nil {
		s.logger.ErrorContext(ctx, "Failed to register webhook",
			"namespace", req.Namespace,
			"events", req.Events,
			"url", req.URL,
			"error", err,
		)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to register webhook")
		return nil, fmt.Errorf("failed to register webhook: %w", err)
	}

	// Update metrics
	if s.metrics != nil {
		s.metrics.WebhookRegistrations.Add(ctx, 1)
		s.metrics.ActiveWebhooks.Add(ctx, 1)
	}

	s.logger.InfoContext(ctx, "Enhanced webhook registered successfully",
		"webhook_id", webhookReg.ID,
		"namespace", req.Namespace,
		"events", req.Events,
		"url", req.URL,
		"http_config_provided", req.HTTPConfig != nil,
	)

	span.SetStatus(otelcodes.Ok, "webhook created successfully")
	return webhookReg, nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookService) UnregisterWebhook(ctx context.Context, webhookID string, namespace string) error {
	s.logger.InfoContext(ctx, "Processing webhook un registration request",
		"webhook_id", webhookID,
		"namespace", namespace,
	)
	if webhookID == "" {
		return fmt.Errorf("webhook_id is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %w", err)
	}

	// Check if webhook exists in namespace
	_, err = s.webhookRepo.GetWebhookByID(ctx, tenantID, id, namespace)
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

// ListWebhooks lists all registered webhooks with optional namespace and other filters.
// When namespace is empty, returns webhooks across all namespaces.
func (s *WebhookService) ListWebhooks(ctx context.Context, namespace string, webhookID string, event string, activeOnly bool, limit, offset int32) ([]*store.WebhookRegistration, int32, error) {
	s.logger.InfoContext(ctx, "Processing list webhooks request",
		"namespace", namespace,
		"webhook_id", webhookID,
		"event", event,
		"active_only", activeOnly,
		"limit", limit,
		"offset", offset,
	)

	tenantID := tenant.DefaultTenantID

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	if webhookID != "" {
		id, err := uuid.Parse(webhookID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid webhook ID: %w", err)
		}

		// When looking up by ID, namespace can be empty — try without namespace filter
		if namespace != "" {
			reg, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, id, namespace)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to retrieve webhook: %w", err)
			}
			if reg == nil || (activeOnly && !reg.Active) {
				return []*store.WebhookRegistration{}, 0, nil
			}
			return []*store.WebhookRegistration{reg}, 1, nil
		}
		// Without namespace, fall through to paginated list which will find it
	}

	registrations, totalCount, err := s.webhookRepo.ListWebhooksPaginated(ctx, tenantID, namespace, event, activeOnly, int(limit), int(offset))
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list webhooks",
			"namespace", namespace,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to list webhooks: %w", err)
	}

	s.logger.InfoContext(ctx, "Listed webhooks successfully",
		"namespace", namespace,
		"count", len(registrations),
		"total", totalCount,
	)
	return registrations, int32(totalCount), nil
}

// PushEvent pushes an event
func (s *WebhookService) PushEvent(ctx context.Context, namespace string, event string, payload map[string]any, ttlSeconds int64, metadata map[string]string, labels map[string]string) (string, []string, error) {
	ctx, span := s.tracer.Start(ctx, "event.push",
		trace.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.String("event", event),
		),
	)
	defer span.End()

	s.logger.InfoContext(ctx, "Processing push event request",
		"namespace", namespace,
		"event", event,
	)

	tenantID := tenant.DefaultTenantID

	// Validate required fields
	if namespace == "" {
		err := fmt.Errorf("namespace is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "namespace is required")
		return "", nil, err
	}
	if event == "" {
		err := fmt.Errorf("event is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event is required")
		return "", nil, err
	}
	if err := validateLabels(labels, "labels"); err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "invalid labels")
		return "", nil, err
	}

	// Lookup registered event, auto-registering if it doesn't exist yet.
	eventReg, err := s.webhookRepo.GetEventByName(ctx, tenantID, event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event lookup failed")
		s.logger.ErrorContext(ctx, "Failed to lookup event registration", "event", event, "error", err)
		return "", nil, fmt.Errorf("failed to lookup event registration: %w", err)
	}
	if eventReg == nil {
		// Auto-register the event so callers don't have to pre-register every
		// event type. The registration is created without a schema, so any
		// payload is accepted. Users can later update it with a description and
		// JSON schema via the RegisterEvent / UpdateEvent API.
		eventReg = &store.EventRegistration{
			Name:   event,
			Active: true,
		}
		if err := s.webhookRepo.RegisterEvent(ctx, tenantID, eventReg); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "auto-registration failed")
			s.logger.ErrorContext(ctx, "Failed to auto-register event", "event", event, "error", err)
			return "", nil, fmt.Errorf("failed to auto-register event: %w", err)
		}
		s.logger.InfoContext(ctx, "Auto-registered new event type", "event", event)
	}
	if !eventReg.Active {
		err := svcerrors.FailedPreconditionf("event '%s' is inactive", event)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event inactive")
		s.logger.ErrorContext(ctx, "Event is inactive", "event", event)
		return "", nil, err
	}

	// Soft schema validation: validate payload against event schema if present.
	// Events are always accepted and stored regardless of schema match.
	// Invalid payloads are tagged (schema_valid=false) with per-field warnings.
	var warnings []string
	schemaValid := true
	if len(eventReg.Schema) != 0 && payload != nil {
		if err := ValidateJSONSchema(eventReg.Schema, payload); err != nil {
			var schemaErr *SchemaValidationError
			if errors.As(err, &schemaErr) {
				schemaValid = false
				warnings = schemaErr.Warnings()
				s.logger.WarnContext(ctx, "Payload does not match event schema (accepted with warnings)",
					"event", event,
					"warning_count", len(warnings),
				)
				span.SetAttributes(attribute.Bool("schema_valid", false))
			} else {
				// Non-schema error (e.g., schema compilation failure) -- still accept
				schemaValid = false
				warnings = []string{err.Error()}
				s.logger.WarnContext(ctx, "Schema validation encountered unexpected error (accepted with warnings)",
					"event", event,
					"error", err,
				)
			}
		}
	}

	// TTL=0 means no expiry (default). Only positive values enable expiry.
	ttl := ttlSeconds
	if ttl < 0 {
		ttl = 0
	}

	// Generate event ID
	eventID := uuid.New().String()

	// Store the event record in database first
	eventRecord := &store.EventRecord{
		ID:          uuid.MustParse(eventID),
		Namespace:   namespace,
		Event:       event,
		Payload:     payload,
		TTL:         ttl,
		Metadata:    metadata,
		Labels:      labels,
		SchemaValid: schemaValid,
		CreatedAt:   time.Now(),
	}

	if err := s.webhookRepo.StoreEvent(ctx, tenantID, eventRecord); err != nil {
		s.logger.ErrorContext(ctx, "Failed to store event record", "error", err, "event_id", eventID)
		return "", nil, fmt.Errorf("failed to store event record: %w", err)
	}

	// Create event processing job with minimal data
	eventArgs := queue.EventArgs{
		EventID:    eventID,
		Namespace:  namespace,
		Event:      event,
		TTLSeconds: ttl,
		Metadata:   metadata,
		Labels:     labels,
		CreatedAt:  eventRecord.CreatedAt,
		TenantID:   tenantID.String(),
	}

	// Insert the event processing job
	_, err = s.jobInserter.Insert(ctx, eventArgs)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to schedule event processing job",
			"event_id", eventID,
			"namespace", namespace,
			"event", event,
			"error", err,
		)
		// Compensation: delete the orphaned event record since the job failed.
		// This prevents events from existing in the DB without a corresponding
		// processing job (cross-driver: sqlx event store + pgx River job).
		if delErr := s.webhookRepo.DeleteEventByID(ctx, tenantID, eventRecord.ID); delErr != nil {
			s.logger.ErrorContext(ctx, "Failed to compensate: could not delete orphaned event record",
				"event_id", eventID,
				"delete_error", delErr,
			)
		}
		return "", nil, fmt.Errorf("failed to schedule event processing: %w", err)
	}

	// Record metrics
	if s.metrics != nil {
		s.metrics.EventsPushed.Add(ctx, 1)
	}

	span.SetStatus(otelcodes.Ok, "event scheduled successfully")

	s.logger.InfoContext(ctx, "Event processing scheduled successfully",
		"event_id", eventID,
		"namespace", namespace,
		"event", event,
	)
	return eventID, warnings, nil
}

// RePushEvent replays a previously pushed event as if it were pushed fresh.
// It loads the original event record and calls PushEvent with the same payload,
// namespace, event name, metadata, and labels. The payload is validated against
// the CURRENT event type schema. Returns a new event_id and any warnings.
func (s *WebhookService) RePushEvent(ctx context.Context, eventID string) (string, []string, error) {
	ctx, span := s.tracer.Start(ctx, "event.repush",
		trace.WithAttributes(
			attribute.String("original_event_id", eventID),
		),
	)
	defer span.End()

	s.logger.InfoContext(ctx, "Processing single event re-push",
		"original_event_id", eventID,
	)

	tenantID := tenant.DefaultTenantID

	// Parse and validate event ID
	id, err := uuid.Parse(eventID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "invalid event ID")
		return "", nil, svcerrors.InvalidInputf("invalid event ID: %v", err)
	}

	// Load original event record
	original, err := s.webhookRepo.GetEventByID(ctx, tenantID, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to load original event")
		s.logger.ErrorContext(ctx, "Failed to load original event for re-push",
			"event_id", eventID,
			"error", err,
		)
		return "", nil, fmt.Errorf("failed to load original event: %w", err)
	}
	if original == nil {
		err := fmt.Errorf("event not found: %s", eventID)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event not found")
		return "", nil, err
	}

	// Re-push through the standard PushEvent pipeline.
	// This gives us: current schema validation, new event_id, fan-out to matching subscriptions.
	newEventID, warnings, err := s.PushEvent(ctx, original.Namespace, original.Event, original.Payload, original.TTL, original.Metadata, original.Labels)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "re-push failed")
		s.logger.ErrorContext(ctx, "Failed to re-push event",
			"original_event_id", eventID,
			"error", err,
		)
		return "", nil, fmt.Errorf("failed to re-push event: %w", err)
	}

	span.SetStatus(otelcodes.Ok, "event re-pushed successfully")
	span.SetAttributes(attribute.String("new_event_id", newEventID))

	s.logger.InfoContext(ctx, "Event re-pushed successfully",
		"original_event_id", eventID,
		"new_event_id", newEventID,
	)
	return newEventID, warnings, nil
}

// GetEventRecord retrieves a single pushed event instance by UUID with delivery statistics.
func (s *WebhookService) GetEventRecord(ctx context.Context, eventID string) (*store.EventRecord, int32, int32, int32, int32, error) {
	ctx, span := s.tracer.Start(ctx, "event.get_record",
		trace.WithAttributes(
			attribute.String("event_id", eventID),
		),
	)
	defer span.End()

	tenantID := tenant.DefaultTenantID

	// Parse and validate event ID
	id, err := uuid.Parse(eventID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "invalid event ID")
		return nil, 0, 0, 0, 0, svcerrors.InvalidInputf("invalid event ID: %v", err)
	}

	// Load event record
	record, err := s.webhookRepo.GetEventByID(ctx, tenantID, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to load event record")
		return nil, 0, 0, 0, 0, fmt.Errorf("failed to load event record: %w", err)
	}
	if record == nil {
		return nil, 0, 0, 0, 0, nil
	}

	// Get delivery statistics
	webhookCount, successCount, failedCount, pendingCount, err := s.webhookRepo.GetEventDeliveryStats(ctx, tenantID, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to get delivery stats")
		s.logger.ErrorContext(ctx, "Failed to get delivery stats for event record",
			"event_id", eventID,
			"error", err,
		)
		// Return the record without stats rather than failing entirely
		return record, 0, 0, 0, 0, nil
	}

	return record, webhookCount, successCount, failedCount, pendingCount, nil
}

// GetDeliveryStatus gets the status of a webhook delivery.
// When namespace is empty, looks up by delivery ID alone.
func (s *WebhookService) GetDeliveryStatus(ctx context.Context, deliveryID string, namespace string) (*store.WebhookDelivery, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetDeliveryStatus")
	defer span.End()

	s.logger.InfoContext(ctx, "Getting webhook delivery status",
		"delivery_id", deliveryID,
		"namespace", namespace)

	if deliveryID == "" {
		return nil, fmt.Errorf("delivery ID is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := uuid.Parse(deliveryID)
	if err != nil {
		return nil, fmt.Errorf("invalid delivery ID: %w", err)
	}

	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, tenantID, id, namespace)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get delivery by ID", "error", err)
		return nil, fmt.Errorf("failed to retrieve delivery status: %w", err)
	}
	if delivery == nil {
		return nil, fmt.Errorf("delivery not found")
	}
	return delivery, nil
}

// GetDeliveryAttempts retrieves individual attempt history for a delivery.
// Returns all recorded health events for the delivery, ordered by timestamp ascending.
func (s *WebhookService) GetDeliveryAttempts(ctx context.Context, deliveryID string) ([]*store.WebhookHealthEvent, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetDeliveryAttempts")
	defer span.End()

	tenantID := tenant.DefaultTenantID

	s.logger.InfoContext(ctx, "Getting delivery attempts", "delivery_id", deliveryID, "tenant_id", tenantID.String())

	if deliveryID == "" {
		return nil, fmt.Errorf("delivery ID is required")
	}

	id, err := uuid.Parse(deliveryID)
	if err != nil {
		return nil, fmt.Errorf("invalid delivery ID: %w", err)
	}

	attempts, err := s.webhookRepo.GetDeliveryAttempts(ctx, tenantID, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get delivery attempts", "error", err)
		return nil, fmt.Errorf("failed to retrieve delivery attempts: %w", err)
	}

	return attempts, nil
}

// ListDeliveries retrieves delivery history with filters.
// Supports filtering by namespace, webhook, event, status, error_category,
// subscription, and time range via the DeliveryFilter struct.
// When PrepareRetry is true, snapshots all matching delivery IDs into a batch job and returns the batch ID.
func (s *WebhookService) ListDeliveries(ctx context.Context, filter store.DeliveryFilter) ([]*store.WebhookDelivery, int32, string, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListDeliveries")
	defer span.End()

	s.logger.InfoContext(ctx, "Listing deliveries",
		"namespace", filter.Namespace,
		"webhook_id", filter.WebhookID,
		"event_id", filter.EventID,
		"status", filter.Status,
		"error_category", filter.ErrorCategory,
		"prepare_retry", filter.PrepareRetry,
		"limit", filter.Limit,
		"offset", filter.Offset)

	tenantID := tenant.DefaultTenantID

	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	deliveries, totalCount, err := s.webhookRepo.ListDeliveriesFiltered(ctx, tenantID, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list deliveries", "error", err)
		return nil, 0, "", fmt.Errorf("failed to retrieve deliveries: %w", err)
	}

	// Snapshot matching IDs into a batch job if requested
	var retryID string
	if filter.PrepareRetry {
		ids, err := s.webhookRepo.SnapshotDeliveryIDs(ctx, tenantID, filter)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to snapshot delivery IDs for retry", "error", err)
			return nil, 0, "", fmt.Errorf("failed to prepare retry: %w", err)
		}
		if len(ids) > 0 {
			filterMap := map[string]any{
				"namespace": filter.Namespace,
			}
			if filter.WebhookID != nil {
				filterMap["webhook_id"] = filter.WebhookID.String()
			}
			if filter.EventID != nil {
				filterMap["event_id"] = filter.EventID.String()
			}
			if filter.Status != nil {
				filterMap["status"] = *filter.Status
			}
			if filter.ErrorCategory != nil {
				filterMap["error_category"] = *filter.ErrorCategory
			}
			batchData := &store.BatchJobData{
				ItemIDs: ids,
				Filter:  filterMap,
			}
			batchJob, err := s.webhookRepo.CreateBatchJob(ctx, tenantID, filter.Namespace, store.BatchTypeDeliveryRetry, batchData)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to create batch job for retry", "error", err)
				return nil, 0, "", fmt.Errorf("failed to create retry batch: %w", err)
			}
			retryID = batchJob.ID.String()
			s.logger.InfoContext(ctx, "Created retry batch job",
				"retry_id", retryID,
				"delivery_count", len(ids))
		}
	}

	return deliveries, int32(totalCount), retryID, nil
}

// RetryDelivery manually retries failed or pending webhook deliveries
func (s *WebhookService) RetryDelivery(ctx context.Context, namespace string, deliveryID string, webhookID string, force bool) ([]string, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RetryDelivery")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing retry delivery request",
		"delivery_id", deliveryID,
		"webhook_id", webhookID,
		"namespace", namespace,
		"force", force)

	tenantID := tenant.DefaultTenantID

	// Validate required fields
	if deliveryID == "" && webhookID == "" {
		return nil, 0, svcerrors.InvalidInput("either delivery_id or webhook_id is required")
	}

	// Namespace is required for webhook-level retry (multiple deliveries),
	// but optional for single-delivery retry (delivery_id is globally unique within a tenant).
	if namespace == "" && webhookID != "" {
		return nil, 0, svcerrors.InvalidInput("namespace is required for webhook-level retry")
	}

	if deliveryID != "" && webhookID != "" {
		return nil, 0, svcerrors.InvalidInput("only one of delivery_id or webhook_id can be specified")
	}

	var deliveriesToResubmit []*store.WebhookDelivery

	if deliveryID != "" {
		id, err := uuid.Parse(deliveryID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid delivery ID: %w", err)
		}

		// Resubmit specific delivery
		delivery, err := s.webhookRepo.GetDeliveryByID(ctx, tenantID, id, namespace)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get delivery", "error", err)
			return nil, 0, fmt.Errorf("failed to retrieve delivery: %w", err)
		}

		if delivery == nil {
			return nil, 0, fmt.Errorf("delivery not found")
		}

		// Check if delivery can be resubmitted
		if !force && delivery.Status == store.StatusSuccess {
			return nil, 0, svcerrors.FailedPrecondition("delivery already succeeded. Use force to resubmit anyway")
		}

		deliveriesToResubmit = []*store.WebhookDelivery{delivery}
	} else {
		id, err := uuid.Parse(webhookID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid webhook ID: %w", err)
		}

		// Resubmit all failed/pending deliveries for webhook
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, id, namespace)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get webhook", "error", err)
			return nil, 0, fmt.Errorf("failed to retrieve webhook: %w", err)
		}

		if webhook == nil {
			return nil, 0, fmt.Errorf("webhook not found")
		}

		// Get retriable deliveries
		deliveriesToResubmit, err = s.webhookRepo.GetRetriableDeliveries(ctx, tenantID, id, namespace, force)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get retriable deliveries", "error", err)
			return nil, 0, fmt.Errorf("failed to retrieve deliveries: %w", err)
		}

		if len(deliveriesToResubmit) == 0 {
			message := "No failed or pending deliveries found"
			if !force {
				message += ". Use force to resubmit all deliveries"
			}
			s.logger.InfoContext(ctx, message)
			return []string{}, 0, nil
		}
	}

	// Process each delivery for resubmission
	var resubmittedIDs []string
	var resubmittedCount int32

	for _, delivery := range deliveriesToResubmit {
		// Reset delivery status to pending
		err := s.webhookRepo.ResetDeliveryForRetry(ctx, delivery.ID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to reset delivery for retry",
				"delivery_id", delivery.ID,
				"error", err)
			continue
		}

		// Get webhook info for queuing
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, delivery.WebhookID, namespace)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get webhook for delivery",
				"webhook_id", delivery.WebhookID,
				"delivery_id", delivery.ID,
				"error", err)
			continue
		}

		// Queue the webhook for delivery.
		// Manual retries never expire -- use far-future sentinel so TTL doesn't apply.
		_, err = s.jobInserter.Insert(ctx, &queue.WebhookArgs{
			DeliveryID: delivery.ID.String(),
			WebhookID:  delivery.WebhookID.String(),
			EventID:    delivery.EventID.String(),
			ExpiresAt:  store.NoExpiryTime,
			Namespace:  webhook.Namespace,
			TenantID:   tenantID.String(),
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to queue webhook for resubmission",
				"delivery_id", delivery.ID,
				"webhook_id", delivery.WebhookID,
				"error", err)
			continue
		}

		resubmittedIDs = append(resubmittedIDs, delivery.ID.String())
		resubmittedCount++
	}

	if resubmittedCount == 0 {
		return nil, 0, svcerrors.FailedPrecondition("failed to resubmit any deliveries")
	}

	s.logger.InfoContext(ctx, "Webhook deliveries resubmitted successfully",
		"resubmitted_count", resubmittedCount,
		"total_requested", len(deliveriesToResubmit))

	return resubmittedIDs, resubmittedCount, nil
}

// PauseWebhook temporarily disables webhook deliveries
func (s *WebhookService) PauseWebhook(ctx context.Context, webhookID string, namespace string, reason string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.PauseWebhook")
	defer span.End()

	s.logger.InfoContext(ctx, "Pausing webhook", "webhook_id", webhookID, "namespace", namespace, "reason", reason)
	if webhookID == "" {
		return fmt.Errorf("webhook ID is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %w", err)
	}

	webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, id, namespace)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get webhook", "error", err)
		return fmt.Errorf("failed to retrieve webhook: %w", err)
	}
	if webhook == nil {
		return fmt.Errorf("webhook not found")
	}
	if !webhook.Active {
		return fmt.Errorf("webhook is already paused")
	}
	webhook.Active = false
	webhook.UpdatedAt = time.Now()
	err = s.webhookRepo.UpdateWebhook(ctx, tenantID, webhook)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to pause webhook", "error", err)
		return fmt.Errorf("failed to pause webhook: %w", err)
	}
	s.logger.InfoContext(ctx, "Webhook paused successfully", "webhook_id", webhookID)
	return nil
}

// ResumeWebhook re-enables webhook deliveries
func (s *WebhookService) ResumeWebhook(ctx context.Context, webhookID string, namespace string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ResumeWebhook")
	defer span.End()

	s.logger.InfoContext(ctx, "Resuming webhook",
		"webhook_id", webhookID,
		"namespace", namespace)

	if webhookID == "" {
		return fmt.Errorf("webhook ID is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %w", err)
	}

	webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, id, namespace)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get webhook", "error", err)
		return fmt.Errorf("failed to retrieve webhook: %w", err)
	}
	if webhook == nil {
		return fmt.Errorf("webhook not found")
	}
	if webhook.Active {
		return fmt.Errorf("webhook is already active")
	}
	webhook.Active = true
	webhook.UpdatedAt = time.Now()
	err = s.webhookRepo.UpdateWebhook(ctx, tenantID, webhook)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to resume webhook", "error", err)
		return fmt.Errorf("failed to resume webhook: %w", err)
	}
	s.logger.InfoContext(ctx, "Webhook resumed successfully", "webhook_id", webhookID)
	return nil
}

// WebhookHealthData represents webhook health information
type WebhookHealthData struct {
	WebhookID            string              `json:"webhook_id"`
	Health               store.WebhookHealth `json:"health"`
	TotalDeliveries      int                 `json:"total_deliveries"`
	SuccessfulDeliveries int                 `json:"successful_deliveries"`
	FailedDeliveries     int                 `json:"failed_deliveries"`
	ConsecutiveFailures  int                 `json:"consecutive_failures"`
	LastSuccessAt        *time.Time          `json:"last_success_at"`
	LastFailureAt        *time.Time          `json:"last_failure_at"`
	SuccessRate          float64             `json:"success_rate"`
	AvgResponseTime      int                 `json:"avg_response_time"` // milliseconds
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`

	// Error category breakdown
	ClientErrors           int `json:"client_errors"`            // 4xx responses
	ServerErrors           int `json:"server_errors"`            // 5xx responses
	TimeoutErrors          int `json:"timeout_errors"`           // Timeouts
	NetworkErrors          int `json:"network_errors"`           // DNS, TLS, connection refused, and other network errors
	UnexpectedStatusErrors int `json:"unexpected_status_errors"` // 2xx/3xx not in expected_status_codes
}

// HealthSummaryData represents health summary information
type HealthSummaryData struct {
	HealthyCount   int `json:"healthy_count"`
	DegradedCount  int `json:"degraded_count"`
	UnhealthyCount int `json:"unhealthy_count"`
	UnknownCount   int `json:"unknown_count"`
	TotalCount     int `json:"total_count"`
}

// NamespaceStatsData represents namespace statistics
type NamespaceStatsData struct {
	TotalWebhooks        int     `json:"total_webhooks"`
	ActiveWebhooks       int     `json:"active_webhooks"`
	TotalDeliveries      int     `json:"total_deliveries"`
	SuccessfulDeliveries int     `json:"successful_deliveries"`
	FailedDeliveries     int     `json:"failed_deliveries"`
	PendingDeliveries    int     `json:"pending_deliveries"`
	SuccessRate          float64 `json:"success_rate"`
}

// generateSamplePayload generates a sample payload from the given schema using schemagen
func generateSamplePayload(schema map[string]any) (map[string]any, error) {
	if len(schema) == 0 {
		return map[string]any{}, nil
	}

	// Convert schema to JSON bytes
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Create generator and generate sample
	generator := schemagen.NewGenerator().SetGenerateAllFields(true)
	sample, err := generator.Generate(schemaBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sample payload: %w", err)
	}

	// Convert to map[string]any
	if sampleMap, ok := sample.(map[string]any); ok {
		return sampleMap, nil
	}

	return map[string]any{}, nil
}

// RegisterEvent registers a new event type
func (s *WebhookService) RegisterEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) (string, time.Time, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RegisterEvent")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing event registration request", "name", name, "description", description)
	if name == "" {
		return "", time.Time{}, fmt.Errorf("event name is required")
	}

	tenantID := tenant.DefaultTenantID

	// Event types are tenant-scoped (shared across namespaces)

	existingEvent, err := s.webhookRepo.GetEventByName(ctx, tenantID, name)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to check existing event", "error", err)
		return "", time.Time{}, fmt.Errorf("failed to check existing event: %w", err)
	}
	if existingEvent != nil {
		return "", time.Time{}, fmt.Errorf("event already exists")
	}

	// Generate sample payload from schema
	samplePayload, err := generateSamplePayload(schema)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to generate sample payload, using empty payload", "error", err)
		samplePayload = map[string]any{}
	}

	event := &store.EventRegistration{
		Name:          name,
		Description:   description,
		Schema:        schema,
		SamplePayload: samplePayload,
		Metadata:      metadata,
		Active:        active,
	}
	if err := s.webhookRepo.RegisterEvent(ctx, tenantID, event); err != nil {
		s.logger.ErrorContext(ctx, "Failed to register event",
			"name", name,
			"error", err,
		)
		return "", time.Time{}, fmt.Errorf("failed to register event: %w", err)
	}
	s.logger.InfoContext(ctx, "Event registered successfully",
		"name", name,
		"description", description,
	)
	return event.Name, event.CreatedAt, nil
}

// ListEvents lists all registered events
func (s *WebhookService) ListEvents(ctx context.Context, activeOnly bool, limit, offset int32) ([]*store.EventRegistration, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEvents")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing list events request",
		"active_only", activeOnly, "limit", limit, "offset", offset)

	tenantID := tenant.DefaultTenantID

	// Event types are tenant-scoped

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	events, totalCount, err := s.webhookRepo.ListEventsPaginated(ctx, tenantID, activeOnly, int(limit), int(offset))
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list events", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve events: %w", err)
	}
	s.logger.InfoContext(ctx, "Listed events successfully",
		"count", len(events),
		"total", totalCount,
	)
	return events, int32(totalCount), nil
}

// UpdateEvent updates an event registration
func (s *WebhookService) UpdateEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.UpdateEvent")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing event update request",
		"name", name,
		"description", description)

	// Validate required fields
	if name == "" {
		return fmt.Errorf("event name is required")
	}

	tenantID := tenant.DefaultTenantID

	// Event types are tenant-scoped

	// Check if event exists
	existingEvent, err := s.webhookRepo.GetEventByName(ctx, tenantID, name)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get event", "error", err)
		return fmt.Errorf("failed to retrieve event: %w", err)
	}

	if existingEvent == nil {
		return fmt.Errorf("event not found")
	}

	// Update event fields
	existingEvent.Description = description
	existingEvent.Schema = schema
	existingEvent.Metadata = metadata
	existingEvent.Active = active

	// Generate sample payload from schema
	samplePayload, err := generateSamplePayload(schema)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to generate sample payload, using empty payload", "error", err)
		samplePayload = map[string]any{}
	}
	existingEvent.SamplePayload = samplePayload

	// Update the event
	if err := s.webhookRepo.UpdateEvent(ctx, tenantID, existingEvent); err != nil {
		s.logger.ErrorContext(ctx, "Failed to update event",
			"name", name,
			"error", err,
		)
		return fmt.Errorf("failed to update event: %w", err)
	}

	s.logger.InfoContext(ctx, "Event updated successfully", "name", name)
	return nil
}

// DeleteEvent deletes an event registration
func (s *WebhookService) DeleteEvent(ctx context.Context, name string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.DeleteEvent")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing event deletion request", "name", name)

	// Validate required fields
	if name == "" {
		return fmt.Errorf("event name is required")
	}

	tenantID := tenant.DefaultTenantID

	// Event types are tenant-scoped

	// Check if event exists
	existingEvent, err := s.webhookRepo.GetEventByName(ctx, tenantID, name)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get event", "error", err)
		return fmt.Errorf("failed to retrieve event: %w", err)
	}

	if existingEvent == nil {
		return fmt.Errorf("event not found")
	}

	// Delete the event
	if err := s.webhookRepo.DeleteEvent(ctx, tenantID, name); err != nil {
		s.logger.ErrorContext(ctx, "Failed to delete event",
			"name", name,
			"error", err,
		)
		return fmt.Errorf("failed to delete event: %w", err)
	}

	s.logger.InfoContext(ctx, "Event deleted successfully", "name", name)
	return nil
}

// GetEvent retrieves an event registration by name
func (s *WebhookService) GetEvent(ctx context.Context, name string) (*store.EventRegistration, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetEvent")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing get event request", "name", name)
	if name == "" {
		return nil, fmt.Errorf("event name is required")
	}

	tenantID := tenant.DefaultTenantID

	// Event types are tenant-scoped

	event, err := s.webhookRepo.GetEventByName(ctx, tenantID, name)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get event", "error", err)
		return nil, fmt.Errorf("failed to retrieve event: %w", err)
	}

	return event, nil
}

// GetWebhookHealth retrieves health metrics for a webhook
func (s *WebhookService) GetWebhookHealth(ctx context.Context, webhookID string, namespace string) (*WebhookHealthData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetWebhookHealth")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing get webhook health request",
		"webhook_id", webhookID,
		"namespace", namespace)

	// Validate required fields
	if webhookID == "" {
		return nil, fmt.Errorf("webhook ID is required")
	}

	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook ID: %w", err)
	}

	// Get webhook to verify it exists and get current health
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, id, namespace)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhook")
		s.logger.ErrorContext(ctx, "Failed to get webhook", "error", err)
		return nil, fmt.Errorf("webhook not found: %w", err)
	}

	if webhook == nil {
		span.SetStatus(otelcodes.Error, "Webhook not found")
		s.logger.ErrorContext(ctx, "Webhook not found", "webhook_id", webhookID)
		return nil, fmt.Errorf("webhook not found")
	}

	// Get health state (current status and consecutive failures)
	healthState, err := s.webhookRepo.GetWebhookHealthState(ctx, id)
	if err != nil {
		// If no health state exists yet, return basic health info
		s.logger.InfoContext(ctx, "No health state found for webhook", "webhook_id", webhookID)
		return &WebhookHealthData{
			WebhookID: webhookID,
			Health:    webhook.Health,
		}, nil
	}

	// Get health summary for the last 24 hours
	healthSummary, err := s.webhookRepo.GetWebhookHealthSummary(ctx, id, 24)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get health summary", "error", err)
		// Continue with just the state info
	}

	// Convert to response format
	healthData := &WebhookHealthData{
		WebhookID:           webhookID,
		Health:              webhook.Health,
		ConsecutiveFailures: healthState.ConsecutiveFailures,
		LastSuccessAt:       healthState.LastSuccessAt,
		LastFailureAt:       healthState.LastFailureAt,
		CreatedAt:           healthState.CreatedAt,
		UpdatedAt:           healthState.UpdatedAt,
	}

	// Add summary data if available
	if healthSummary != nil {
		healthData.TotalDeliveries = healthSummary.TotalDeliveries
		healthData.SuccessfulDeliveries = healthSummary.SuccessfulDeliveries
		healthData.FailedDeliveries = healthSummary.FailedDeliveries
		healthData.SuccessRate = healthSummary.SuccessRate
		healthData.AvgResponseTime = healthSummary.AvgResponseTime
		healthData.ClientErrors = healthSummary.ClientErrors
		healthData.ServerErrors = healthSummary.ServerErrors
		healthData.TimeoutErrors = healthSummary.TimeoutErrors
		healthData.NetworkErrors = healthSummary.NetworkErrors
		healthData.UnexpectedStatusErrors = healthSummary.UnexpectedStatusErrors
	}

	s.logger.InfoContext(ctx, "Webhook health retrieved successfully",
		"webhook_id", webhookID,
		"health", webhook.Health,
		"success_rate", healthData.SuccessRate)

	return healthData, nil
}

// ListWebhooksByHealth retrieves webhooks filtered by health status
func (s *WebhookService) ListWebhooksByHealth(ctx context.Context, health store.WebhookHealth, limit, offset int32) ([]*store.WebhookRegistration, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListWebhooksByHealth")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing list webhooks by health request", "health", health, "limit", limit, "offset", offset)

	tenantID := tenant.DefaultTenantID

	// This is a cross-namespace query — only tenant-level roles can do this

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Get webhooks by health status
	webhooksList, totalCount, err := s.webhookRepo.GetWebhooksByHealthPaginated(ctx, tenantID, health, int(limit), int(offset))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhooks by health")
		s.logger.ErrorContext(ctx, "Failed to get webhooks by health", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve webhooks: %w", err)
	}

	s.logger.InfoContext(ctx, "Webhooks retrieved successfully",
		"health", health,
		"count", len(webhooksList),
		"total", totalCount)

	return webhooksList, int32(totalCount), nil
}

// GetHealthSummary retrieves a summary of webhook health across all namespaces
func (s *WebhookService) GetHealthSummary(ctx context.Context) (*HealthSummaryData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetHealthSummary")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing get health summary request")

	tenantID := tenant.DefaultTenantID

	// Health summary is a cross-namespace query — only tenant-level roles can do this

	// Get health summary from repository
	summary, err := s.webhookRepo.GetHealthSummary(ctx, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get health summary")
		s.logger.ErrorContext(ctx, "Failed to get health summary", "error", err)
		return nil, fmt.Errorf("failed to retrieve health summary: %w", err)
	}

	// Convert to response format
	healthSummary := &HealthSummaryData{
		HealthyCount:   summary[store.HealthHealthy],
		DegradedCount:  summary[store.HealthDegraded],
		UnhealthyCount: summary[store.HealthUnhealthy],
		UnknownCount:   summary[store.HealthUnknown],
	}

	// Calculate total
	healthSummary.TotalCount = healthSummary.HealthyCount + healthSummary.DegradedCount +
		healthSummary.UnhealthyCount + healthSummary.UnknownCount

	s.logger.InfoContext(ctx, "Health summary retrieved successfully",
		"healthy", healthSummary.HealthyCount,
		"degraded", healthSummary.DegradedCount,
		"unhealthy", healthSummary.UnhealthyCount,
		"unknown", healthSummary.UnknownCount,
		"total", healthSummary.TotalCount)

	return healthSummary, nil
}

// GetNamespaceStats retrieves statistics for a namespace, or across all namespaces if empty
func (s *WebhookService) GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStatsData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetNamespaceStats")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing get namespace stats request", "namespace", namespace)

	tenantID := tenant.DefaultTenantID

	stats, err := s.webhookRepo.GetNamespaceStats(ctx, tenantID, namespace)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get namespace stats", "error", err)
		return nil, err
	}

	res := &NamespaceStatsData{
		TotalWebhooks:        stats.TotalWebhooks,
		ActiveWebhooks:       stats.ActiveWebhooks,
		TotalDeliveries:      stats.TotalDeliveries,
		SuccessfulDeliveries: stats.SuccessfulDeliveries,
		FailedDeliveries:     stats.FailedDeliveries,
		PendingDeliveries:    stats.PendingDeliveries,
		SuccessRate:          stats.SuccessRate,
	}

	s.logger.InfoContext(ctx, "Namespace stats retrieved successfully",
		"namespace", namespace,
		"total_webhooks", res.TotalWebhooks,
		"active_webhooks", res.ActiveWebhooks,
		"success_rate", res.SuccessRate)
	return res, nil
}

// UpdateWebhookConfig updates webhook configuration.
// When updateMask is non-empty, only the listed field paths are applied.
// When updateMask is empty, falls back to legacy behavior (all non-zero fields applied).
//
// Supported mask paths:
//
//	"url", "active", "description", "events", "headers",
//	"secret_headers", "http_config", "http_config.webhook_secret"
func (s *WebhookService) UpdateWebhookConfig(ctx context.Context, webhookID string, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string, httpConfig *HTTPConfigUpdate, secretHeaders map[string]string, updateMask []string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.UpdateWebhookConfig")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing update webhook config request",
		"webhook_id", webhookID,
		"namespace", namespace,
		"update_mask", updateMask)

	if webhookID == "" {
		return fmt.Errorf("webhook ID is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	webhookUUID, err := uuid.Parse(webhookID)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %w", err)
	}

	webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, webhookUUID, namespace)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get webhook", "error", err)
		return fmt.Errorf("failed to retrieve webhook: %w", err)
	}
	if webhook == nil {
		return fmt.Errorf("webhook not found")
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

	// Update subscriptions if events are provided
	if shouldUpdate("events") && len(events) > 0 {
		var newSubs []*store.EventSubscription
		for _, event := range events {
			newSubs = append(newSubs, &store.EventSubscription{
				EventName: event,
			})
		}
		if err := s.webhookRepo.ReplaceWebhookSubscriptions(ctx, tenantID, webhookUUID, namespace, newSubs); err != nil {
			s.logger.ErrorContext(ctx, "Failed to replace webhook subscriptions",
				"webhook_id", webhookID,
				"error", err)
			return fmt.Errorf("failed to update webhook subscriptions: %w", err)
		}
	}
	if shouldUpdate("url") && url != "" {
		normalizedURL := strings.TrimSpace(url)
		if normalizedURL == "" {
			return fmt.Errorf("URL is required")
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
	// Legacy timeout (deprecated, but still supported)
	if !useMask && timeout > 0 {
		webhook.Timeout = timeout
	}
	// Apply HTTP config updates if provided
	if shouldUpdate("http_config") && httpConfig != nil {
		if httpConfig.MaxRetries > 0 {
			webhook.MaxRetries = httpConfig.MaxRetries
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
		webhook.CaptureResponseBody = httpConfig.CaptureResponseBody
		webhook.FollowRedirects = httpConfig.FollowRedirects
		webhook.VerifySSL = httpConfig.VerifySSL
	}
	// Encrypt and set secret headers if in mask (or legacy non-empty)
	if shouldUpdate("secret_headers") && len(secretHeaders) > 0 {
		encrypted, err := s.EncryptSecretHeaders(secretHeaders)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret headers: %w", err)
		}
		webhook.SecretHeaders = encrypted
	}
	err = s.webhookRepo.UpdateWebhook(ctx, tenantID, webhook)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update webhook config",
			"webhook_id", webhookID,
			"error", err)
		return fmt.Errorf("failed to update webhook configuration: %w", err)
	}
	s.logger.InfoContext(ctx, "Webhook configuration updated successfully",
		"webhook_id", webhookID)
	return nil
}

// SchemaValidationError represents a structured schema validation failure
// with per-field error details that can be displayed to the user.
type SchemaValidationError struct {
	Message string            `json:"message"`
	Details map[string]string `json:"details"` // field path -> error message
}

func (e *SchemaValidationError) Error() string {
	if len(e.Details) == 0 {
		return e.Message
	}
	var parts []string
	for field, msg := range e.Details {
		if field == "" {
			parts = append(parts, msg)
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
		}
	}
	return fmt.Sprintf("%s: %s", e.Message, strings.Join(parts, "; "))
}

// Warnings returns per-field validation messages suitable for API responses.
// Each warning is a human-readable string describing a specific validation failure.
func (e *SchemaValidationError) Warnings() []string {
	var warnings []string
	for field, msg := range e.Details {
		if field == "" {
			warnings = append(warnings, msg)
		} else {
			warnings = append(warnings, fmt.Sprintf("field '%s': %s", field, msg))
		}
	}
	if len(warnings) == 0 {
		warnings = append(warnings, e.Message)
	}
	return warnings
}

// ValidateJSONSchema validates a payload against a JSON schema string.
// Returns a SchemaValidationError with detailed per-field errors on failure.
func ValidateJSONSchema(schema map[string]any, payload map[string]any) error {
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	sch, err := compiler.Compile(schemaJSON)
	if err != nil {
		return fmt.Errorf("invalid event schema: %w", err)
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	result := sch.ValidateJSON(payloadBytes)
	if result == nil || result.Valid {
		return nil
	}

	// Extract detailed per-field errors from the evaluation result
	details := result.GetDetailedErrors()

	return &SchemaValidationError{
		Message: "payload validation failed",
		Details: details,
	}
}

// ListEventReports lists event records with delivery statistics in descending order by creation time.
// Supports filtering by namespace, event name, schema_valid, labels, and time range.
// When PrepareRepush is true, snapshots all matching event IDs into a batch job and returns the batch ID.
func (s *WebhookService) ListEventReports(ctx context.Context, filter store.EventReportFilter) ([]*store.EventReportWithStats, int32, string, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEventReports")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing list event reports request",
		"namespace", filter.Namespace,
		"event_name", filter.EventName,
		"prepare_repush", filter.PrepareRepush,
		"limit", filter.Limit,
		"offset", filter.Offset)

	tenantID := tenant.DefaultTenantID

	// Set default limit if not provided or out of range
	if filter.Limit <= 0 {
		filter.Limit = 50
	} else if filter.Limit > 1000 {
		filter.Limit = 1000
	}

	// Ensure offset is not negative
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	events, totalCount, err := s.webhookRepo.ListEventReportsFiltered(ctx, tenantID, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list event reports", "namespace", filter.Namespace, "event_name", filter.EventName, "error", err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, 0, "", fmt.Errorf("failed to list event reports: %w", err)
	}

	// Snapshot matching IDs into a batch job if requested
	var repushID string
	if filter.PrepareRepush {
		ids, err := s.webhookRepo.SnapshotEventIDs(ctx, tenantID, filter)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to snapshot event IDs for repush", "error", err)
			return nil, 0, "", fmt.Errorf("failed to prepare repush: %w", err)
		}
		if len(ids) > 0 {
			filterMap := map[string]any{
				"namespace": filter.Namespace,
			}
			if filter.EventName != nil {
				filterMap["event_name"] = *filter.EventName
			}
			if filter.SchemaValid != nil {
				filterMap["schema_valid"] = *filter.SchemaValid
			}
			if len(filter.Labels) > 0 {
				filterMap["labels"] = filter.Labels
			}
			batchData := &store.BatchJobData{
				ItemIDs: ids,
				Filter:  filterMap,
			}
			batchJob, err := s.webhookRepo.CreateBatchJob(ctx, tenantID, filter.Namespace, store.BatchTypeEventRepush, batchData)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to create batch job for repush", "error", err)
				return nil, 0, "", fmt.Errorf("failed to create repush batch: %w", err)
			}
			repushID = batchJob.ID.String()
			s.logger.InfoContext(ctx, "Created repush batch job",
				"repush_id", repushID,
				"event_count", len(ids))
		}
	}

	s.logger.InfoContext(ctx, "Successfully listed event reports",
		"namespace", filter.Namespace,
		"event_name", filter.EventName,
		"count", len(events),
		"total", totalCount)

	span.SetAttributes(
		attribute.String("namespace", filter.Namespace),
		attribute.Int("count", len(events)),
		attribute.Int("total", totalCount),
	)
	if filter.EventName != nil {
		span.SetAttributes(attribute.String("event_name", *filter.EventName))
	}

	return events, int32(totalCount), repushID, nil
}

// Subscription Management Implementation

func (s *WebhookService) CreateSubscription(ctx context.Context, webhookID, eventName, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string) (string, time.Time, error) {
	s.logger.InfoContext(ctx, "Creating subscription", "webhook_id", webhookID, "event_name", eventName, "namespace", namespace)

	if namespace == "" {
		return "", time.Time{}, fmt.Errorf("namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid webhook ID: %w", err)
	}

	if err := validateLabels(labelFilters, "label_filters"); err != nil {
		return "", time.Time{}, err
	}

	sub := &store.EventSubscription{
		WebhookID:         id,
		EventName:         eventName,
		Namespace:         namespace,
		Headers:           headers,
		Method:            method,
		Timeout:           timeout,
		TransformEnabled:  transformEnabled,
		TransformTemplate: transformTemplate,
		LabelFilters:      labelFilters,
	}

	if err := s.webhookRepo.CreateSubscription(ctx, tenantID, sub); err != nil {
		return "", time.Time{}, err
	}

	return sub.ID.String(), sub.CreatedAt, nil
}

func (s *WebhookService) GetSubscription(ctx context.Context, subscriptionID string, namespace string) (*store.EventSubscription, error) {
	tenantID := tenant.DefaultTenantID

	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	id, err := uuid.Parse(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription ID: %w", err)
	}

	sub, err := s.webhookRepo.GetSubscription(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if sub != nil && sub.Namespace != namespace {
		return nil, fmt.Errorf("subscription not found in namespace")
	}
	return sub, nil
}

func (s *WebhookService) ListSubscriptions(ctx context.Context, namespace string, webhookID string, eventName string, limit, offset int32) ([]*store.EventSubscription, int32, error) {
	if namespace == "" {
		return nil, 0, fmt.Errorf("namespace is required")
	}

	tenantID := tenant.DefaultTenantID

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var subs []*store.EventSubscription
	var totalCount int
	var err error

	if webhookID != "" {
		var id uuid.UUID
		id, err = uuid.Parse(webhookID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid webhook ID: %w", err)
		}
		subs, err = s.webhookRepo.ListSubscriptions(ctx, tenantID, id)
		totalCount = len(subs)
		// Apply pagination manually for now if repo doesn't support it for ListSubscriptions
		if int(offset) < len(subs) {
			end := int(offset + limit)
			if end > len(subs) {
				end = len(subs)
			}
			subs = subs[int(offset):end]
		} else {
			subs = []*store.EventSubscription{}
		}
	} else if eventName != "" {
		subs, err = s.webhookRepo.GetSubscriptionsByEvent(ctx, tenantID, namespace, eventName, nil)
		totalCount = len(subs)
		if int(offset) < len(subs) {
			end := int(offset + limit)
			if end > len(subs) {
				end = len(subs)
			}
			subs = subs[int(offset):end]
		} else {
			subs = []*store.EventSubscription{}
		}
	} else {
		// List all subscriptions in namespace
		subs, totalCount, err = s.webhookRepo.ListSubscriptionsByNamespace(ctx, tenantID, namespace, int(limit), int(offset))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list subscriptions: %w", err)
		}
	}

	return subs, int32(totalCount), err
}

func (s *WebhookService) UpdateSubscription(ctx context.Context, subscriptionID string, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string) error {
	tenantID := tenant.DefaultTenantID

	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	id, err := uuid.Parse(subscriptionID)
	if err != nil {
		return fmt.Errorf("invalid subscription ID: %w", err)
	}

	if err := validateLabels(labelFilters, "label_filters"); err != nil {
		return err
	}

	sub, err := s.webhookRepo.GetSubscription(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if sub == nil || sub.Namespace != namespace {
		return fmt.Errorf("subscription not found in namespace")
	}

	sub.Headers = headers
	sub.Method = method
	sub.Timeout = timeout
	sub.TransformEnabled = transformEnabled
	sub.TransformTemplate = transformTemplate
	sub.LabelFilters = labelFilters

	return s.webhookRepo.UpdateSubscription(ctx, tenantID, sub)
}

func (s *WebhookService) DeleteSubscription(ctx context.Context, subscriptionID string, namespace string) error {
	tenantID := tenant.DefaultTenantID

	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	id, err := uuid.Parse(subscriptionID)
	if err != nil {
		return fmt.Errorf("invalid subscription ID: %w", err)
	}

	sub, err := s.webhookRepo.GetSubscription(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if sub == nil || sub.Namespace != namespace {
		return fmt.Errorf("subscription not found in namespace")
	}

	return s.webhookRepo.DeleteSubscription(ctx, tenantID, sub.ID)
}

func (s *WebhookService) TestSubscriptionTemplate(ctx context.Context, eventName, transformTemplate, namespace string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.TestSubscriptionTemplate")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing test subscription template request", "event_name", eventName, "namespace", namespace)

	if eventName == "" {
		return "", fmt.Errorf("event name is required")
	}

	tenantID := tenant.DefaultTenantID

	event, err := s.webhookRepo.GetEventByName(ctx, tenantID, eventName)
	if err != nil {
		return "", fmt.Errorf("failed to get event: %w", err)
	}
	if event == nil {
		return "", fmt.Errorf("event not found")
	}

	engine := client.NewTemplateEngine()

	// Create context for template
	data := client.NewWebhookTemplateContext(
		"dry-run-event-id",
		eventName,
		time.Now().UTC().Format(time.RFC3339),
		1,
		event.SamplePayload,
	)

	result, err := engine.TransformPayload(transformTemplate, data)
	if err != nil {
		return "", svcerrors.Wrapf(err, codes.InvalidArgument, "template transformation failed: %v", err)
	}

	return string(result), nil
}

func (s *WebhookService) GetTemplateFunctions() []TemplateFunctionInfo {
	functions := client.GetTemplateFunctions()
	res := make([]TemplateFunctionInfo, len(functions))
	for i, f := range functions {
		res[i] = TemplateFunctionInfo{
			Name:        f.Name,
			Description: f.Description,
		}
	}
	return res
}

// --- Batch Operations ---

// RePushEvents starts async processing of a batch re-push.
// The batch must exist, belong to the tenant, be of type event_repush, and be in pending status.
func (s *WebhookService) RePushEvents(ctx context.Context, repushID string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RePushEvents")
	defer span.End()

	tenantID := tenant.DefaultTenantID

	batchUUID, err := uuid.Parse(repushID)
	if err != nil {
		return svcerrors.InvalidInputf("invalid repush_id: %v", err)
	}

	batch, err := s.webhookRepo.GetBatchJob(ctx, tenantID, batchUUID)
	if err != nil {
		return fmt.Errorf("failed to get batch job: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch job not found")
	}
	if batch.JobType != store.BatchTypeEventRepush {
		return svcerrors.FailedPrecondition("batch job is not an event repush")
	}
	if batch.Status != store.BatchStatusPending {
		return svcerrors.FailedPreconditionf("batch job is not in pending status (current: %s)", batch.Status)
	}
	if time.Now().After(batch.ExpiresAt) {
		return svcerrors.FailedPrecondition("batch job has expired")
	}

	// Transition to processing and enqueue the River job
	if err := s.webhookRepo.UpdateBatchJobStatus(ctx, batchUUID, store.BatchStatusProcessing); err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	_, err = s.jobInserter.Insert(ctx, &queue.BatchJobArgs{
		TenantID: tenantID.String(),
		BatchID:  repushID,
	})
	if err != nil {
		// Roll back status on enqueue failure
		_ = s.webhookRepo.UpdateBatchJobStatus(ctx, batchUUID, store.BatchStatusPending)
		return fmt.Errorf("failed to enqueue batch job: %w", err)
	}

	s.logger.InfoContext(ctx, "Batch re-push enqueued", "repush_id", repushID, "total", batch.Total)
	return nil
}

// GetRepushStatus returns the current state of a batch re-push.
func (s *WebhookService) GetRepushStatus(ctx context.Context, repushID string) (*store.BatchJob, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetRepushStatus")
	defer span.End()

	tenantID := tenant.DefaultTenantID

	batchUUID, err := uuid.Parse(repushID)
	if err != nil {
		return nil, svcerrors.InvalidInputf("invalid repush_id: %v", err)
	}

	batch, err := s.webhookRepo.GetBatchJob(ctx, tenantID, batchUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch job: %w", err)
	}
	if batch == nil {
		return nil, fmt.Errorf("batch job not found")
	}
	if batch.JobType != store.BatchTypeEventRepush {
		return nil, svcerrors.FailedPrecondition("batch job is not an event repush")
	}
	return batch, nil
}

// CancelRepush aborts a pending or in-progress batch re-push.
func (s *WebhookService) CancelRepush(ctx context.Context, repushID string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.CancelRepush")
	defer span.End()

	tenantID := tenant.DefaultTenantID

	batchUUID, err := uuid.Parse(repushID)
	if err != nil {
		return svcerrors.InvalidInputf("invalid repush_id: %v", err)
	}

	batch, err := s.webhookRepo.GetBatchJob(ctx, tenantID, batchUUID)
	if err != nil {
		return fmt.Errorf("failed to get batch job: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch job not found")
	}
	if batch.JobType != store.BatchTypeEventRepush {
		return svcerrors.FailedPrecondition("batch job is not an event repush")
	}
	if batch.Status == store.BatchStatusCompleted || batch.Status == store.BatchStatusCancelled {
		return svcerrors.FailedPreconditionf("batch job is already in terminal state: %s", batch.Status)
	}

	if err := s.webhookRepo.UpdateBatchJobStatus(ctx, batchUUID, store.BatchStatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel batch job: %w", err)
	}

	s.logger.InfoContext(ctx, "Batch re-push cancelled", "repush_id", repushID)
	return nil
}

// RetryDeliveries starts async processing of a batch delivery retry.
func (s *WebhookService) RetryDeliveries(ctx context.Context, retryID string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RetryDeliveries")
	defer span.End()

	tenantID := tenant.DefaultTenantID

	batchUUID, err := uuid.Parse(retryID)
	if err != nil {
		return svcerrors.InvalidInputf("invalid retry_id: %v", err)
	}

	batch, err := s.webhookRepo.GetBatchJob(ctx, tenantID, batchUUID)
	if err != nil {
		return fmt.Errorf("failed to get batch job: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch job not found")
	}
	if batch.JobType != store.BatchTypeDeliveryRetry {
		return svcerrors.FailedPrecondition("batch job is not a delivery retry")
	}
	if batch.Status != store.BatchStatusPending {
		return svcerrors.FailedPreconditionf("batch job is not in pending status (current: %s)", batch.Status)
	}
	if time.Now().After(batch.ExpiresAt) {
		return svcerrors.FailedPrecondition("batch job has expired")
	}

	if err := s.webhookRepo.UpdateBatchJobStatus(ctx, batchUUID, store.BatchStatusProcessing); err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	_, err = s.jobInserter.Insert(ctx, &queue.BatchJobArgs{
		TenantID: tenantID.String(),
		BatchID:  retryID,
	})
	if err != nil {
		_ = s.webhookRepo.UpdateBatchJobStatus(ctx, batchUUID, store.BatchStatusPending)
		return fmt.Errorf("failed to enqueue batch job: %w", err)
	}

	s.logger.InfoContext(ctx, "Batch delivery retry enqueued", "retry_id", retryID, "total", batch.Total)
	return nil
}

// GetRetryStatus returns the current state of a batch delivery retry.
func (s *WebhookService) GetRetryStatus(ctx context.Context, retryID string) (*store.BatchJob, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetRetryStatus")
	defer span.End()

	tenantID := tenant.DefaultTenantID

	batchUUID, err := uuid.Parse(retryID)
	if err != nil {
		return nil, svcerrors.InvalidInputf("invalid retry_id: %v", err)
	}

	batch, err := s.webhookRepo.GetBatchJob(ctx, tenantID, batchUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch job: %w", err)
	}
	if batch == nil {
		return nil, fmt.Errorf("batch job not found")
	}
	if batch.JobType != store.BatchTypeDeliveryRetry {
		return nil, svcerrors.FailedPrecondition("batch job is not a delivery retry")
	}
	return batch, nil
}

// CancelRetry aborts a pending or in-progress batch delivery retry.
func (s *WebhookService) CancelRetry(ctx context.Context, retryID string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.CancelRetry")
	defer span.End()

	tenantID := tenant.DefaultTenantID

	batchUUID, err := uuid.Parse(retryID)
	if err != nil {
		return svcerrors.InvalidInputf("invalid retry_id: %v", err)
	}

	batch, err := s.webhookRepo.GetBatchJob(ctx, tenantID, batchUUID)
	if err != nil {
		return fmt.Errorf("failed to get batch job: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch job not found")
	}
	if batch.JobType != store.BatchTypeDeliveryRetry {
		return svcerrors.FailedPrecondition("batch job is not a delivery retry")
	}
	if batch.Status == store.BatchStatusCompleted || batch.Status == store.BatchStatusCancelled {
		return svcerrors.FailedPreconditionf("batch job is already in terminal state: %s", batch.Status)
	}

	if err := s.webhookRepo.UpdateBatchJobStatus(ctx, batchUUID, store.BatchStatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel batch job: %w", err)
	}

	s.logger.InfoContext(ctx, "Batch delivery retry cancelled", "retry_id", retryID)
	return nil
}
