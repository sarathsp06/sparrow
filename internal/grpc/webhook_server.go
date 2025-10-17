package grpc

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

	"github.com/sarathsp06/httpqueue/internal/jobs"
	"github.com/sarathsp06/httpqueue/internal/logger"
	"github.com/sarathsp06/httpqueue/internal/observability"
	"github.com/sarathsp06/httpqueue/internal/queue"
	"github.com/sarathsp06/httpqueue/internal/webhooks"
	pb "github.com/sarathsp06/httpqueue/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WebhookServer implements the WebhookService gRPC interface
type WebhookServer struct {
	pb.UnimplementedWebhookServiceServer
	queueManager *queue.Manager
	webhookRepo  *webhooks.Repository
	logger       *slog.Logger
	tracer       trace.Tracer
	metrics      *observability.HTTPQueueMetrics
}

// NewWebhookServer creates a new WebhookServer instance
func NewWebhookServer(queueManager *queue.Manager, webhookRepo *webhooks.Repository) *WebhookServer {
	metrics, err := observability.NewHTTPQueueMetrics()
	if err != nil {
		// Log error but continue without metrics
		log := logger.NewLogger("grpc-webhook-server")
		log.Error("Failed to initialize metrics", "error", err)
	}

	return &WebhookServer{
		queueManager: queueManager,
		webhookRepo:  webhookRepo,
		logger:       logger.NewLogger("grpc-webhook-server"),
		tracer:       observability.GetTracer("httpqueue.grpc.webhook"),
		metrics:      metrics,
	}
}

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookServer) RegisterWebhook(ctx context.Context, req *pb.RegisterWebhookRequest) (*pb.RegisterWebhookResponse, error) {
	ctx, span := s.tracer.Start(ctx, "webhook.register",
		trace.WithAttributes(
			attribute.String("namespace", req.Namespace),
			attribute.StringSlice("events", req.Events),
			attribute.String("url", req.Url),
		),
	)
	defer span.End()

	s.logger.Info("Received webhook registration request",
		"namespace", req.Namespace,
		"events", req.Events,
		"url", req.Url,
	)

	// Validate required fields
	if req.Namespace == "" {
		span.RecordError(fmt.Errorf("namespace is required"))
		span.SetStatus(otelcodes.Error, "namespace is required")
		return nil, status.Error(codes.InvalidArgument, "namespace is required")
	}
	if len(req.Events) == 0 {
		span.RecordError(fmt.Errorf("at least one event is required"))
		span.SetStatus(otelcodes.Error, "at least one event is required")
		return nil, status.Error(codes.InvalidArgument, "at least one event is required")
	}
	if req.Url == "" {
		span.RecordError(fmt.Errorf("URL is required"))
		span.SetStatus(otelcodes.Error, "URL is required")
		return nil, status.Error(codes.InvalidArgument, "URL is required")
	}

	// Validate events are not empty
	for _, event := range req.Events {
		if event == "" {
			span.RecordError(fmt.Errorf("event names cannot be empty"))
			span.SetStatus(otelcodes.Error, "event names cannot be empty")
			return nil, status.Error(codes.InvalidArgument, "event names cannot be empty")
		}
	}

	// Set default timeout
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	span.SetAttributes(attribute.Int("timeout", int(timeout)))

	// Create webhook registration (method is always POST)
	registration := &webhooks.WebhookRegistration{
		Namespace:   req.Namespace,
		Events:      req.Events,
		URL:         req.Url,
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
			"url", req.Url,
			"error", err,
		)
		return nil, status.Errorf(codes.Internal, "failed to register webhook: %v", err)
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
		"url", req.Url,
	)

	return &pb.RegisterWebhookResponse{
		WebhookId: registration.ID,
		Success:   true,
		Message:   "Webhook registered successfully",
		CreatedAt: registration.CreatedAt.Unix(),
	}, nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookServer) UnregisterWebhook(ctx context.Context, req *pb.UnregisterWebhookRequest) (*pb.UnregisterWebhookResponse, error) {
	s.logger.Info("Received webhook unregistration request",
		"webhook_id", req.WebhookId,
	)

	if req.WebhookId == "" {
		return nil, status.Error(codes.InvalidArgument, "webhook_id is required")
	}

	// Remove the registration
	if err := s.webhookRepo.UnregisterWebhook(ctx, req.WebhookId); err != nil {
		s.logger.Error("Failed to unregister webhook",
			"webhook_id", req.WebhookId,
			"error", err,
		)
		return nil, status.Errorf(codes.Internal, "failed to unregister webhook: %v", err)
	}

	s.logger.Info("Webhook unregistered successfully",
		"webhook_id", req.WebhookId,
	)

	return &pb.UnregisterWebhookResponse{
		Success: true,
		Message: "Webhook unregistered successfully",
	}, nil
}

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookServer) PushEvent(ctx context.Context, req *pb.PushEventRequest) (*pb.PushEventResponse, error) {
	ctx, span := s.tracer.Start(ctx, "event.push",
		trace.WithAttributes(
			attribute.String("namespace", req.Namespace),
			attribute.String("event", req.Event),
		),
	)
	defer span.End()

	s.logger.Info("Received push event request",
		"namespace", req.Namespace,
		"event", req.Event,
	)

	// Validate required fields
	if req.Namespace == "" {
		span.RecordError(fmt.Errorf("namespace is required"))
		span.SetStatus(otelcodes.Error, "namespace is required")
		return nil, status.Error(codes.InvalidArgument, "namespace is required")
	}
	if req.Event == "" {
		span.RecordError(fmt.Errorf("event is required"))
		span.SetStatus(otelcodes.Error, "event is required")
		return nil, status.Error(codes.InvalidArgument, "event is required")
	}

	// Validate JSON payload
	if req.Payload != "" {
		var payload interface{}
		if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "invalid JSON payload")
			return nil, status.Errorf(codes.InvalidArgument, "invalid JSON payload: %v", err)
		}
	}

	// Set default TTL if not provided
	ttl := req.TtlSeconds
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
		return nil, status.Errorf(codes.Internal, "failed to get registered webhooks: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to schedule event processing: %v", err)
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

	return &pb.PushEventResponse{
		EventId:           eventID,
		WebhooksTriggered: int32(len(registeredWebhooks)),
		WebhookIds:        webhookIDs,
		Success:           true,
		Message:           fmt.Sprintf("Event scheduled for processing, %d webhooks will be triggered", len(registeredWebhooks)),
	}, nil
}

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookServer) GetWebhookStatus(ctx context.Context, req *pb.GetWebhookStatusRequest) (*pb.GetWebhookStatusResponse, error) {
	s.logger.Info("Received webhook status request")

	var deliveries []*webhooks.WebhookDelivery
	var err error

	switch id := req.Identifier.(type) {
	case *pb.GetWebhookStatusRequest_WebhookId:
		if id.WebhookId == "" {
			return nil, status.Error(codes.InvalidArgument, "webhook_id is required")
		}
		deliveries, err = s.webhookRepo.GetDeliveriesByWebhook(ctx, id.WebhookId)
	case *pb.GetWebhookStatusRequest_EventId:
		if id.EventId == "" {
			return nil, status.Error(codes.InvalidArgument, "event_id is required")
		}
		deliveries, err = s.webhookRepo.GetDeliveriesByEvent(ctx, id.EventId)
	default:
		return nil, status.Error(codes.InvalidArgument, "either webhook_id or event_id is required")
	}

	if err != nil {
		s.logger.Error("Failed to get webhook deliveries", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get webhook status: %v", err)
	}

	// Convert to protobuf format
	pbDeliveries := make([]*pb.WebhookDelivery, len(deliveries))
	for i, d := range deliveries {
		pbDeliveries[i] = &pb.WebhookDelivery{
			DeliveryId:   d.ID,
			WebhookId:    d.WebhookID,
			EventId:      d.EventID,
			Status:       convertDeliveryStatus(d.Status),
			AttemptCount: int32(d.AttemptCount),
			MaxAttempts:  int32(d.MaxAttempts),
			CreatedAt:    d.CreatedAt.Unix(),
			ExpiresAt:    d.ExpiresAt.Unix(),
			ResponseCode: int32(d.ResponseCode),
			ResponseBody: d.ResponseBody,
			ErrorMessage: d.ErrorMessage,
		}

		if d.LastAttemptedAt != nil {
			pbDeliveries[i].LastAttemptedAt = d.LastAttemptedAt.Unix()
		}
		if d.NextRetryAt != nil {
			pbDeliveries[i].NextRetryAt = d.NextRetryAt.Unix()
		}
	}

	return &pb.GetWebhookStatusResponse{
		Deliveries:      pbDeliveries,
		TotalDeliveries: int32(len(deliveries)),
		Success:         true,
		Message:         fmt.Sprintf("Found %d webhook deliveries", len(deliveries)),
	}, nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookServer) ListWebhooks(ctx context.Context, req *pb.ListWebhooksRequest) (*pb.ListWebhooksResponse, error) {
	s.logger.Info("Received list webhooks request",
		"namespace", req.Namespace,
		"event", req.Event,
		"active_only", req.ActiveOnly,
	)

	if req.Namespace == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace is required")
	}

	// Get webhooks from repository
	registrations, err := s.webhookRepo.ListWebhooks(ctx, req.Namespace, req.ActiveOnly)
	if err != nil {
		s.logger.Error("Failed to list webhooks",
			"namespace", req.Namespace,
			"error", err,
		)
		return nil, status.Errorf(codes.Internal, "failed to list webhooks: %v", err)
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

	// Convert to protobuf format
	pbWebhooks := make([]*pb.RegisteredWebhook, len(filteredRegistrations))
	for i, reg := range filteredRegistrations {
		pbWebhooks[i] = &pb.RegisteredWebhook{
			WebhookId:   reg.ID,
			Namespace:   reg.Namespace,
			Events:      reg.Events,
			Url:         reg.URL,
			Headers:     reg.Headers,
			Timeout:     int32(reg.Timeout),
			Active:      reg.Active,
			Description: reg.Description,
			CreatedAt:   reg.CreatedAt.Unix(),
			UpdatedAt:   reg.UpdatedAt.Unix(),
		}
	}

	s.logger.Info("Listed webhooks successfully",
		"namespace", req.Namespace,
		"total_count", len(pbWebhooks),
	)

	return &pb.ListWebhooksResponse{
		Webhooks:   pbWebhooks,
		TotalCount: int32(len(pbWebhooks)),
		Success:    true,
		Message:    fmt.Sprintf("Found %d webhooks", len(pbWebhooks)),
	}, nil
}

// Helper function to convert delivery status
func convertDeliveryStatus(status webhooks.WebhookDeliveryStatus) pb.WebhookDeliveryStatus {
	switch status {
	case webhooks.StatusPending:
		return pb.WebhookDeliveryStatus_DELIVERY_PENDING
	case webhooks.StatusSending:
		return pb.WebhookDeliveryStatus_DELIVERY_SENDING
	case webhooks.StatusSuccess:
		return pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	case webhooks.StatusFailed:
		return pb.WebhookDeliveryStatus_DELIVERY_FAILED
	case webhooks.StatusRetrying:
		return pb.WebhookDeliveryStatus_DELIVERY_RETRYING
	case webhooks.StatusExpired:
		return pb.WebhookDeliveryStatus_DELIVERY_EXPIRED
	default:
		return pb.WebhookDeliveryStatus_DELIVERY_UNKNOWN
	}
}
