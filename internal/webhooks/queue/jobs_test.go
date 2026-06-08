package queue

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"
)

func TestJobInserter_InsertOpts_Merge(t *testing.T) {
	inserter := &jobInserter{}
	ctx := context.Background()

	args := WebhookArgs{
		MaxAttempts: 10,
	}

	// Get the opts from the inserter, which should merge metadata into the base opts
	opts := inserter.InsertOpts(ctx, args)

	// FIXED BEHAVIOR:
	assert.Equal(t, QueueWebhookDelivery, opts.Queue)
	assert.Equal(t, 10, opts.MaxAttempts)
	assert.NotEmpty(t, opts.Metadata)

	var carrier propagation.MapCarrier
	err := json.Unmarshal(opts.Metadata, &carrier)
	assert.NoError(t, err)
}

// Mocking River client is hard, so let's focus on the logic in InsertOpts first.
// If we fix InsertOpts to merge, the problem is solved.
