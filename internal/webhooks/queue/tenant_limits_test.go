package queue

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

func TestTenantGateCapsEachTenant(t *testing.T) {
	ctx := context.Background()
	capped, free := uuid.New(), uuid.New()
	g := newTenantGate(func(_ context.Context, id uuid.UUID) TenantLimits {
		if id == capped {
			return TenantLimits{MaxInFlight: 1}
		}
		return TenantLimits{}
	})

	release, ok := g.acquire(ctx, capped)
	if !ok {
		t.Fatal("first job of a capped tenant was refused")
	}
	if _, ok := g.acquire(ctx, capped); ok {
		t.Fatal("second job ran while the tenant was at its cap")
	}
	for range 5 {
		if _, ok := g.acquire(ctx, free); !ok {
			t.Fatal("a tenant without a cap was refused")
		}
	}
	release()
	if _, ok := g.acquire(ctx, capped); !ok {
		t.Fatal("capped tenant still refused after its job finished")
	}
}

func TestTenantGateNilAdmitsEverything(t *testing.T) {
	g := newTenantGate(nil)
	if g != nil {
		t.Fatal("no limiter should give no gate")
	}
	for range 3 {
		release, ok := g.acquire(context.Background(), uuid.New())
		if !ok {
			t.Fatal("self-hosted Sparrow refused a job")
		}
		release()
	}
}

func TestFairnessSnoozeIsASnooze(t *testing.T) {
	var snooze *river.JobSnoozeError
	if !errors.As(fairnessSnooze(), &snooze) {
		t.Fatal("fairnessSnooze must return a River snooze, which uses no retry attempt")
	}
}

func TestJobInserterRoutesTenantQueue(t *testing.T) {
	ctx := context.Background()
	free, paid := uuid.New(), uuid.New()
	ji := &jobInserter{
		tenantLimits: func(_ context.Context, id uuid.UUID) TenantLimits {
			switch id {
			case free:
				return TenantLimits{Queue: "free"}
			case paid:
				return TenantLimits{Queue: "undeclared"}
			}
			return TenantLimits{}
		},
		extraQueues: map[string]int{"free": 2},
	}
	ji.logger = slog.New(slog.DiscardHandler)

	cases := []struct {
		name string
		args river.JobArgs
		want string
	}{
		{"event job of a routed tenant", EventArgs{TenantID: free.String()}, "free"},
		{"delivery job of a routed tenant", &WebhookArgs{TenantID: free.String()}, "free"},
		{"undeclared queue falls back", WebhookArgs{TenantID: paid.String()}, QueueWebhookDelivery},
		{"tenant without a queue", EventArgs{TenantID: uuid.NewString()}, QueueEventProcessing},
		{"batch jobs are never routed", BatchJobArgs{TenantID: free.String()}, QueueBatchJobs},
	}
	for _, tc := range cases {
		if got := ji.InsertOpts(ctx, tc.args).Queue; got != tc.want {
			t.Errorf("%s: queue %q, want %q", tc.name, got, tc.want)
		}
	}

	self := &jobInserter{logger: slog.New(slog.DiscardHandler)}
	if got := self.InsertOpts(ctx, EventArgs{TenantID: free.String()}).Queue; got != QueueEventProcessing {
		t.Errorf("self-hosted inserter routed a job to %q", got)
	}
}

// TestTenantGateUnderConcurrency hammers the gate from many goroutines and
// checks that no tenant ever runs more than its cap at once, that tenants
// without a cap are never refused, and (under -race) that the gate's
// bookkeeping is safe.
func TestTenantGateUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	capOne, capThree, uncapped := uuid.New(), uuid.New(), uuid.New()
	caps := map[uuid.UUID]int{capOne: 1, capThree: 3}
	g := newTenantGate(func(_ context.Context, id uuid.UUID) TenantLimits {
		return TenantLimits{MaxInFlight: caps[id]}
	})

	var mu sync.Mutex
	running := map[uuid.UUID]int{}
	peak := map[uuid.UUID]int{}
	refusedUncapped := 0

	var wg sync.WaitGroup
	for i := range 60 {
		id := []uuid.UUID{capOne, capThree, uncapped}[i%3]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				release, ok := g.acquire(ctx, id)
				if !ok {
					if id == uncapped {
						mu.Lock()
						refusedUncapped++
						mu.Unlock()
					}
					continue
				}
				mu.Lock()
				running[id]++
				peak[id] = max(peak[id], running[id])
				mu.Unlock()
				runtime.Gosched()
				mu.Lock()
				running[id]--
				mu.Unlock()
				release()
			}
		}()
	}
	wg.Wait()

	if peak[capOne] > 1 || peak[capThree] > 3 {
		t.Fatalf("cap exceeded: peak %d for cap 1, %d for cap 3", peak[capOne], peak[capThree])
	}
	if peak[capOne] != 1 || peak[capThree] < 2 {
		t.Fatalf("gate admitted too little: peak %d for cap 1, %d for cap 3", peak[capOne], peak[capThree])
	}
	if refusedUncapped != 0 {
		t.Fatalf("an uncapped tenant was refused %d times", refusedUncapped)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.running) != 0 {
		t.Fatalf("gate still counts running jobs after all released: %v", g.running)
	}
}
