package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivertype"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	// Queue names
	QueueEventProcessing = "events"
	QueueWebhookDelivery = "webhooks"
	QueueBatchJobs       = "batch_jobs"
	QueueDefault         = river.QueueDefault
)

// EventArgs represents an event processing job
// Contains only essential identifiers - the payload is stored in the database
type EventArgs struct {
	TenantID   string            `json:"tenant_id"`
	EventID    string            `json:"event_id"`
	Consumer   string            `json:"consumer"`
	Event      string            `json:"event"`
	TTLSeconds int64             `json:"ttl_seconds"`
	Metadata   map[string]string `json:"metadata"`
	Labels     map[string]string `json:"labels"`
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
	TenantID       string    `json:"tenant_id"`
	DeliveryID     string    `json:"delivery_id"`
	WebhookID      string    `json:"webhook_id"`
	SubscriptionID string    `json:"subscription_id"`
	EventID        string    `json:"event_id"`
	ExpiresAt      time.Time `json:"expires_at"`
	Consumer       string    `json:"consumer"`
	MaxAttempts    int       `json:"max_attempts"`
	// RetryBackoffSeconds is the webhook's configured base retry delay;
	// 0 falls back to River's default retry policy. See WebhookWorker.NextRetry.
	RetryBackoffSeconds int `json:"retry_backoff_seconds,omitempty"`
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

// BatchJobArgs represents a batch job processing task (event repush or delivery retry).
// The batch data (item IDs, type) is stored in the batch_jobs table; this job just
// carries the reference.
type BatchJobArgs struct {
	TenantID string `json:"tenant_id"`
	BatchID  string `json:"batch_id"`
}

var _ river.JobArgsWithInsertOpts = (*BatchJobArgs)(nil)

func (BatchJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueBatchJobs,
	}
}

// Kind returns the job kind for River queue
func (BatchJobArgs) Kind() string {
	return "batch_processing"
}

type jobInserter struct {
	client *river.Client[pgx.Tx]
	// txClient inserts inside a database/sql transaction (the one the
	// repositories use), so rows and their jobs commit together. It never
	// works jobs.
	txClient *river.Client[*sql.Tx]
	logger   *slog.Logger

	// tenantLimits and extraQueues route a tenant's event fan-out and
	// delivery jobs to its own queue (TenantLimits.Queue). Nil: no routing.
	tenantLimits TenantLimiter
	extraQueues  map[string]int
}

// NewJobInserter creates a new JobInserter. txClient may be nil when
// BatchInsertTx is never called.
func NewJobInserter(client *river.Client[pgx.Tx], txClient *river.Client[*sql.Tx]) *jobInserter {
	return &jobInserter{
		client:   client,
		txClient: txClient,
		logger:   slog.Default().With("component", "job-inserter"),
	}
}

// NewTxInsertClient returns an insert-only River client for database/sql
// transactions.
func NewTxInsertClient() (*river.Client[*sql.Tx], error) {
	return river.NewClient(riverdatabasesql.New(nil), &river.Config{})
}

// Insert inserts a job into the queue.
func (j *jobInserter) Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	return j.client.Insert(ctx, args, j.InsertOpts(ctx, args))
}

func (j *jobInserter) InsertOpts(ctx context.Context, args river.JobArgs) *river.InsertOpts {
	var opts river.InsertOpts
	if optArgs, ok := args.(river.JobArgsWithInsertOpts); ok {
		opts = optArgs.InsertOpts()
	}

	// get trace id and set that as metadata
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	carrierJSON, err := json.Marshal(carrier)
	if err != nil {
		j.logger.ErrorContext(ctx, "Failed to marshal trace metadata", "error", err)
	}
	opts.Metadata = carrierJSON
	if queue := j.tenantQueue(ctx, args); queue != "" {
		opts.Queue = queue
	}
	return &opts
}

// tenantQueue returns the queue a tenant's event fan-out or delivery job
// goes to, or "" for the job's default queue.
func (j *jobInserter) tenantQueue(ctx context.Context, args river.JobArgs) string {
	if j.tenantLimits == nil {
		return ""
	}
	var tenantArg string
	switch a := args.(type) {
	case EventArgs:
		tenantArg = a.TenantID
	case *EventArgs:
		tenantArg = a.TenantID
	case WebhookArgs:
		tenantArg = a.TenantID
	case *WebhookArgs:
		tenantArg = a.TenantID
	default:
		return ""
	}
	tenantID, err := uuid.Parse(tenantArg)
	if err != nil {
		return ""
	}
	queue := j.tenantLimits(ctx, tenantID).Queue
	if _, declared := j.extraQueues[queue]; !declared {
		if queue != "" {
			j.logger.WarnContext(ctx, "Tenant queue is not declared, using the default queue", "queue", queue, "tenant_id", tenantID)
		}
		return ""
	}
	return queue
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
		j.logger.ErrorContext(ctx, "Failed to batch insert jobs", "error", err)
		return nil, fmt.Errorf("failed to batch insert jobs: %w", err)
	}
	return insertResults, nil
}

// BatchInsertTx inserts jobs inside tx: they become visible, and are
// worked, only if tx commits.
func (j *jobInserter) BatchInsertTx(ctx context.Context, tx *sql.Tx, args []river.JobArgs) error {
	if j.txClient == nil {
		return fmt.Errorf("job inserter has no transactional client")
	}
	params := make([]river.InsertManyParams, 0, len(args))
	for _, arg := range args {
		params = append(params, river.InsertManyParams{Args: arg, InsertOpts: j.InsertOpts(ctx, arg)})
	}
	if _, err := j.txClient.InsertManyTx(ctx, tx, params); err != nil {
		return fmt.Errorf("failed to batch insert jobs in transaction: %w", err)
	}
	return nil
}

// QueueManagerInterface defines the interface for queue management.
//
//go:generate go tool gowrap gen -g -i JobInserter -t ../../../templates/opentelemetry.tmpl -o JobInserter_otel.go
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error)
	BatchInsert(ctx context.Context, args []river.JobArgs) ([]*rivertype.JobInsertResult, error)
	BatchInsertTx(ctx context.Context, tx *sql.Tx, args []river.JobArgs) error
}

var _ JobInserter = (*jobInserter)(nil)
