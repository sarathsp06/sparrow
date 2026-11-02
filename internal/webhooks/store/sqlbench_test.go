//go:build sqlbench

package store

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/storage/postgres"
	"github.com/sarathsp06/sparrow/internal/tenant"
)

// TestSQLBench times the repository's heavy read paths against a database
// seeded by scripts/sqlbench/seed.sql. It calls the real repository methods,
// so it always measures the SQL that ships. Run it through
// scripts/sqlbench/run.sh, or directly:
//
//	SQLBENCH_DSN=postgres://... go test -tags sqlbench -run TestSQLBench -v ./internal/webhooks/store/
//
// Each case runs SQLBENCH_RUNS times (default 7) on one connection, so pgx's
// cached prepared statements and Postgres's generic plans come into play
// after the fifth run, as in production. The last case folds the health
// evaluator backlog the seed leaves behind; it writes, so it reports
// "no backlog" on a second run until the database is re-seeded.
func TestSQLBench(t *testing.T) {
	dsn := os.Getenv("SQLBENCH_DSN")
	if dsn == "" {
		t.Skip("SQLBENCH_DSN not set")
	}
	runs := 7
	if v := os.Getenv("SQLBENCH_RUNS"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &runs); err != nil || runs < 1 {
			t.Fatalf("bad SQLBENCH_RUNS %q", v)
		}
	}
	db, err := postgres.Open(dsn, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	db.SetMaxOpenConns(1)
	r := NewRepository(db)
	ctx := context.Background()
	tid := tenant.DefaultTenantID

	// Shapes from seed.sql: the first webhook carries ~10% of deliveries.
	var hotWebhook uuid.UUID
	if err := db.GetContext(ctx, &hotWebhook, `SELECT id FROM webhook_registrations ORDER BY created_at LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	const hot, mid, sparse = "consumer_0001", "consumer_0150", "consumer_0252"
	failed, paused := "failed", "paused"

	type result struct {
		name          string
		first, median time.Duration
		maxT          time.Duration
		note          string
	}
	var results []result
	bench := func(name string, f func() (string, error)) {
		var times []time.Duration
		var note string
		for i := 0; i < runs; i++ {
			start := time.Now()
			n, err := f()
			times = append(times, time.Since(start))
			if err != nil {
				note = "error: " + err.Error()
				break
			}
			note = n
		}
		sorted := append([]time.Duration(nil), times...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		results = append(results, result{name, times[0], sorted[len(sorted)/2], sorted[len(sorted)-1], note})
	}
	rows := func(n int, more bool) string { return fmt.Sprintf("%d rows, more=%v", n, more) }

	list := func(f DeliveryFilter) func() (string, error) {
		return func() (string, error) {
			f.Limit = 50
			items, more, err := r.ListDeliveriesFiltered(ctx, tid, f)
			return rows(len(items), more), err
		}
	}
	bench("deliveries: first page, no filter", list(DeliveryFilter{}))
	bench("deliveries: status=failed", list(DeliveryFilter{Status: &failed}))
	bench("deliveries: hot webhook", list(DeliveryFilter{Consumer: hot, WebhookID: &hotWebhook}))
	bench("deliveries: consumer, hot", list(DeliveryFilter{Consumer: hot}))
	bench("deliveries: consumer, mid", list(DeliveryFilter{Consumer: mid}))
	bench("deliveries: consumer, sparse", list(DeliveryFilter{Consumer: sparse}))
	bench("deliveries: consumer hot, status=failed", list(DeliveryFilter{Consumer: hot, Status: &failed}))

	// Cursor depth: walk 200 pages once, then time the 201st.
	var deep *PageCursor
	for i := 0; i < 200; i++ {
		items, more, err := r.ListDeliveriesFiltered(ctx, tid, DeliveryFilter{Limit: 50, After: deep})
		if err != nil || !more || len(items) == 0 {
			break
		}
		last := items[len(items)-1]
		deep = &PageCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	bench("deliveries: page 201 by cursor", list(DeliveryFilter{After: deep}))

	bench("deliveries: count paused, hot webhook (resume)", func() (string, error) {
		n, err := r.CountDeliveries(ctx, tid, DeliveryFilter{Consumer: hot, WebhookID: &hotWebhook, Status: &paused})
		return fmt.Sprintf("count=%d", n), err
	})
	bench("retry snapshot: consumer hot, failed", func() (string, error) {
		ids, err := r.SnapshotDeliveryIDs(ctx, tid, DeliveryFilter{Consumer: hot, Status: &failed})
		if err != nil && strings.Contains(err.Error(), "narrow the filter") {
			return "rejected: over the batch limit (expected at scale 1)", nil
		}
		return fmt.Sprintf("%d ids", len(ids)), err
	})
	bench("retry snapshot: consumer mid, failed", func() (string, error) {
		ids, err := r.SnapshotDeliveryIDs(ctx, tid, DeliveryFilter{Consumer: mid, Status: &failed})
		return fmt.Sprintf("%d ids", len(ids)), err
	})

	events := func(f EventReportFilter) func() (string, error) {
		return func() (string, error) {
			f.Limit = 50
			items, more, err := r.ListEventReportsFiltered(ctx, tid, f)
			return rows(len(items), more), err
		}
	}
	bench("events: first page, no filter", events(EventReportFilter{}))
	bench("events: consumer, hot", events(EventReportFilter{Consumer: hot}))
	orderCreated := "order.created"
	bench("events: consumer hot, event=order.created", events(EventReportFilter{Consumer: hot, EventName: &orderCreated}))

	bench("stats: consumer, hot", func() (string, error) {
		s, err := r.GetConsumerStats(ctx, tid, hot)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d webhooks, %d attempts", s.TotalWebhooks, s.TotalDeliveries), nil
	})
	bench("stats: all consumers", func() (string, error) {
		s, err := r.GetConsumerStats(ctx, tid, "")
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d webhooks, %d attempts", s.TotalWebhooks, s.TotalDeliveries), nil
	})
	bench("webhook health summary: hot webhook, 24h", func() (string, error) {
		s, err := r.GetWebhookHealthSummary(ctx, tid, hotWebhook, DefaultHealthRules.WindowHours)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d deliveries", s.TotalDeliveries), nil
	})
	var hotEvent string
	if err := db.GetContext(ctx, &hotEvent, `SELECT event_name FROM event_subscriptions WHERE consumer = $1 GROUP BY 1 ORDER BY count(*) DESC LIMIT 1`, hot); err != nil {
		t.Fatal(err)
	}
	bench("fan-out: subscriptions for an event, hot consumer", func() (string, error) {
		subs, err := r.GetSubscriptionsWithWebhooksByEvent(ctx, tid, hot, hotEvent, map[string]string{"env": "production"})
		return fmt.Sprintf("%d subscriptions", len(subs)), err
	})
	bench("webhooks: list, hot consumer", func() (string, error) {
		items, total, err := r.ListWebhooksPaginated(ctx, tid, hot, "", false, "", 50, 0)
		return fmt.Sprintf("%d of %d", len(items), total), err
	})
	bench("consumers: list (loose index scan)", func() (string, error) {
		names, err := r.ListConsumers(ctx, tid, "", 50)
		return fmt.Sprintf("%d consumers", len(names)), err
	})

	// Health evaluator: fold the backlog after the seed's watermark, in the
	// worker's default passes.
	var passes, folded int
	var slowest time.Duration
	start := time.Now()
	for passes < 1000 {
		t0 := time.Now()
		res, err := r.EvaluateHealth(ctx, time.Now(), 0)
		if err != nil {
			t.Fatal(err)
		}
		if d := time.Since(t0); d > slowest {
			slowest = d
		}
		if res.Events == 0 {
			break
		}
		passes++
		folded += res.Events
		if !res.More {
			break
		}
	}
	note := "no backlog (re-seed to measure)"
	if folded > 0 {
		note = fmt.Sprintf("%d events in %d passes, %s total", folded, passes, time.Since(start).Round(time.Millisecond))
	}
	results = append(results, result{"health evaluator: fold backlog (slowest pass)", slowest, slowest, slowest, note})

	var b strings.Builder
	fmt.Fprintf(&b, "\n%-52s %10s %10s %10s  %s\n", "case", "first", "median", "max", "result")
	for _, x := range results {
		fmt.Fprintf(&b, "%-52s %10s %10s %10s  %s\n", x.name, ms(x.first), ms(x.median), ms(x.maxT), x.note)
	}
	t.Log(b.String())
	if out := os.Getenv("SQLBENCH_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, x := range results {
		if strings.HasPrefix(x.note, "error:") {
			t.Errorf("%s: %s", x.name, x.note)
		}
	}
}

func ms(d time.Duration) string { return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000) }
