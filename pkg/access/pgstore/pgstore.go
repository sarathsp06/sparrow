// Package pgstore is a PostgreSQL access.Store using database/sql. Create the
// tables with Schema (or copy it into your migrations).
package pgstore

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sarathsp06/sparrow/pkg/access"
)

// Schema creates the access_tokens and access_invites tables: SchemaParts
// joined, in order. Apply the parts as separate migrations when upgrading.
var Schema = strings.Join(SchemaParts, "\n")

// SchemaParts are the schema changes in the order they were introduced.
var SchemaParts = []string{schemaBase, schemaIdempotency}

//go:embed schema.sql
var schemaBase string

//go:embed schema_002_idempotency.sql
var schemaIdempotency string

// Store is a PostgreSQL access.Store.
type Store struct {
	db *sql.DB
}

// New returns a Store on db. The driver must accept $n placeholders.
func New(db *sql.DB) *Store { return &Store{db: db} }

const tokenCols = `id, realm, scope, name, created_by, created_at, expires_at, revoked_at, last_used_at, idempotency_key`
const inviteCols = `id, realm, scope, name, token_ttl_seconds, created_by, created_at, expires_at, redeemed_at, cancelled_at, token_id`

type scanner interface{ Scan(dest ...any) error }

// CreateToken implements access.Store.
func (s *Store) CreateToken(ctx context.Context, t access.Token, secretHash []byte) error {
	return insertToken(ctx, s.db, t, secretHash)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertToken(ctx context.Context, db execer, t access.Token, secretHash []byte) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO access_tokens (id, realm, scope, name, secret_hash, created_by, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.Realm, t.Scope, t.Name, secretHash, t.CreatedBy, t.CreatedAt, t.ExpiresAt)
	if err != nil {
		return fmt.Errorf("pgstore: create token: %w", err)
	}
	return nil
}

// CreateTokenIdempotent implements access.Store. The holder row is locked
// while it is checked; a concurrent first insert of the same key loses on the
// unique index and returns the winner's token instead.
func (s *Store) CreateTokenIdempotent(ctx context.Context, t access.Token, secretHash, sealedSecret []byte, now time.Time) (access.Token, []byte, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return access.Token{}, nil, false, fmt.Errorf("pgstore: create idempotent token: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	scope := ""
	if t.Scope != nil {
		scope = *t.Scope
	}
	holder := func(lock string) (access.Token, []byte, error) {
		var sealed []byte
		got, err := scanToken(tx.QueryRowContext(ctx,
			`SELECT `+tokenCols+`, sealed_secret FROM access_tokens
			 WHERE realm = $1 AND COALESCE(scope, '') = $2 AND idempotency_key = $3`+lock,
			t.Realm, scope, t.IdempotencyKey), &sealed)
		return got, sealed, err
	}

	got, sealed, err := holder(" FOR UPDATE")
	switch {
	case err == nil && got.Status(now) == access.StatusActive:
		return got, sealed, false, tx.Commit()
	case err == nil:
		if _, err := tx.ExecContext(ctx,
			`UPDATE access_tokens SET idempotency_key = NULL, sealed_secret = NULL WHERE id = $1`, got.ID); err != nil {
			return access.Token{}, nil, false, fmt.Errorf("pgstore: release idempotency key: %w", err)
		}
	case !errors.Is(err, sql.ErrNoRows):
		return access.Token{}, nil, false, err
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO access_tokens (id, realm, scope, name, secret_hash, created_by, created_at, expires_at, idempotency_key, sealed_secret)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (realm, COALESCE(scope, ''), idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING`,
		t.ID, t.Realm, t.Scope, t.Name, secretHash, t.CreatedBy, t.CreatedAt, t.ExpiresAt, t.IdempotencyKey, sealedSecret)
	if err != nil {
		return access.Token{}, nil, false, fmt.Errorf("pgstore: create idempotent token: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return access.Token{}, nil, false, err
	} else if n == 0 {
		// A concurrent call inserted the key first and has committed.
		got, sealed, err := holder("")
		if err != nil {
			return access.Token{}, nil, false, err
		}
		return got, sealed, false, tx.Commit()
	}
	if err := tx.Commit(); err != nil {
		return access.Token{}, nil, false, fmt.Errorf("pgstore: create idempotent token: %w", err)
	}
	return t, sealedSecret, true, nil
}

// TokenByHash implements access.Store.
func (s *Store) TokenByHash(ctx context.Context, secretHash []byte) (access.Token, error) {
	t, err := scanToken(s.db.QueryRowContext(ctx, `SELECT `+tokenCols+` FROM access_tokens WHERE secret_hash = $1`, secretHash))
	if errors.Is(err, sql.ErrNoRows) {
		return access.Token{}, access.ErrNotFound
	}
	return t, err
}

// ListTokens implements access.Store.
func (s *Store) ListTokens(ctx context.Context, realm string) ([]access.Token, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+tokenCols+` FROM access_tokens WHERE realm = $1 ORDER BY created_at DESC, id DESC`, realm)
	if err != nil {
		return nil, fmt.Errorf("pgstore: list tokens: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	var out []access.Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RevokeToken implements access.Store.
func (s *Store) RevokeToken(ctx context.Context, realm, id string, at time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE access_tokens SET revoked_at = COALESCE(revoked_at, $3), idempotency_key = NULL, sealed_secret = NULL
		 WHERE realm = $1 AND id = $2`, realm, id, at)
	if err != nil {
		return fmt.Errorf("pgstore: revoke token: %w", err)
	}
	return oneRow(res)
}

// TouchToken implements access.Store.
func (s *Store) TouchToken(ctx context.Context, id string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE access_tokens SET last_used_at = $2 WHERE id = $1`, id, at)
	return err
}

// PurgeTokens implements access.Store. access_invites.token_id is
// ON DELETE SET NULL, so redeemed invites survive their token.
func (s *Store) PurgeTokens(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM access_tokens WHERE expires_at < $1 OR revoked_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("pgstore: purge tokens: %w", err)
	}
	return res.RowsAffected()
}

// CreateInvite implements access.Store.
func (s *Store) CreateInvite(ctx context.Context, inv access.Invite, secretHash []byte) error {
	var ttl *int64
	if inv.TokenTTL != nil {
		secs := int64(*inv.TokenTTL / time.Second)
		ttl = &secs
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO access_invites (id, realm, scope, name, secret_hash, token_ttl_seconds, created_by, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		inv.ID, inv.Realm, inv.Scope, inv.Name, secretHash, ttl, inv.CreatedBy, inv.CreatedAt, inv.ExpiresAt)
	if err != nil {
		return fmt.Errorf("pgstore: create invite: %w", err)
	}
	return nil
}

// ListInvites implements access.Store.
func (s *Store) ListInvites(ctx context.Context, realm string) ([]access.Invite, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+inviteCols+` FROM access_invites WHERE realm = $1 ORDER BY created_at DESC, id DESC`, realm)
	if err != nil {
		return nil, fmt.Errorf("pgstore: list invites: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	var out []access.Invite
	for rows.Next() {
		inv, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// CancelInvite implements access.Store.
func (s *Store) CancelInvite(ctx context.Context, realm, id string, at time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE access_invites SET cancelled_at = $3
		 WHERE realm = $1 AND id = $2 AND redeemed_at IS NULL AND cancelled_at IS NULL AND expires_at > $3`,
		realm, id, at)
	if err != nil {
		return fmt.Errorf("pgstore: cancel invite: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM access_invites WHERE realm = $1 AND id = $2)`, realm, id).Scan(&exists); err != nil {
		return fmt.Errorf("pgstore: cancel invite: %w", err)
	}
	if exists {
		return access.ErrInvalidInvite
	}
	return access.ErrNotFound
}

// RedeemInvite implements access.Store. The invite row is locked for the
// duration of the transaction, so concurrent redeems serialize and only the
// first sees it pending.
func (s *Store) RedeemInvite(ctx context.Context, inviteHash []byte, at time.Time, mint func(access.Invite) (access.Token, []byte, error)) (access.Invite, access.Token, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return access.Invite{}, access.Token{}, fmt.Errorf("pgstore: redeem invite: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	inv, err := scanInvite(tx.QueryRowContext(ctx, `SELECT `+inviteCols+` FROM access_invites WHERE secret_hash = $1 FOR UPDATE`, inviteHash))
	if errors.Is(err, sql.ErrNoRows) {
		return access.Invite{}, access.Token{}, access.ErrInvalidInvite
	}
	if err != nil {
		return access.Invite{}, access.Token{}, err
	}
	if inv.Status(at) != access.StatusPending {
		return access.Invite{}, access.Token{}, access.ErrInvalidInvite
	}

	t, hash, err := mint(inv)
	if err != nil {
		return access.Invite{}, access.Token{}, err
	}
	if err := insertToken(ctx, tx, t, hash); err != nil {
		return access.Invite{}, access.Token{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE access_invites SET redeemed_at = $2, token_id = $3 WHERE id = $1`, inv.ID, at, t.ID); err != nil {
		return access.Invite{}, access.Token{}, fmt.Errorf("pgstore: redeem invite: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return access.Invite{}, access.Token{}, fmt.Errorf("pgstore: redeem invite: %w", err)
	}
	at = at.UTC()
	inv.RedeemedAt, inv.TokenID = &at, &t.ID
	return inv, t, nil
}

// scanToken scans tokenCols, then any extra columns into extra.
func scanToken(row scanner, extra ...any) (access.Token, error) {
	var t access.Token
	var scope, key sql.NullString
	var exp, rev, used sql.NullTime
	if err := row.Scan(append([]any{&t.ID, &t.Realm, &scope, &t.Name, &t.CreatedBy, &t.CreatedAt, &exp, &rev, &used, &key}, extra...)...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, err
		}
		return t, fmt.Errorf("pgstore: scan token: %w", err)
	}
	t.CreatedAt = t.CreatedAt.UTC()
	t.Scope, t.ExpiresAt, t.RevokedAt, t.LastUsedAt = str(scope), tm(exp), tm(rev), tm(used)
	t.IdempotencyKey = key.String
	return t, nil
}

func scanInvite(row scanner) (access.Invite, error) {
	var inv access.Invite
	var scope, tokenID sql.NullString
	var ttl sql.NullInt64
	var redeemed, cancelled sql.NullTime
	if err := row.Scan(&inv.ID, &inv.Realm, &scope, &inv.Name, &ttl, &inv.CreatedBy, &inv.CreatedAt, &inv.ExpiresAt, &redeemed, &cancelled, &tokenID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inv, err
		}
		return inv, fmt.Errorf("pgstore: scan invite: %w", err)
	}
	inv.CreatedAt, inv.ExpiresAt = inv.CreatedAt.UTC(), inv.ExpiresAt.UTC()
	inv.Scope, inv.TokenID, inv.RedeemedAt, inv.CancelledAt = str(scope), str(tokenID), tm(redeemed), tm(cancelled)
	if ttl.Valid {
		d := time.Duration(ttl.Int64) * time.Second
		inv.TokenTTL = &d
	}
	return inv, nil
}

func oneRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return access.ErrNotFound
	}
	return nil
}

func str(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func tm(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time.UTC()
	return &t
}
