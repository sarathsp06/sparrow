package grpc

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/sarathsp06/sparrow/proto"
)

// PushEvent pushes an event that triggers registered webhooks
func (s *WebhookServer) PushEvent(ctx context.Context, req *pb.PushEventRequest) (*pb.PushEventResponse, error) {
	eventID, err := s.service.PushEvent(ctx, req.Namespace, req.Event, req.Payload.AsMap(), req.TtlSeconds, req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push event: %v", err)
	}
	return &pb.PushEventResponse{
		EventId: eventID,
		Success: true,
		Message: "Event pushed successfully",
	}, nil
}

// RegisterEvent registers a new event type
func (s *WebhookServer) RegisterEvent(ctx context.Context, req *pb.RegisterEventRequest) (*pb.RegisterEventResponse, error) {
	// Convert JSON schema string to map[string]any
	var schema map[string]any
	if req.Schema != "" {
		if err := json.Unmarshal([]byte(req.Schema), &schema); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid schema JSON: %v", err)
		}
	}

	eventID, createdAt, err := s.service.RegisterEvent(ctx, req.Name, req.Description, schema, req.Metadata, req.Active)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register event: %v", err)
	}
	return &pb.RegisterEventResponse{
		EventId:   eventID,
		Success:   true,
		Message:   "Event registered successfully",
		CreatedAt: createdAt,
	}, nil
}

// ListEvents lists all registered events
func (s *WebhookServer) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	events, err := s.service.ListEvents(ctx, req.ActiveOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list events: %v", err)
	}

	pbEvents := make([]*pb.RegisteredEvent, len(events))
	for i, event := range events {
		pbEvents[i] = &pb.RegisteredEvent{
			EventId:     event.ID.String(),
			Name:        event.Name,
			Description: event.Description,
			Active:      event.Active,
			CreatedAt:   event.CreatedAt.Unix(),
			UpdatedAt:   event.UpdatedAt.Unix(),
		}

		// Convert schema map to JSON string for protobuf
		if event.Schema != nil {
			schemaJSON, err := json.Marshal(event.Schema)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to marshal event schema: %v", err)
			}
			pbEvents[i].Schema = string(schemaJSON)
		}

		// Convert metadata to protobuf format
		pbEvents[i].Metadata = event.Metadata
	}

	return &pb.ListEventsResponse{
		Events:     pbEvents,
		TotalCount: int32(len(pbEvents)),
		Success:    true,
	}, nil
}

// UpdateEvent updates an event registration
func (s *WebhookServer) UpdateEvent(ctx context.Context, req *pb.UpdateEventRequest) (*pb.UpdateEventResponse, error) {
	// Convert JSON schema string to map[string]any
	var schema map[string]any
	if req.Schema != "" {
		if err := json.Unmarshal([]byte(req.Schema), &schema); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid schema JSON: %v", err)
		}
	}

	err := s.service.UpdateEvent(ctx, req.Name, req.Description, schema, req.Metadata, req.Active)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update event: %v", err)
	}
	return &pb.UpdateEventResponse{
		Success: true,
		Message: "Event updated successfully",
	}, nil
}

// DeleteEvent deletes an event registration
func (s *WebhookServer) DeleteEvent(ctx context.Context, req *pb.DeleteEventRequest) (*pb.DeleteEventResponse, error) {
	err := s.service.DeleteEvent(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete event: %v", err)
	}
	return &pb.DeleteEventResponse{
		Success: true,
		Message: "Event deleted successfully",
	}, nil
}

// ListEventReports lists all events in descending order for a given namespace
func (s *WebhookServer) ListEventReports(ctx context.Context, req *pb.ListEventReportsRequest) (*pb.ListEventReportsResponse, error) {
	// Validate request
	if req.Namespace == "" {
		return nil, status.Errorf(codes.InvalidArgument, "namespace is required")
	}

	// Set default values
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000
	}

	offset := req.Offset
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
			CreatedAt:            event.CreatedAt.Unix(),
			TtlSeconds:           event.TTL,
			WebhookCount:         event.WebhookCount,
			SuccessfulDeliveries: event.SuccessfulDeliveries,
			FailedDeliveries:     event.FailedDeliveries,
			PendingDeliveries:    event.PendingDeliveries,
		}
		pbEvents = append(pbEvents, pbEvent)
	}

	return &pb.ListEventReportsResponse{
		Events:     pbEvents,
		TotalCount: totalCount,
		Success:    true,
		Message:    "Event reports retrieved successfully",
	}, nil
}
