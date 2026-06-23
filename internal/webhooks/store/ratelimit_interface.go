package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RateLimitRepository defines operations for per-webhook rate limiting.
type RateLimitRepository interface {
	AcquireDeliverySlot(ctx context.Context, webhookID uuid.UUID) (time.Time, float64, error)
	UpsertRateLimitState(ctx context.Context, webhookID uuid.UUID) error
	DeleteRateLimitState(ctx context.Context, webhookID uuid.UUID) error
}
