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
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
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

// maskEncryptedSecret decrypts an encrypted webhook secret and masks it for
// safe display in API responses. Returns "" if the secret is empty or
// decryption fails (e.g. key rotated/removed).
func maskEncryptedSecret(encrypted []byte, svc webhooks.WebhookServiceInterface) string {
	if len(encrypted) == 0 {
		return ""
	}
	plaintext, err := svc.DecryptWebhookSecret(encrypted)
	if err != nil || plaintext == "" {
		// Can't decrypt — show a generic mask so the user knows a secret exists
		return "••••••"
	}
	return maskSecret(plaintext)
}

// toGRPCError maps service-layer errors to appropriate gRPC status codes.
//
// Resolution order:
//  1. ServiceError — service/validation code explicitly marks errors as
//     client-safe with a gRPC code. This is the preferred mechanism.
//  2. Storage sentinel errors — ErrNotFound, ErrAlreadyExists, etc.
//  3. String-matching fallback — legacy allowlist for errors that haven't
//     been converted to ServiceError yet.
//  4. Default — codes.Internal with only the fallbackMsg (no internals leaked).
func toGRPCError(ctx context.Context, err error, fallbackMsg string) error {
	if err == nil {
		return nil
	}

	// 1. ServiceError — explicitly marked as client-safe by service layer.
	var svcErr *svcerrors.ServiceError
	if errors.As(err, &svcErr) {
		return status.Errorf(svcErr.GRPCCode, "%s", svcErr.ClientMessage())
	}

	// 2. Storage sentinel errors (more reliable than string matching).
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

	// 3. String-matching fallback — covers errors not yet converted to ServiceError.
	errMsg := err.Error()

	// "not found" anywhere in the message
	if strings.Contains(errMsg, "not found") {
		return status.Errorf(codes.NotFound, "%s: %v", fallbackMsg, err)
	}

	// Required-field / basic validation errors
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
		strings.Contains(errMsg, "namespace is required for namespace-scoped access") ||
		// URL validation / SSRF errors
		strings.Contains(errMsg, "not allowed") ||
		strings.Contains(errMsg, "only http and https are allowed") ||
		strings.Contains(errMsg, "must have a non-empty host") ||
		strings.Contains(errMsg, "cannot resolve URL host") ||
		strings.Contains(errMsg, "resolves to blocked address") ||
		strings.HasPrefix(errMsg, "invalid URL") ||
		// Label validation errors
		strings.HasPrefix(errMsg, "labels:") ||
		strings.HasPrefix(errMsg, "label_filters:") ||
		// Template errors
		strings.Contains(errMsg, "template transformation failed") {
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// State / precondition errors (operation not valid in current state)
	if strings.Contains(errMsg, "is inactive") ||
		strings.Contains(errMsg, "already succeeded") ||
		strings.Contains(errMsg, "encryption is required") ||
		strings.Contains(errMsg, "encryption key not configured") ||
		strings.Contains(errMsg, "batch job is not") ||
		strings.Contains(errMsg, "batch job has expired") ||
		strings.Contains(errMsg, "not in pending status") ||
		strings.Contains(errMsg, "is already in terminal state") ||
		strings.Contains(errMsg, "only one of") ||
		strings.Contains(errMsg, "failed to resubmit") {
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	}

	// 4. Default — log the real error but do NOT expose internals to the client.
	slog.ErrorContext(ctx, "internal error", "fallback_msg", fallbackMsg, "error", err)
	return status.Errorf(codes.Internal, "%s", fallbackMsg)
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

// batchJobToProto converts a store.BatchJob to the protobuf BatchJobStatus message.
func batchJobToProto(batch *store.BatchJob) *pb.BatchJobStatus {
	if batch == nil {
		return nil
	}
	return &pb.BatchJobStatus{
		Status:    string(batch.Status),
		Total:     int32(batch.Total),
		Processed: int32(batch.Processed),
		Failed:    int32(batch.Failed),
		CreatedAt: convertTimeToProto(batch.CreatedAt),
		ExpiresAt: convertTimeToProto(batch.ExpiresAt),
	}
}
