package queue

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// TenantLimits bound one tenant's share of the shared queues. Zero values
// mean no limit and the default queue.
type TenantLimits struct {
	// MaxInFlight caps the tenant's event fan-out and delivery jobs running
	// at once in this process. With N processes a tenant can run up to
	// N × MaxInFlight.
	MaxInFlight int
	// Queue, when set, is one of WorkerPoolConfig.ExtraQueues: the tenant's
	// event fan-out and delivery jobs are inserted there instead of the
	// default queues.
	Queue string
}

// TenantLimiter returns the limits for a tenant. It is called for every
// event fan-out and delivery job, at insert and at work time, so it must be
// fast (cache what it looks up).
type TenantLimiter func(ctx context.Context, tenantID uuid.UUID) TenantLimits

// tenantGate counts each tenant's running jobs in this process. A nil gate
// or one without a limiter admits everything, which is self-hosted Sparrow.
type tenantGate struct {
	limits  TenantLimiter
	mu      sync.Mutex
	running map[uuid.UUID]int
}

func newTenantGate(limits TenantLimiter) *tenantGate {
	if limits == nil {
		return nil
	}
	return &tenantGate{limits: limits, running: map[uuid.UUID]int{}}
}

// acquire admits a job for tenantID unless the tenant is at its cap. The
// returned release must be called when the job ends.
func (g *tenantGate) acquire(ctx context.Context, tenantID uuid.UUID) (release func(), ok bool) {
	if g == nil {
		return func() {}, true
	}
	limit := g.limits(ctx, tenantID).MaxInFlight
	if limit <= 0 {
		return func() {}, true
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.running[tenantID] >= limit {
		return nil, false
	}
	g.running[tenantID]++
	return func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.running[tenantID]--; g.running[tenantID] <= 0 {
			delete(g.running, tenantID)
		}
	}, true
}

// fairnessSnooze puts a job back for one to two seconds. River does not
// count a snooze as an attempt, and no delivery attempt or health event is
// recorded, so a tenant at its cap never burns retries or hurts its
// receivers' health.
func fairnessSnooze() error {
	return river.JobSnooze(time.Second + rand.N(time.Second)) //nolint:gosec // jitter, not security
}
