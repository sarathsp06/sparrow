package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/sarathsp06/sparrow/proto"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// PushEvent pushes an event that triggers registered webhooks.
// Schema validation is soft: events are always accepted and stored.
// If the payload does not match the registered schema, warnings are
// returned in the response but the event is still processed.
func (s *WebhookServer) PushEvent(ctx context.Context, req *pb.PushEventRequest) (*pb.PushEventResponse, error) {
	var payload map[string]any
	if req.Payload != nil {
		payload = req.Payload.AsMap()
	}
	eventID, warnings, err := s.service.PushEvent(ctx, req.Namespace, req.Event, payload, req.TtlSeconds, req.Metadata, req.Labels)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to push event")
	}
	return &pb.PushEventResponse{
		EventId:  eventID,
		Warnings: warnings,
	}, nil
}

// RegisterEvent registers a new event type
func (s *WebhookServer) RegisterEvent(ctx context.Context, req *pb.RegisterEventRequest) (*pb.RegisterEventResponse, error) {
	// Convert JSON schema Struct to map[string]any
	var schema map[string]any
	if req.Schema != nil {
		schema = req.Schema.AsMap()
	}

	eventName, createdAt, err := s.service.RegisterEvent(ctx, req.Name, req.Description, schema, req.Metadata, req.Active)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to register event")
	}
	return &pb.RegisterEventResponse{
		EventId:   eventName, // Deprecated field — now carries the event name
		CreatedAt: timestamppb.New(createdAt),
	}, nil
}

// ListEvents lists all registered events
func (s *WebhookServer) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	var limit, offset int32
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		offset = req.Pagination.Offset
	}

	events, totalCount, err := s.service.ListEvents(ctx, req.ActiveOnly, limit, offset)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to list events")
	}

	pbEvents := make([]*pb.RegisteredEvent, len(events))
	for i, event := range events {
		pbEv, err := convertEventToProto(event)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert event: %v", err)
		}
		pbEvents[i] = pbEv
	}

	return &pb.ListEventsResponse{
		Events: pbEvents,
		Pagination: &pb.PaginationResponse{
			TotalCount: totalCount,
			Limit:      limit,
			Offset:     offset,
		},
	}, nil
}

// UpdateEvent updates an event registration
func (s *WebhookServer) UpdateEvent(ctx context.Context, req *pb.UpdateEventRequest) (*pb.UpdateEventResponse, error) {
	// Convert JSON schema Struct to map[string]any
	var schema map[string]any
	if req.Schema != nil {
		schema = req.Schema.AsMap()
	}

	err := s.service.UpdateEvent(ctx, req.Name, req.Description, schema, req.Metadata, req.Active)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to update event")
	}
	return &pb.UpdateEventResponse{}, nil
}

// DeleteEvent deletes an event registration
func (s *WebhookServer) DeleteEvent(ctx context.Context, req *pb.DeleteEventRequest) (*pb.DeleteEventResponse, error) {
	err := s.service.DeleteEvent(ctx, req.Name)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to delete event")
	}
	return &pb.DeleteEventResponse{}, nil
}

// GetEvent retrieves an event type by name
func (s *WebhookServer) GetEvent(ctx context.Context, req *pb.GetEventRequest) (*pb.GetEventResponse, error) {
	event, err := s.service.GetEvent(ctx, req.Name)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to get event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}

	pbEvent, err := convertEventToProto(event)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert event: %v", err)
	}

	return &pb.GetEventResponse{
		Event: pbEvent,
	}, nil
}

// ListEventReports lists all events in descending order, with optional filters
func (s *WebhookServer) ListEventReports(ctx context.Context, req *pb.ListEventReportsRequest) (*pb.ListEventReportsResponse, error) {
	// Build filter from request
	filter := store.EventReportFilter{
		Namespace: req.Namespace,
	}

	// Pagination
	if req.Pagination != nil {
		filter.Limit = int(req.Pagination.Limit)
		filter.Offset = int(req.Pagination.Offset)
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	} else if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	// Optional event name
	if req.EventName != nil {
		filter.EventName = req.EventName
	}

	// Schema valid filter
	if req.SchemaValid != nil {
		filter.SchemaValid = req.SchemaValid
	}

	// Labels filter
	if len(req.Labels) > 0 {
		filter.Labels = req.Labels
	}

	// Time range filters
	if req.CreatedAfter != nil {
		t := req.CreatedAfter.AsTime()
		filter.CreatedAfter = &t
	}
	if req.CreatedBefore != nil {
		t := req.CreatedBefore.AsTime()
		filter.CreatedBefore = &t
	}

	// Batch snapshot opt-in
	filter.PrepareRepush = req.PrepareRepush

	// Call service method
	events, totalCount, repushID, err := s.service.ListEventReports(ctx, filter)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to list event reports")
	}

	// Convert events to protobuf format
	var pbEvents []*pb.EventReport
	for _, event := range events {
		// Convert payload to protobuf Struct
		payloadStruct, err := convertMapToStruct(event.Payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert payload: %v", err)
		}

		// Convert metadata to map[string]string
		metadata := make(map[string]string)
		for k, v := range event.Metadata {
			metadata[k] = v
		}

		// Use the delivery statistics from the database query
		pbEvent := &pb.EventReport{
			EventId:              event.ID.String(),
			Namespace:            event.Namespace,
			EventName:            event.Event,
			Payload:              payloadStruct,
			Metadata:             metadata,
			CreatedAt:            convertTimeToProto(event.CreatedAt),
			TtlSeconds:           event.TTL,
			WebhookCount:         event.WebhookCount,
			SuccessfulDeliveries: event.SuccessfulDeliveries,
			FailedDeliveries:     event.FailedDeliveries,
			PendingDeliveries:    event.PendingDeliveries,
			SchemaValid:          event.SchemaValid,
		}
		pbEvents = append(pbEvents, pbEvent)
	}

	return &pb.ListEventReportsResponse{
		Events: pbEvents,
		Pagination: &pb.PaginationResponse{
			TotalCount: totalCount,
			Limit:      int32(filter.Limit),
			Offset:     int32(filter.Offset),
		},
		RepushId: repushID,
	}, nil
}

// RePushEvents starts a batch re-push of events previously snapshotted via ListEventReports.
func (s *WebhookServer) RePushEvents(ctx context.Context, req *pb.RePushEventsRequest) (*pb.RePushEventsResponse, error) {
	if req.RepushId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "repush_id is required")
	}

	if err := s.service.RePushEvents(ctx, req.RepushId); err != nil {
		return nil, toGRPCError(ctx, err, "failed to start batch re-push")
	}

	// Fetch the batch to return total/status
	batch, err := s.service.GetRepushStatus(ctx, req.RepushId)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to get repush status")
	}

	return &pb.RePushEventsResponse{
		RepushId: req.RepushId,
		Total:    int32(batch.Total),
		Status:   string(batch.Status),
	}, nil
}

// GetRepushStatus returns the current progress of a batch re-push operation.
func (s *WebhookServer) GetRepushStatus(ctx context.Context, req *pb.GetRepushStatusRequest) (*pb.GetRepushStatusResponse, error) {
	if req.RepushId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "repush_id is required")
	}

	batch, err := s.service.GetRepushStatus(ctx, req.RepushId)
	if err != nil {
		return nil, toGRPCError(ctx, err, "failed to get repush status")
	}

	return &pb.GetRepushStatusResponse{
		Batch: batchJobToProto(batch),
	}, nil
}

// CancelRepush aborts a pending or in-progress batch re-push.
func (s *WebhookServer) CancelRepush(ctx context.Context, req *pb.CancelRepushRequest) (*pb.CancelRepushResponse, error) {
	if req.RepushId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "repush_id is required")
	}

	if err := s.service.CancelRepush(ctx, req.RepushId); err != nil {
		return nil, toGRPCError(ctx, err, "failed to cancel repush")
	}

	return &pb.CancelRepushResponse{
		Status: string(store.BatchStatusCancelled),
	}, nil
}
