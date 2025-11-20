package grpc

import (
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
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

// Helper function to convert map[string]any to protobuf Struct
func convertMapToStruct(m map[string]any) (*structpb.Struct, error) {
	if m == nil {
		return nil, nil
	}

	return structpb.NewStruct(m)
}
