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
	// RevokeToken marks a token revoked (idempotent) and drops its external
	// id and sealed secret. ErrNotFound if it is not in realm.
	RevokeToken(ctx context.Context, realm, id string, at time.Time) error
	// TouchToken records a use of the token. Best effort.
	TouchToken(ctx context.Context, id string, at time.Time) error
	// GetOrCreateToken stores t (whose ExternalID is set) with the hash and
	// the sealed form of its secret, unless a token in t.Realm and t.Scope
	// with the same external id is active at now: then it stores nothing and
	// returns that token and its sealed secret with created=false. A token
	// with that id that is no longer active gives it up first. Concurrent
	// calls with one external id must leave exactly one active token for it.
	GetOrCreateToken(ctx context.Context, t Token, secretHash, sealedSecret []byte, now time.Time) (got Token, gotSealed []byte, created bool, err error)
	// PurgeTokens deletes every token that expired or was revoked before
	// cutoff, in all realms, and returns how many it deleted. Invites that
	// created a purged token keep their other fields; their TokenID becomes nil.
	PurgeTokens(ctx context.Context, cutoff time.Time) (int64, error)

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
