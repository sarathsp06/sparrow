package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	jsonschema "github.com/kaptinlin/jsonschema"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/observability"
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

//go:generate gowrap gen -i WebhookServiceInterface -t opentelemetry -o WebhookServiceInterface_otel.go
type WebhookServiceInterface interface {
	RegisterWebhook(ctx context.Context, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string) (string, int64, error)
	UnregisterWebhook(ctx context.Context, webhookID string) error
	PushEvent(ctx context.Context, namespace string, event string, payload map[string]any, ttlSeconds int64, metadata map[string]string) (string, error)
	GetWebhookStatus(ctx context.Context, namespace string, webhookID string) ([]*store.WebhookDelivery, int32, error)
	ListWebhooks(ctx context.Context, namespace string, event string, activeOnly bool) ([]*store.WebhookRegistration, error)
	GetRegisteredWebhooks(ctx context.Context, namespace string, webhookID string, activeOnly bool) ([]*store.WebhookRegistration, error)
	ListRegisteredWebhooksByEvent(ctx context.Context, namespace string, event string, activeOnly bool) ([]*store.WebhookRegistration, error)
	GetWebhookDeliveryStatus(ctx context.Context, deliveryID string, namespace string) (*store.WebhookDelivery, error)
	ResendWebhook(ctx context.Context, deliveryID string, namespace string, forceResend bool) (string, error)
	PauseWebhook(ctx context.Context, webhookID string, namespace string, reason string) error
	ResumeWebhook(ctx context.Context, webhookID string, namespace string) error
	GetWebhookDeliveryHistory(ctx context.Context, webhookID string, namespace string, limit int32, offset int32) ([]*store.WebhookDelivery, int32, error)
	RegisterEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) (string, int64, error)
	GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStatsData, error)
	UpdateWebhookConfig(ctx context.Context, webhookID string, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string) error
	UpdateEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) error
	DeleteEvent(ctx context.Context, name string) error
	GetWebhookHealth(ctx context.Context, webhookID string, namespace string) (*WebhookHealthData, error)
	GetHealthSummary(ctx context.Context) (*HealthSummaryData, error)
	ListEvents(ctx context.Context, activeOnly bool) ([]*store.EventRegistration, error)
	ListWebhooksByHealth(ctx context.Context, health store.WebhookHealth) ([]*store.WebhookRegistration, error)
	ResubmitWebhook(ctx context.Context, deliveryID string, webhookID string, namespace string, force bool) ([]string, int32, error)
	ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int32) ([]*store.EventReportWithStats, int32, error)
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

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookService) RegisterWebhook(ctx context.Context, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string) (string, int64, error) {
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
		return "", 0, fmt.Errorf("namespace is required")
	}
	if len(events) == 0 {
		return "", 0, fmt.Errorf("at least one event is required")
	}
	if url == "" {
		return "", 0, fmt.Errorf("URL is required")
	}
	s.logger.Info("Validating event names", "events", events, "contains_empty", slices.Contains(events, ""))
	if slices.Contains(events, "") {
		s.logger.Error("Event names validation failed", "events", events)
		return "", 0, fmt.Errorf("event names cannot be empty")
	}
	if timeout <= 0 {
		timeout = 30
	}
	registration := &store.WebhookRegistration{
		Namespace:   namespace,
		Events:      events,
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
		return "", 0, fmt.Errorf("failed to register webhook: %w", err)
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
	return registration.ID, registration.CreatedAt.Unix(), nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookService) UnregisterWebhook(ctx context.Context, webhookID string) error {
	s.logger.Info("Processing webhook un registration request",
		"webhook_id", webhookID,
	)
	if webhookID == "" {
		return fmt.Errorf("webhook_id is required")
	}
	if err := s.webhookRepo.UnregisterWebhook(ctx, webhookID); err != nil {
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

	// Create event processing job
	eventArgs := queue.EventArgs{
		EventID:    eventID,
		Namespace:  namespace,
		Event:      event,
		Payload:    payload,
		TTLSeconds: ttl,
		Metadata:   metadata,
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

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookService) GetWebhookStatus(ctx context.Context, webhookID string, eventID string) ([]*store.WebhookDelivery, int32, error) {
	s.logger.Info("Processing webhook status request")
	var deliveries []*store.WebhookDelivery
	var err error
	if webhookID != "" {
		deliveries, err = s.webhookRepo.GetDeliveriesByWebhook(ctx, webhookID)
	} else if eventID != "" {
		deliveries, err = s.webhookRepo.GetDeliveriesByEvent(ctx, eventID)
	} else {
		return nil, 0, fmt.Errorf("either webhookID or eventID is required")
	}
	if err != nil {
		s.logger.Error("Failed to get webhook deliveries", "error", err)
		return nil, 0, fmt.Errorf("failed to get webhook status: %w", err)
	}
	return deliveries, int32(len(deliveries)), nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookService) ListWebhooks(ctx context.Context, namespace string, event string, activeOnly bool) ([]*store.WebhookRegistration, error) {
	s.logger.Info("Processing list webhooks request",
		"namespace", namespace,
		"event", event,
		"active_only", activeOnly,
	)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	registrations, err := s.webhookRepo.ListWebhooks(ctx, namespace, activeOnly)
	if err != nil {
		s.logger.Error("Failed to list webhooks",
			"namespace", namespace,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	var filteredRegistrations []*store.WebhookRegistration
	if event != "" {
		for _, reg := range registrations {
			for _, evt := range reg.Events {
				if evt == event {
					filteredRegistrations = append(filteredRegistrations, reg)
					break
				}
			}
		}
	} else {
		filteredRegistrations = registrations
	}
	s.logger.Info("Listed webhooks successfully",
		"namespace", namespace,
		"total_count", len(filteredRegistrations),
	)
	return filteredRegistrations, nil
}

// GetRegisteredWebhooks gets registered webhooks by namespace and optional webhook ID
func (s *WebhookService) GetRegisteredWebhooks(ctx context.Context, namespace string, webhookID string, activeOnly bool) ([]*store.WebhookRegistration, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetRegisteredWebhooks")
	defer span.End()

	s.logger.Info("Getting registered webhooks",
		"namespace", namespace,
		"webhook_id", webhookID,
		"active_only", activeOnly)

	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	var regs []*store.WebhookRegistration
	var err error

	if webhookID != "" {
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, webhookID, namespace)
		if err != nil {
			s.logger.Error("Failed to get webhook by ID", "error", err)
			return nil, fmt.Errorf("failed to retrieve webhook: %w", err)
		}
		if webhook != nil && (!activeOnly || webhook.Active) {
			regs = []*store.WebhookRegistration{webhook}
		}
	} else {
		regs, err = s.webhookRepo.GetWebhooksByNamespace(ctx, namespace, activeOnly)
		if err != nil {
			s.logger.Error("Failed to get webhooks by namespace", "error", err)
			return nil, fmt.Errorf("failed to retrieve webhooks: %w", err)
		}
	}

	return regs, nil
}

// ListRegisteredWebhooksByEvent lists webhooks registered for a specific event
func (s *WebhookService) ListRegisteredWebhooksByEvent(ctx context.Context, namespace string, event string, activeOnly bool) ([]*store.WebhookRegistration, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListRegisteredWebhooksByEvent")
	defer span.End()

	s.logger.Info("Listing webhooks by event",
		"namespace", namespace,
		"event", event,
		"active_only", activeOnly)

	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if event == "" {
		return nil, fmt.Errorf("event is required")
	}

	allWebhooks, err := s.webhookRepo.GetWebhooksByEvent(ctx, namespace, event)
	if err != nil {
		s.logger.Error("Failed to get webhooks by event", "error", err)
		return nil, fmt.Errorf("failed to retrieve webhooks: %w", err)
	}

	var webhooks []*store.WebhookRegistration
	for _, wh := range allWebhooks {
		if !activeOnly || wh.Active {
			webhooks = append(webhooks, wh)
		}
	}

	return webhooks, nil
}

// GetWebhookDeliveryStatus gets the status of a webhook delivery
func (s *WebhookService) GetWebhookDeliveryStatus(ctx context.Context, deliveryID string, namespace string) (*store.WebhookDelivery, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetWebhookDeliveryStatus")
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

	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, deliveryID, namespace)
	if err != nil {
		s.logger.Error("Failed to get delivery by ID", "error", err)
		return nil, fmt.Errorf("failed to retrieve delivery status: %w", err)
	}
	if delivery == nil {
		return nil, fmt.Errorf("delivery not found")
	}
	return delivery, nil
}

// ResendWebhook resends a failed webhook delivery
func (s *WebhookService) ResendWebhook(ctx context.Context, deliveryID string, namespace string, forceResend bool) (string, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ResendWebhook")
	defer span.End()

	s.logger.Info("Resending webhook",
		"delivery_id", deliveryID,
		"namespace", namespace,
		"force_resend", forceResend)

	if deliveryID == "" {
		return "", fmt.Errorf("delivery ID is required")
	}

	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	// Get the original delivery
	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, deliveryID, namespace)
	if err != nil {
		s.logger.Error("Failed to get delivery", "error", err)
		return "", fmt.Errorf("failed to retrieve delivery: %w", err)
	}
	if delivery == nil {
		return "", fmt.Errorf("delivery not found")
	}
	if !forceResend && delivery.Status == store.StatusSuccess {
		return "", fmt.Errorf("delivery already succeeded. Use force_resend to resend anyway")
	}
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, delivery.WebhookID, namespace)
	if err != nil {
		s.logger.Error("Failed to get webhook", "error", err)
		return "", fmt.Errorf("failed to retrieve webhook: %w", err)
	}
	if webhook == nil {
		return "", fmt.Errorf("webhook not found")
	}
	if !webhook.Active {
		return "", fmt.Errorf("webhook is not active")
	}
	newDelivery := &store.WebhookDelivery{
		ID:          uuid.NewString(),
		WebhookID:   delivery.WebhookID,
		EventID:     delivery.EventID,
		Status:      store.StatusPending,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		MaxAttempts: delivery.MaxAttempts,
	}
	err = s.webhookRepo.CreateDelivery(ctx, newDelivery)
	if err != nil {
		s.logger.Error("Failed to create new delivery", "error", err)
		return "", fmt.Errorf("failed to create resend delivery: %w", err)
	}
	_, err = s.jobInserter.Insert(ctx, &queue.WebhookArgs{
		DeliveryID: newDelivery.ID,
		WebhookID:  newDelivery.WebhookID,
		URL:        webhook.URL,
		Headers:    webhook.Headers,
		Payload:    map[string]any{"TODO": "fetch original payload"}, // TODO: fetch original payload
		ExpiresAt:  newDelivery.ExpiresAt,
		Namespace:  webhook.Namespace,
	})
	if err != nil {
		s.logger.Error("Failed to queue webhook", "error", err)
		return "", fmt.Errorf("failed to queue webhook for delivery: %w", err)
	}
	s.logger.Info("Webhook resend queued successfully",
		"original_delivery_id", deliveryID,
		"new_delivery_id", newDelivery.ID)
	return newDelivery.ID, nil
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
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, webhookID, namespace)
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
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, webhookID, namespace)
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

// GetWebhookDeliveryHistory gets delivery history for a webhook
func (s *WebhookService) GetWebhookDeliveryHistory(ctx context.Context, webhookID string, namespace string, limit int32, offset int32) ([]*store.WebhookDelivery, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetWebhookDeliveryHistory")
	defer span.End()

	s.logger.Info("Getting webhook delivery history",
		"webhook_id", webhookID,
		"namespace", namespace,
		"limit", limit,
		"offset", offset)

	if webhookID == "" {
		return nil, 0, fmt.Errorf("webhook ID is required")
	}

	if namespace == "" {
		return nil, 0, fmt.Errorf("namespace is required")
	}

	// Set default pagination values
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	deliveries, totalCount, err := s.webhookRepo.GetDeliveriesByWebhookID(ctx, webhookID, namespace, int(limit), int(offset))
	if err != nil {
		s.logger.Error("Failed to get delivery history", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve delivery history: %w", err)
	}

	return deliveries, int32(totalCount), nil
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

// RegisterEvent registers a new event type
func (s *WebhookService) RegisterEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) (string, int64, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RegisterEvent")
	defer span.End()

	s.logger.Info("Processing event registration request",
		"name", name,
		"description", description)
	if name == "" {
		return "", 0, fmt.Errorf("event name is required")
	}
	existingEvent, err := s.webhookRepo.GetEventByName(ctx, name)
	if err != nil {
		s.logger.Error("Failed to check existing event", "error", err)
		return "", 0, fmt.Errorf("failed to check existing event: %w", err)
	}
	if existingEvent != nil {
		return "", 0, fmt.Errorf("event already exists")
	}
	event := &store.EventRegistration{
		Name:        name,
		Description: description,
		Schema:      schema,
		Metadata:    metadata,
		Active:      active,
	}
	if !active {
		event.Active = true
	}
	if err := s.webhookRepo.RegisterEvent(ctx, event); err != nil {
		s.logger.Error("Failed to register event",
			"name", name,
			"error", err,
		)
		return "", 0, fmt.Errorf("failed to register event: %w", err)
	}
	s.logger.Info("Event registered successfully",
		"event_id", event.ID,
		"name", name,
		"description", description,
	)
	return event.ID, event.CreatedAt.Unix(), nil
}

// ListEvents lists all registered events
func (s *WebhookService) ListEvents(ctx context.Context, activeOnly bool) ([]*store.EventRegistration, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEvents")
	defer span.End()

	s.logger.Info("Processing list events request",
		"active_only", activeOnly)
	events, err := s.webhookRepo.ListEvents(ctx, activeOnly)
	if err != nil {
		s.logger.Error("Failed to list events", "error", err)
		return nil, fmt.Errorf("failed to retrieve events: %w", err)
	}
	s.logger.Info("Listed events successfully",
		"total_count", len(events),
	)
	return events, nil
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

	// Get webhook to verify it exists and get current health
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, webhookID, namespace)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhook")
		s.logger.Error("Failed to get webhook", "error", err)
		return nil, fmt.Errorf("webhook not found: %w", err)
	}

	// Get health state (current status and consecutive failures)
	healthState, err := s.webhookRepo.GetWebhookHealthState(ctx, webhookID)
	if err != nil {
		// If no health state exists yet, return basic health info
		s.logger.Info("No health state found for webhook", "webhook_id", webhookID)
		return &WebhookHealthData{
			WebhookID: webhookID,
			Health:    webhook.Health,
		}, nil
	}

	// Get health summary for the last 24 hours
	healthSummary, err := s.webhookRepo.GetWebhookHealthSummary(ctx, webhookID, 24)
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
func (s *WebhookService) ListWebhooksByHealth(ctx context.Context, health store.WebhookHealth) ([]*store.WebhookRegistration, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListWebhooksByHealth")
	defer span.End()

	s.logger.Info("Processing list webhooks by health request", "health", health)

	// Get webhooks by health status
	webhooksList, err := s.webhookRepo.GetWebhooksByHealth(ctx, health)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhooks by health")
		s.logger.Error("Failed to get webhooks by health", "error", err)
		return nil, fmt.Errorf("failed to retrieve webhooks: %w", err)
	}

	s.logger.Info("Webhooks retrieved successfully",
		"health", health,
		"count", len(webhooksList))

	return webhooksList, nil
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

// ResubmitWebhook manually retries failed or pending webhook deliveries
func (s *WebhookService) ResubmitWebhook(ctx context.Context, deliveryID string, webhookID string, namespace string, force bool) ([]string, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ResubmitWebhook")
	defer span.End()

	s.logger.Info("Processing resubmit webhook request",
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
		// Resubmit specific delivery
		delivery, err := s.webhookRepo.GetDeliveryByID(ctx, deliveryID, namespace)
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
		// Resubmit all failed/pending deliveries for webhook
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, webhookID, namespace)
		if err != nil {
			s.logger.Error("Failed to get webhook", "error", err)
			return nil, 0, fmt.Errorf("failed to retrieve webhook: %w", err)
		}

		if webhook == nil {
			return nil, 0, fmt.Errorf("webhook not found")
		}

		// Get retriable deliveries
		deliveriesToResubmit, err = s.webhookRepo.GetRetriableDeliveries(ctx, webhookID, namespace, force)
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
			DeliveryID: delivery.ID,
			WebhookID:  delivery.WebhookID,
			URL:        webhook.URL,
			Headers:    webhook.Headers,
			Payload:    map[string]any{"TODO": "TODO"}, // Will be populated by the event data
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

		resubmittedIDs = append(resubmittedIDs, delivery.ID)
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

// GetNamespaceStats retrieves statistics for a namespace
func (s *WebhookService) GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStatsData, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetNamespaceStats")
	defer span.End()

	s.logger.Info("Processing get namespace stats request", "namespace", namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	allWebhooks, err := s.webhookRepo.ListWebhooks(ctx, namespace, false)
	if err != nil {
		s.logger.Error("Failed to get webhooks for stats", "error", err)
		return nil, err
	}
	activeWebhooks, err := s.webhookRepo.ListWebhooks(ctx, namespace, true)
	if err != nil {
		s.logger.Error("Failed to get active webhooks for stats", "error", err)
		return nil, err
	}
	totalDeliveries := 0
	successfulDeliveries := 0
	failedDeliveries := 0
	pendingDeliveries := 0
	for _, webhook := range allWebhooks {
		deliveries, err := s.webhookRepo.GetDeliveriesByWebhook(ctx, webhook.ID)
		if err != nil {
			continue
		}
		totalDeliveries += len(deliveries)
		for _, delivery := range deliveries {
			switch delivery.Status {
			case store.StatusSuccess:
				successfulDeliveries++
			case store.StatusFailed, store.StatusExpired:
				failedDeliveries++
			case store.StatusPending, store.StatusSending, store.StatusRetrying:
				pendingDeliveries++
			}
		}
	}
	var successRate float64
	if totalDeliveries > 0 {
		successRate = float64(successfulDeliveries) / float64(totalDeliveries)
	}
	stats := &NamespaceStatsData{
		TotalWebhooks:        len(allWebhooks),
		ActiveWebhooks:       len(activeWebhooks),
		TotalDeliveries:      totalDeliveries,
		SuccessfulDeliveries: successfulDeliveries,
		FailedDeliveries:     failedDeliveries,
		PendingDeliveries:    pendingDeliveries,
		SuccessRate:          successRate,
	}
	s.logger.Info("Namespace stats retrieved successfully",
		"namespace", namespace,
		"total_webhooks", stats.TotalWebhooks,
		"active_webhooks", stats.ActiveWebhooks,
		"success_rate", successRate)
	return stats, nil
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
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, webhookID, namespace)
	if err != nil {
		s.logger.Error("Failed to get webhook", "error", err)
		return fmt.Errorf("failed to retrieve webhook: %w", err)
	}
	if webhook == nil {
		return fmt.Errorf("webhook not found")
	}
	if len(events) > 0 {
		webhook.Events = events
	}
	if url != "" {
		webhook.URL = url
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
		s.logger.Error("Failed to list event reports",
			"namespace", namespace,
			"event_name", eventName,
			"error", err)
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
