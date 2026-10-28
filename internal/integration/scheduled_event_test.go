//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_ScheduledEvent verifies that a push with scheduled_at stores the
// event immediately but defers the fan-out: the River event_processing job is
// scheduled for the requested time and no deliveries exist before it fires.
func TestE2E_ScheduledEvent(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	when := time.Now().Add(1 * time.Hour).Truncate(time.Second).UTC()
	eventID, _, _, _, err := env.webhookSvc.PushEvent(
		ctx, "sched-ns", "sched.event",
		map[string]any{"hello": "later"},
		0, nil, nil, nil, &when,
	)
	require.NoError(t, err, "push scheduled event")

	// The event row carries scheduled_at.
	var stored time.Time
	require.NoError(t, env.sqlxDB.GetContext(ctx, &stored,
		`SELECT scheduled_at FROM event_records WHERE id = $1`, eventID))
	assert.True(t, stored.Equal(when), "event_records.scheduled_at: got %v want %v", stored, when)

	// The fan-out job is parked until then: state scheduled, at the right time.
	var job struct {
		State       string    `db:"state"`
		ScheduledAt time.Time `db:"scheduled_at"`
	}
	require.NoError(t, env.sqlxDB.GetContext(ctx, &job,
		`SELECT state::text AS state, scheduled_at FROM river_job
		 WHERE kind = 'event_processing' AND args->>'event_id' = $1`, eventID))
	assert.Equal(t, "scheduled", job.State)
	assert.True(t, job.ScheduledAt.Equal(when), "river_job.scheduled_at: got %v want %v", job.ScheduledAt, when)

	// No deliveries yet: fan-out has not run.
	var n int
	require.NoError(t, env.sqlxDB.GetContext(ctx, &n,
		`SELECT count(*) FROM webhook_deliveries WHERE event_id = $1`, eventID))
	assert.Zero(t, n, "no deliveries before the scheduled time")
}
