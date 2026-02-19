package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	urlpkg "net/url"
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
	UpdateWebhookConfig(ctx context.Context, webhookID string, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string) error
	PauseWebhook(ctx context.Context, webhookID string, namespace string, reason string) error
	ResumeWebhook(ctx context.Context, webhookID string, namespace string) error
	GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStatsData, error)

	// Event Management
	RegisterEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) (string, time.Time, error)
	ListEvents(ctx context.Context, activeOnly bool, limit, offset int32) ([]*store.EventRegistration, int32, error)
	UpdateEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) error
	DeleteEvent(ctx context.Context, name string) error
	PushEvent(ctx context.Context, namespace string, event string, payload map[string]any, ttlSeconds int64, metadata map[string]string) (string, error)
	ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int32) ([]*store.EventReportWithStats, int32, error)

	// Subscription Management
	CreateSubscription(ctx context.Context, webhookID, eventName, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string) (string, time.Time, error)
	GetSubscription(ctx context.Context, subscriptionID string, namespace string) (*store.EventSubscription, error)
	ListSubscriptions(ctx context.Context, namespace string, webhookID string, eventName string, limit, offset int32) ([]*store.EventSubscription, int32, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string) error
	DeleteSubscription(ctx context.Context, subscriptionID string, namespace string) error

	// Delivery Management
	GetDeliveryStatus(ctx context.Context, deliveryID string, namespace string) (*store.WebhookDelivery, error)
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

	s.logger.Info("Processing webhook registration request",
		"namespace", namespace,
		"events", events,
		"url", url,
	)

	if namespace == "" {
		return "", time.Time{}, fmt.Errorf("namespace is required")
	}
	if len(events) == 0 {
		return "", time.Time{}, fmt.Errorf("at least one event is required")
	}
	if url == "" {
		return "", time.Time{}, fmt.Errorf("URL is required")
	}
	s.logger.Info("Validating event names", "events", events, "contains_empty", slices.Contains(events, ""))
	if slices.Contains(events, "") {
		s.logger.Error("Event names validation failed", "events", events)
		return "", time.Time{}, fmt.Errorf("event names cannot be empty")
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
	if err := s.webhookRepo.RegisterWebhook(ctx, registration); err != nil {
		s.logger.Error("Failed to register webhook",
			"namespace", namespace,
			"events", events,
			"url", url,
			"error", err,
		)
		return "", time.Time{}, fmt.Errorf("failed to register webhook: %w", err)
	}

	// Create subscriptions for each event
	for _, event := range events {
		sub := &store.EventSubscription{
			WebhookID: registration.ID,
			EventName: event,
			Namespace: namespace,
		}
		if err := s.webhookRepo.CreateSubscription(ctx, sub); err != nil {
			s.logger.Error("Failed to create subscription", "webhook_id", registration.ID, "event", event, "error", err)
		}
	}

	if s.metrics != nil {
		s.metrics.WebhookRegistrations.Add(ctx, 1)
		s.metrics.ActiveWebhooks.Add(ctx, 1)
	}
	s.logger.Info("Webhook registered successfully",
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

	s.logger.Info("Processing enhanced webhook creation request",
		"namespace", req.Namespace,
		"events", req.Events,
		"url", req.URL,
	)

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
		events, _, err := s.webhookRepo.ListEventsPaginated(ctx, false, 1000, 0)
		if err != nil {
			s.logger.Warn("Failed to validate event names", "error", err)
		} else {
			eventExists := false
			for _, registeredEvent := range events {
				if registeredEvent.Name == event {
					eventExists = true
					break
				}
			}
			if !eventExists {
				s.logger.Warn("Event not registered", "event", event, "namespace", req.Namespace)
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

	// Register the webhook
	if err := s.webhookRepo.RegisterWebhook(ctx, storeWebhook); err != nil {
		s.logger.Error("Failed to register webhook",
			"namespace", req.Namespace,
			"events", req.Events,
			"url", req.URL,
			"error", err,
		)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to register webhook")
		return nil, fmt.Errorf("failed to register webhook: %w", err)
	}

	// Create subscriptions for each event
	for _, event := range req.Events {
		sub := &store.EventSubscription{
			WebhookID: storeWebhook.ID,
			EventName: event,
			Namespace: req.Namespace,
		}
		if err := s.webhookRepo.CreateSubscription(ctx, sub); err != nil {
			s.logger.Error("Failed to create subscription", "webhook_id", storeWebhook.ID, "event", event, "error", err)
		}
	}

	// Update metrics
	if s.metrics != nil {
		s.metrics.WebhookRegistrations.Add(ctx, 1)
		s.metrics.ActiveWebhooks.Add(ctx, 1)
	}

	s.logger.Info("Enhanced webhook registered successfully",
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
	s.logger.Info("Processing webhook un registration request",
		"webhook_id", webhookID,
		"namespace", namespace,
	)
	if webhookID == "" {
		return fmt.Errorf("webhook_id is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %w", err)
	}

	// Check if webhook exists in namespace
	_, err = s.webhookRepo.GetWebhookByID(ctx, id, namespace)
	if err != nil {
		return fmt.Errorf("failed to retrieve webhook: %w", err)
	}

	if err := s.webhookRepo.UnregisterWebhook(ctx, id); err != nil {
		s.logger.Error("Failed to unregister webhook",
			"webhook_id", webhookID,
			"error", err,
		)
		return fmt.Errorf("failed to unregister webhook: %w", err)
	}
	s.logger.Info("Webhook unregistered successfully",
		"webhook_id", webhookID,
	)
	return nil
}

// ListWebhooks lists all registered webhooks for a namespace with optional filters
func (s *WebhookService) ListWebhooks(ctx context.Context, namespace string, webhookID string, event string, activeOnly bool, limit, offset int32) ([]*store.WebhookRegistration, int32, error) {
	s.logger.Info("Processing list webhooks request",
		"namespace", namespace,
		"webhook_id", webhookID,
		"event", event,
		"active_only", activeOnly,
		"limit", limit,
		"offset", offset,
	)
	if namespace == "" {
		return nil, 0, fmt.Errorf("namespace is required")
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

		reg, err := s.webhookRepo.GetWebhookByID(ctx, id, namespace)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to retrieve webhook: %w", err)
		}
		if reg == nil || (activeOnly && !reg.Active) {
			return []*store.WebhookRegistration{}, 0, nil
		}
		return []*store.WebhookRegistration{reg}, 1, nil
	}

	registrations, totalCount, err := s.webhookRepo.ListWebhooksPaginated(ctx, namespace, event, activeOnly, int(limit), int(offset))
	if err != nil {
		s.logger.Error("Failed to list webhooks",
			"namespace", namespace,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to list webhooks: %w", err)
	}

	s.logger.Info("Listed webhooks successfully",
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

	s.logger.Info("Processing push event request",
		"namespace", namespace,
		"event", event,
	)

	// Validate required fields
	if namespace == "" {
		err := fmt.Errorf("namespace is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "namespace is required")
		return "", err
	}
	if event == "" {
		err := fmt.Errorf("event is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event is required")
		return "", err
	}

	// Lookup registered event
	eventReg, err := s.webhookRepo.GetEventByName(ctx, event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event lookup failed")
		s.logger.Error("Failed to lookup event registration", "event", event, "error", err)
		return "", fmt.Errorf("failed to lookup event registration: %w", err)
	}
	if eventReg == nil || !eventReg.Active {
		err := fmt.Errorf("event '%s' is not registered or inactive", event)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event not registered or inactive")
		s.logger.Error("Event not registered or inactive", "event", event)
		return "", err
	}

	// Validate payload against event schema if present
	if len(eventReg.Schema) != 0 && payload != nil {
		if err := ValidateJSONSchema(eventReg.Schema, payload); err != nil {
			s.logger.Error("Payload does not match event schema", "event", event, "error", err)
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

	if err := s.webhookRepo.StoreEvent(ctx, eventRecord); err != nil {
		s.logger.Error("Failed to store event record", "error", err, "event_id", eventID)
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
	}

	// Insert the event processing job
	_, err = s.jobInserter.Insert(ctx, eventArgs)
	if err != nil {
		s.logger.Error("Failed to schedule event processing job",
			"event_id", eventID,
			"namespace", namespace,
			"event", event,
			"error", err,
		)
		return "", fmt.Errorf("failed to schedule event processing: %w", err)
	}

	// Record metrics
	if s.metrics != nil {
		s.metrics.EventsPushed.Add(ctx, 1)
	}

	span.SetStatus(otelcodes.Ok, "event scheduled successfully")

	s.logger.Info("Event processing scheduled successfully",
		"event_id", eventID,
		"namespace", namespace,
		"event", event,
	)
	return eventID, nil
}

// GetDeliveryStatus gets the status of a webhook delivery
func (s *WebhookService) GetDeliveryStatus(ctx context.Context, deliveryID string, namespace string) (*store.WebhookDelivery, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetDeliveryStatus")
	defer span.End()

	s.logger.Info("Getting webhook delivery status",
		"delivery_id", deliveryID,
		"namespace", namespace)

	if deliveryID == "" {
		return nil, fmt.Errorf("delivery ID is required")
	}
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	id, err := uuid.Parse(deliveryID)
	if err != nil {
		return nil, fmt.Errorf("invalid delivery ID: %w", err)
	}

	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, id, namespace)
	if err != nil {
		s.logger.Error("Failed to get delivery by ID", "error", err)
		return nil, fmt.Errorf("failed to retrieve delivery status: %w", err)
	}
	if delivery == nil {
		return nil, fmt.Errorf("delivery not found")
	}
	return delivery, nil
}

// ListDeliveries retrieves delivery history with filters
func (s *WebhookService) ListDeliveries(ctx context.Context, namespace string, webhookID string, eventID string, limit, offset int32) ([]*store.WebhookDelivery, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListDeliveries")
	defer span.End()

	s.logger.Info("Listing deliveries", "namespace", namespace, "webhook_id", webhookID, "event_id", eventID, "limit", limit, "offset", offset)

	if namespace == "" {
		return nil, 0, fmt.Errorf("namespace is required")
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
		id, err := uuid.Parse(webhookID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid webhook ID: %w", err)
		}
		deliveries, totalCount, err = s.webhookRepo.GetDeliveriesByWebhookID(ctx, id, namespace, int(limit), int(offset))
	} else if eventID != "" {
		id, err := uuid.Parse(eventID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid event ID: %w", err)
		}
		deliveries, totalCount, err = s.webhookRepo.GetDeliveriesByEventPaginated(ctx, id, namespace, int(limit), int(offset))
	} else {
		// List all deliveries for namespace
		deliveries, totalCount, err = s.webhookRepo.ListDeliveriesPaginated(ctx, namespace, int(limit), int(offset))
	}

	if err != nil {
		s.logger.Error("Failed to list deliveries", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve deliveries: %w", err)
	}

	return deliveries, int32(totalCount), nil
}

// RetryDelivery manually retries failed or pending webhook deliveries
func (s *WebhookService) RetryDelivery(ctx context.Context, namespace string, deliveryID string, webhookID string, force bool) ([]string, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RetryDelivery")
	defer span.End()

	s.logger.Info("Processing retry delivery request",
		"delivery_id", deliveryID,
		"webhook_id", webhookID,
		"namespace", namespace,
		"force", force)

	// Validate required fields
	if namespace == "" {
		return nil, 0, fmt.Errorf("namespace is required")
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
		delivery, err := s.webhookRepo.GetDeliveryByID(ctx, id, namespace)
		if err != nil {
			s.logger.Error("Failed to get delivery", "error", err)
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
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, id, namespace)
		if err != nil {
			s.logger.Error("Failed to get webhook", "error", err)
			return nil, 0, fmt.Errorf("failed to retrieve webhook: %w", err)
		}

		if webhook == nil {
			return nil, 0, fmt.Errorf("webhook not found")
		}

		// Get retriable deliveries
		deliveriesToResubmit, err = s.webhookRepo.GetRetriableDeliveries(ctx, id, namespace, force)
		if err != nil {
			s.logger.Error("Failed to get retriable deliveries", "error", err)
			return nil, 0, fmt.Errorf("failed to retrieve deliveries: %w", err)
		}

		if len(deliveriesToResubmit) == 0 {
			message := "No failed or pending deliveries found"
			if !force {
				message += ". Use force to resubmit all deliveries"
			}
			s.logger.Info(message)
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
			s.logger.Error("Failed to reset delivery for retry",
				"delivery_id", delivery.ID,
				"error", err)
			continue
		}

		// Get webhook info for queuing
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, delivery.WebhookID, namespace)
		if err != nil {
			s.logger.Error("Failed to get webhook for delivery",
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
		})
		if err != nil {
			s.logger.Error("Failed to queue webhook for resubmission",
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

	s.logger.Info("Webhook deliveries resubmitted successfully",
		"resubmitted_count", resubmittedCount,
		"total_requested", len(deliveriesToResubmit))

	return resubmittedIDs, resubmittedCount, nil
}

// PauseWebhook temporarily disables webhook deliveries
func (s *WebhookService) PauseWebhook(ctx context.Context, webhookID string, namespace string, reason string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.PauseWebhook")
	defer span.End()

	s.logger.Info("Pausing webhook", "webhook_id", webhookID, "namespace", namespace, "reason", reason)
	if webhookID == "" {
		return fmt.Errorf("webhook ID is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %w", err)
	}

	webhook, err := s.webhookRepo.GetWebhookByID(ctx, id, namespace)
	if err != nil {
		s.logger.Error("Failed to get webhook", "error", err)
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
	err = s.webhookRepo.UpdateWebhook(ctx, webhook)
	if err != nil {
		s.logger.Error("Failed to pause webhook", "error", err)
		return fmt.Errorf("failed to pause webhook: %w", err)
	}
	s.logger.Info("Webhook paused successfully", "webhook_id", webhookID)
	return nil
}

// ResumeWebhook re-enables webhook deliveries
func (s *WebhookService) ResumeWebhook(ctx context.Context, webhookID string, namespace string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ResumeWebhook")
	defer span.End()

	s.logger.Info("Resuming webhook",
		"webhook_id", webhookID,
		"namespace", namespace)

	if webhookID == "" {
		return fmt.Errorf("webhook ID is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %w", err)
	}

	webhook, err := s.webhookRepo.GetWebhookByID(ctx, id, namespace)
	if err != nil {
		s.logger.Error("Failed to get webhook", "error", err)
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
	err = s.webhookRepo.UpdateWebhook(ctx, webhook)
	if err != nil {
		s.logger.Error("Failed to resume webhook", "error", err)
		return fmt.Errorf("failed to resume webhook: %w", err)
	}
	s.logger.Info("Webhook resumed successfully", "webhook_id", webhookID)
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

	s.logger.Info("Processing event registration request", "name", name, "description", description)
	if name == "" {
		return "", time.Time{}, fmt.Errorf("event name is required")
	}
	existingEvent, err := s.webhookRepo.GetEventByName(ctx, name)
	if err != nil {
		s.logger.Error("Failed to check existing event", "error", err)
		return "", time.Time{}, fmt.Errorf("failed to check existing event: %w", err)
	}
	if existingEvent != nil {
		return "", time.Time{}, fmt.Errorf("event already exists")
	}

	// Generate sample payload from schema
	samplePayload, err := generateSamplePayload(schema)
	if err != nil {
		s.logger.Warn("Failed to generate sample payload, using empty payload", "error", err)
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
	if !active {
		event.Active = true
	}
	if err := s.webhookRepo.RegisterEvent(ctx, event); err != nil {
		s.logger.Error("Failed to register event",
			"name", name,
			"error", err,
		)
		return "", time.Time{}, fmt.Errorf("failed to register event: %w", err)
	}
	s.logger.Info("Event registered successfully",
		"event_id", event.ID,
		"name", name,
		"description", description,
	)
	return event.ID.String(), event.CreatedAt, nil
}

// ListEvents lists all registered events
func (s *WebhookService) ListEvents(ctx context.Context, activeOnly bool, limit, offset int32) ([]*store.EventRegistration, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEvents")
	defer span.End()

	s.logger.Info("Processing list events request",
		"active_only", activeOnly, "limit", limit, "offset", offset)

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	events, totalCount, err := s.webhookRepo.ListEventsPaginated(ctx, activeOnly, int(limit), int(offset))
	if err != nil {
		s.logger.Error("Failed to list events", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve events: %w", err)
	}
	s.logger.Info("Listed events successfully",
		"count", len(events),
		"total", totalCount,
	)
	return events, int32(totalCount), nil
}

// UpdateEvent updates an event registration
func (s *WebhookService) UpdateEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.UpdateEvent")
	defer span.End()

	s.logger.Info("Processing event update request",
		"name", name,
		"description", description)

	// Validate required fields
	if name == "" {
		return fmt.Errorf("event name is required")
	}

	// Check if event exists
	existingEvent, err := s.webhookRepo.GetEventByName(ctx, name)
	if err != nil {
		s.logger.Error("Failed to get event", "error", err)
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
		s.logger.Warn("Failed to generate sample payload, using empty payload", "error", err)
		samplePayload = map[string]any{}
	}
	existingEvent.SamplePayload = samplePayload

	// Update the event
	if err := s.webhookRepo.UpdateEvent(ctx, existingEvent); err != nil {
		s.logger.Error("Failed to update event",
			"name", name,
			"error", err,
		)
		return fmt.Errorf("failed to update event: %w", err)
	}

	s.logger.Info("Event updated successfully", "name", name)
	return nil
}

// DeleteEvent deletes an event registration
func (s *WebhookService) DeleteEvent(ctx context.Context, name string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.DeleteEvent")
	defer span.End()

	s.logger.Info("Processing event deletion request", "name", name)

	// Validate required fields
	if name == "" {
		return fmt.Errorf("event name is required")
	}

	// Check if event exists
	existingEvent, err := s.webhookRepo.GetEventByName(ctx, name)
	if err != nil {
		s.logger.Error("Failed to get event", "error", err)
		return fmt.Errorf("failed to retrieve event: %w", err)
	}

	if existingEvent == nil {
		return fmt.Errorf("event not found")
	}

	// Delete the event
	if err := s.webhookRepo.DeleteEvent(ctx, name); err != nil {
		s.logger.Error("Failed to delete event",
			"name", name,
			"error", err,
		)
		return fmt.Errorf("failed to delete event: %w", err)
	}

	s.logger.Info("Event deleted successfully", "name", name)
	return nil
}

// GetWebhookHealth retrieves health metrics for a webhook
func (s *WebhookService) GetWebhookHealth(ctx context.Context, webhookID string, namespace string) (*WebhookHealthData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetWebhookHealth")
	defer span.End()

	s.logger.Info("Processing get webhook health request",
		"webhook_id", webhookID,
		"namespace", namespace)

	// Validate required fields
	if webhookID == "" {
		return nil, fmt.Errorf("webhook ID is required")
	}

	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	id, err := uuid.Parse(webhookID)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook ID: %w", err)
	}

	// Get webhook to verify it exists and get current health
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, id, namespace)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhook")
		s.logger.Error("Failed to get webhook", "error", err)
		return nil, fmt.Errorf("webhook not found: %w", err)
	}

	if webhook == nil {
		span.SetStatus(otelcodes.Error, "Webhook not found")
		s.logger.Error("Webhook not found", "webhook_id", webhookID)
		return nil, fmt.Errorf("webhook not found")
	}

	// Get health state (current status and consecutive failures)
	healthState, err := s.webhookRepo.GetWebhookHealthState(ctx, id)
	if err != nil {
		// If no health state exists yet, return basic health info
		s.logger.Info("No health state found for webhook", "webhook_id", webhookID)
		return &WebhookHealthData{
			WebhookID: webhookID,
			Health:    webhook.Health,
		}, nil
	}

	// Get health summary for the last 24 hours
	healthSummary, err := s.webhookRepo.GetWebhookHealthSummary(ctx, id, 24)
	if err != nil {
		s.logger.Error("Failed to get health summary", "error", err)
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
	}

	s.logger.Info("Webhook health retrieved successfully",
		"webhook_id", webhookID,
		"health", webhook.Health,
		"success_rate", healthData.SuccessRate)

	return healthData, nil
}

// ListWebhooksByHealth retrieves webhooks filtered by health status
func (s *WebhookService) ListWebhooksByHealth(ctx context.Context, health store.WebhookHealth, limit, offset int32) ([]*store.WebhookRegistration, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListWebhooksByHealth")
	defer span.End()

	s.logger.Info("Processing list webhooks by health request", "health", health, "limit", limit, "offset", offset)

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Get webhooks by health status
	webhooksList, totalCount, err := s.webhookRepo.GetWebhooksByHealthPaginated(ctx, health, int(limit), int(offset))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhooks by health")
		s.logger.Error("Failed to get webhooks by health", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve webhooks: %w", err)
	}

	s.logger.Info("Webhooks retrieved successfully",
		"health", health,
		"count", len(webhooksList),
		"total", totalCount)

	return webhooksList, int32(totalCount), nil
}

// GetHealthSummary retrieves a summary of webhook health across all namespaces
func (s *WebhookService) GetHealthSummary(ctx context.Context) (*HealthSummaryData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetHealthSummary")
	defer span.End()

	s.logger.Info("Processing get health summary request")

	// Get health summary from repository
	summary, err := s.webhookRepo.GetHealthSummary(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get health summary")
		s.logger.Error("Failed to get health summary", "error", err)
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

	s.logger.Info("Health summary retrieved successfully",
		"healthy", healthSummary.HealthyCount,
		"degraded", healthSummary.DegradedCount,
		"unhealthy", healthSummary.UnhealthyCount,
		"unknown", healthSummary.UnknownCount,
		"total", healthSummary.TotalCount)

	return healthSummary, nil
}

// GetNamespaceStats retrieves statistics for a namespace
func (s *WebhookService) GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStatsData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetNamespaceStats")
	defer span.End()

	s.logger.Info("Processing get namespace stats request", "namespace", namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	stats, err := s.webhookRepo.GetNamespaceStats(ctx, namespace)
	if err != nil {
		s.logger.Error("Failed to get namespace stats", "error", err)
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

	s.logger.Info("Namespace stats retrieved successfully",
		"namespace", namespace,
		"total_webhooks", res.TotalWebhooks,
		"active_webhooks", res.ActiveWebhooks,
		"success_rate", res.SuccessRate)
	return res, nil
}

// UpdateWebhookConfig updates webhook configuration
func (s *WebhookService) UpdateWebhookConfig(ctx context.Context, webhookID string, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.UpdateWebhookConfig")
	defer span.End()

	s.logger.Info("Processing update webhook config request",
		"webhook_id", webhookID,
		"namespace", namespace)

	if webhookID == "" {
		return fmt.Errorf("webhook ID is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	webhookUUID, err := uuid.Parse(webhookID)
	if err != nil {
		return fmt.Errorf("invalid webhook ID: %w", err)
	}

	webhook, err := s.webhookRepo.GetWebhookByID(ctx, webhookUUID, namespace)
	if err != nil {
		s.logger.Error("Failed to get webhook", "error", err)
		return fmt.Errorf("failed to retrieve webhook: %w", err)
	}
	if webhook == nil {
		return fmt.Errorf("webhook not found")
	}
	// Update subscriptions if events are provided
	if len(events) > 0 {
		// Delete existing subscriptions
		existingSubs, err := s.webhookRepo.ListSubscriptions(ctx, webhookUUID)
		if err != nil {
			s.logger.Error("Failed to get existing subscriptions", "error", err)
		} else {
			for _, sub := range existingSubs {
				if err := s.webhookRepo.DeleteSubscription(ctx, sub.ID); err != nil {
					s.logger.Error("Failed to delete subscription", "subscription_id", sub.ID, "error", err)
				}
			}
		}
		// Create new subscriptions
		for _, event := range events {
			sub := &store.EventSubscription{
				WebhookID: webhookUUID,
				EventName: event,
				Namespace: namespace,
			}
			if err := s.webhookRepo.CreateSubscription(ctx, sub); err != nil {
				s.logger.Error("Failed to create subscription", "webhook_id", webhookID, "event", event, "error", err)
			}
		}
	}
	if url != "" {
		normalizedURL := strings.TrimSpace(url)
		if normalizedURL == "" {
			return fmt.Errorf("URL is required")
		}
		if _, err := urlpkg.ParseRequestURI(normalizedURL); err != nil {
			return fmt.Errorf("invalid URL: %w", err)
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
	err = s.webhookRepo.UpdateWebhook(ctx, webhook)
	if err != nil {
		s.logger.Error("Failed to update webhook config",
			"webhook_id", webhookID,
			"error", err)
		return fmt.Errorf("failed to update webhook configuration: %w", err)
	}
	s.logger.Info("Webhook configuration updated successfully",
		"webhook_id", webhookID)
	return nil
}

// ValidateJSONSchema validates a payload against a JSON schema string
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
	if result := sch.ValidateJSON(payloadBytes); result != nil && !result.Valid {
		return fmt.Errorf("payload validation failed: %v", result)
	}
	return nil
}

// ListEventReports lists event records with delivery statistics in descending order by creation time
func (s *WebhookService) ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int32) ([]*store.EventReportWithStats, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEventReports")
	defer span.End()

	s.logger.Info("Processing list event reports request",
		"namespace", namespace,
		"event_name", eventName,
		"limit", limit,
		"offset", offset)

	// Validate input parameters
	if namespace == "" {
		err := fmt.Errorf("namespace is required")
		span.SetStatus(otelcodes.Error, err.Error())
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

	events, totalCount, err := s.webhookRepo.ListEventReportsWithStats(ctx, namespace, eventName, int(limit), int(offset))
	if err != nil {
		s.logger.Error("Failed to list event reports", "namespace", namespace, "event_name", eventName, "error", err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, 0, fmt.Errorf("failed to list event reports: %w", err)
	}

	s.logger.Info("Successfully listed event reports",
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
	s.logger.Info("Creating subscription", "webhook_id", webhookID, "event_name", eventName, "namespace", namespace)

	if namespace == "" {
		return "", time.Time{}, fmt.Errorf("namespace is required")
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

	if err := s.webhookRepo.CreateSubscription(ctx, sub); err != nil {
		return "", time.Time{}, err
	}

	return sub.ID.String(), sub.CreatedAt, nil
}

func (s *WebhookService) GetSubscription(ctx context.Context, subscriptionID string, namespace string) (*store.EventSubscription, error) {
	id, err := uuid.Parse(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription ID: %w", err)
	}

	sub, err := s.webhookRepo.GetSubscription(ctx, id)
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
		id, err := uuid.Parse(webhookID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid webhook ID: %w", err)
		}
		subs, err = s.webhookRepo.ListSubscriptions(ctx, id)
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
		subs, err = s.webhookRepo.GetSubscriptionsByEvent(ctx, namespace, eventName)
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
		// This might need a new repo method for listing all subs in namespace
		return nil, 0, fmt.Errorf("either webhook_id or event_name is required")
	}

	return subs, int32(totalCount), err
}

func (s *WebhookService) UpdateSubscription(ctx context.Context, subscriptionID string, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string) error {
	id, err := uuid.Parse(subscriptionID)
	if err != nil {
		return fmt.Errorf("invalid subscription ID: %w", err)
	}

	sub, err := s.webhookRepo.GetSubscription(ctx, id)
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

	return s.webhookRepo.UpdateSubscription(ctx, sub)
}

func (s *WebhookService) DeleteSubscription(ctx context.Context, subscriptionID string, namespace string) error {
	id, err := uuid.Parse(subscriptionID)
	if err != nil {
		return fmt.Errorf("invalid subscription ID: %w", err)
	}

	sub, err := s.webhookRepo.GetSubscription(ctx, id)
	if err != nil {
		return err
	}
	if sub == nil || sub.Namespace != namespace {
		return fmt.Errorf("subscription not found in namespace")
	}

	return s.webhookRepo.DeleteSubscription(ctx, sub.ID)
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
