package queue

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// fakeBatchRepo keeps one batch row and applies progress and status writes to
// it, failing any write made on a context that is already done.
type fakeBatchRepo struct {
	store.BatchRepository
	batch *store.BatchJob
}

func (f *fakeBatchRepo) GetBatchJob(ctx context.Context, _, _ uuid.UUID) (*store.BatchJob, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b := *f.batch
	return &b, nil
}

func (f *fakeBatchRepo) UpdateBatchJobProgress(ctx context.Context, _, _ uuid.UUID, processed, failed int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.batch.Processed += processed
	f.batch.Failed += failed
	return nil
}

func (f *fakeBatchRepo) UpdateBatchJobStatus(ctx context.Context, _, _ uuid.UUID, from, to store.BatchJobStatus) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if f.batch.Status == from {
		f.batch.Status = to
	}
	return nil
}

// fakeRepushEventRepo serves any event and records the re-pushed ones.
type fakeRepushEventRepo struct {
	store.EventRepository
	loaded []uuid.UUID
}

func (f *fakeRepushEventRepo) GetEventByID(ctx context.Context, _, id uuid.UUID) (*store.EventRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.loaded = append(f.loaded, id)
	return &store.EventRecord{ID: id, Consumer: "acme", Event: "order.created"}, nil
}

func (f *fakeRepushEventRepo) StoreEventAndEnqueue(ctx context.Context, _ uuid.UUID, _ *store.EventRecord, enqueue store.EventEnqueuer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return enqueue(ctx, nil)
}

func newBatchFixture(t *testing.T, n, done int) (*BatchJobWorker, *fakeBatchRepo, *fakeRepushEventRepo, []string, *river.Job[BatchJobArgs]) {
	t.Helper()
	ids := make([]string, n)
	for i := range ids {
		ids[i] = uuid.NewString()
	}
	data, err := json.Marshal(store.BatchJobData{ItemIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	batchID := uuid.New()
	batches := &fakeBatchRepo{batch: &store.BatchJob{
		ID: batchID, JobType: store.BatchTypeEventRepush, Status: store.BatchStatusProcessing,
		Data: data, Total: n, Processed: done,
	}}
	events := &fakeRepushEventRepo{}
	w := &BatchJobWorker{batchRepo: batches, eventRepo: events, jobInserter: noopJobInserter{}, logger: slog.Default()}
	job := &river.Job[BatchJobArgs]{JobRow: &rivertype.JobRow{Metadata: []byte(`{}`)}, Args: BatchJobArgs{TenantID: uuid.NewString(), BatchID: batchID.String()}}
	return w, batches, events, ids, job
}

// A retried batch job resumes after the items an earlier run counted, so a
// timeout or shutdown never re-pushes the same events twice.
func TestBatchJobWorkerResumesAfterRecordedProgress(t *testing.T) {
	w, batches, events, ids, job := newBatchFixture(t, 5, 3)

	if err := w.Work(context.Background(), job); err != nil {
		t.Fatalf("Work: %v", err)
	}
	if len(events.loaded) != 2 || events.loaded[0].String() != ids[3] || events.loaded[1].String() != ids[4] {
		t.Fatalf("re-pushed %v, want only the last two items %v", events.loaded, ids[3:])
	}
	if batches.batch.Processed != 5 || batches.batch.Status != store.BatchStatusCompleted {
		t.Fatalf("batch processed=%d status=%s, want 5/completed", batches.batch.Processed, batches.batch.Status)
	}
}

// A run whose context dies mid-batch still records its progress (the writes
// must not use the dead context), leaves the batch processing, and returns an
// error so River retries it.
func TestBatchJobWorkerInterruptedRecordsProgress(t *testing.T) {
	w, batches, events, _, job := newBatchFixture(t, 5, 0)
	ctx, cancel := context.WithCancel(context.Background())
	stopAfter := 2
	w.eventRepo = &cancellingEventRepo{fakeRepushEventRepo: events, after: stopAfter, cancel: cancel}

	err := w.Work(ctx, job)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Work error = %v, want context.Canceled so River retries", err)
	}
	if batches.batch.Processed != stopAfter {
		t.Fatalf("recorded processed=%d, want %d", batches.batch.Processed, stopAfter)
	}
	if batches.batch.Status != store.BatchStatusProcessing {
		t.Fatalf("status = %s, want processing until the retry finishes", batches.batch.Status)
	}
}

// cancellingEventRepo cancels the job context once after events stored.
type cancellingEventRepo struct {
	*fakeRepushEventRepo
	after  int
	stored int
	cancel context.CancelFunc
}

func (c *cancellingEventRepo) StoreEventAndEnqueue(ctx context.Context, tenantID uuid.UUID, e *store.EventRecord, enqueue store.EventEnqueuer) error {
	if err := c.fakeRepushEventRepo.StoreEventAndEnqueue(ctx, tenantID, e, enqueue); err != nil {
		return err
	}
	c.stored++
	if c.stored == c.after {
		c.cancel()
	}
	return nil
}
