// Package health provides health checking functionality for the Sparrow webhook service
package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sarathsp06/sparrow"
)

// Checker provides health check functionality
type Checker struct {
	dbPool    *pgxpool.Pool
	startTime time.Time
	draining  atomic.Bool
}

// SetDraining marks the process as shutting down: /ready answers 503 from
// then on, so load balancers stop routing to it while in-flight work ends.
// /health is unaffected; the process is still alive.
func (hc *Checker) SetDraining() {
	hc.draining.Store(true)
}

// NewChecker creates a new health checker
func NewChecker(dbPool *pgxpool.Pool, startTime time.Time) *Checker {
	return &Checker{
		dbPool:    dbPool,
		startTime: startTime,
	}
}

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status    string         `json:"status"`
	Version   string         `json:"version"`
	Timestamp string         `json:"timestamp"`
	Uptime    string         `json:"uptime"`
	Checks    map[string]any `json:"checks"`
	Service   map[string]any `json:"service"`
}

// ReadyResponse represents the readiness check response structure
type ReadyResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// Health is the liveness check: it reports whether the process is up and
// serving, and nothing else. It never touches the database: a database
// outage must make the instance unready (see Ready), not get it restarted,
// which would not fix the database and would drop in-flight deliveries.
func (hc *Checker) Health(ctx context.Context) (HealthResponse, int) {
	return HealthResponse{
		Status:    "healthy",
		Version:   sparrow.Version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    time.Since(hc.startTime).Round(time.Second).String(),
		Checks: map[string]any{
			"process": map[string]any{"status": "healthy"},
		},
		Service: map[string]any{
			"name":        "sparrow",
			"description": "Webhook delivery system",
		},
	}, http.StatusOK
}

// Ready performs a readiness check: the process is not shutting down and
// the database is reachable.
func (hc *Checker) Ready(ctx context.Context) (ReadyResponse, int) {
	if hc.draining.Load() {
		return ReadyResponse{
			Status:    "draining",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   sparrow.Version,
		}, http.StatusServiceUnavailable
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := hc.dbPool.Ping(pingCtx); err != nil {
		// Log server-side only; /ready is unauthenticated, so never echo
		// raw error strings to the caller.
		slog.ErrorContext(ctx, "readiness check: database ping failed", "error", err)
		return ReadyResponse{
			Status:    "not ready",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   sparrow.Version,
		}, http.StatusServiceUnavailable
	}

	return ReadyResponse{
		Status:    "ready",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   sparrow.Version,
	}, http.StatusOK
}

// HealthHandler returns an HTTP handler for health checks
func (hc *Checker) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		healthResponse, httpStatus := hc.Health(r.Context())

		w.WriteHeader(httpStatus)
		if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
			http.Error(w, "Failed to encode health response", http.StatusInternalServerError)
		}
	}
}

// ReadyHandler returns an HTTP handler for readiness checks
func (hc *Checker) ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		readyResponse, httpStatus := hc.Ready(r.Context())

		w.WriteHeader(httpStatus)
		if err := json.NewEncoder(w).Encode(readyResponse); err != nil {
			http.Error(w, "Failed to encode ready response", http.StatusInternalServerError)
		}
	}
}
