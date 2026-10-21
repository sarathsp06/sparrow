package tlsrevocation

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sarathsp06/sparrow/pkg/safetransport"
)

func TestNilRefresherIsOff(t *testing.T) {
	var r *Refresher
	if r.Mode() != "off" || r.Status() != nil {
		t.Fatalf("nil refresher: mode %q, status %v", r.Mode(), r.Status())
	}
	if err := r.RequestRefresh(t.Context()); !errors.Is(err, ErrDisabled) {
		t.Fatalf("got %v, want ErrDisabled", err)
	}
}

// TestRequestRefreshReachesListeners runs two refreshers on one database, as
// two instances would, and checks one request refreshes both. Needs
// DATABASE_URL (CI provides one).
func TestRequestRefreshReachesListeners(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	refreshed := make(chan string, 4)
	var instances []*Refresher
	for _, name := range []string{"a", "b"} {
		listening := make(chan struct{})
		r := New(pool, safetransport.NewRevocationChecker(safetransport.RevocationOptions{}), "soft")
		r.onListen = func() { close(listening) }
		r.onRefresh = func() { refreshed <- name }
		go r.Listen(ctx)
		select {
		case <-listening:
		case <-ctx.Done():
			t.Fatal("listener never started")
		}
		instances = append(instances, r)
	}

	if err := instances[0].RequestRefresh(ctx); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for len(got) < 2 {
		select {
		case name := <-refreshed:
			got[name] = true
		case <-ctx.Done():
			t.Fatalf("refreshed instances: %v, want a and b", got)
		}
	}

	if err := instances[0].RequestRefresh(ctx); !errors.Is(err, ErrTooSoon) {
		t.Fatalf("second request within a minute: got %v, want ErrTooSoon", err)
	}
}
