package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sarathsp06/sparrow/internal/jobs"
	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks"
)

// WebhookService contains the core business logic for webhook operations
type WebhookService struct {
	queueManager *queue.Manager
	webhookRepo  *webhooks.Repository
	logger       *slog.Logger
	tracer       trace.Tracer
	metrics      *observability.SparrowMetrics
}

// RegisterWebhookRequest represents a webhook registration request
type RegisterWebhookRequest struct {
	Namespace   string
	Events      []string
	URL         string
	Headers     map[string]string
	Timeout     int32
	Active      bool
	Description string
}

// RegisterWebhookResponse represents a webhook registration response
type RegisterWebhookResponse struct {
	WebhookID string
	Success   bool
	Message   string
	CreatedAt int64
}

// UnregisterWebhookRequest represents a webhook unregistration request
type UnregisterWebhookRequest struct {
	WebhookID string
}

// UnregisterWebhookResponse represents a webhook unregistration response
type UnregisterWebhookResponse struct {
	Success bool
	Message string
}

// PushEventRequest represents an event push request
type PushEventRequest struct {
	Namespace  string
	Event      string
	Payload    string
	TTLSeconds int64
	Metadata   map[string]string
}

// PushEventResponse represents an event push response
type PushEventResponse struct {
	EventID           string
	WebhooksTriggered int32
	WebhookIDs        []string
	Success           bool
	Message           string
}

// GetWebhookStatusRequest represents a webhook status request
type GetWebhookStatusRequest struct {
	WebhookID string
	EventID   string
}

// GetWebhookStatusResponse represents a webhook status response
type GetWebhookStatusResponse struct {
	Deliveries      []*webhooks.WebhookDelivery
	TotalDeliveries int32
	Success         bool
	Message         string
}

// ListWebhooksRequest represents a list webhooks request
type ListWebhooksRequest struct {
	Namespace  string
	Event      string
	ActiveOnly bool
}

// ListWebhooksResponse represents a list webhooks response
type ListWebhooksResponse struct {
	Webhooks   []*webhooks.WebhookRegistration
	TotalCount int32
	Success    bool
	Message    string
}

// GetRegisteredWebhooksRequest represents a get registered webhooks request
type GetRegisteredWebhooksRequest struct {
	WebhookID  string
	Namespace  string
	ActiveOnly bool
}

// GetRegisteredWebhooksResponse represents a get registered webhooks response
type GetRegisteredWebhooksResponse struct {
	Webhooks   []*webhooks.WebhookRegistration
	TotalCount int32
	Success    bool
	Message    string
}

// ListRegisteredWebhooksByEventRequest represents a list webhooks by event request
type ListRegisteredWebhooksByEventRequest struct {
	Namespace  string
	Event      string
	ActiveOnly bool
}

// ListRegisteredWebhooksByEventResponse represents a list webhooks by event response
type ListRegisteredWebhooksByEventResponse struct {
	Webhooks   []*webhooks.WebhookRegistration
	Event      string
	Namespace  string
	TotalCount int32
	Success    bool
	Message    string
}

// GetWebhookDeliveryStatusRequest represents a get delivery status request
type GetWebhookDeliveryStatusRequest struct {
	DeliveryID string
	Namespace  string
}

// GetWebhookDeliveryStatusResponse represents a get delivery status response
type GetWebhookDeliveryStatusResponse struct {
	Delivery *webhooks.WebhookDelivery
	Success  bool
	Message  string
}

// ResendWebhookRequest represents a resend webhook request
type ResendWebhookRequest struct {
	DeliveryID  string
	Namespace   string
	ForceResend bool
}

// ResendWebhookResponse represents a resend webhook response
type ResendWebhookResponse struct {
	NewDeliveryID string
	Success       bool
	Message       string
}

// PauseWebhookRequest represents a pause webhook request
type PauseWebhookRequest struct {
	WebhookID string
	Namespace string
	Reason    string
}

// PauseWebhookResponse represents a pause webhook response
type PauseWebhookResponse struct {
	Success bool
	Message string
}

// ResumeWebhookRequest represents a resume webhook request
type ResumeWebhookRequest struct {
	WebhookID string
	Namespace string
}

// ResumeWebhookResponse represents a resume webhook response
type ResumeWebhookResponse struct {
	Success bool
	Message string
}

// GetWebhookDeliveryHistoryRequest represents a get delivery history request
type GetWebhookDeliveryHistoryRequest struct {
	WebhookID string
	Namespace string
	Limit     int32
	Offset    int32
}

// GetWebhookDeliveryHistoryResponse represents a get delivery history response
type GetWebhookDeliveryHistoryResponse struct {
	Deliveries []*webhooks.WebhookDelivery
	TotalCount int32
	Success    bool
	Message    string
}

// NewWebhookService creates a new WebhookService instance
func NewWebhookService(queueManager *queue.Manager, webhookRepo *webhooks.Repository) *WebhookService {
	metrics, err := observability.NewSparrowMetrics()
	if err != nil {
		// Log error but continue without metrics
		log := logger.NewLogger("webhook-service")
		log.Error("Failed to initialize metrics", "error", err)
	}

	return &WebhookService{
		queueManager: queueManager,
		webhookRepo:  webhookRepo,
		logger:       logger.NewLogger("webhook-service"),
		tracer:       observability.GetTracer("sparrow.service.webhook"),
		metrics:      metrics,
	}
}

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookService) RegisterWebhook(ctx context.Context, req *RegisterWebhookRequest) (*RegisterWebhookResponse, error) {
	ctx, span := s.tracer.Start(ctx, "webhook.register",
		trace.WithAttributes(
			attribute.String("namespace", req.Namespace),
			attribute.StringSlice("events", req.Events),
			attribute.String("url", req.URL),
		),
	)
	defer span.End()

	s.logger.Info("Processing webhook registration request",
		"namespace", req.Namespace,
		"events", req.Events,
		"url", req.URL,
	)

	// Validate required fields
	if req.Namespace == "" {
		err := fmt.Errorf("namespace is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "namespace is required")
		return nil, err
	}
	if len(req.Events) == 0 {
		err := fmt.Errorf("at least one event is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "at least one event is required")
		return nil, err
	}
	if req.URL == "" {
		err := fmt.Errorf("URL is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "URL is required")
		return nil, err
	}

	// Validate events are not empty
	for _, event := range req.Events {
		if event == "" {
			err := fmt.Errorf("event names cannot be empty")
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "event names cannot be empty")
			return nil, err
		}
	}

	// Set default timeout
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	span.SetAttributes(attribute.Int("timeout", int(timeout)))

	// Create webhook registration
	registration := &webhooks.WebhookRegistration{
		Namespace:   req.Namespace,
		Events:      req.Events,
		URL:         req.URL,
		Headers:     req.Headers,
		Timeout:     int(timeout),
		Active:      req.Active,
		Description: req.Description,
	}

	// Store the registration
	if err := s.webhookRepo.RegisterWebhook(ctx, registration); err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to register webhook")
		s.logger.Error("Failed to register webhook",
			"namespace", req.Namespace,
			"events", req.Events,
			"url", req.URL,
			"error", err,
		)
		return nil, fmt.Errorf("failed to register webhook: %w", err)
	}

	// Record metrics
	if s.metrics != nil {
		s.metrics.WebhookRegistrations.Add(ctx, 1)
		s.metrics.ActiveWebhooks.Add(ctx, 1)
	}

	span.SetAttributes(attribute.String("webhook_id", registration.ID))
	span.SetStatus(otelcodes.Ok, "webhook registered successfully")

	s.logger.Info("Webhook registered successfully",
		"webhook_id", registration.ID,
		"namespace", req.Namespace,
		"events", req.Events,
		"url", req.URL,
	)

	return &RegisterWebhookResponse{
		WebhookID: registration.ID,
		Success:   true,
		Message:   "Webhook registered successfully",
		CreatedAt: registration.CreatedAt.Unix(),
	}, nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookService) UnregisterWebhook(ctx context.Context, req *UnregisterWebhookRequest) (*UnregisterWebhookResponse, error) {
	s.logger.Info("Processing webhook unregistration request",
		"webhook_id", req.WebhookID,
	)

	if req.WebhookID == "" {
		return nil, fmt.Errorf("webhook_id is required")
	}

	// Remove the registration
	if err := s.webhookRepo.UnregisterWebhook(ctx, req.WebhookID); err != nil {
		s.logger.Error("Failed to unregister webhook",
			"webhook_id", req.WebhookID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to unregister webhook: %w", err)
	}

	s.logger.Info("Webhook unregistered successfully",
		"webhook_id", req.WebhookID,
	)

	return &UnregisterWebhookResponse{
		Success: true,
		Message: "Webhook unregistered successfully",
	}, nil
}

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookService) PushEvent(ctx context.Context, req *PushEventRequest) (*PushEventResponse, error) {
	ctx, span := s.tracer.Start(ctx, "event.push",
		trace.WithAttributes(
			attribute.String("namespace", req.Namespace),
			attribute.String("event", req.Event),
		),
	)
	defer span.End()

	s.logger.Info("Processing push event request",
		"namespace", req.Namespace,
		"event", req.Event,
	)

	// Validate required fields
	if req.Namespace == "" {
		err := fmt.Errorf("namespace is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "namespace is required")
		return nil, err
	}
	if req.Event == "" {
		err := fmt.Errorf("event is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event is required")
		return nil, err
	}

	// Validate JSON payload
	if req.Payload != "" {
		var payload interface{}
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "invalid JSON payload")
			return nil, fmt.Errorf("invalid JSON payload: %w", err)
		}
	}

	// Set default TTL if not provided
	ttl := req.TTLSeconds
	if ttl <= 0 {
		ttl = 3600 // Default 1 hour
	}

	// Generate event ID
	eventID := uuid.New().String()

	// Create event processing job
	eventArgs := jobs.EventArgs{
		EventID:    eventID,
		Namespace:  req.Namespace,
		Event:      req.Event,
		Payload:    req.Payload,
		TTLSeconds: ttl,
		Metadata:   req.Metadata,
		CreatedAt:  time.Now(),
	}

	// Find registered webhooks first to know how many will be triggered
	registeredWebhooks, err := s.webhookRepo.GetWebhooksByEvent(ctx, req.Namespace, req.Event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to get registered webhooks")
		s.logger.Error("Failed to get registered webhooks",
			"namespace", req.Namespace,
			"event", req.Event,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get registered webhooks: %w", err)
	}

	span.SetAttributes(
		attribute.String("event_id", eventID),
		attribute.Int("webhooks_count", len(registeredWebhooks)),
	)

	webhookIDs := make([]string, len(registeredWebhooks))
	for i, wh := range registeredWebhooks {
		webhookIDs[i] = wh.ID
	}

	// Insert the event processing job
	_, err = s.queueManager.GetClient().Insert(ctx, eventArgs, &river.InsertOpts{
		Queue: "events",
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to schedule event processing")
		s.logger.Error("Failed to schedule event processing job",
			"event_id", eventID,
			"namespace", req.Namespace,
			"event", req.Event,
			"error", err,
		)
		return nil, fmt.Errorf("failed to schedule event processing: %w", err)
	}

	// Record metrics
	if s.metrics != nil {
		s.metrics.EventsPushed.Add(ctx, 1)
	}

	span.SetStatus(otelcodes.Ok, "event scheduled successfully")

	s.logger.Info("Event processing scheduled successfully",
		"event_id", eventID,
		"namespace", req.Namespace,
		"event", req.Event,
		"webhooks_to_trigger", len(registeredWebhooks),
	)

	return &PushEventResponse{
		EventID:           eventID,
		WebhooksTriggered: int32(len(registeredWebhooks)),
		WebhookIDs:        webhookIDs,
		Success:           true,
		Message:           fmt.Sprintf("Event scheduled for processing, %d webhooks will be triggered", len(registeredWebhooks)),
	}, nil
}

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookService) GetWebhookStatus(ctx context.Context, req *GetWebhookStatusRequest) (*GetWebhookStatusResponse, error) {
	s.logger.Info("Processing webhook status request")

	var deliveries []*webhooks.WebhookDelivery
	var err error

	if req.WebhookID != "" {
		deliveries, err = s.webhookRepo.GetDeliveriesByWebhook(ctx, req.WebhookID)
	} else if req.EventID != "" {
		deliveries, err = s.webhookRepo.GetDeliveriesByEvent(ctx, req.EventID)
	} else {
		return nil, fmt.Errorf("either webhook_id or event_id is required")
	}

	if err != nil {
		s.logger.Error("Failed to get webhook deliveries", "error", err)
		return nil, fmt.Errorf("failed to get webhook status: %w", err)
	}

	return &GetWebhookStatusResponse{
		Deliveries:      deliveries,
		TotalDeliveries: int32(len(deliveries)),
		Success:         true,
		Message:         fmt.Sprintf("Found %d webhook deliveries", len(deliveries)),
	}, nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookService) ListWebhooks(ctx context.Context, req *ListWebhooksRequest) (*ListWebhooksResponse, error) {
	s.logger.Info("Processing list webhooks request",
		"namespace", req.Namespace,
		"event", req.Event,
		"active_only", req.ActiveOnly,
	)

	if req.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	// Get webhooks from repository
	registrations, err := s.webhookRepo.ListWebhooks(ctx, req.Namespace, req.ActiveOnly)
	if err != nil {
		s.logger.Error("Failed to list webhooks",
			"namespace", req.Namespace,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	// Filter by event if specified
	var filteredRegistrations []*webhooks.WebhookRegistration
	if req.Event != "" {
		for _, reg := range registrations {
			// Check if the webhook listens to the requested event
			for _, event := range reg.Events {
				if event == req.Event {
					filteredRegistrations = append(filteredRegistrations, reg)
					break
				}
			}
		}
	} else {
		filteredRegistrations = registrations
	}

	s.logger.Info("Listed webhooks successfully",
		"namespace", req.Namespace,
		"total_count", len(filteredRegistrations),
	)

	return &ListWebhooksResponse{
		Webhooks:   filteredRegistrations,
		TotalCount: int32(len(filteredRegistrations)),
		Success:    true,
		Message:    fmt.Sprintf("Found %d webhooks", len(filteredRegistrations)),
	}, nil
}

// GetRegisteredWebhooks gets registered webhooks by namespace and optional webhook ID
func (s *WebhookService) GetRegisteredWebhooks(ctx context.Context, req *GetRegisteredWebhooksRequest) (*GetRegisteredWebhooksResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetRegisteredWebhooks")
	defer span.End()

	s.logger.Info("Getting registered webhooks",
		"namespace", req.Namespace,
		"webhook_id", req.WebhookID,
		"active_only", req.ActiveOnly)

	if req.Namespace == "" {
		return &GetRegisteredWebhooksResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	var regs []*webhooks.WebhookRegistration
	var err error

	if req.WebhookID != "" {
		// Get specific webhook
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, req.WebhookID, req.Namespace)
		if err != nil {
			s.logger.Error("Failed to get webhook by ID", "error", err)
			return &GetRegisteredWebhooksResponse{
				Success: false,
				Message: "Failed to retrieve webhook",
			}, err
		}
		if webhook != nil && (!req.ActiveOnly || webhook.Active) {
			regs = []*webhooks.WebhookRegistration{webhook}
		}
	} else {
		// Get all webhooks for namespace
		regs, err = s.webhookRepo.GetWebhooksByNamespace(ctx, req.Namespace, req.ActiveOnly)
		if err != nil {
			s.logger.Error("Failed to get webhooks by namespace", "error", err)
			return &GetRegisteredWebhooksResponse{
				Success: false,
				Message: "Failed to retrieve webhooks",
			}, err
		}
	}

	return &GetRegisteredWebhooksResponse{
		Webhooks:   regs,
		TotalCount: int32(len(regs)),
		Success:    true,
		Message:    "Webhooks retrieved successfully",
	}, nil
}

// ListRegisteredWebhooksByEvent lists webhooks registered for a specific event
func (s *WebhookService) ListRegisteredWebhooksByEvent(ctx context.Context, req *ListRegisteredWebhooksByEventRequest) (*ListRegisteredWebhooksByEventResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListRegisteredWebhooksByEvent")
	defer span.End()

	s.logger.Info("Listing webhooks by event",
		"namespace", req.Namespace,
		"event", req.Event,
		"active_only", req.ActiveOnly)

	if req.Namespace == "" {
		return &ListRegisteredWebhooksByEventResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	if req.Event == "" {
		return &ListRegisteredWebhooksByEventResponse{
			Success: false,
			Message: "Event is required",
		}, nil
	}

	allWebhooks, err := s.webhookRepo.GetWebhooksByEvent(ctx, req.Namespace, req.Event)
	if err != nil {
		s.logger.Error("Failed to get webhooks by event", "error", err)
		return &ListRegisteredWebhooksByEventResponse{
			Success: false,
			Message: "Failed to retrieve webhooks",
		}, err
	}

	// Filter by active status if requested
	var webhooks []*webhooks.WebhookRegistration
	for _, wh := range allWebhooks {
		if !req.ActiveOnly || wh.Active {
			webhooks = append(webhooks, wh)
		}
	}

	return &ListRegisteredWebhooksByEventResponse{
		Webhooks:   webhooks,
		Event:      req.Event,
		Namespace:  req.Namespace,
		TotalCount: int32(len(webhooks)),
		Success:    true,
		Message:    "Webhooks retrieved successfully",
	}, nil
}

// GetWebhookDeliveryStatus gets the status of a webhook delivery
func (s *WebhookService) GetWebhookDeliveryStatus(ctx context.Context, req *GetWebhookDeliveryStatusRequest) (*GetWebhookDeliveryStatusResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetWebhookDeliveryStatus")
	defer span.End()

	s.logger.Info("Getting webhook delivery status",
		"delivery_id", req.DeliveryID,
		"namespace", req.Namespace)

	if req.DeliveryID == "" {
		return &GetWebhookDeliveryStatusResponse{
			Success: false,
			Message: "Delivery ID is required",
		}, nil
	}

	if req.Namespace == "" {
		return &GetWebhookDeliveryStatusResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, req.DeliveryID, req.Namespace)
	if err != nil {
		s.logger.Error("Failed to get delivery by ID", "error", err)
		return &GetWebhookDeliveryStatusResponse{
			Success: false,
			Message: "Failed to retrieve delivery status",
		}, err
	}

	if delivery == nil {
		return &GetWebhookDeliveryStatusResponse{
			Success: false,
			Message: "Delivery not found",
		}, nil
	}

	return &GetWebhookDeliveryStatusResponse{
		Delivery: delivery,
		Success:  true,
		Message:  "Delivery status retrieved successfully",
	}, nil
}

// ResendWebhook resends a failed webhook delivery
func (s *WebhookService) ResendWebhook(ctx context.Context, req *ResendWebhookRequest) (*ResendWebhookResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ResendWebhook")
	defer span.End()

	s.logger.Info("Resending webhook",
		"delivery_id", req.DeliveryID,
		"namespace", req.Namespace,
		"force_resend", req.ForceResend)

	if req.DeliveryID == "" {
		return &ResendWebhookResponse{
			Success: false,
			Message: "Delivery ID is required",
		}, nil
	}

	if req.Namespace == "" {
		return &ResendWebhookResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	// Get the original delivery
	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, req.DeliveryID, req.Namespace)
	if err != nil {
		s.logger.Error("Failed to get delivery", "error", err)
		return &ResendWebhookResponse{
			Success: false,
			Message: "Failed to retrieve delivery",
		}, err
	}

	if delivery == nil {
		return &ResendWebhookResponse{
			Success: false,
			Message: "Delivery not found",
		}, nil
	}

	// Check if delivery can be resent
	if !req.ForceResend && delivery.Status == webhooks.StatusSuccess {
		return &ResendWebhookResponse{
			Success: false,
			Message: "Delivery already succeeded. Use force_resend to resend anyway",
		}, nil
	}

	// Get the webhook to check if it's still active
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, delivery.WebhookID, req.Namespace)
	if err != nil {
		s.logger.Error("Failed to get webhook", "error", err)
		return &ResendWebhookResponse{
			Success: false,
			Message: "Failed to retrieve webhook",
		}, err
	}

	if webhook == nil {
		return &ResendWebhookResponse{
			Success: false,
			Message: "Webhook not found",
		}, nil
	}

	if !webhook.Active {
		return &ResendWebhookResponse{
			Success: false,
			Message: "Webhook is not active",
		}, nil
	}

	// Create new delivery record
	newDelivery := &webhooks.WebhookDelivery{
		ID:          generateDeliveryID(),
		WebhookID:   delivery.WebhookID,
		EventID:     delivery.EventID,
		Status:      webhooks.StatusPending,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour), // 24 hour TTL
		MaxAttempts: delivery.MaxAttempts,
	}

	// Save new delivery
	err = s.webhookRepo.CreateDelivery(ctx, newDelivery)
	if err != nil {
		s.logger.Error("Failed to create new delivery", "error", err)
		return &ResendWebhookResponse{
			Success: false,
			Message: "Failed to create resend delivery",
		}, err
	}

	// Queue the webhook for delivery - get URL and headers from the webhook registration
	err = s.queueManager.QueueWebhook(ctx, &jobs.WebhookArgs{
		DeliveryID: newDelivery.ID,
		WebhookID:  newDelivery.WebhookID,
		URL:        webhook.URL,
		Headers:    webhook.Headers,
		Payload:    "", // Will be populated by the event data
		ExpiresAt:  newDelivery.ExpiresAt,
		Namespace:  webhook.Namespace,
	})
	if err != nil {
		s.logger.Error("Failed to queue webhook", "error", err)
		return &ResendWebhookResponse{
			Success: false,
			Message: "Failed to queue webhook for delivery",
		}, err
	}

	s.logger.Info("Webhook resend queued successfully",
		"original_delivery_id", req.DeliveryID,
		"new_delivery_id", newDelivery.ID)

	return &ResendWebhookResponse{
		NewDeliveryID: newDelivery.ID,
		Success:       true,
		Message:       "Webhook resend queued successfully",
	}, nil
}

// PauseWebhook temporarily disables webhook deliveries
func (s *WebhookService) PauseWebhook(ctx context.Context, req *PauseWebhookRequest) (*PauseWebhookResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.PauseWebhook")
	defer span.End()

	s.logger.Info("Pausing webhook",
		"webhook_id", req.WebhookID,
		"namespace", req.Namespace,
		"reason", req.Reason)

	if req.WebhookID == "" {
		return &PauseWebhookResponse{
			Success: false,
			Message: "Webhook ID is required",
		}, nil
	}

	if req.Namespace == "" {
		return &PauseWebhookResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	// Get the webhook
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, req.WebhookID, req.Namespace)
	if err != nil {
		s.logger.Error("Failed to get webhook", "error", err)
		return &PauseWebhookResponse{
			Success: false,
			Message: "Failed to retrieve webhook",
		}, err
	}

	if webhook == nil {
		return &PauseWebhookResponse{
			Success: false,
			Message: "Webhook not found",
		}, nil
	}

	if !webhook.Active {
		return &PauseWebhookResponse{
			Success: false,
			Message: "Webhook is already paused",
		}, nil
	}

	// Update webhook to inactive
	webhook.Active = false
	webhook.UpdatedAt = time.Now()

	err = s.webhookRepo.UpdateWebhook(ctx, webhook)
	if err != nil {
		s.logger.Error("Failed to pause webhook", "error", err)
		return &PauseWebhookResponse{
			Success: false,
			Message: "Failed to pause webhook",
		}, err
	}

	s.logger.Info("Webhook paused successfully", "webhook_id", req.WebhookID)

	return &PauseWebhookResponse{
		Success: true,
		Message: "Webhook paused successfully",
	}, nil
}

// ResumeWebhook re-enables webhook deliveries
func (s *WebhookService) ResumeWebhook(ctx context.Context, req *ResumeWebhookRequest) (*ResumeWebhookResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ResumeWebhook")
	defer span.End()

	s.logger.Info("Resuming webhook",
		"webhook_id", req.WebhookID,
		"namespace", req.Namespace)

	if req.WebhookID == "" {
		return &ResumeWebhookResponse{
			Success: false,
			Message: "Webhook ID is required",
		}, nil
	}

	if req.Namespace == "" {
		return &ResumeWebhookResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	// Get the webhook
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, req.WebhookID, req.Namespace)
	if err != nil {
		s.logger.Error("Failed to get webhook", "error", err)
		return &ResumeWebhookResponse{
			Success: false,
			Message: "Failed to retrieve webhook",
		}, err
	}

	if webhook == nil {
		return &ResumeWebhookResponse{
			Success: false,
			Message: "Webhook not found",
		}, nil
	}

	if webhook.Active {
		return &ResumeWebhookResponse{
			Success: false,
			Message: "Webhook is already active",
		}, nil
	}

	// Update webhook to active
	webhook.Active = true
	webhook.UpdatedAt = time.Now()

	err = s.webhookRepo.UpdateWebhook(ctx, webhook)
	if err != nil {
		s.logger.Error("Failed to resume webhook", "error", err)
		return &ResumeWebhookResponse{
			Success: false,
			Message: "Failed to resume webhook",
		}, err
	}

	s.logger.Info("Webhook resumed successfully", "webhook_id", req.WebhookID)

	return &ResumeWebhookResponse{
		Success: true,
		Message: "Webhook resumed successfully",
	}, nil
}

// GetWebhookDeliveryHistory gets delivery history for a webhook
func (s *WebhookService) GetWebhookDeliveryHistory(ctx context.Context, req *GetWebhookDeliveryHistoryRequest) (*GetWebhookDeliveryHistoryResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetWebhookDeliveryHistory")
	defer span.End()

	s.logger.Info("Getting webhook delivery history",
		"webhook_id", req.WebhookID,
		"namespace", req.Namespace,
		"limit", req.Limit,
		"offset", req.Offset)

	if req.WebhookID == "" {
		return &GetWebhookDeliveryHistoryResponse{
			Success: false,
			Message: "Webhook ID is required",
		}, nil
	}

	if req.Namespace == "" {
		return &GetWebhookDeliveryHistoryResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	// Set default pagination values
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	deliveries, totalCount, err := s.webhookRepo.GetDeliveriesByWebhookID(ctx, req.WebhookID, req.Namespace, int(limit), int(offset))
	if err != nil {
		s.logger.Error("Failed to get delivery history", "error", err)
		return &GetWebhookDeliveryHistoryResponse{
			Success: false,
			Message: "Failed to retrieve delivery history",
		}, err
	}

	return &GetWebhookDeliveryHistoryResponse{
		Deliveries: deliveries,
		TotalCount: int32(totalCount),
		Success:    true,
		Message:    "Delivery history retrieved successfully",
	}, nil
}

// RegisterEventRequest represents a register event request
type RegisterEventRequest struct {
	Name        string
	Description string
	Schema      string
	Metadata    map[string]string
	Active      bool
}

// RegisterEventResponse represents a register event response
type RegisterEventResponse struct {
	EventID   string
	Success   bool
	Message   string
	CreatedAt int64
}

// ListEventsRequest represents a list events request
type ListEventsRequest struct {
	ActiveOnly bool
}

// ListEventsResponse represents a list events response
type ListEventsResponse struct {
	Events     []*webhooks.EventRegistration
	TotalCount int32
	Success    bool
	Message    string
}

// UpdateEventRequest represents an update event request
type UpdateEventRequest struct {
	Name        string
	Description string
	Schema      string
	Metadata    map[string]string
	Active      bool
}

// UpdateEventResponse represents an update event response
type UpdateEventResponse struct {
	Success bool
	Message string
}

// DeleteEventRequest represents a delete event request
type DeleteEventRequest struct {
	Name string
}

// DeleteEventResponse represents a delete event response
type DeleteEventResponse struct {
	Success bool
	Message string
}

// RegisterEvent registers a new event type
func (s *WebhookService) RegisterEvent(ctx context.Context, req *RegisterEventRequest) (*RegisterEventResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RegisterEvent")
	defer span.End()

	s.logger.Info("Processing event registration request",
		"name", req.Name,
		"description", req.Description)

	// Validate required fields
	if req.Name == "" {
		return &RegisterEventResponse{
			Success: false,
			Message: "Event name is required",
		}, nil
	}

	// Check if event already exists
	existingEvent, err := s.webhookRepo.GetEventByName(ctx, req.Name)
	if err != nil {
		s.logger.Error("Failed to check existing event", "error", err)
		return &RegisterEventResponse{
			Success: false,
			Message: "Failed to check existing event",
		}, err
	}

	if existingEvent != nil {
		return &RegisterEventResponse{
			Success: false,
			Message: "Event already exists",
		}, nil
	}

	// Create event registration
	event := &webhooks.EventRegistration{
		Name:        req.Name,
		Description: req.Description,
		Schema:      req.Schema,
		Metadata:    req.Metadata,
		Active:      req.Active,
	}

	// Default to active if not specified
	if !req.Active {
		event.Active = true
	}

	// Store the event registration
	if err := s.webhookRepo.RegisterEvent(ctx, event); err != nil {
		s.logger.Error("Failed to register event",
			"name", req.Name,
			"error", err,
		)
		return &RegisterEventResponse{
			Success: false,
			Message: "Failed to register event",
		}, err
	}

	s.logger.Info("Event registered successfully",
		"event_id", event.ID,
		"name", req.Name,
		"description", req.Description,
	)

	return &RegisterEventResponse{
		EventID:   event.ID,
		Success:   true,
		Message:   "Event registered successfully",
		CreatedAt: event.CreatedAt.Unix(),
	}, nil
}

// ListEvents lists all registered events
func (s *WebhookService) ListEvents(ctx context.Context, req *ListEventsRequest) (*ListEventsResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEvents")
	defer span.End()

	s.logger.Info("Processing list events request",
		"active_only", req.ActiveOnly)

	// Get events from repository
	events, err := s.webhookRepo.ListEvents(ctx, req.ActiveOnly)
	if err != nil {
		s.logger.Error("Failed to list events", "error", err)
		return &ListEventsResponse{
			Success: false,
			Message: "Failed to retrieve events",
		}, err
	}

	s.logger.Info("Listed events successfully",
		"total_count", len(events),
	)

	return &ListEventsResponse{
		Events:     events,
		TotalCount: int32(len(events)),
		Success:    true,
		Message:    fmt.Sprintf("Found %d events", len(events)),
	}, nil
}

// UpdateEvent updates an event registration
func (s *WebhookService) UpdateEvent(ctx context.Context, req *UpdateEventRequest) (*UpdateEventResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.UpdateEvent")
	defer span.End()

	s.logger.Info("Processing event update request",
		"name", req.Name,
		"description", req.Description)

	// Validate required fields
	if req.Name == "" {
		return &UpdateEventResponse{
			Success: false,
			Message: "Event name is required",
		}, nil
	}

	// Check if event exists
	existingEvent, err := s.webhookRepo.GetEventByName(ctx, req.Name)
	if err != nil {
		s.logger.Error("Failed to get event", "error", err)
		return &UpdateEventResponse{
			Success: false,
			Message: "Failed to retrieve event",
		}, err
	}

	if existingEvent == nil {
		return &UpdateEventResponse{
			Success: false,
			Message: "Event not found",
		}, nil
	}

	// Update event fields
	existingEvent.Description = req.Description
	existingEvent.Schema = req.Schema
	existingEvent.Metadata = req.Metadata
	existingEvent.Active = req.Active

	// Update the event
	if err := s.webhookRepo.UpdateEvent(ctx, existingEvent); err != nil {
		s.logger.Error("Failed to update event",
			"name", req.Name,
			"error", err,
		)
		return &UpdateEventResponse{
			Success: false,
			Message: "Failed to update event",
		}, err
	}

	s.logger.Info("Event updated successfully", "name", req.Name)

	return &UpdateEventResponse{
		Success: true,
		Message: "Event updated successfully",
	}, nil
}

// DeleteEvent deletes an event registration
func (s *WebhookService) DeleteEvent(ctx context.Context, req *DeleteEventRequest) (*DeleteEventResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.DeleteEvent")
	defer span.End()

	s.logger.Info("Processing event deletion request", "name", req.Name)

	// Validate required fields
	if req.Name == "" {
		return &DeleteEventResponse{
			Success: false,
			Message: "Event name is required",
		}, nil
	}

	// Check if event exists
	existingEvent, err := s.webhookRepo.GetEventByName(ctx, req.Name)
	if err != nil {
		s.logger.Error("Failed to get event", "error", err)
		return &DeleteEventResponse{
			Success: false,
			Message: "Failed to retrieve event",
		}, err
	}

	if existingEvent == nil {
		return &DeleteEventResponse{
			Success: false,
			Message: "Event not found",
		}, nil
	}

	// Delete the event
	if err := s.webhookRepo.DeleteEvent(ctx, req.Name); err != nil {
		s.logger.Error("Failed to delete event",
			"name", req.Name,
			"error", err,
		)
		return &DeleteEventResponse{
			Success: false,
			Message: "Failed to delete event",
		}, err
	}

	s.logger.Info("Event deleted successfully", "name", req.Name)

	return &DeleteEventResponse{
		Success: true,
		Message: "Event deleted successfully",
	}, nil
}

// Helper function to generate delivery ID
func generateDeliveryID() string {
	return uuid.New().String()
}
