package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/sarathsp06/sparrow/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	// Job kinds
	JobKindEventProcessing = "event_processing"
	JobKindWebhookDelivery = "webhook_delivery"

	// Queue names
	QueueEventProcessing = "events"
	QueueWebhookDelivery = "webhooks"
	QueueDefault         = river.QueueDefault
)

// EventArgs represents an event processing job
type EventArgs struct {
	EventID    string            `json:"event_id"`
	Namespace  string            `json:"namespace"`
	Event      string            `json:"event"`
	Payload    map[string]any    `json:"payload"`
	TTLSeconds int64             `json:"ttl_seconds"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
}

var _ river.JobArgsWithInsertOpts = (*EventArgs)(nil)

func (EventArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueEventProcessing,
	}
}

// Kind returns the job kind for River queue
func (EventArgs) Kind() string {
	return "event_processing"
}

// WebhookArgs represents a webhook delivery job
type WebhookArgs struct {
	DeliveryID string            `json:"delivery_id"`
	WebhookID  string            `json:"webhook_id"`
	EventID    string            `json:"event_id"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers"`
	Payload    map[string]any    `json:"payload"`
	Timeout    int               `json:"timeout"`
	ExpiresAt  time.Time         `json:"expires_at"`
	Namespace  string            `json:"namespace"`
	Event      string            `json:"event"`
}

var _ river.JobArgsWithInsertOpts = (*WebhookArgs)(nil)

func (WebhookArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueWebhookDelivery,
	}
}

// Kind returns the job kind for River queue
func (WebhookArgs) Kind() string {
	return "webhook_delivery"
}

type jobInserter struct {
	client *river.Client[pgx.Tx]
	logger *slog.Logger
}

// NewJobInserter creates a new JobInserter
func NewJobInserter(client *river.Client[pgx.Tx]) *jobInserter {
	return &jobInserter{
		client: client,
		logger: logger.NewLogger("job-inserter"),
	}
}

// Insert inserts a job into the queue.
func (j *jobInserter) Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	// get trace id and set that as metadata
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	carrierJSON, err := json.Marshal(carrier)
	if err != nil {
		j.logger.Error("Failed to marshal trace metadata", "error", err)
		return nil, fmt.Errorf("failed to marshal trace metadata: %w", err)
	}
	return j.client.Insert(ctx, args, &river.InsertOpts{
		Metadata: carrierJSON,
	})
}

// QueueManagerInterface defines the interface for queue management.
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error)
}

var _ JobInserter = (*jobInserter)(nil)
