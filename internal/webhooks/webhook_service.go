package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	jsonschema "github.com/kaptinlin/jsonschema"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/webhooks/jobs"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	pb "github.com/sarathsp06/sparrow/proto"
)

// WebhookService contains the core business logic for webhook operations
type WebhookService struct {
	queueManager *queue.Manager
	webhookRepo  *store.Repository
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
	Deliveries      []*store.WebhookDelivery
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
	Webhooks   []*store.WebhookRegistration
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
	Webhooks   []*store.WebhookRegistration
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
	Webhooks   []*store.WebhookRegistration
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
	Delivery *store.WebhookDelivery
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
	Deliveries []*store.WebhookDelivery
	TotalCount int32
	Success    bool
	Message    string
}

// NewWebhookService creates a new WebhookService instance
func NewWebhookService(queueManager *queue.Manager, webhookRepo *store.Repository) *WebhookService {
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
	registration := &store.WebhookRegistration{
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

	// Lookup registered event
	eventReg, err := s.webhookRepo.GetEventByName(ctx, req.Event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event lookup failed")
		s.logger.Error("Failed to lookup event registration", "event", req.Event, "error", err)
		return nil, fmt.Errorf("failed to lookup event registration: %w", err)
	}
	if eventReg == nil || !eventReg.Active {
		err := fmt.Errorf("event '%s' is not registered", req.Event)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event not registered")
		s.logger.Error("Event not registered", "event", req.Event)
		return nil, err
	}

	// Validate payload against event schema if present
	if eventReg.Schema != "" && req.Payload != "" {
		var payload interface{}
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "invalid JSON payload")
			return nil, fmt.Errorf("invalid JSON payload: %w", err)
		}
		// Validate against JSON schema
		if err := ValidateJSONSchema(eventReg.Schema, payload); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "payload does not match schema")
			s.logger.Error("Payload does not match event schema", "event", req.Event, "error", err)
			return nil, fmt.Errorf("payload does not match event schema: %w", err)
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

	var deliveries []*store.WebhookDelivery
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
	var filteredRegistrations []*store.WebhookRegistration
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

	var regs []*store.WebhookRegistration
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
			regs = []*store.WebhookRegistration{webhook}
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
	var webhooks []*store.WebhookRegistration
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
	if !req.ForceResend && delivery.Status == store.StatusSuccess {
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
	newDelivery := &store.WebhookDelivery{
		ID:          generateDeliveryID(),
		WebhookID:   delivery.WebhookID,
		EventID:     delivery.EventID,
		Status:      store.StatusPending,
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
	Events     []*store.EventRegistration
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

// GetWebhookHealthRequest represents a get webhook health request
type GetWebhookHealthRequest struct {
	WebhookID string
	Namespace string
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

// GetWebhookHealthResponse represents a get webhook health response
type GetWebhookHealthResponse struct {
	Success   bool                `json:"success"`
	Message   string              `json:"message"`
	WebhookID string              `json:"webhook_id"`
	Health    store.WebhookHealth `json:"health"`
	Metrics   *WebhookHealthData  `json:"metrics,omitempty"`
}

// ListWebhooksByHealthRequest represents a list webhooks by health request
type ListWebhooksByHealthRequest struct {
	Health store.WebhookHealth
}

// ListWebhooksByHealthResponse represents a list webhooks by health response
type ListWebhooksByHealthResponse struct {
	Success    bool                         `json:"success"`
	Message    string                       `json:"message"`
	Webhooks   []*store.WebhookRegistration `json:"webhooks"`
	TotalCount int32                        `json:"total_count"`
}

// GetHealthSummaryRequest represents a get health summary request
type GetHealthSummaryRequest struct {
	// No parameters needed
}

// HealthSummaryData represents health summary information
type HealthSummaryData struct {
	HealthyCount   int `json:"healthy_count"`
	DegradedCount  int `json:"degraded_count"`
	UnhealthyCount int `json:"unhealthy_count"`
	UnknownCount   int `json:"unknown_count"`
	TotalCount     int `json:"total_count"`
}

// GetHealthSummaryResponse represents a get health summary response
type GetHealthSummaryResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message"`
	Summary *HealthSummaryData `json:"summary"`
}

// ResubmitWebhookRequest represents a resubmit webhook request
type ResubmitWebhookRequest struct {
	DeliveryID string // Resubmit specific delivery (mutually exclusive with WebhookID)
	WebhookID  string // Resubmit all failed/pending deliveries for webhook (mutually exclusive with DeliveryID)
	Namespace  string // Namespace for authorization
	Force      bool   // Force retry even for non-failed deliveries
}

// ResubmitWebhookResponse represents a resubmit webhook response
type ResubmitWebhookResponse struct {
	Success          bool     `json:"success"`
	Message          string   `json:"message"`
	ResubmittedCount int32    `json:"resubmitted_count"`
	DeliveryIDs      []string `json:"delivery_ids"`
}

// GetNamespaceStatsRequest represents a get namespace stats request
type GetNamespaceStatsRequest struct {
	Namespace string
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

// GetNamespaceStatsResponse represents a get namespace stats response
type GetNamespaceStatsResponse struct {
	Success   bool                `json:"success"`
	Message   string              `json:"message"`
	Namespace string              `json:"namespace"`
	Stats     *NamespaceStatsData `json:"stats"`
}

// UpdateWebhookConfigRequest represents an update webhook config request
type UpdateWebhookConfigRequest struct {
	WebhookID string
	Namespace string
	Updates   *store.WebhookUpdateFields
}

// UpdateWebhookConfigResponse represents an update webhook config response
type UpdateWebhookConfigResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
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
	event := &store.EventRegistration{
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

// GetWebhookHealth retrieves health metrics for a webhook
func (s *WebhookService) GetWebhookHealth(ctx context.Context, req *GetWebhookHealthRequest) (*GetWebhookHealthResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetWebhookHealth")
	defer span.End()

	s.logger.Info("Processing get webhook health request",
		"webhook_id", req.WebhookID,
		"namespace", req.Namespace)

	// Validate required fields
	if req.WebhookID == "" {
		return &GetWebhookHealthResponse{
			Success: false,
			Message: "Webhook ID is required",
		}, nil
	}

	if req.Namespace == "" {
		return &GetWebhookHealthResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	// Get webhook to verify it exists and get current health
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, req.WebhookID, req.Namespace)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhook")
		s.logger.Error("Failed to get webhook", "error", err)
		return &GetWebhookHealthResponse{
			Success: false,
			Message: "Webhook not found",
		}, err
	}

	// Get health state (current status and consecutive failures)
	healthState, err := s.webhookRepo.GetWebhookHealthState(ctx, req.WebhookID)
	if err != nil {
		// If no health state exists yet, return basic health info
		s.logger.Info("No health state found for webhook", "webhook_id", req.WebhookID)
		return &GetWebhookHealthResponse{
			Success:   true,
			Message:   "Webhook health retrieved (no metrics available yet)",
			WebhookID: req.WebhookID,
			Health:    webhook.Health,
		}, nil
	}

	// Get health summary for the last 24 hours
	healthSummary, err := s.webhookRepo.GetWebhookHealthSummary(ctx, req.WebhookID, 24)
	if err != nil {
		s.logger.Error("Failed to get health summary", "error", err)
		// Continue with just the state info
	}

	// Convert to response format
	healthData := &WebhookHealthData{
		WebhookID:           req.WebhookID,
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

	var successRate float64
	if healthSummary != nil {
		successRate = healthSummary.SuccessRate
	}

	s.logger.Info("Webhook health retrieved successfully",
		"webhook_id", req.WebhookID,
		"health", webhook.Health,
		"success_rate", successRate)

	return &GetWebhookHealthResponse{
		Success:   true,
		Message:   "Webhook health retrieved successfully",
		WebhookID: req.WebhookID,
		Health:    webhook.Health,
		Metrics:   healthData,
	}, nil
}

// ListWebhooksByHealth retrieves webhooks filtered by health status
func (s *WebhookService) ListWebhooksByHealth(ctx context.Context, req *ListWebhooksByHealthRequest) (*ListWebhooksByHealthResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListWebhooksByHealth")
	defer span.End()

	s.logger.Info("Processing list webhooks by health request", "health", req.Health)

	// Get webhooks by health status
	webhooksList, err := s.webhookRepo.GetWebhooksByHealth(ctx, req.Health)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get webhooks by health")
		s.logger.Error("Failed to get webhooks by health", "error", err)
		return &ListWebhooksByHealthResponse{
			Success: false,
			Message: "Failed to retrieve webhooks",
		}, err
	}

	s.logger.Info("Webhooks retrieved successfully",
		"health", req.Health,
		"count", len(webhooksList))

	return &ListWebhooksByHealthResponse{
		Success:    true,
		Message:    "Webhooks retrieved successfully",
		Webhooks:   webhooksList,
		TotalCount: int32(len(webhooksList)),
	}, nil
}

// GetHealthSummary retrieves a summary of webhook health across all namespaces
func (s *WebhookService) GetHealthSummary(ctx context.Context, req *GetHealthSummaryRequest) (*GetHealthSummaryResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetHealthSummary")
	defer span.End()

	s.logger.Info("Processing get health summary request")

	// Get health summary from repository
	summary, err := s.webhookRepo.GetHealthSummary(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "Failed to get health summary")
		s.logger.Error("Failed to get health summary", "error", err)
		return &GetHealthSummaryResponse{
			Success: false,
			Message: "Failed to retrieve health summary",
		}, err
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

	return &GetHealthSummaryResponse{
		Success: true,
		Message: "Health summary retrieved successfully",
		Summary: healthSummary,
	}, nil
}

// ResubmitWebhook manually retries failed or pending webhook deliveries
func (s *WebhookService) ResubmitWebhook(ctx context.Context, req *ResubmitWebhookRequest) (*ResubmitWebhookResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ResubmitWebhook")
	defer span.End()

	s.logger.Info("Processing resubmit webhook request",
		"delivery_id", req.DeliveryID,
		"webhook_id", req.WebhookID,
		"namespace", req.Namespace,
		"force", req.Force)

	// Validate required fields
	if req.Namespace == "" {
		return &ResubmitWebhookResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	if req.DeliveryID == "" && req.WebhookID == "" {
		return &ResubmitWebhookResponse{
			Success: false,
			Message: "Either delivery_id or webhook_id is required",
		}, nil
	}

	if req.DeliveryID != "" && req.WebhookID != "" {
		return &ResubmitWebhookResponse{
			Success: false,
			Message: "Only one of delivery_id or webhook_id can be specified",
		}, nil
	}

	var deliveriesToResubmit []*store.WebhookDelivery

	if req.DeliveryID != "" {
		// Resubmit specific delivery
		delivery, err := s.webhookRepo.GetDeliveryByID(ctx, req.DeliveryID, req.Namespace)
		if err != nil {
			s.logger.Error("Failed to get delivery", "error", err)
			return &ResubmitWebhookResponse{
				Success: false,
				Message: "Failed to retrieve delivery",
			}, err
		}

		if delivery == nil {
			return &ResubmitWebhookResponse{
				Success: false,
				Message: "Delivery not found",
			}, nil
		}

		// Check if delivery can be resubmitted
		if !req.Force && delivery.Status == store.StatusSuccess {
			return &ResubmitWebhookResponse{
				Success: false,
				Message: "Delivery already succeeded. Use force to resubmit anyway",
			}, nil
		}

		deliveriesToResubmit = []*store.WebhookDelivery{delivery}
	} else {
		// Resubmit all failed/pending deliveries for webhook
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, req.WebhookID, req.Namespace)
		if err != nil {
			s.logger.Error("Failed to get webhook", "error", err)
			return &ResubmitWebhookResponse{
				Success: false,
				Message: "Failed to retrieve webhook",
			}, err
		}

		if webhook == nil {
			return &ResubmitWebhookResponse{
				Success: false,
				Message: "Webhook not found",
			}, nil
		}

		// Get retriable deliveries
		deliveriesToResubmit, err = s.webhookRepo.GetRetriableDeliveries(ctx, req.WebhookID, req.Namespace, req.Force)
		if err != nil {
			s.logger.Error("Failed to get retriable deliveries", "error", err)
			return &ResubmitWebhookResponse{
				Success: false,
				Message: "Failed to retrieve deliveries",
			}, err
		}

		if len(deliveriesToResubmit) == 0 {
			message := "No failed or pending deliveries found"
			if !req.Force {
				message += ". Use force to resubmit all deliveries"
			}
			return &ResubmitWebhookResponse{
				Success:          true,
				Message:          message,
				ResubmittedCount: 0,
			}, nil
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
		webhook, err := s.webhookRepo.GetWebhookByID(ctx, delivery.WebhookID, req.Namespace)
		if err != nil {
			s.logger.Error("Failed to get webhook for delivery",
				"webhook_id", delivery.WebhookID,
				"delivery_id", delivery.ID,
				"error", err)
			continue
		}

		// Queue the webhook for delivery
		err = s.queueManager.QueueWebhook(ctx, &jobs.WebhookArgs{
			DeliveryID: delivery.ID,
			WebhookID:  delivery.WebhookID,
			URL:        webhook.URL,
			Headers:    webhook.Headers,
			Payload:    "", // Will be populated by the event data
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
		return &ResubmitWebhookResponse{
			Success: false,
			Message: "Failed to resubmit any deliveries",
		}, nil
	}

	s.logger.Info("Webhook deliveries resubmitted successfully",
		"resubmitted_count", resubmittedCount,
		"total_requested", len(deliveriesToResubmit))

	return &ResubmitWebhookResponse{
		Success:          true,
		Message:          fmt.Sprintf("Successfully resubmitted %d deliveries", resubmittedCount),
		ResubmittedCount: resubmittedCount,
		DeliveryIDs:      resubmittedIDs,
	}, nil
}

// GetNamespaceStats retrieves statistics for a namespace
func (s *WebhookService) GetNamespaceStats(ctx context.Context, req *GetNamespaceStatsRequest) (*GetNamespaceStatsResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetNamespaceStats")
	defer span.End()

	s.logger.Info("Processing get namespace stats request", "namespace", req.Namespace)

	// Validate required fields
	if req.Namespace == "" {
		return &GetNamespaceStatsResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	// Get webhook counts
	allWebhooks, err := s.webhookRepo.ListWebhooks(ctx, req.Namespace, false)
	if err != nil {
		s.logger.Error("Failed to get webhooks for stats", "error", err)
		return &GetNamespaceStatsResponse{
			Success: false,
			Message: "Failed to retrieve webhook statistics",
		}, err
	}

	activeWebhooks, err := s.webhookRepo.ListWebhooks(ctx, req.Namespace, true)
	if err != nil {
		s.logger.Error("Failed to get active webhooks for stats", "error", err)
		return &GetNamespaceStatsResponse{
			Success: false,
			Message: "Failed to retrieve webhook statistics",
		}, err
	}

	// Calculate delivery stats (simplified - would need more complex queries for production)
	totalDeliveries := 0
	successfulDeliveries := 0
	failedDeliveries := 0
	pendingDeliveries := 0

	for _, webhook := range allWebhooks {
		deliveries, err := s.webhookRepo.GetDeliveriesByWebhook(ctx, webhook.ID)
		if err != nil {
			continue // Skip webhooks with delivery errors
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

	// Calculate success rate
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
		"namespace", req.Namespace,
		"total_webhooks", stats.TotalWebhooks,
		"active_webhooks", stats.ActiveWebhooks,
		"success_rate", successRate)

	return &GetNamespaceStatsResponse{
		Success:   true,
		Message:   "Namespace statistics retrieved successfully",
		Namespace: req.Namespace,
		Stats:     stats,
	}, nil
}

// UpdateWebhookConfig updates webhook configuration
func (s *WebhookService) UpdateWebhookConfig(ctx context.Context, req *UpdateWebhookConfigRequest) (*UpdateWebhookConfigResponse, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.UpdateWebhookConfig")
	defer span.End()

	s.logger.Info("Processing update webhook config request",
		"webhook_id", req.WebhookID,
		"namespace", req.Namespace)

	// Validate required fields
	if req.WebhookID == "" {
		return &UpdateWebhookConfigResponse{
			Success: false,
			Message: "Webhook ID is required",
		}, nil
	}

	if req.Namespace == "" {
		return &UpdateWebhookConfigResponse{
			Success: false,
			Message: "Namespace is required",
		}, nil
	}

	// Get the existing webhook
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, req.WebhookID, req.Namespace)
	if err != nil {
		s.logger.Error("Failed to get webhook", "error", err)
		return &UpdateWebhookConfigResponse{
			Success: false,
			Message: "Failed to retrieve webhook",
		}, err
	}

	if webhook == nil {
		return &UpdateWebhookConfigResponse{
			Success: false,
			Message: "Webhook not found",
		}, nil
	}

	// Apply updates if provided
	if req.Updates != nil {
		if len(req.Updates.Events) > 0 {
			webhook.Events = req.Updates.Events
		}
		if req.Updates.URL != "" {
			webhook.URL = req.Updates.URL
		}
		if req.Updates.Headers != nil {
			webhook.Headers = req.Updates.Headers
		}
		if req.Updates.Timeout > 0 {
			webhook.Timeout = req.Updates.Timeout
		}
		webhook.Active = req.Updates.Active
		if req.Updates.Description != "" {
			webhook.Description = req.Updates.Description
		}
	}

	// Update the webhook
	err = s.webhookRepo.UpdateWebhook(ctx, webhook)
	if err != nil {
		s.logger.Error("Failed to update webhook config",
			"webhook_id", req.WebhookID,
			"error", err)
		return &UpdateWebhookConfigResponse{
			Success: false,
			Message: "Failed to update webhook configuration",
		}, err
	}

	s.logger.Info("Webhook configuration updated successfully",
		"webhook_id", req.WebhookID)

	return &UpdateWebhookConfigResponse{
		Success: true,
		Message: "Webhook configuration updated successfully",
	}, nil
}

// Helper function to generate delivery ID
func generateDeliveryID() string {
	return uuid.New().String()
}

// ============================================================================
// VALIDATION METHODS
// ============================================================================

// Regular expressions for validation
var (
	namespaceRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	eventRegex     = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// ValidateNamespace validates namespace field (required for most operations)
func (s *WebhookService) ValidateNamespace(namespace string) *ValidationError {
	if namespace == "" {
		return &ValidationError{
			Field:   "namespace",
			Message: MsgRequired,
			Code:    ErrorCodeRequired,
		}
	}

	if len(namespace) > 64 {
		return &ValidationError{
			Field:   "namespace",
			Message: MsgNamespaceTooLong,
			Code:    ErrorCodeTooLong,
		}
	}

	if !namespaceRegex.MatchString(namespace) {
		return &ValidationError{
			Field:   "namespace",
			Message: MsgInvalidNamespace,
			Code:    ErrorCodeInvalidFormat,
		}
	}

	return nil
}

// ValidateURL validates webhook URL
func (s *WebhookService) ValidateURL(urlStr string) *ValidationError {
	if urlStr == "" {
		return &ValidationError{
			Field:   "url",
			Message: MsgRequired,
			Code:    ErrorCodeRequired,
		}
	}

	u, err := url.Parse(urlStr)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return &ValidationError{
			Field:   "url",
			Message: MsgInvalidURL,
			Code:    ErrorCodeInvalidFormat,
		}
	}

	return nil
}

// ValidateEvents validates event names array
func (s *WebhookService) ValidateEvents(events []string) *ValidationErrors {
	var errors ValidationErrors

	if len(events) == 0 {
		errors.Add("events", MsgRequired, ErrorCodeRequired)
		return &errors
	}

	if len(events) > 50 {
		errors.Add("events", MsgTooManyEvents, ErrorCodeTooLong)
		return &errors
	}

	seen := make(map[string]bool)
	for i, event := range events {
		field := fmt.Sprintf("events[%d]", i)

		if event == "" {
			errors.Add(field, MsgRequired, ErrorCodeRequired)
			continue
		}

		if !eventRegex.MatchString(event) {
			errors.Add(field, MsgInvalidEvent, ErrorCodeInvalidFormat)
			continue
		}

		if seen[event] {
			errors.Add(field, "duplicate event name", ErrorCodeDuplicate)
			continue
		}

		seen[event] = true
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidateTimeout validates timeout value
func (s *WebhookService) ValidateTimeout(timeout int32) *ValidationError {
	if timeout < 1 || timeout > 300 {
		return &ValidationError{
			Field:   "timeout",
			Message: MsgInvalidTimeout,
			Code:    ErrorCodeInvalid,
		}
	}

	return nil
}

// ValidateHeaders validates HTTP headers
func (s *WebhookService) ValidateHeaders(headers map[string]string) *ValidationError {
	if headers == nil {
		return nil
	}

	totalSize := 0
	for key, value := range headers {
		totalSize += len(key) + len(value) + 4 // key: value\r\n
	}

	if totalSize > 8*1024 { // 8KB limit
		return &ValidationError{
			Field:   "headers",
			Message: MsgHeadersTooLarge,
			Code:    ErrorCodeTooLong,
		}
	}

	return nil
}

// ValidateJSON validates that a string is valid JSON
func (s *WebhookService) ValidateJSON(jsonStr string) *ValidationError {
	if jsonStr == "" {
		return nil
	}

	var js json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &js); err != nil {
		return &ValidationError{
			Field:   "payload",
			Message: MsgInvalidJSON,
			Code:    ErrorCodeInvalidFormat,
		}
	}

	if len(jsonStr) > 1024*1024 { // 1MB limit
		return &ValidationError{
			Field:   "payload",
			Message: MsgPayloadTooLarge,
			Code:    ErrorCodeTooLong,
		}
	}

	return nil
}

// ValidateDescription validates description field
func (s *WebhookService) ValidateDescription(description string) *ValidationError {
	if len(description) > 500 {
		return &ValidationError{
			Field:   "description",
			Message: MsgDescriptionTooLong,
			Code:    ErrorCodeTooLong,
		}
	}

	return nil
}

// ValidateEventName validates a single event name
func (s *WebhookService) ValidateEventName(event string) *ValidationError {
	if event == "" {
		return &ValidationError{
			Field:   "event",
			Message: MsgRequired,
			Code:    ErrorCodeRequired,
		}
	}

	if !eventRegex.MatchString(event) {
		return &ValidationError{
			Field:   "event",
			Message: MsgInvalidEvent,
			Code:    ErrorCodeInvalidFormat,
		}
	}

	return nil
}

// ValidateWebhookID validates webhook ID format
func (s *WebhookService) ValidateWebhookID(webhookID string) *ValidationError {
	if webhookID == "" {
		return &ValidationError{
			Field:   "webhook_id",
			Message: MsgRequired,
			Code:    ErrorCodeRequired,
		}
	}

	// Assuming UUIDs or similar format
	if len(webhookID) < 10 || len(webhookID) > 64 {
		return &ValidationError{
			Field:   "webhook_id",
			Message: "webhook ID must be between 10 and 64 characters",
			Code:    ErrorCodeInvalidFormat,
		}
	}

	return nil
}

// ValidateDeliveryID validates delivery ID format
func (s *WebhookService) ValidateDeliveryID(deliveryID string) *ValidationError {
	if deliveryID == "" {
		return &ValidationError{
			Field:   "delivery_id",
			Message: MsgRequired,
			Code:    ErrorCodeRequired,
		}
	}

	// Assuming UUIDs or similar format
	if len(deliveryID) < 10 || len(deliveryID) > 64 {
		return &ValidationError{
			Field:   "delivery_id",
			Message: "delivery ID must be between 10 and 64 characters",
			Code:    ErrorCodeInvalidFormat,
		}
	}

	return nil
}

// ValidateDateRange validates start and end date parameters
func (s *WebhookService) ValidateDateRange(startDate, endDate string) *ValidationErrors {
	var errors ValidationErrors

	if startDate != "" {
		if _, err := time.Parse(time.RFC3339, startDate); err != nil {
			errors.Add("start_date", "must be a valid RFC3339 timestamp", ErrorCodeInvalidFormat)
		}
	}

	if endDate != "" {
		if _, err := time.Parse(time.RFC3339, endDate); err != nil {
			errors.Add("end_date", "must be a valid RFC3339 timestamp", ErrorCodeInvalidFormat)
		}
	}

	if startDate != "" && endDate != "" {
		start, startErr := time.Parse(time.RFC3339, startDate)
		end, endErr := time.Parse(time.RFC3339, endDate)

		if startErr == nil && endErr == nil && start.After(end) {
			errors.Add("start_date", "start date must be before end date", ErrorCodeInvalid)
		}
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidateLimit validates pagination limit
func (s *WebhookService) ValidateLimit(limit int32) *ValidationError {
	if limit < 1 || limit > 1000 {
		return &ValidationError{
			Field:   "limit",
			Message: "limit must be between 1 and 1000",
			Code:    ErrorCodeInvalid,
		}
	}

	return nil
}

// ValidateOffset validates pagination offset
func (s *WebhookService) ValidateOffset(offset int32) *ValidationError {
	if offset < 0 {
		return &ValidationError{
			Field:   "offset",
			Message: "offset must be non-negative",
			Code:    ErrorCodeInvalid,
		}
	}

	return nil
}

// ValidateStatus validates delivery status
func (s *WebhookService) ValidateStatus(status string) *ValidationError {
	if status == "" {
		return nil // Status is optional in filters
	}

	validStatuses := map[string]bool{
		"pending":   true,
		"success":   true,
		"failed":    true,
		"retrying":  true,
		"cancelled": true,
	}

	if !validStatuses[strings.ToLower(status)] {
		return &ValidationError{
			Field:   "status",
			Message: "status must be one of: pending, success, failed, retrying, cancelled",
			Code:    ErrorCodeInvalid,
		}
	}

	return nil
}

// ============================================================================
// REQUEST VALIDATION METHODS
// ============================================================================

// ValidateRegisterWebhookRequest validates RegisterWebhookRequest
func (s *WebhookService) ValidateRegisterWebhookRequest(req *pb.RegisterWebhookRequest) *ValidationErrors {
	var errors ValidationErrors

	// Namespace is required
	if err := s.ValidateNamespace(req.GetNamespace()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// URL is required
	if err := s.ValidateURL(req.GetUrl()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// Events are required
	if eventsErr := s.ValidateEvents(req.GetEvents()); eventsErr != nil {
		errors.Errors = append(errors.Errors, eventsErr.Errors...)
	}

	// Optional fields
	if req.GetTimeout() > 0 {
		if err := s.ValidateTimeout(req.GetTimeout()); err != nil {
			errors.Errors = append(errors.Errors, *err)
		}
	}

	if err := s.ValidateHeaders(req.GetHeaders()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	if err := s.ValidateDescription(req.GetDescription()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidateUnregisterWebhookRequest validates UnregisterWebhookRequest
func (s *WebhookService) ValidateUnregisterWebhookRequest(req *pb.UnregisterWebhookRequest) *ValidationErrors {
	var errors ValidationErrors

	// Webhook ID is required
	if err := s.ValidateWebhookID(req.GetWebhookId()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidatePushEventRequest validates PushEventRequest
func (s *WebhookService) ValidatePushEventRequest(req *pb.PushEventRequest) *ValidationErrors {
	var errors ValidationErrors

	// Namespace is required
	if err := s.ValidateNamespace(req.GetNamespace()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// Event is required
	if err := s.ValidateEventName(req.GetEvent()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// Validate payload if provided
	if req.GetPayload() != "" {
		if err := s.ValidateJSON(req.GetPayload()); err != nil {
			errors.Errors = append(errors.Errors, *err)
		}
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidateGetWebhookStatusRequest validates GetWebhookStatusRequest
func (s *WebhookService) ValidateGetWebhookStatusRequest(req *pb.GetWebhookStatusRequest) *ValidationErrors {
	var errors ValidationErrors

	// Namespace is required
	if err := s.ValidateNamespace(req.GetNamespace()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// Either webhook_id or event_id should be provided
	if req.GetWebhookId() == "" && req.GetEventId() == "" {
		errors.Add("identifier", "either webhook_id or event_id must be provided", ErrorCodeRequired)
	}

	// If webhook_id is provided, validate it
	if req.GetWebhookId() != "" {
		if err := s.ValidateWebhookID(req.GetWebhookId()); err != nil {
			errors.Errors = append(errors.Errors, *err)
		}
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidateListWebhooksRequest validates ListWebhooksRequest
func (s *WebhookService) ValidateListWebhooksRequest(req *pb.ListWebhooksRequest) *ValidationErrors {
	var errors ValidationErrors

	// Namespace is required
	if err := s.ValidateNamespace(req.GetNamespace()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// Optional event filter validation
	if req.GetEvent() != "" {
		if err := s.ValidateEventName(req.GetEvent()); err != nil {
			errors.Errors = append(errors.Errors, *err)
		}
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidateGetWebhookHealthRequest validates GetWebhookHealthRequest
func (s *WebhookService) ValidateGetWebhookHealthRequest(req *pb.GetWebhookHealthRequest) *ValidationErrors {
	var errors ValidationErrors

	// Namespace is required
	if err := s.ValidateNamespace(req.GetNamespace()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// Webhook ID is required
	if err := s.ValidateWebhookID(req.GetWebhookId()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidateResubmitWebhookRequest validates ResubmitWebhookRequest
func (s *WebhookService) ValidateResubmitWebhookRequest(req *pb.ResubmitWebhookRequest) *ValidationErrors {
	var errors ValidationErrors

	// Namespace is required
	if err := s.ValidateNamespace(req.GetNamespace()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// Delivery ID is required
	if err := s.ValidateDeliveryID(req.GetDeliveryId()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidateGetNamespaceStatsRequest validates GetNamespaceStatsRequest
func (s *WebhookService) ValidateGetNamespaceStatsRequest(req *pb.GetNamespaceStatsRequest) *ValidationErrors {
	var errors ValidationErrors

	// Namespace is required
	if err := s.ValidateNamespace(req.GetNamespace()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidateUpdateWebhookConfigRequest validates UpdateWebhookConfigRequest
func (s *WebhookService) ValidateUpdateWebhookConfigRequest(req *pb.UpdateWebhookConfigRequest) *ValidationErrors {
	var errors ValidationErrors

	// Namespace is required
	if err := s.ValidateNamespace(req.GetNamespace()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// Webhook ID is required
	if err := s.ValidateWebhookID(req.GetWebhookId()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ValidatePauseWebhookRequest validates PauseWebhookRequest
func (s *WebhookService) ValidatePauseWebhookRequest(req *pb.PauseWebhookRequest) *ValidationErrors {
	var errors ValidationErrors

	// Namespace is required
	if err := s.ValidateNamespace(req.GetNamespace()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	// Webhook ID is required
	if err := s.ValidateWebhookID(req.GetWebhookId()); err != nil {
		errors.Errors = append(errors.Errors, *err)
	}

	if errors.HasErrors() {
		return &errors
	}

	return nil
}

// ============================================================================
// VALIDATION HELPER METHODS
// ============================================================================

// ValidateAndConvertToError validates using a validation function and converts to a regular error
func (s *WebhookService) ValidateAndConvertToError(validationFunc func() *ValidationErrors) error {
	if validationErrors := validationFunc(); validationErrors != nil {
		return fmt.Errorf("validation failed: %s", validationErrors.Error())
	}
	return nil
}

// ValidateJSONSchema validates a payload against a JSON schema string
func ValidateJSONSchema(schema string, payload interface{}) error {
	compiler := jsonschema.NewCompiler()
	sch, err := compiler.Compile([]byte(schema))
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
