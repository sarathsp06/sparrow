package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

	"github.com/sarathsp06/sparrow/internal/auth"
	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type WebhookService struct {
	jobInserter queue.JobInserter
	webhookRepo store.RepositoryInterface
	logger      *slog.Logger
	tracer      trace.Tracer
	metrics     *observability.SparrowMetrics
}

//go:generate gowrap gen -i WebhookServiceInterface -t ../../templates/opentelemetry.tmpl -o WebhookServiceInterface_otel.go
type WebhookServiceInterface interface {
	// Webhook Management
	RegisterWebhook(ctx context.Context, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string) (string, time.Time, error)
	CreateWebhook(ctx context.Context, req WebhookRegistrationRequest) (*WebhookRegistration, error)
	UnregisterWebhook(ctx context.Context, webhookID string, namespace string) error
	ListWebhooks(ctx context.Context, namespace string, webhookID string, event string, activeOnly bool, limit, offset int32) ([]*store.WebhookRegistration, int32, error)
	UpdateWebhookConfig(ctx context.Context, webhookID string, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string, httpConfig *HTTPConfigUpdate) error
	PauseWebhook(ctx context.Context, webhookID string, namespace string, reason string) error
	ResumeWebhook(ctx context.Context, webhookID string, namespace string) error
	GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStatsData, error)

	// Event Management
	RegisterEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) (string, time.Time, error)
	ListEvents(ctx context.Context, activeOnly bool, limit, offset int32) ([]*store.EventRegistration, int32, error)
	UpdateEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) error
	DeleteEvent(ctx context.Context, name string) error
	GetEvent(ctx context.Context, name string) (*store.EventRegistration, error)
	PushEvent(ctx context.Context, namespace string, event string, payload map[string]any, ttlSeconds int64, metadata map[string]string) (string, error)
	ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int32) ([]*store.EventReportWithStats, int32, error)

	// Subscription Management
	CreateSubscription(ctx context.Context, webhookID, eventName, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string) (string, time.Time, error)
	GetSubscription(ctx context.Context, subscriptionID string, namespace string) (*store.EventSubscription, error)
	ListSubscriptions(ctx context.Context, namespace string, webhookID string, eventName string, limit, offset int32) ([]*store.EventSubscription, int32, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string) error
	DeleteSubscription(ctx context.Context, subscriptionID string, namespace string) error
	TestSubscriptionTemplate(ctx context.Context, eventName, transformTemplate, namespace string) (string, error)

	// Delivery Management
	GetDeliveryStatus(ctx context.Context, deliveryID string, namespace string) (*store.WebhookDelivery, error)
	GetDeliveryAttempts(ctx context.Context, deliveryID string) ([]*store.WebhookHealthEvent, error)
	ListDeliveries(ctx context.Context, namespace string, webhookID string, eventID string, limit, offset int32) ([]*store.WebhookDelivery, int32, error)
	RetryDelivery(ctx context.Context, namespace string, deliveryID string, webhookID string, force bool) ([]string, int32, error)

	// Health Management
	GetWebhookHealth(ctx context.Context, webhookID string, namespace string) (*WebhookHealthData, error)
	ListWebhooksByHealth(ctx context.Context, health store.WebhookHealth, limit, offset int32) ([]*store.WebhookRegistration, int32, error)
	GetHealthSummary(ctx context.Context) (*HealthSummaryData, error)

	// Metadata
	GetTemplateFunctions() []TemplateFunctionInfo

	// Repository access
	GetWebhookRepo() store.RepositoryInterface
}

type TemplateFunctionInfo struct {
	Name        string
	Description string
}

var _ WebhookServiceInterface = (*WebhookService)(nil)

// NewWebhookService creates a new WebhookService instance
func NewWebhookService(queueManager queue.JobInserter, webhookRepo store.RepositoryInterface) *WebhookService {
	metrics, err := observability.NewSparrowMetrics()
	if err != nil {
		// Log error but continue without metrics
		log := logger.NewLogger("webhook-service")
		log.Error("Failed to initialize metrics", "error", err)
	}

	return &WebhookService{
		jobInserter: queueManager,
		webhookRepo: webhookRepo,
		logger:      logger.NewLogger("webhook-service"),
		tracer:      observability.GetTracer("sparrow.service.webhook"),
		metrics:     metrics,
	}
}

// GetWebhookRepo returns the repository interface for direct access
func (s *WebhookService) GetWebhookRepo() store.RepositoryInterface {
	return s.webhookRepo
}

func (s *WebhookService) RegisterWebhook(ctx context.Context, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string) (string, time.Time, error) {
	ctx, span := s.tracer.Start(ctx, "webhook.register",
		trace.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.StringSlice("events", events),
			attribute.String("url", url),
		),
	)
	defer span.End()

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	s.logger.InfoContext(ctx, "Processing webhook registration request",
		"namespace", namespace,
		"events", events,
		"url", url,
	)

	if namespace == "" {
		return "", time.Time{}, fmt.Errorf("namespace is required")
	}
	if err := authInfo.Require(auth.PermWebhookCreate, namespace); err != nil {
		return "", time.Time{}, err
	}
	if url == "" {
		return "", time.Time{}, fmt.Errorf("URL is required")
	}
	if err := ValidateWebhookURL(url); err != nil {
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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if req.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if err := authInfo.Require(auth.PermWebhookCreate, req.Namespace); err != nil {
		return nil, err
	}

	// Validate webhook URL against SSRF
	if err := ValidateWebhookURL(req.URL); err != nil {
		return nil, err
	}

	// Convert request to internal webhook registration
	webhookReg, err := req.ToWebhookRegistration()
	if err != nil {
		return nil, fmt.Errorf("failed to convert webhook registration request: %w", err)
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
		WebhookSecret:         webhookReg.HTTPConfig.WebhookSecret,
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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if err := authInfo.Require(auth.PermWebhookDelete, namespace); err != nil {
		return err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Enforce namespace scoping: namespace-scoped identities must provide
	// a namespace they have access to.
	if err := authInfo.CanAccessNamespace(namespace, auth.PermWebhookRead); err != nil {
		return nil, 0, err
	}

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
func (s *WebhookService) PushEvent(ctx context.Context, namespace string, event string, payload map[string]any, ttlSeconds int64, metadata map[string]string) (string, error) {
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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Validate required fields
	if namespace == "" {
		err := fmt.Errorf("namespace is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "namespace is required")
		return "", err
	}
	if err := authInfo.Require(auth.PermEventPush, namespace); err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "permission denied")
		return "", err
	}
	if event == "" {
		err := fmt.Errorf("event is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event is required")
		return "", err
	}

	// Lookup registered event, auto-registering if it doesn't exist yet.
	eventReg, err := s.webhookRepo.GetEventByName(ctx, tenantID, event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event lookup failed")
		s.logger.ErrorContext(ctx, "Failed to lookup event registration", "event", event, "error", err)
		return "", fmt.Errorf("failed to lookup event registration: %w", err)
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
			return "", fmt.Errorf("failed to auto-register event: %w", err)
		}
		s.logger.InfoContext(ctx, "Auto-registered new event type", "event", event)
	}
	if !eventReg.Active {
		err := fmt.Errorf("event '%s' is inactive", event)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event inactive")
		s.logger.ErrorContext(ctx, "Event is inactive", "event", event)
		return "", err
	}

	// Validate payload against event schema if present
	if len(eventReg.Schema) != 0 && payload != nil {
		if err := ValidateJSONSchema(eventReg.Schema, payload); err != nil {
			s.logger.ErrorContext(ctx, "Payload does not match event schema", "event", event, "error", err)
			return "", fmt.Errorf("payload does not match event schema: %w", err)
		}
	}

	// Set default TTL if not provided
	ttl := ttlSeconds
	if ttl <= 0 {
		ttl = 3600 * 24 // Default 1 day
	}

	// Generate event ID
	eventID := uuid.New().String()

	// Store the event record in database first
	eventRecord := &store.EventRecord{
		ID:        uuid.MustParse(eventID),
		Namespace: namespace,
		Event:     event,
		Payload:   payload,
		TTL:       ttl,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	if err := s.webhookRepo.StoreEvent(ctx, tenantID, eventRecord); err != nil {
		s.logger.ErrorContext(ctx, "Failed to store event record", "error", err, "event_id", eventID)
		return "", fmt.Errorf("failed to store event record: %w", err)
	}

	// Create event processing job with minimal data
	eventArgs := queue.EventArgs{
		EventID:    eventID,
		Namespace:  namespace,
		Event:      event,
		TTLSeconds: ttl,
		Metadata:   metadata,
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
		return "", fmt.Errorf("failed to schedule event processing: %w", err)
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
	return eventID, nil
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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if err := authInfo.CanAccessNamespace(namespace, auth.PermDeliveryRead); err != nil {
		return nil, err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	s.logger.InfoContext(ctx, "Getting delivery attempts", "delivery_id", deliveryID, "tenant_id", tenantID.String())

	if deliveryID == "" {
		return nil, fmt.Errorf("delivery ID is required")
	}

	id, err := uuid.Parse(deliveryID)
	if err != nil {
		return nil, fmt.Errorf("invalid delivery ID: %w", err)
	}

	// Look up the delivery first (tenant-scoped, no namespace filter) to find its webhook.
	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, tenantID, id, "")
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get delivery for authorization", "error", err)
		return nil, fmt.Errorf("failed to retrieve delivery: %w", err)
	}
	if delivery == nil {
		return nil, fmt.Errorf("delivery not found")
	}

	// Resolve the namespace from the parent webhook so we can authorize properly.
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenantID, delivery.WebhookID, "")
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get webhook for namespace resolution", "error", err)
		return nil, fmt.Errorf("failed to resolve delivery namespace: %w", err)
	}
	if webhook == nil {
		return nil, fmt.Errorf("webhook not found for delivery")
	}

	// Now authorize with the correct namespace instead of empty string.
	if err := authInfo.Require(auth.PermDeliveryRead, webhook.Namespace); err != nil {
		return nil, err
	}

	attempts, err := s.webhookRepo.GetDeliveryAttempts(ctx, tenantID, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get delivery attempts", "error", err)
		return nil, fmt.Errorf("failed to retrieve delivery attempts: %w", err)
	}

	return attempts, nil
}

// ListDeliveries retrieves delivery history with filters.
// When namespace is empty, returns deliveries across all namespaces.
func (s *WebhookService) ListDeliveries(ctx context.Context, namespace string, webhookID string, eventID string, limit, offset int32) ([]*store.WebhookDelivery, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListDeliveries")
	defer span.End()

	s.logger.InfoContext(ctx, "Listing deliveries", "namespace", namespace, "webhook_id", webhookID, "event_id", eventID, "limit", limit, "offset", offset)

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Enforce namespace scoping for delivery reads
	if err := authInfo.CanAccessNamespace(namespace, auth.PermDeliveryRead); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var deliveries []*store.WebhookDelivery
	var totalCount int
	var err error

	if webhookID != "" {
		var id uuid.UUID
		id, err = uuid.Parse(webhookID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid webhook ID: %w", err)
		}
		deliveries, totalCount, err = s.webhookRepo.GetDeliveriesByWebhookID(ctx, tenantID, id, namespace, int(limit), int(offset))
	} else if eventID != "" {
		var id uuid.UUID
		id, err = uuid.Parse(eventID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid event ID: %w", err)
		}
		deliveries, totalCount, err = s.webhookRepo.GetDeliveriesByEventPaginated(ctx, tenantID, id, namespace, int(limit), int(offset))
	} else {
		// List all deliveries, optionally filtered by namespace
		deliveries, totalCount, err = s.webhookRepo.ListDeliveriesPaginated(ctx, tenantID, namespace, int(limit), int(offset))
	}

	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list deliveries", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve deliveries: %w", err)
	}

	return deliveries, int32(totalCount), nil
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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Validate required fields
	if namespace == "" {
		return nil, 0, fmt.Errorf("namespace is required")
	}

	if err := authInfo.Require(auth.PermDeliveryRetry, namespace); err != nil {
		return nil, 0, err
	}

	if deliveryID == "" && webhookID == "" {
		return nil, 0, fmt.Errorf("either delivery_id or webhook_id is required")
	}

	if deliveryID != "" && webhookID != "" {
		return nil, 0, fmt.Errorf("only one of delivery_id or webhook_id can be specified")
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
			return nil, 0, fmt.Errorf("delivery already succeeded. Use force to resubmit anyway")
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

		// Queue the webhook for delivery
		_, err = s.jobInserter.Insert(ctx, &queue.WebhookArgs{
			DeliveryID: delivery.ID.String(),
			WebhookID:  delivery.WebhookID.String(),
			EventID:    delivery.EventID.String(),
			ExpiresAt:  delivery.ExpiresAt,
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
		return nil, 0, fmt.Errorf("failed to resubmit any deliveries")
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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if err := authInfo.Require(auth.PermWebhookPause, namespace); err != nil {
		return err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if err := authInfo.Require(auth.PermWebhookPause, namespace); err != nil {
		return err
	}

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
	ClientErrors  int `json:"client_errors"`  // 4xx responses
	ServerErrors  int `json:"server_errors"`  // 5xx responses
	TimeoutErrors int `json:"timeout_errors"` // Timeouts
	NetworkErrors int `json:"network_errors"` // DNS, TLS, connection refused, and other network errors
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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Event types are tenant-scoped (shared across namespaces)
	if err := authInfo.Require(auth.PermEventTypeCreate, ""); err != nil {
		return "", time.Time{}, err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Event types are tenant-scoped
	if err := authInfo.Require(auth.PermEventTypeRead, ""); err != nil {
		return nil, 0, err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Event types are tenant-scoped
	if err := authInfo.Require(auth.PermEventTypeUpdate, ""); err != nil {
		return err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Event types are tenant-scoped
	if err := authInfo.Require(auth.PermEventTypeDelete, ""); err != nil {
		return err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Event types are tenant-scoped
	if err := authInfo.Require(auth.PermEventTypeRead, ""); err != nil {
		return nil, err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if err := authInfo.Require(auth.PermHealthRead, namespace); err != nil {
		return nil, err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// This is a cross-namespace query — only tenant-level roles can do this
	if err := authInfo.Require(auth.PermHealthRead, ""); err != nil {
		return nil, 0, err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Health summary is a cross-namespace query — only tenant-level roles can do this
	if err := authInfo.Require(auth.PermHealthRead, ""); err != nil {
		return nil, err
	}

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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Enforce namespace scoping for stats
	if err := authInfo.CanAccessNamespace(namespace, auth.PermNamespaceRead); err != nil {
		return nil, err
	}

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

// UpdateWebhookConfig updates webhook configuration
func (s *WebhookService) UpdateWebhookConfig(ctx context.Context, webhookID string, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string, httpConfig *HTTPConfigUpdate) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.UpdateWebhookConfig")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing update webhook config request",
		"webhook_id", webhookID,
		"namespace", namespace)

	if webhookID == "" {
		return fmt.Errorf("webhook ID is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if err := authInfo.Require(auth.PermWebhookUpdate, namespace); err != nil {
		return err
	}

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
	// Update subscriptions if events are provided
	if len(events) > 0 {
		// Build new subscriptions slice
		var newSubs []*store.EventSubscription
		for _, event := range events {
			newSubs = append(newSubs, &store.EventSubscription{
				EventName: event,
			})
		}
		// Atomically replace all subscriptions in a single transaction
		if err := s.webhookRepo.ReplaceWebhookSubscriptions(ctx, tenantID, webhookUUID, namespace, newSubs); err != nil {
			s.logger.ErrorContext(ctx, "Failed to replace webhook subscriptions",
				"webhook_id", webhookID,
				"error", err)
			return fmt.Errorf("failed to update webhook subscriptions: %w", err)
		}
	}
	if url != "" {
		normalizedURL := strings.TrimSpace(url)
		if normalizedURL == "" {
			return fmt.Errorf("URL is required")
		}
		if err := ValidateWebhookURL(normalizedURL); err != nil {
			return err
		}
		webhook.URL = normalizedURL
	}
	if headers != nil {
		webhook.Headers = headers
	}
	if timeout > 0 {
		webhook.Timeout = timeout
	}
	webhook.Active = active
	if description != "" {
		webhook.Description = description
	}
	// Apply HTTP config updates if provided
	if httpConfig != nil {
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
		if httpConfig.WebhookSecret != "" {
			webhook.WebhookSecret = httpConfig.WebhookSecret
		}
		if httpConfig.UserAgent != "" {
			webhook.UserAgent = httpConfig.UserAgent
		}
		if httpConfig.ContentType != "" {
			webhook.ContentType = httpConfig.ContentType
		}
		// Booleans are applied directly (can't distinguish "not set" from "set to false")
		webhook.CaptureResponseBody = httpConfig.CaptureResponseBody
		webhook.FollowRedirects = httpConfig.FollowRedirects
		webhook.VerifySSL = httpConfig.VerifySSL
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
// When namespace is empty, returns reports across all namespaces.
func (s *WebhookService) ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int32) ([]*store.EventReportWithStats, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEventReports")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing list event reports request",
		"namespace", namespace,
		"event_name", eventName,
		"limit", limit,
		"offset", offset)

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// Enforce namespace scoping for event reports
	if err := authInfo.CanAccessNamespace(namespace, auth.PermEventRead); err != nil {
		return nil, 0, err
	}

	// Set default limit if not provided or out of range
	if limit <= 0 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000
	}

	// Ensure offset is not negative
	if offset < 0 {
		offset = 0
	}

	events, totalCount, err := s.webhookRepo.ListEventReportsWithStats(ctx, tenantID, namespace, eventName, int(limit), int(offset))
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list event reports", "namespace", namespace, "event_name", eventName, "error", err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, 0, fmt.Errorf("failed to list event reports: %w", err)
	}

	s.logger.InfoContext(ctx, "Successfully listed event reports",
		"namespace", namespace,
		"event_name", eventName,
		"count", len(events),
		"total", totalCount)

	span.SetAttributes(
		attribute.String("namespace", namespace),
		attribute.Int("count", len(events)),
		attribute.Int("total", totalCount),
	)
	if eventName != nil {
		span.SetAttributes(attribute.String("event_name", *eventName))
	}

	return events, int32(totalCount), nil
}

// Subscription Management Implementation

func (s *WebhookService) CreateSubscription(ctx context.Context, webhookID, eventName, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string) (string, time.Time, error) {
	s.logger.InfoContext(ctx, "Creating subscription", "webhook_id", webhookID, "event_name", eventName, "namespace", namespace)

	if namespace == "" {
		return "", time.Time{}, fmt.Errorf("namespace is required")
	}

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if err := authInfo.Require(auth.PermSubscriptionCreate, namespace); err != nil {
		return "", time.Time{}, err
	}

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid webhook ID: %w", err)
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
	}

	if err := s.webhookRepo.CreateSubscription(ctx, tenantID, sub); err != nil {
		return "", time.Time{}, err
	}

	return sub.ID.String(), sub.CreatedAt, nil
}

func (s *WebhookService) GetSubscription(ctx context.Context, subscriptionID string, namespace string) (*store.EventSubscription, error) {
	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if err := authInfo.Require(auth.PermSubscriptionRead, namespace); err != nil {
		return nil, err
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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if err := authInfo.Require(auth.PermSubscriptionRead, namespace); err != nil {
		return nil, 0, err
	}

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
		subs, err = s.webhookRepo.GetSubscriptionsByEvent(ctx, tenantID, namespace, eventName)
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

func (s *WebhookService) UpdateSubscription(ctx context.Context, subscriptionID string, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string) error {
	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if err := authInfo.Require(auth.PermSubscriptionUpdate, namespace); err != nil {
		return err
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

	sub.Headers = headers
	sub.Method = method
	sub.Timeout = timeout
	sub.TransformEnabled = transformEnabled
	sub.TransformTemplate = transformTemplate

	return s.webhookRepo.UpdateSubscription(ctx, tenantID, sub)
}

func (s *WebhookService) DeleteSubscription(ctx context.Context, subscriptionID string, namespace string) error {
	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if err := authInfo.Require(auth.PermSubscriptionDelete, namespace); err != nil {
		return err
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

	authInfo := auth.MustFromContext(ctx)
	tenantID := authInfo.TenantID

	// TestSubscriptionTemplate is namespace-scoped when namespace is provided
	if namespace != "" {
		if err := authInfo.Require(auth.PermSubscriptionRead, namespace); err != nil {
			return "", err
		}
	} else {
		// Without a namespace, require tenant-wide access
		if err := authInfo.Require(auth.PermSubscriptionRead, ""); err != nil {
			return "", err
		}
	}

	event, err := s.webhookRepo.GetEventByName(ctx, tenantID, eventName)
	if err != nil {
		return "", fmt.Errorf("failed to get event: %w", err)
	}
	if event == nil {
		return "", fmt.Errorf("event not found")
	}

	engine := client.NewTemplateEngine()

	// Create context for template
	data := client.WebhookTemplateContext{
		EventID:   "dry-run-event-id",
		EventName: eventName,
		Payload:   event.SamplePayload,
	}

	result, err := engine.TransformPayload(transformTemplate, data)
	if err != nil {
		return "", fmt.Errorf("template transformation failed: %w", err)
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
