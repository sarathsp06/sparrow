package store

import (
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// Repository provides data access layer for webhook operations.
// It handles CRUD operations for webhooks, events, deliveries, and health tracking.
// All query/exec operations use the conn field (storage.DBTX) so that the
// repository works transparently against either a plain connection or a transaction.
//
// Use WithConn to obtain a repository clone that runs against a specific
// connection (e.g. a transaction obtained from storage.WithTransaction):
//
//	storage.WithTransaction(db, func(tx storage.DBTX) error {
//	    return repo.WithConn(tx).RegisterWebhook(ctx, ...)
//	})
//
// Each domain's narrow interface (WebhookRepository, EventRepository, ...) is
// defined alongside its implementation. Methods are distributed across
// separate files based on their primary table:
// - webhook_repository.go: webhook_registrations + rate limit state
// - event_repository.go: event_records table operations
// - event_type_repository.go: event_registrations table operations
// - delivery_repository.go: webhook_deliveries table operations
// - health_repository.go: webhook_health_* table operations
// - subscription_repository.go: event_subscriptions table operations
// - batch_repository.go: batch_jobs table operations
type Repository struct {
	db   storage.DB   // full connection — used for Beginx/Ping/Close
	conn storage.DBTX // query/exec target — either db or a transaction
}

// NewRepository creates a new Repository instance with the provided database connection.
// The storage.DB interface allows for dependency injection and easier testing with mock implementations.
func NewRepository(db storage.DB) *Repository {
	return &Repository{
		db:   db,
		conn: db, // default: queries go directly to the pool
	}
}

// WithConn returns a shallow copy of the repository that executes all
// queries against conn instead of the original database pool. This is
// the primary mechanism for enlisting a repository in an external
// transaction started via storage.WithTransaction.
func (r *Repository) WithConn(conn storage.DBTX) *Repository {
	return &Repository{
		db:   r.db,
		conn: conn,
	}
}

// RunInTransaction executes fn within a database transaction. The fn
// receives a transactional RepositoryInterface backed by the same tx.
func (r *Repository) RunInTransaction(fn func(RepositoryInterface) error) error {
	return storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		return fn(r.WithConn(tx))
	})
}
