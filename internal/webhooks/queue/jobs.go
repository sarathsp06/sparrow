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
	"github.com/lib/pq"
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
	// ScheduledAt delays the fan-out: River holds the job until then.
	// Zero means run immediately.
	ScheduledAt time.Time `json:"scheduled_at,omitzero"`
}

var _ river.JobArgsWithInsertOpts = (*EventArgs)(nil)

func (a EventArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueEventProcessing,
		ScheduledAt: a.ScheduledAt,
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

// traceCarrier extracts the string values of job metadata into a trace
// carrier. River writes non-string values of its own into metadata (a
// "snoozes" counter on every snoozed job), so the metadata cannot be
// unmarshalled into the carrier's map[string]string directly.
func traceCarrier(metadata []byte) propagation.MapCarrier {
	carrier := make(propagation.MapCarrier)
	var raw map[string]json.RawMessage
	if json.Unmarshal(metadata, &raw) != nil {
		return carrier
	}
	for k, v := range raw {
		var s string
		if json.Unmarshal(v, &s) == nil {
			carrier[k] = s
		}
	}
	return carrier
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

// CancelWaitingDeliveryJobsTx cancels the webhook delivery jobs of these
// deliveries that have not been picked up by a worker (available, scheduled
// or retryable: a retrying delivery's job waits out its backoff in one of
// these), and returns the deliveries whose job it cancelled. The jobs are
// locked with SKIP LOCKED, so one a worker is fetching right now is left to
// it, and the cancel commits or rolls back with tx. A manual retry uses it to
// replace a retrying delivery's retry chain with a new job instead of adding
// a second one.
func (j *jobInserter) CancelWaitingDeliveryJobsTx(ctx context.Context, tx *sql.Tx, deliveryIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(deliveryIDs) == 0 {
		return nil, nil
	}
	if j.txClient == nil {
		return nil, fmt.Errorf("job inserter has no transactional client")
	}
	ids := make([]string, len(deliveryIDs))
	for i, id := range deliveryIDs {
		ids[i] = id.String()
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id, args->>'delivery_id'
		FROM river_job
		WHERE queue = $1
		  AND kind = $2
		  AND state IN ('available', 'scheduled', 'retryable')
		  AND args->>'delivery_id' = ANY($3)
		FOR UPDATE SKIP LOCKED`,
		QueueWebhookDelivery, WebhookArgs{}.Kind(), pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("find waiting delivery jobs: %w", err)
	}
	type waitingJob struct {
		id         int64
		deliveryID string
	}
	var jobs []waitingJob
	for rows.Next() {
		var wj waitingJob
		if err := rows.Scan(&wj.id, &wj.deliveryID); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan waiting delivery job: %w", err)
		}
		jobs = append(jobs, wj)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("find waiting delivery jobs: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find waiting delivery jobs: %w", err)
	}

	seen := make(map[uuid.UUID]bool, len(jobs))
	claimed := make([]uuid.UUID, 0, len(jobs))
	for _, wj := range jobs {
		if _, err := j.txClient.JobCancelTx(ctx, tx, wj.id); err != nil {
			return nil, fmt.Errorf("cancel delivery job %d: %w", wj.id, err)
		}
		id, err := uuid.Parse(wj.deliveryID)
		if err != nil || seen[id] {
			continue
		}
		seen[id] = true
		claimed = append(claimed, id)
	}
	return claimed, nil
}

// QueueManagerInterface defines the interface for queue management.
//
//go:generate go tool gowrap gen -g -i JobInserter -t ../../../templates/opentelemetry.tmpl -o JobInserter_otel.go
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error)
	BatchInsert(ctx context.Context, args []river.JobArgs) ([]*rivertype.JobInsertResult, error)
	BatchInsertTx(ctx context.Context, tx *sql.Tx, args []river.JobArgs) error
	// CancelWaitingDeliveryJobsTx cancels, inside tx, the delivery jobs of
	// these deliveries that are waiting to run, and returns the deliveries
	// whose job it cancelled (see store.RetryClaimer).
	CancelWaitingDeliveryJobsTx(ctx context.Context, tx *sql.Tx, deliveryIDs []uuid.UUID) ([]uuid.UUID, error)
}

var _ JobInserter = (*jobInserter)(nil)
