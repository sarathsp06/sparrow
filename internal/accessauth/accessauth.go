// Package accessauth is Sparrow's adapter for pkg/access: it fixes the
// secret prefixes, maps the realm to the default tenant and the scope to a
// consumer, and holds Sparrow's lifetime rules for tokens and invites.
//
// Everything generic (issuing, hashing, redeeming, caching) lives in
// pkg/access; nothing here should be needed by another application.
package accessauth

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/pgstore"
	"github.com/sarathsp06/sparrow/pkg/crypto"
)

// Secret prefixes. Distinctive so leaked secrets are easy to spot and scan for.
const (
	TokenPrefix  = "sparrow_tk_"
	InvitePrefix = "sparrow_inv_"
)

// MasterKeyName is how SPARROW_API_KEY appears as a principal and in created_by.
const MasterKeyName = "master key"

// Realm is the tenant every credential belongs to (only the default tenant today).
func Realm() string { return tenant.DefaultTenantID.String() }

// Lifetime rules.
//   - Tenant-wide tokens act exactly like SPARROW_API_KEY. They expire after
//     the operator's default (SPARROW_TOKEN_DEFAULT_TTL, 90 days unless
//     changed) unless a TTL is given, or never when explicitly asked for.
//   - Consumer tokens reach outside users, so they always expire.
//   - Invites are links in chat: short-lived by default.
const (
	TenantTokenDefaultTTL   = 90 * 24 * time.Hour
	ConsumerTokenDefaultTTL = 7 * 24 * time.Hour
	ConsumerTokenMaxTTL     = 30 * 24 * time.Hour
	InviteDefaultTTL        = 24 * time.Hour
	InviteMaxTTL            = 7 * 24 * time.Hour
	// TokenRetention is how long expired and revoked tokens stay listed
	// before the daily purge deletes them.
	TokenRetention = 7 * 24 * time.Hour
)

// TokenTTL applies the lifetime rules to a requested TTL (0 = default).
// neverExpires asks for a tenant-wide token without expiry; tenantDefault is
// the lifetime of a tenant-wide token when neither is given (0 = never).
// A returned 0 means the token never expires.
func TokenTTL(consumer *string, requested time.Duration, neverExpires bool, tenantDefault time.Duration) (time.Duration, error) {
	if neverExpires && requested > 0 {
		return 0, fmt.Errorf("set either a TTL or never_expires, not both")
	}
	if consumer == nil {
		switch {
		case neverExpires:
			return 0, nil
		case requested > 0:
			return requested, nil
		default:
			return tenantDefault, nil
		}
	}
	if neverExpires {
		return 0, fmt.Errorf("consumer tokens always expire (at most %d days)", ConsumerTokenMaxTTL/(24*time.Hour))
	}
	if requested == 0 {
		return ConsumerTokenDefaultTTL, nil
	}
	if requested > ConsumerTokenMaxTTL {
		return 0, fmt.Errorf("consumer tokens can live at most %d days", ConsumerTokenMaxTTL/(24*time.Hour))
	}
	return requested, nil
}

// InviteTTL applies the lifetime rules to a requested invite TTL (0 = default).
func InviteTTL(requested time.Duration) (time.Duration, error) {
	if requested == 0 {
		return InviteDefaultTTL, nil
	}
	if requested > InviteMaxTTL {
		return 0, fmt.Errorf("invites can live at most %d days", InviteMaxTTL/(24*time.Hour))
	}
	return requested, nil
}

// ValidateConsumer checks a consumer name for a consumer-scoped token or
// invite. Consumers are addressed as one URL path segment
// (/v1/consumers/{consumer}/...), which the portal gateway builds from the
// token's scope, so a scope must be a single, literal segment. The reserved
// system consumer is refused: only tenant-wide credentials may reach it.
func ValidateConsumer(consumer string) error {
	switch {
	case consumer == tenant.SystemConsumer:
		return fmt.Errorf("consumer %q is reserved for Sparrow's system events and needs a tenant-wide credential", consumer)
	case strings.TrimSpace(consumer) != consumer || consumer == "":
		return errors.New("consumer must be non-empty with no leading or trailing spaces")
	case len(consumer) > 255:
		return errors.New("consumer is longer than 255 characters")
	case consumer == "." || consumer == ".." || strings.ContainsAny(consumer, "/\\?#%"):
		return fmt.Errorf("consumer %q is not a valid name (no /, \\, ?, #, %%, or dot segments)", consumer)
	}
	for _, r := range consumer {
		if unicode.IsControl(r) {
			return errors.New("consumer must not contain control characters")
		}
	}
	return nil
}

// New builds the access service over Postgres. apiKey (SPARROW_API_KEY) is
// registered as the root key when set; sealer (see Sealer) stores the secrets
// of tokens with an external id and may be nil to disable them.
func New(db *sql.DB, apiKey string, sealer access.SecretSealer) (*access.Service, error) {
	return NewWithStore(pgstore.New(db), apiKey, sealer)
}

// NewWithStore is New with any store (tests use memstore).
func NewWithStore(store access.Store, apiKey string, sealer access.SecretSealer) (*access.Service, error) {
	var roots []access.RootKey
	if apiKey != "" {
		roots = []access.RootKey{{Secret: apiKey, Realm: Realm(), Name: MasterKeyName}}
	}
	return access.New(access.Config{Store: store, RootKeys: roots, Sealer: sealer, TokenPrefix: TokenPrefix, InvitePrefix: InvitePrefix})
}

// Sealer seals the stored secrets of tokens with an external id with Sparrow's
// envelope encryption (the SPARROW_ENCRYPTION_KEYS keyring), like every other
// secret at rest.
func Sealer(svc *crypto.Service) access.SecretSealer { return cryptoSealer{svc} }

type cryptoSealer struct{ svc *crypto.Service }

func (c cryptoSealer) Seal(p []byte) ([]byte, error) { return c.svc.EnvelopeEncrypt(p) }
func (c cryptoSealer) Open(s []byte) ([]byte, error) { return c.svc.EnvelopeDecrypt(s) }
