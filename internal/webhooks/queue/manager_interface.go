package queue

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/sarathsp06/sparrow/internal/webhooks/jobs"
)

// QueueManagerInterface defines the interface for queue management.
type QueueManagerInterface interface {
	QueueWebhook(ctx context.Context, args *jobs.WebhookArgs) error
	QueueEvent(ctx context.Context, args *jobs.EventArgs) error
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

var _ QueueManagerInterface = (*Manager)(nil)
