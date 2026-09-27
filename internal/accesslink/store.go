package accesslink

import (
	"context"
	"time"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// PostgresClaims tracks redeemed nonces in access_link_redemptions.
type PostgresClaims struct {
	DB storage.DBTX
}

// Claim inserts the nonce; a conflict means the link was already used.
// Rows for links that have expired are dropped on the way, so the table only
// ever holds links that could still verify.
func (p PostgresClaims) Claim(ctx context.Context, nonce string, expiresAt time.Time) (bool, error) {
	if _, err := p.DB.ExecContext(ctx, `DELETE FROM access_link_redemptions WHERE expires_at < NOW()`); err != nil {
		return false, err
	}
	res, err := p.DB.ExecContext(ctx,
		`INSERT INTO access_link_redemptions (nonce, expires_at) VALUES ($1, $2) ON CONFLICT (nonce) DO NOTHING`,
		nonce, expiresAt)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}
