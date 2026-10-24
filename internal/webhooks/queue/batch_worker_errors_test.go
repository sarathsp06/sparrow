package queue

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/sarathsp06/sparrow/internal/storage"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// scriptedBatchRepo answers GetBatchJob and UpdateBatchJobStatus as told and
// records the status writes.
type scriptedBatchRepo struct {
	store.BatchRepository
	batch     *store.BatchJob
	getErr    error
	statusErr error
	statuses  []store.BatchJobStatus
}

func (s *scriptedBatchRepo) GetBatchJob(context.Context, uuid.UUID, uuid.UUID) (*store.BatchJob, error) {
	return s.batch, s.getErr
}

func (s *scriptedBatchRepo) UpdateBatchJobStatus(_ context.Context, _, _ uuid.UUID, _, to store.BatchJobStatus) error {
	s.statuses = append(s.statuses, to)
	return s.statusErr
}

func batchJob(tenantID, batchID string) *river.Job[BatchJobArgs] {
	return &river.Job[BatchJobArgs]{
		JobRow: &rivertype.JobRow{Metadata: []byte(`not json`)},
		Args:   BatchJobArgs{TenantID: tenantID, BatchID: batchID},
	}
}

// TestBatchJobWorkerStopsEarly covers every way a batch job ends before it
// handles an item: malformed args cancel the job, a missing batch errors so
// River retries, a cancelled batch is skipped, and a batch that cannot be
// handled is marked failed (a lost race on that write is not an error).
func TestBatchJobWorkerStopsEarly(t *testing.T) {
	ctx := context.Background()
	tenantID, batchID := uuid.NewString(), uuid.NewString()

	for _, tc := range []struct {
		name       string
		job        *river.Job[BatchJobArgs]
		repo       *scriptedBatchRepo
		wantErr    string
		wantCancel bool
		wantStatus []store.BatchJobStatus
	}{
		{name: "bad tenant id", job: batchJob("nope", batchID), repo: &scriptedBatchRepo{}, wantErr: "invalid tenant ID", wantCancel: true},
		{name: "bad batch id", job: batchJob(tenantID, "nope"), repo: &scriptedBatchRepo{}, wantErr: "invalid batch ID", wantCancel: true},
		{name: "lookup fails", job: batchJob(tenantID, batchID), repo: &scriptedBatchRepo{getErr: errors.New("db down")}, wantErr: "db down"},
		{name: "batch missing", job: batchJob(tenantID, batchID), repo: &scriptedBatchRepo{}, wantErr: "not found"},
		{name: "cancelled", job: batchJob(tenantID, batchID), repo: &scriptedBatchRepo{batch: &store.BatchJob{Status: store.BatchStatusCancelled}}},
		{
			name: "unreadable data", job: batchJob(tenantID, batchID),
			repo:    &scriptedBatchRepo{batch: &store.BatchJob{Status: store.BatchStatusProcessing, Data: []byte(`{`)}, statusErr: errors.New("write failed")},
			wantErr: "parse batch data", wantStatus: []store.BatchJobStatus{store.BatchStatusFailed},
		},
		{
			name: "unknown job type", job: batchJob(tenantID, batchID),
			repo:    &scriptedBatchRepo{batch: &store.BatchJob{Status: store.BatchStatusProcessing, JobType: "mystery", Data: []byte(`{"item_ids":[]}`)}, statusErr: storage.ErrNotFound},
			wantErr: "unknown batch job type", wantStatus: []store.BatchJobStatus{store.BatchStatusFailed},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &BatchJobWorker{batchRepo: tc.repo, logger: slog.Default()}
			err := w.Work(ctx, tc.job)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Work = %v, want nil", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Work = %v, want an error containing %q", err, tc.wantErr)
			}
			var cancel *river.JobCancelError
			if got := errors.As(err, &cancel); got != tc.wantCancel {
				t.Fatalf("JobCancel = %v, want %v (err %v)", got, tc.wantCancel, err)
			}
			if len(tc.repo.statuses) != len(tc.wantStatus) || (len(tc.wantStatus) > 0 && tc.repo.statuses[0] != tc.wantStatus[0]) {
				t.Fatalf("status writes = %v, want %v", tc.repo.statuses, tc.wantStatus)
			}
		})
	}
}

type cleanupRepo struct {
	store.BatchRepository
	deleted int
	err     error
}

func (c cleanupRepo) CleanupExpiredBatchJobs(context.Context) (int, error) { return c.deleted, c.err }

func TestBatchCleanupWorker(t *testing.T) {
	job := &river.Job[BatchCleanupArgs]{JobRow: &rivertype.JobRow{}}
	if err := NewBatchCleanupWorker(cleanupRepo{deleted: 3}).Work(context.Background(), job); err != nil {
		t.Fatalf("Work = %v", err)
	}
	if err := NewBatchCleanupWorker(cleanupRepo{}).Work(context.Background(), job); err != nil {
		t.Fatalf("Work with nothing to delete = %v", err)
	}
	boom := errors.New("db down")
	if err := NewBatchCleanupWorker(cleanupRepo{err: boom}).Work(context.Background(), job); !errors.Is(err, boom) {
		t.Fatalf("Work = %v, want the repository error so River retries", err)
	}
}
