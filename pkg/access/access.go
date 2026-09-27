// Package access issues and checks revocable access tokens and one-time
// invites for an HTTP API. It is deliberately small and knows nothing about
// the application using it.
//
// Concepts:
//
//   - Realm: an isolation boundary (a tenant, an organization). Every token
//     and invite belongs to exactly one realm.
//   - Scope: an optional, application-defined restriction. A nil scope means
//     full access within the realm; a non-nil scope is opaque to this package
//     and interpreted by the application (e.g. "only this customer's data").
//   - Root key: a static secret from configuration (an admin API key). It
//     authenticates as a full-access principal and needs no storage.
//   - Token: a stored, named, optionally expiring credential that can be
//     revoked individually. Only a SHA-256 hash of its secret is stored.
//   - Invite: a stored, single-use, expiring secret meant to travel in a link.
//     Redeeming it creates a token for the invitee; the invite is then spent.
//
// A Service ties these to a Store (memstore for tests, pgstore for Postgres).
// Package httpauth adapts it to net/http.
package access

import (
	"errors"
	"fmt"
	"time"
)

// Principal is who a request authenticated as.
type Principal struct {
	Realm string
	// Scope is nil for full access within Realm.
	Scope *string
	// Name is the token's name, or the root key's name.
	Name string
	// TokenID is empty for root keys.
	TokenID string
	// Root is true for a static root key.
	Root bool
}

// FullAccess reports whether the principal is unrestricted within its realm.
func (p Principal) FullAccess() bool { return p.Scope == nil }

// Token is a stored credential. The secret is never part of it.
type Token struct {
	ID         string
	Realm      string
	Scope      *string
	Name       string
	CreatedBy  string
	CreatedAt  time.Time
	ExpiresAt  *time.Time // nil = never expires
	RevokedAt  *time.Time
	LastUsedAt *time.Time
}

// Status is a token's or invite's state at a point in time.
type Status string

// Token and invite statuses.
const (
	StatusActive    Status = "active"
	StatusRevoked   Status = "revoked"
	StatusExpired   Status = "expired"
	StatusPending   Status = "pending"
	StatusRedeemed  Status = "redeemed"
	StatusCancelled Status = "cancelled"
)

// Status returns active, revoked, or expired as of now.
func (t Token) Status(now time.Time) Status {
	switch {
	case t.RevokedAt != nil:
		return StatusRevoked
	case t.ExpiresAt != nil && !now.Before(*t.ExpiresAt):
		return StatusExpired
	default:
		return StatusActive
	}
}

// Invite is a stored single-use invitation. The secret is never part of it.
type Invite struct {
	ID    string
	Realm string
	Scope *string
	// Name becomes the name of the token created on redemption.
	Name string
	// TokenTTL is the lifetime of the token created on redemption; nil = never expires.
	TokenTTL    *time.Duration
	CreatedBy   string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	RedeemedAt  *time.Time
	CancelledAt *time.Time
	// TokenID is the token created on redemption.
	TokenID *string
}

// Status returns pending, redeemed, cancelled, or expired as of now.
func (i Invite) Status(now time.Time) Status {
	switch {
	case i.RedeemedAt != nil:
		return StatusRedeemed
	case i.CancelledAt != nil:
		return StatusCancelled
	case !now.Before(i.ExpiresAt):
		return StatusExpired
	default:
		return StatusPending
	}
}

var (
	// ErrNotFound is returned for a token or invite that does not exist in the realm.
	ErrNotFound = errors.New("access: not found")
	// ErrInvalidInvite covers unknown, expired, cancelled, and already-used invites.
	ErrInvalidInvite = errors.New("access: invite is invalid, expired, cancelled, or already used")
	// ErrUnavailable wraps store failures during authentication, so callers can
	// answer "try again" (503) instead of "you are signed out" (401).
	ErrUnavailable = errors.New("access: credential store unavailable")
	// ErrInvalidRequest is returned for malformed create requests.
	ErrInvalidRequest = errors.New("access: invalid request")
)

// Reason says why a credential was rejected. Revealing it is safe: only the
// holder of the credential learns it.
type Reason string

// Rejection reasons.
const (
	ReasonMissing Reason = "missing"
	ReasonInvalid Reason = "invalid"
	ReasonExpired Reason = "expired"
	ReasonRevoked Reason = "revoked"
)

// AuthError is returned by Authenticate for a missing or unacceptable credential.
type AuthError struct {
	Reason Reason
}

func (e *AuthError) Error() string {
	switch e.Reason {
	case ReasonMissing:
		return "access: no credential provided"
	case ReasonExpired:
		return "access: credential expired"
	case ReasonRevoked:
		return "access: credential revoked"
	default:
		return "access: invalid credential"
	}
}

func invalidRequest(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRequest, fmt.Sprintf(format, args...))
}
