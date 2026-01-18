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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/sarathsp06/sparrow/internal/logger"
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
// Contains only essential identifiers - the payload is stored in the database
type EventArgs struct {
	EventID    string            `json:"event_id"`
	Namespace  string            `json:"namespace"`
	Event      string            `json:"event"`
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
// Contains only essential identifiers - webhook config and event payload retrieved from database
type WebhookArgs struct {
	DeliveryID     string    `json:"delivery_id"`
	WebhookID      string    `json:"webhook_id"`
	SubscriptionID string    `json:"subscription_id"`
	EventID        string    `json:"event_id"`
	ExpiresAt      time.Time `json:"expires_at"`
	Namespace      string    `json:"namespace"`
	MaxAttempts    int       `json:"max_attempts"`
}

var _ river.JobArgsWithInsertOpts = (*WebhookArgs)(nil)

func (w WebhookArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueWebhookDelivery,
		MaxAttempts: w.MaxAttempts,
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
	return j.client.Insert(ctx, args, j.InsertOpts(ctx, args))
}

func (j *jobInserter) InsertOpts(ctx context.Context, args river.JobArgs) *river.InsertOpts {
	// get trace id and set that as metadata
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	carrierJSON, err := json.Marshal(carrier)
	if err != nil {
		j.logger.Error("Failed to marshal trace metadata", "error", err)
	}
	return &river.InsertOpts{
		Metadata: carrierJSON,
	}
}

// BatchInsert inserts multiple jobs into the queue.
// The trace id is set as metadata for each job.
// If any errors occur during the batch insert, the entire operation is failed.
func (j *jobInserter) BatchInsert(ctx context.Context, args []river.JobArgs) ([]*rivertype.JobInsertResult, error) {
	params := make([]river.InsertManyParams, 0, len(args))
	for _, arg := range args {
		params = append(params, river.InsertManyParams{
			Args:       arg,
			InsertOpts: j.InsertOpts(ctx, arg),
		})
	}
	insertResults, err := j.client.InsertMany(ctx, params)
	if err != nil {
		j.logger.Error("Failed to batch insert jobs", "error", err)
		return nil, fmt.Errorf("failed to batch insert jobs: %w", err)
	}
	return insertResults, nil
}

// QueueManagerInterface defines the interface for queue management.
//
//go:generate gowrap gen -i JobInserter -t ../../../templates/opentelemetry.tmpl -o JobInserter_otel.go
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error)
	BatchInsert(ctx context.Context, args []river.JobArgs) ([]*rivertype.JobInsertResult, error)
}

var _ JobInserter = (*jobInserter)(nil)
