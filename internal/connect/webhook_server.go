package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
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
	pb "github.com/sarathsp06/sparrow/proto"
	"github.com/sarathsp06/sparrow/proto/protoconnect"
)

// WebhookConnectServer implements the WebhookService Connect-RPC interface
type WebhookConnectServer struct {
	queueManager *queue.Manager
	webhookRepo  *webhooks.Repository
	logger       *slog.Logger
	tracer       trace.Tracer
	metrics      *observability.SparrowMetrics
}

// NewWebhookConnectServer creates a new Connect-RPC server instance
func NewWebhookConnectServer(queueManager *queue.Manager, webhookRepo *webhooks.Repository) *WebhookConnectServer {
	metrics, err := observability.NewSparrowMetrics()
	if err != nil {
		// Log error but continue without metrics
		log := logger.NewLogger("connect-webhook-server")
		log.Error("Failed to initialize metrics", "error", err)
	}

	return &WebhookConnectServer{
		queueManager: queueManager,
		webhookRepo:  webhookRepo,
		logger:       logger.NewLogger("connect-webhook-server"),
		tracer:       observability.GetTracer("sparrow.connect.webhook"),
		metrics:      metrics,
	}
}

// RegisterWebhook registers a URL for specific events in a namespace
func (s *WebhookConnectServer) RegisterWebhook(
	ctx context.Context,
	req *connect.Request[pb.RegisterWebhookRequest],
) (*connect.Response[pb.RegisterWebhookResponse], error) {
	ctx, span := s.tracer.Start(ctx, "connect.webhook.register",
		trace.WithAttributes(
			attribute.String("namespace", req.Msg.Namespace),
			attribute.StringSlice("events", req.Msg.Events),
			attribute.String("url", req.Msg.Url),
		),
	)
	defer span.End()

	s.logger.Info("Connect: Received webhook registration request",
		"namespace", req.Msg.Namespace,
		"events", req.Msg.Events,
		"url", req.Msg.Url,
	)

	// Validate required fields
	if req.Msg.Namespace == "" {
		span.RecordError(fmt.Errorf("namespace is required"))
		span.SetStatus(otelcodes.Error, "namespace is required")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("namespace is required"))
	}
	if len(req.Msg.Events) == 0 {
		span.RecordError(fmt.Errorf("at least one event is required"))
		span.SetStatus(otelcodes.Error, "at least one event is required")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("at least one event is required"))
	}
	if req.Msg.Url == "" {
		span.RecordError(fmt.Errorf("URL is required"))
		span.SetStatus(otelcodes.Error, "URL is required")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("URL is required"))
	}

	// Validate events are not empty
	for _, event := range req.Msg.Events {
		if event == "" {
			span.RecordError(fmt.Errorf("event names cannot be empty"))
			span.SetStatus(otelcodes.Error, "event names cannot be empty")
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("event names cannot be empty"))
		}
	}

	// Set default timeout
	timeout := req.Msg.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	span.SetAttributes(attribute.Int("timeout", int(timeout)))

	// Create webhook registration
	registration := &webhooks.WebhookRegistration{
		Namespace:   req.Msg.Namespace,
		Events:      req.Msg.Events,
		URL:         req.Msg.Url,
		Headers:     req.Msg.Headers,
		Timeout:     int(timeout),
		Active:      req.Msg.Active,
		Description: req.Msg.Description,
	}

	// Store the registration
	if err := s.webhookRepo.RegisterWebhook(ctx, registration); err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to register webhook")
		s.logger.Error("Failed to register webhook",
			"namespace", req.Msg.Namespace,
			"events", req.Msg.Events,
			"url", req.Msg.Url,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register webhook: %w", err))
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
		"namespace", req.Msg.Namespace,
		"events", req.Msg.Events,
		"url", req.Msg.Url,
	)

	result := &pb.RegisterWebhookResponse{
		WebhookId: registration.ID,
		Success:   true,
		Message:   "Webhook registered successfully",
		CreatedAt: registration.CreatedAt.Unix(),
	}

	return connect.NewResponse(result), nil
}

// UnregisterWebhook removes a webhook registration
func (s *WebhookConnectServer) UnregisterWebhook(
	ctx context.Context,
	req *connect.Request[pb.UnregisterWebhookRequest],
) (*connect.Response[pb.UnregisterWebhookResponse], error) {
	ctx, span := s.tracer.Start(ctx, "connect.webhook.unregister")
	defer span.End()

	s.logger.Info("Connect: Received webhook unregistration request",
		"webhook_id", req.Msg.WebhookId,
	)

	if req.Msg.WebhookId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("webhook_id is required"))
	}

	// Remove the registration
	if err := s.webhookRepo.UnregisterWebhook(ctx, req.Msg.WebhookId); err != nil {
		s.logger.Error("Failed to unregister webhook",
			"webhook_id", req.Msg.WebhookId,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unregister webhook: %w", err))
	}

	s.logger.Info("Webhook unregistered successfully",
		"webhook_id", req.Msg.WebhookId,
	)

	result := &pb.UnregisterWebhookResponse{
		Success: true,
		Message: "Webhook unregistered successfully",
	}

	return connect.NewResponse(result), nil
}

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookConnectServer) PushEvent(
	ctx context.Context,
	req *connect.Request[pb.PushEventRequest],
) (*connect.Response[pb.PushEventResponse], error) {
	ctx, span := s.tracer.Start(ctx, "connect.event.push",
		trace.WithAttributes(
			attribute.String("namespace", req.Msg.Namespace),
			attribute.String("event", req.Msg.Event),
		),
	)
	defer span.End()

	s.logger.Info("Connect: Received push event request",
		"namespace", req.Msg.Namespace,
		"event", req.Msg.Event,
	)

	// Validate required fields
	if req.Msg.Namespace == "" {
		span.RecordError(fmt.Errorf("namespace is required"))
		span.SetStatus(otelcodes.Error, "namespace is required")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("namespace is required"))
	}
	if req.Msg.Event == "" {
		span.RecordError(fmt.Errorf("event is required"))
		span.SetStatus(otelcodes.Error, "event is required")
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("event is required"))
	}

	// Validate JSON payload
	if req.Msg.Payload != "" {
		var payload interface{}
		if err := json.Unmarshal([]byte(req.Msg.Payload), &payload); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "invalid JSON payload")
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid JSON payload: %w", err))
		}
	}

	// Set default TTL if not provided
	ttl := req.Msg.TtlSeconds
	if ttl <= 0 {
		ttl = 3600 // Default 1 hour
	}

	// Generate event ID
	eventID := uuid.New().String()

	// Create event processing job
	eventArgs := jobs.EventArgs{
		EventID:    eventID,
		Namespace:  req.Msg.Namespace,
		Event:      req.Msg.Event,
		Payload:    req.Msg.Payload,
		TTLSeconds: ttl,
		Metadata:   req.Msg.Metadata,
		CreatedAt:  time.Now(),
	}

	// Find registered webhooks first to know how many will be triggered
	registeredWebhooks, err := s.webhookRepo.GetWebhooksByEvent(ctx, req.Msg.Namespace, req.Msg.Event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to get registered webhooks")
		s.logger.Error("Failed to get registered webhooks",
			"namespace", req.Msg.Namespace,
			"event", req.Msg.Event,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get registered webhooks: %w", err))
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
			"namespace", req.Msg.Namespace,
			"event", req.Msg.Event,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to schedule event processing: %w", err))
	}

	// Record metrics
	if s.metrics != nil {
		s.metrics.EventsPushed.Add(ctx, 1)
	}

	span.SetStatus(otelcodes.Ok, "event scheduled successfully")

	s.logger.Info("Event processing scheduled successfully",
		"event_id", eventID,
		"namespace", req.Msg.Namespace,
		"event", req.Msg.Event,
		"webhooks_to_trigger", len(registeredWebhooks),
	)

	result := &pb.PushEventResponse{
		EventId:           eventID,
		WebhooksTriggered: int32(len(registeredWebhooks)),
		WebhookIds:        webhookIDs,
		Success:           true,
		Message:           fmt.Sprintf("Event scheduled for processing, %d webhooks will be triggered", len(registeredWebhooks)),
	}

	return connect.NewResponse(result), nil
}

// GetWebhookStatus gets the status of webhook deliveries
func (s *WebhookConnectServer) GetWebhookStatus(
	ctx context.Context,
	req *connect.Request[pb.GetWebhookStatusRequest],
) (*connect.Response[pb.GetWebhookStatusResponse], error) {
	ctx, span := s.tracer.Start(ctx, "connect.webhook.status")
	defer span.End()

	s.logger.Info("Connect: Received webhook status request")

	var deliveries []*webhooks.WebhookDelivery
	var err error

	switch id := req.Msg.Identifier.(type) {
	case *pb.GetWebhookStatusRequest_WebhookId:
		if id.WebhookId == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("webhook_id is required"))
		}
		deliveries, err = s.webhookRepo.GetDeliveriesByWebhook(ctx, id.WebhookId)
	case *pb.GetWebhookStatusRequest_EventId:
		if id.EventId == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("event_id is required"))
		}
		deliveries, err = s.webhookRepo.GetDeliveriesByEvent(ctx, id.EventId)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("either webhook_id or event_id is required"))
	}

	if err != nil {
		s.logger.Error("Failed to get webhook deliveries", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get webhook status: %w", err))
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

	result := &pb.GetWebhookStatusResponse{
		Deliveries:      pbDeliveries,
		TotalDeliveries: int32(len(deliveries)),
		Success:         true,
		Message:         fmt.Sprintf("Found %d webhook deliveries", len(deliveries)),
	}

	return connect.NewResponse(result), nil
}

// ListWebhooks lists all registered webhooks for a namespace
func (s *WebhookConnectServer) ListWebhooks(
	ctx context.Context,
	req *connect.Request[pb.ListWebhooksRequest],
) (*connect.Response[pb.ListWebhooksResponse], error) {
	ctx, span := s.tracer.Start(ctx, "connect.webhook.list")
	defer span.End()

	s.logger.Info("Connect: Received list webhooks request",
		"namespace", req.Msg.Namespace,
		"event", req.Msg.Event,
		"active_only", req.Msg.ActiveOnly,
	)

	if req.Msg.Namespace == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("namespace is required"))
	}

	// Get webhooks from repository
	registrations, err := s.webhookRepo.ListWebhooks(ctx, req.Msg.Namespace, req.Msg.ActiveOnly)
	if err != nil {
		s.logger.Error("Failed to list webhooks",
			"namespace", req.Msg.Namespace,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list webhooks: %w", err))
	}

	// Filter by event if specified
	var filteredRegistrations []*webhooks.WebhookRegistration
	if req.Msg.Event != "" {
		for _, reg := range registrations {
			// Check if the webhook listens to the requested event
			for _, event := range reg.Events {
				if event == req.Msg.Event {
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
		"namespace", req.Msg.Namespace,
		"total_count", len(pbWebhooks),
	)

	result := &pb.ListWebhooksResponse{
		Webhooks:   pbWebhooks,
		TotalCount: int32(len(pbWebhooks)),
		Success:    true,
		Message:    fmt.Sprintf("Found %d webhooks", len(pbWebhooks)),
	}

	return connect.NewResponse(result), nil
}

// convertDeliveryStatus converts internal status to protobuf status
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

// Handler returns the Connect-RPC handler
func (s *WebhookConnectServer) Handler() (string, http.Handler) {
	// Create simple handler
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		log.Fatal(err)
	}
	path, handler := protoconnect.NewWebhookServiceHandler(s, connect.WithInterceptors(otelInterceptor))
	return path, handler
}
