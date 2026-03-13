package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	pb "github.com/sarathsp06/sparrow/proto"
)

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookServer) PushEvent(ctx context.Context, req *pb.PushEventRequest) (*pb.PushEventResponse, error) {
	var payload map[string]any
	if req.Payload != nil {
		payload = req.Payload.AsMap()
	}
	eventID, err := s.service.PushEvent(ctx, req.Namespace, req.Event, payload, req.TtlSeconds, req.Metadata)
	if err != nil {
		// Return InvalidArgument for schema validation errors so the client
		// gets actionable feedback instead of a generic Internal error.
		var schemaErr *webhooks.SchemaValidationError
		if errors.As(err, &schemaErr) {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to push event: %v", err)
	}
	return &pb.PushEventResponse{
		EventId: eventID,
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
		return nil, status.Errorf(codes.Internal, "failed to register event: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to list events: %v", err)
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
		return nil, status.Errorf(codes.Internal, "Failed to update event: %v", err)
	}
	return &pb.UpdateEventResponse{}, nil
}

// DeleteEvent deletes an event registration
func (s *WebhookServer) DeleteEvent(ctx context.Context, req *pb.DeleteEventRequest) (*pb.DeleteEventResponse, error) {
	err := s.service.DeleteEvent(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete event: %v", err)
	}
	return &pb.DeleteEventResponse{}, nil
}

// GetEvent retrieves an event type by name
func (s *WebhookServer) GetEvent(ctx context.Context, req *pb.GetEventRequest) (*pb.GetEventResponse, error) {
	event, err := s.service.GetEvent(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get event: %v", err)
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

// ListEventReports lists all events in descending order, optionally filtered by namespace
func (s *WebhookServer) ListEventReports(ctx context.Context, req *pb.ListEventReportsRequest) (*pb.ListEventReportsResponse, error) {
	// Set default values
	var limit, offset int32
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		offset = req.Pagination.Offset
	}
	// Deprecated req.Limit/req.Offset fields are intentionally not read.
	// Clients should migrate to the pagination field.

	if limit <= 0 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000
	}

	if offset < 0 {
		offset = 0
	}

	// Convert optional event name
	var eventName *string
	if req.EventName != nil {
		eventName = req.EventName
	}

	// Call service method
	events, totalCount, err := s.service.ListEventReports(ctx, req.Namespace, eventName, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list event reports: %v", err)
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
		}
		pbEvents = append(pbEvents, pbEvent)
	}

	return &pb.ListEventReportsResponse{
		Events: pbEvents,
		Pagination: &pb.PaginationResponse{
			TotalCount: totalCount,
			Limit:      limit,
			Offset:     offset,
		},
	}, nil
}
