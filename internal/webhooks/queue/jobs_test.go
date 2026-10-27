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

// River merges a numeric "snoozes" counter into job metadata on every
// snooze; the trace carrier must tolerate it (a fairness-capped tenant
// snoozes on every run, and this used to log an ERROR each time).
func TestTraceCarrierToleratesRiverMetadata(t *testing.T) {
	carrier := traceCarrier([]byte(`{"traceparent":"00-abc-def-01","snoozes":3}`))
	assert.Equal(t, "00-abc-def-01", carrier["traceparent"])
	_, hasSnoozes := carrier["snoozes"]
	assert.False(t, hasSnoozes)

	assert.Empty(t, traceCarrier([]byte(`not json`)))
}

// Mocking River client is hard, so let's focus on the logic in InsertOpts first.
// If we fix InsertOpts to merge, the problem is solved.
