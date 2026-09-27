package access

import (
	"context"
	"time"
)

// Store persists tokens and invites. Implementations must be safe for
// concurrent use; storetest.Run checks the full contract.
//
// Lists are per realm and unfiltered: the tables are small, and filtering is
// done by Service.
type Store interface {
	// CreateToken stores t with the hash of its secret.
	CreateToken(ctx context.Context, t Token, secretHash []byte) error
	// TokenByHash returns the token whose secret hashes to secretHash, or ErrNotFound.
	TokenByHash(ctx context.Context, secretHash []byte) (Token, error)
	// ListTokens returns every token in realm, newest first.
	ListTokens(ctx context.Context, realm string) ([]Token, error)
	// RevokeToken marks a token revoked (idempotent). ErrNotFound if it is not in realm.
	RevokeToken(ctx context.Context, realm, id string, at time.Time) error
	// TouchToken records a use of the token. Best effort.
	TouchToken(ctx context.Context, id string, at time.Time) error

	// CreateInvite stores inv with the hash of its secret.
	CreateInvite(ctx context.Context, inv Invite, secretHash []byte) error
	// ListInvites returns every invite in realm, newest first.
	ListInvites(ctx context.Context, realm string) ([]Invite, error)
	// CancelInvite cancels a pending invite. ErrNotFound if it is not in realm,
	// ErrInvalidInvite if it is no longer pending.
	CancelInvite(ctx context.Context, realm, id string, at time.Time) error
	// RedeemInvite atomically checks that the invite hashing to inviteHash is
	// pending at time at, stores the token returned by mint, and marks the
	// invite redeemed by it. Exactly one concurrent caller can succeed; the
	// others get ErrInvalidInvite (also returned for unknown invites).
	RedeemInvite(ctx context.Context, inviteHash []byte, at time.Time, mint func(Invite) (Token, []byte, error)) (Invite, Token, error)
}
