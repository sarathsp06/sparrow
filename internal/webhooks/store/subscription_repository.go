package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// UnsubscribeFromEvent removes an event subscription
func (r *Repository) UnsubscribeFromEvent(ctx context.Context, subscriptionID uuid.UUID) error {
	query := `DELETE FROM event_subscriptions WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, subscriptionID)
	if err != nil {
		return storage.Error(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storage.Error(err)
	}

	if rowsAffected == 0 {
		return storage.Error(sql.ErrNoRows)
	}

	return nil
}
