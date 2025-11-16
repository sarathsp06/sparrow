// Package health provides health checking functionality for the Sparrow webhook service
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sarathsp06/sparrow"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
)

// Checker provides health check functionality
type Checker struct {
	dbPool       *pgxpool.Pool
	queueManager *queue.Manager
	startTime    time.Time
}

// NewChecker creates a new health checker
func NewChecker(dbPool *pgxpool.Pool, queueManager *queue.Manager, startTime time.Time) *Checker {
	return &Checker{
		dbPool:       dbPool,
		queueManager: queueManager,
		startTime:    startTime,
	}
}

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	Timestamp string                 `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Checks    map[string]interface{} `json:"checks"`
	Service   map[string]interface{} `json:"service"`
}

// ReadyResponse represents the readiness check response structure
type ReadyResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// Health performs a comprehensive health check
func (hc *Checker) Health(ctx context.Context) (HealthResponse, int) {
	// Check database connectivity
	dbHealthy := true
	dbStatus := "healthy"
	var dbError string

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := hc.dbPool.Ping(pingCtx); err != nil {
		dbHealthy = false
		dbStatus = "unhealthy"
		dbError = err.Error()
	}

	// Check queue manager status
	queueHealthy := hc.queueManager != nil
	queueStatus := "healthy"
	if !queueHealthy {
		queueStatus = "unhealthy"
	}

	// Overall service health
	overallHealthy := dbHealthy && queueHealthy
	overallStatus := "healthy"
	httpStatus := http.StatusOK

	if !overallHealthy {
		overallStatus = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	// Build health response
	healthResponse := HealthResponse{
		Status:    overallStatus,
		Version:   sparrow.Version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    time.Since(hc.startTime).Round(time.Second).String(),
		Checks: map[string]interface{}{
			"database": map[string]interface{}{
				"status": dbStatus,
				"type":   "postgres",
			},
			"queue": map[string]interface{}{
				"status": queueStatus,
				"type":   "river",
			},
		},
		Service: map[string]interface{}{
			"name":        "sparrow",
			"description": "Webhook delivery system",
			"endpoints": map[string]interface{}{
				"grpc":   "localhost:50051",
				"http":   "localhost:8080",
				"health": "localhost:8080/health",
				"ready":  "localhost:8080/ready",
			},
		},
	}

	// Add database error if present
	if dbError != "" {
		healthResponse.Checks["database"].(map[string]interface{})["error"] = dbError
	}

	return healthResponse, httpStatus
}

// Ready performs a simple readiness check
func (hc *Checker) Ready() ReadyResponse {
	return ReadyResponse{
		Status:    "ready",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   sparrow.Version,
	}
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

		readyResponse := hc.Ready()

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(readyResponse); err != nil {
			http.Error(w, "Failed to encode ready response", http.StatusInternalServerError)
		}
	}
}
