package grpc

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/storage"
	pb "github.com/sarathsp06/sparrow/proto"
)

// Helper function to convert delivery status
func convertDeliveryStatus(status store.WebhookDeliveryStatus) pb.WebhookDeliveryStatus {
	switch status {
	case store.StatusPending:
		return pb.WebhookDeliveryStatus_DELIVERY_PENDING
	case store.StatusSending:
		return pb.WebhookDeliveryStatus_DELIVERY_SENDING
	case store.StatusSuccess:
		return pb.WebhookDeliveryStatus_DELIVERY_SUCCESS
	case store.StatusFailed:
		return pb.WebhookDeliveryStatus_DELIVERY_FAILED
	case store.StatusRetrying:
		return pb.WebhookDeliveryStatus_DELIVERY_RETRYING
	case store.StatusExpired:
		return pb.WebhookDeliveryStatus_DELIVERY_EXPIRED
	default:
		return pb.WebhookDeliveryStatus_DELIVERY_UNSPECIFIED
	}
}

// Helper function to convert webhook health to protobuf
func convertWebhookHealth(health store.WebhookHealth) pb.WebhookHealth {
	switch health {
	case store.HealthHealthy:
		return pb.WebhookHealth_HEALTH_HEALTHY
	case store.HealthDegraded:
		return pb.WebhookHealth_HEALTH_DEGRADED
	case store.HealthUnhealthy:
		return pb.WebhookHealth_HEALTH_UNHEALTHY
	case store.HealthUnknown:
		return pb.WebhookHealth_HEALTH_UNSPECIFIED
	default:
		return pb.WebhookHealth_HEALTH_UNSPECIFIED
	}
}

// convertExpectedStatusCodes converts pq.Int64Array to []int32
func convertExpectedStatusCodes(codes []int64) []int32 {
	if len(codes) == 0 {
		return nil
	}

	result := make([]int32, len(codes))
	for i, code := range codes {
		result[i] = int32(code)
	}
	return result
}

// convertStatusCodesToInt converts []int32 to []int for use in HTTPConfigUpdate
func convertStatusCodesToInt(codes []int32) []int {
	if len(codes) == 0 {
		return nil
	}

	result := make([]int, len(codes))
	for i, code := range codes {
		result[i] = int(code)
	}
	return result
}

// Helper function to convert map[string]any to protobuf Struct
func convertMapToStruct(m map[string]any) (*structpb.Struct, error) {
	if m == nil {
		return nil, nil
	}

	return structpb.NewStruct(m)
}

// convertTimeToProto converts time.Time to timestamppb.Timestamp
func convertTimeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// convertPtrTimeToProto converts *time.Time to timestamppb.Timestamp
func convertPtrTimeToProto(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

// convertEventToProto converts store.EventRegistration to protobuf RegisteredEvent
func convertEventToProto(event *store.EventRegistration) (*pb.RegisteredEvent, error) {
	if event == nil {
		return nil, nil
	}
	pbEvent := &pb.RegisteredEvent{
		EventId:     event.Name, // Deprecated field — now carries the event name for backward compat
		Name:        event.Name,
		Description: event.Description,
		Active:      event.Active,
		CreatedAt:   convertTimeToProto(event.CreatedAt),
		UpdatedAt:   convertTimeToProto(event.UpdatedAt),
		Metadata:    event.Metadata,
	}

	if event.Schema != nil {
		schemaStruct, err := convertMapToStruct(event.Schema)
		if err != nil {
			return nil, err
		}
		pbEvent.Schema = schemaStruct
	}

	if event.SamplePayload != nil {
		samplePayloadStruct, err := convertMapToStruct(event.SamplePayload)
		if err != nil {
			return nil, err
		}
		pbEvent.SamplePayload = samplePayloadStruct
	}

	return pbEvent, nil
}

// maskSecret masks a secret string for safe display in API responses.
// It shows the first 4 characters followed by asterisks.
// Returns an empty string if the input is empty.
func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:4] + strings.Repeat("*", len(secret)-4)
}

// toGRPCError maps service-layer errors to appropriate gRPC status codes.
// It checks for known error types (not found, validation)
// and returns a properly-coded gRPC status error.
// Internal errors are logged server-side but NOT exposed to the client —
// only the fallbackMsg is returned to prevent leaking implementation details.
func toGRPCError(ctx context.Context, err error, fallbackMsg string) error {
	if err == nil {
		return nil
	}

	// Check for known storage-level errors first (more reliable than string matching)
	if errors.Is(err, storage.ErrNotFound) {
		return status.Errorf(codes.NotFound, "%s: %v", fallbackMsg, err)
	}
	if errors.Is(err, storage.ErrForeignKeyViolation) {
		return status.Errorf(codes.FailedPrecondition, "%s: a referenced resource does not exist", fallbackMsg)
	}
	if errors.Is(err, storage.ErrAlreadyExists) {
		return status.Errorf(codes.AlreadyExists, "%s: resource already exists", fallbackMsg)
	}
	if errors.Is(err, storage.ErrNotNullViolation) {
		return status.Errorf(codes.InvalidArgument, "%s: a required field is missing", fallbackMsg)
	}

	// Check for common error message patterns
	errMsg := err.Error()

	// "not found" errors — safe to include in response
	if strings.Contains(errMsg, "not found") {
		return status.Errorf(codes.NotFound, "%s: %v", fallbackMsg, err)
	}

	// Validation / input errors — safe to include in response
	if strings.HasPrefix(errMsg, "namespace is required") ||
		strings.HasPrefix(errMsg, "webhook_id is required") ||
		strings.HasPrefix(errMsg, "webhook ID is required") ||
		strings.HasPrefix(errMsg, "delivery ID is required") ||
		strings.HasPrefix(errMsg, "event name is required") ||
		strings.HasPrefix(errMsg, "event names cannot be empty") ||
		strings.HasPrefix(errMsg, "empty event name not allowed") ||
		strings.HasPrefix(errMsg, "URL is required") ||
		strings.HasPrefix(errMsg, "event is required") ||
		strings.Contains(errMsg, "invalid webhook ID") ||
		strings.Contains(errMsg, "invalid delivery ID") ||
		strings.Contains(errMsg, "invalid subscription ID") ||
		strings.Contains(errMsg, "already exists") ||
		strings.Contains(errMsg, "already paused") ||
		strings.Contains(errMsg, "already active") ||
		strings.Contains(errMsg, "namespace is required for namespace-scoped access") {
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// Default to internal error — log the real error but do NOT expose it to the client.
	// Returning internal error details (SQL errors, stack traces, etc.) is a security risk.
	slog.ErrorContext(ctx, "internal error", "fallback_msg", fallbackMsg, "error", err)
	return status.Errorf(codes.Internal, "%s", fallbackMsg)
}

// paginationDefaults extracts limit/offset from a PaginationRequest with sensible defaults.
func paginationDefaults(p *pb.PaginationRequest) (limit, offset int32) {
	limit = 20
	offset = 0
	if p != nil {
		if p.Limit > 0 {
			limit = p.Limit
		}
		if p.Offset > 0 {
			offset = p.Offset
		}
	}
	if limit > 100 {
		limit = 100
	}
	return limit, offset
}

// maskSecretHeaders decrypts the encrypted secret headers and returns a map
// with all values replaced by "••••••" for safe display in API responses.
// Returns nil if there are no secret headers or decryption fails.
func maskSecretHeaders(encrypted []byte, svc webhooks.WebhookServiceInterface) map[string]string {
	if len(encrypted) == 0 {
		return nil
	}
	decrypted, err := svc.DecryptSecretHeaders(encrypted)
	if err != nil || len(decrypted) == 0 {
		return nil
	}
	masked := make(map[string]string, len(decrypted))
	for k := range decrypted {
		masked[k] = "••••••"
	}
	return masked
}
