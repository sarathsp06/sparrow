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
//  3. Default — codes.Internal with only the fallbackMsg (no internals leaked).
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

	// 3. Default — log the real error but do NOT expose internals to the client.
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

// extractPagination extracts limit and offset from a PaginationRequest.
// Returns (0, 0) when pagination is nil, which the service layer normalizes.
func extractPagination(p *pb.PaginationRequest) (limit, offset int32) {
	if p != nil {
		return p.Limit, p.Offset
	}
	return 0, 0
}

// convertDeliveryToProto converts a store.WebhookDelivery to the protobuf WebhookDelivery message.
func convertDeliveryToProto(d *store.WebhookDelivery) *pb.WebhookDelivery {
	if d == nil {
		return nil
	}
	return &pb.WebhookDelivery{
		DeliveryId:      d.ID.String(),
		WebhookId:       d.WebhookID.String(),
		EventId:         d.EventID.String(),
		Status:          convertDeliveryStatus(d.Status),
		AttemptCount:    int32(d.AttemptCount),
		MaxAttempts:     int32(d.MaxAttempts),
		CreatedAt:       convertTimeToProto(d.CreatedAt),
		LastAttemptedAt: convertPtrTimeToProto(d.LastAttemptedAt),
		NextRetryAt:     convertPtrTimeToProto(d.NextRetryAt),
		ExpiresAt:       convertTimeToProto(d.ExpiresAt),
		ResponseCode:    int32(d.ResponseCode),
		ResponseBody:    d.ResponseBody,
		ErrorMessage:    d.ErrorMessage,
		RequestBody:     d.RequestBody,
		ErrorCategory:   d.ErrorCategory,
	}
}

// convertWebhookRegToProto converts a store.WebhookRegistration to the protobuf
// RegisteredWebhook message. events is the pre-fetched event list for this webhook.
// svc is used for secret masking (decryption).
func convertWebhookRegToProto(reg *store.WebhookRegistration, events []string, svc webhooks.WebhookServiceInterface) *pb.RegisteredWebhook {
	if reg == nil {
		return nil
	}
	return &pb.RegisteredWebhook{
		WebhookId:     reg.ID.String(),
		Namespace:     reg.Namespace,
		Events:        events,
		Url:           reg.URL,
		Headers:       reg.Headers,
		Timeout:       int32(reg.Timeout),
		Active:        reg.Active,
		Description:   reg.Description,
		Health:        convertWebhookHealth(reg.Health),
		CreatedAt:     convertTimeToProto(reg.CreatedAt),
		UpdatedAt:     convertTimeToProto(reg.UpdatedAt),
		SecretHeaders: maskSecretHeaders(reg.SecretHeaders, svc),
		HttpConfig: &pb.WebhookHTTPConfig{
			MaxRetries:            int32(reg.MaxRetries),
			RetryBackoffSeconds:   int32(reg.RetryBackoffSeconds),
			CaptureResponseBody:   reg.CaptureResponseBody,
			FollowRedirects:       reg.FollowRedirects,
			VerifySsl:             reg.VerifySSL,
			RequestTimeoutSeconds: int32(reg.RequestTimeoutSeconds),
			ExpectedStatusCodes:   convertExpectedStatusCodes(reg.ExpectedStatusCodes),
			WebhookSecret:         maskEncryptedSecret(reg.WebhookSecret, svc),
			UserAgent:             reg.UserAgent,
			ContentType:           reg.ContentType,
			RateLimitRps:          float64PtrToFloat32Ptr(reg.RateLimitRPS),
		},
	}
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
