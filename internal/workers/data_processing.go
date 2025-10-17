package workers

import (
	"context"
	"time"

	"github.com/riverqueue/river"
	"github.com/sarathsp06/httpqueue/internal/jobs"
	"github.com/sarathsp06/httpqueue/internal/logger"
)

// DataProcessingWorker handles data processing jobs
type DataProcessingWorker struct {
	river.WorkerDefaults[jobs.DataProcessingArgs]
}

// Work processes the data processing job
func (w DataProcessingWorker) Work(ctx context.Context, job *river.Job[jobs.DataProcessingArgs]) error {
	log := logger.NewLogger("data-processing-worker")

	log.Info("Processing data job",
		"job_id", job.ID,
		"data_id", job.Args.DataID,
		"data_type", job.Args.DataType,
	)

	// Simulate data processing
	time.Sleep(3 * time.Second)

	log.Info("Data processed successfully",
		"job_id", job.ID,
		"data_id", job.Args.DataID,
		"data_type", job.Args.DataType,
	)
	return nil
}
