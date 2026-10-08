package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var DefaultTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// SystemConsumer is the reserved consumer Sparrow's own sparrow.* system
// events are scoped to. Only tenant-wide credentials may act on it.
const SystemConsumer = "_sparrow"

func Bootstrap(ctx context.Context, db *sqlx.DB) error {
	var id string
	err := db.GetContext(ctx, &id, `SELECT id FROM tenants WHERE id = $1`, DefaultTenantID)
	if errors.Is(err, sql.ErrNoRows) {
		slog.WarnContext(ctx, "default tenant not found — run migrations first")
		return fmt.Errorf("bootstrap: default tenant %s not found; run database migrations", DefaultTenantID)
	}
	if err != nil {
		return fmt.Errorf("bootstrap: check default tenant: %w", err)
	}
	slog.InfoContext(ctx, "bootstrap: default tenant verified", "tenant_id", DefaultTenantID)
	return nil
}
