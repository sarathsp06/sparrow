// Package tlsrevocation connects safetransport's in-memory CRL cache to a
// Sparrow cluster: POST /v1/tls/crls:refresh sends a Postgres NOTIFY, and
// every instance listening on the channel refreshes its own cache. No table,
// no scheduled job: CRLs are public and re-downloadable, so each instance
// keeps its own copy.
package tlsrevocation

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sarathsp06/sparrow/pkg/safetransport"
)

// Channel is the NOTIFY channel that asks every instance to refresh its CRLs.
const Channel = "sparrow_crl_refresh"

// RefreshInterval is the minimum time between refresh requests on one
// server, so the endpoint cannot be used to hammer CAs' CRL servers.
const RefreshInterval = time.Minute

// ErrTooSoon is returned by RequestRefresh within RefreshInterval of the
// previous request on this server.
var ErrTooSoon = errors.New("a CRL refresh was requested less than a minute ago")

// ErrDisabled is returned by RequestRefresh when SPARROW_TLS_REVOCATION=off.
var ErrDisabled = errors.New("revocation checking is off (SPARROW_TLS_REVOCATION=off)")

// Refresher owns this instance's revocation checker and its refresh channel.
// A nil *Refresher means revocation checking is off.
type Refresher struct {
	checker *safetransport.RevocationChecker
	pool    *pgxpool.Pool
	mode    string

	mu   sync.Mutex
	last time.Time

	// Test hooks: called once LISTEN is in place and after each refresh.
	onListen, onRefresh func()
}

// New returns a Refresher for checker, which runs in mode (soft or hard).
func New(pool *pgxpool.Pool, checker *safetransport.RevocationChecker, mode string) *Refresher {
	return &Refresher{checker: checker, pool: pool, mode: mode}
}

// Mode reports soft, hard or off.
func (r *Refresher) Mode() string {
	if r == nil {
		return "off"
	}
	return r.mode
}

// Status lists this instance's cached CRLs.
func (r *Refresher) Status() []safetransport.CRLStatus {
	if r == nil {
		return nil
	}
	return r.checker.Status()
}

// RequestRefresh asks every instance, this one included, to download its
// known CRLs now. It returns once the request is sent, not when the
// downloads finish.
func (r *Refresher) RequestRefresh(ctx context.Context) error {
	if r == nil {
		return ErrDisabled
	}
	r.mu.Lock()
	if !r.last.IsZero() && time.Since(r.last) < RefreshInterval {
		r.mu.Unlock()
		return ErrTooSoon
	}
	r.last = time.Now()
	r.mu.Unlock()

	_, err := r.pool.Exec(ctx, "SELECT pg_notify($1, '')", Channel)
	return err
}

// Listen refreshes the checker on every notification until ctx is done,
// reconnecting after connection errors. A notification missed while
// reconnecting is covered by the next handshake older than the max age.
func (r *Refresher) Listen(ctx context.Context) {
	for {
		err := r.listen(ctx)
		if ctx.Err() != nil {
			return
		}
		slog.Warn("CRL refresh listener stopped, reconnecting", "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (r *Refresher) listen(ctx context.Context) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	// The connection stays in LISTEN mode, so it never goes back to the pool.
	pgConn := conn.Hijack()
	defer func() { _ = pgConn.Close(context.Background()) }()

	if _, err := pgConn.Exec(ctx, "LISTEN "+Channel); err != nil {
		return err
	}
	if r.onListen != nil {
		r.onListen()
	}
	for {
		if _, err := pgConn.WaitForNotification(ctx); err != nil {
			return err
		}
		refreshCtx, cancel := context.WithTimeout(ctx, time.Minute)
		if err := r.checker.Refresh(refreshCtx); err != nil {
			slog.Warn("CRL refresh finished with errors", "error", err)
		} else {
			slog.Info("CRLs refreshed", "count", len(r.checker.Status()))
		}
		cancel()
		if r.onRefresh != nil {
			r.onRefresh()
		}
	}
}
