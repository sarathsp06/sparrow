package access

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// RootKey is a static full-access secret from configuration.
type RootKey struct {
	Secret string
	Realm  string
	// Name identifies the key in CreatedBy fields and Principal.Name.
	Name string
}

// Config configures a Service.
type Config struct {
	Store    Store
	RootKeys []RootKey
	// TokenPrefix and InvitePrefix start every secret, making leaked secrets
	// easy to recognize (and to scan for). Defaults: "tk_" and "inv_".
	TokenPrefix  string
	InvitePrefix string
	// CacheTTL is how long a successful token lookup is reused. It bounds how
	// long a revoked token keeps working on other instances. Default 30s;
	// negative disables caching.
	CacheTTL time.Duration
	// TouchInterval throttles last-used updates per token. Default 1m.
	TouchInterval time.Duration
	// Now overrides the clock (tests).
	Now func() time.Time
}

// Service issues and checks tokens and invites.
type Service struct {
	store         Store
	roots         []rootKey
	tokenPrefix   string
	invitePrefix  string
	cacheTTL      time.Duration
	touchInterval time.Duration
	now           func() time.Time

	mu    sync.Mutex
	cache map[string]*cacheEntry // key: hex(secret hash)
}

// rootKey keeps only the hash of a root secret, computed once.
type rootKey struct {
	hash  [sha256.Size]byte
	realm string
	name  string
}

type cacheEntry struct {
	token     Token
	loadedAt  time.Time
	touchedAt time.Time
}

const maxCacheEntries = 10_000

// New returns a Service. It fails without a store or with an empty root key.
func New(cfg Config) (*Service, error) {
	if cfg.Store == nil {
		return nil, errors.New("access: Config.Store is required")
	}
	for _, rk := range cfg.RootKeys {
		if rk.Secret == "" || rk.Realm == "" {
			return nil, errors.New("access: root keys need a secret and a realm")
		}
	}
	s := &Service{
		store:         cfg.Store,
		tokenPrefix:   cmp.Or(cfg.TokenPrefix, "tk_"),
		invitePrefix:  cmp.Or(cfg.InvitePrefix, "inv_"),
		cacheTTL:      cfg.CacheTTL,
		touchInterval: cfg.TouchInterval,
		now:           cfg.Now,
		cache:         map[string]*cacheEntry{},
	}
	for _, rk := range cfg.RootKeys {
		s.roots = append(s.roots, rootKey{hash: sha256.Sum256([]byte(rk.Secret)), realm: rk.Realm, name: cmp.Or(rk.Name, "root key")})
	}
	if s.cacheTTL == 0 {
		s.cacheTTL = 30 * time.Second
	}
	if s.touchInterval == 0 {
		s.touchInterval = time.Minute
	}
	if s.now == nil {
		s.now = time.Now
	}
	return s, nil
}

// CreateTokenRequest describes a new token.
type CreateTokenRequest struct {
	Realm string
	Scope *string
	Name  string
	// TTL is the token's lifetime; 0 means it never expires.
	TTL       time.Duration
	CreatedBy string
}

// CreateToken stores a new token and returns it with its secret. The secret
// is shown exactly once; only its hash is kept.
func (s *Service) CreateToken(ctx context.Context, req CreateTokenRequest) (Token, string, error) {
	if err := validate(req.Realm, req.Scope, req.Name, req.TTL); err != nil {
		return Token{}, "", err
	}
	t, secret, hash, err := s.newToken(req.Realm, req.Scope, req.Name, req.CreatedBy, req.TTL)
	if err != nil {
		return Token{}, "", err
	}
	if err := s.store.CreateToken(ctx, t, hash); err != nil {
		return Token{}, "", err
	}
	return t, secret, nil
}

// ListTokens returns the realm's tokens, newest first.
func (s *Service) ListTokens(ctx context.Context, realm string) ([]Token, error) {
	return s.store.ListTokens(ctx, realm)
}

// RevokeToken revokes a token. It stops working here immediately and on
// other instances within CacheTTL.
func (s *Service) RevokeToken(ctx context.Context, realm, id string) error {
	if err := s.store.RevokeToken(ctx, realm, id, s.now()); err != nil {
		return err
	}
	s.mu.Lock()
	for k, e := range s.cache {
		if e.token.ID == id {
			delete(s.cache, k)
		}
	}
	s.mu.Unlock()
	return nil
}

// CreateInviteRequest describes a new invite.
type CreateInviteRequest struct {
	Realm string
	Scope *string
	Name  string
	// TTL is how long the invite can be redeemed; required.
	TTL time.Duration
	// TokenTTL is the lifetime of the token it creates; 0 means never expires.
	TokenTTL  time.Duration
	CreatedBy string
}

// CreateInvite stores a new invite and returns it with its secret.
func (s *Service) CreateInvite(ctx context.Context, req CreateInviteRequest) (Invite, string, error) {
	if err := validate(req.Realm, req.Scope, req.Name, req.TokenTTL); err != nil {
		return Invite{}, "", err
	}
	if req.TTL <= 0 {
		return Invite{}, "", invalidRequest("invite TTL must be positive")
	}
	secret, hash, err := newSecret(s.invitePrefix)
	if err != nil {
		return Invite{}, "", err
	}
	id, err := newID("inv_")
	if err != nil {
		return Invite{}, "", err
	}
	now := s.now().UTC().Truncate(time.Second)
	inv := Invite{
		ID: id, Realm: req.Realm, Scope: clonePtr(req.Scope), Name: strings.TrimSpace(req.Name),
		CreatedBy: req.CreatedBy, CreatedAt: now, ExpiresAt: now.Add(req.TTL),
	}
	if req.TokenTTL > 0 {
		ttl := req.TokenTTL
		inv.TokenTTL = &ttl
	}
	if err := s.store.CreateInvite(ctx, inv, hash); err != nil {
		return Invite{}, "", err
	}
	return inv, secret, nil
}

// ListInvites returns the realm's invites, newest first.
func (s *Service) ListInvites(ctx context.Context, realm string) ([]Invite, error) {
	return s.store.ListInvites(ctx, realm)
}

// CancelInvite cancels a pending invite.
func (s *Service) CancelInvite(ctx context.Context, realm, id string) error {
	return s.store.CancelInvite(ctx, realm, id, s.now())
}

// RedeemInvite spends an invite and returns the token it creates, with the
// token's secret. Unknown, expired, cancelled, and used invites all return
// ErrInvalidInvite.
func (s *Service) RedeemInvite(ctx context.Context, secret string) (Token, string, error) {
	secret = strings.TrimSpace(secret)
	if !strings.HasPrefix(secret, s.invitePrefix) {
		return Token{}, "", ErrInvalidInvite
	}
	var tokenSecret string
	_, t, err := s.store.RedeemInvite(ctx, hashSecret(secret), s.now(), func(inv Invite) (Token, []byte, error) {
		var ttl time.Duration
		if inv.TokenTTL != nil {
			ttl = *inv.TokenTTL
		}
		tok, sec, hash, err := s.newToken(inv.Realm, inv.Scope, inv.Name, inv.CreatedBy, ttl)
		tokenSecret = sec
		return tok, hash, err
	})
	if err != nil {
		return Token{}, "", err
	}
	return t, tokenSecret, nil
}

// Authenticate resolves a credential (root key or token secret) to a
// principal. It returns *AuthError for missing or unacceptable credentials
// and an error wrapping ErrUnavailable when the store cannot be reached.
func (s *Service) Authenticate(ctx context.Context, credential string) (Principal, error) {
	credential = strings.TrimSpace(credential)
	if credential == "" {
		return Principal{}, &AuthError{Reason: ReasonMissing}
	}
	if p, ok := s.matchRoot(credential); ok {
		return p, nil
	}
	if !strings.HasPrefix(credential, s.tokenPrefix) {
		return Principal{}, &AuthError{Reason: ReasonInvalid}
	}

	hash := hashSecret(credential)
	key := hex.EncodeToString(hash)
	now := s.now()

	entry := s.cached(key, now)
	if entry == nil {
		t, err := s.store.TokenByHash(ctx, hash)
		if errors.Is(err, ErrNotFound) {
			return Principal{}, &AuthError{Reason: ReasonInvalid}
		}
		if err != nil {
			return Principal{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		entry = &cacheEntry{token: t, loadedAt: now}
		if t.LastUsedAt != nil {
			entry.touchedAt = *t.LastUsedAt
		}
		s.remember(key, entry)
	}

	switch entry.token.Status(now) {
	case StatusRevoked:
		return Principal{}, &AuthError{Reason: ReasonRevoked}
	case StatusExpired:
		return Principal{}, &AuthError{Reason: ReasonExpired}
	}

	s.mu.Lock()
	touch := now.Sub(entry.touchedAt) >= s.touchInterval
	if touch {
		entry.touchedAt = now
	}
	s.mu.Unlock()
	if touch {
		// Best effort: a failed touch must not fail the request.
		_ = s.store.TouchToken(ctx, entry.token.ID, now)
	}

	t := entry.token
	return Principal{Realm: t.Realm, Scope: clonePtr(t.Scope), Name: t.Name, TokenID: t.ID}, nil
}

func (s *Service) matchRoot(credential string) (Principal, bool) {
	got := sha256.Sum256([]byte(credential))
	for _, rk := range s.roots {
		if subtle.ConstantTimeCompare(got[:], rk.hash[:]) == 1 {
			return Principal{Realm: rk.realm, Name: rk.name, Root: true}, true
		}
	}
	return Principal{}, false
}

func (s *Service) cached(key string, now time.Time) *cacheEntry {
	if s.cacheTTL < 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.cache[key]
	if !ok || now.Sub(e.loadedAt) >= s.cacheTTL {
		return nil
	}
	return e
}

func (s *Service) remember(key string, e *cacheEntry) {
	if s.cacheTTL < 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cache) >= maxCacheEntries {
		s.cache = map[string]*cacheEntry{}
	}
	s.cache[key] = e
}

func (s *Service) newToken(realm string, scope *string, name, createdBy string, ttl time.Duration) (Token, string, []byte, error) {
	secret, hash, err := newSecret(s.tokenPrefix)
	if err != nil {
		return Token{}, "", nil, err
	}
	id, err := newID("tok_")
	if err != nil {
		return Token{}, "", nil, err
	}
	now := s.now().UTC().Truncate(time.Second)
	t := Token{ID: id, Realm: realm, Scope: clonePtr(scope), Name: strings.TrimSpace(name), CreatedBy: createdBy, CreatedAt: now}
	if ttl > 0 {
		exp := now.Add(ttl)
		t.ExpiresAt = &exp
	}
	return t, secret, hash, nil
}

func validate(realm string, scope *string, name string, ttl time.Duration) error {
	switch {
	case realm == "":
		return invalidRequest("realm is required")
	case strings.TrimSpace(name) == "":
		return invalidRequest("name is required")
	case len(name) > 200:
		return invalidRequest("name is longer than 200 characters")
	case scope != nil && *scope == "":
		return invalidRequest("scope must be nil or non-empty")
	case ttl < 0:
		return invalidRequest("TTL must not be negative")
	}
	return nil
}

// newSecret returns prefix + 256 random bits (base64url) and its SHA-256 hash.
// A plain hash is enough: the secret is random, not a guessable password.
func newSecret(prefix string) (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("access: generate secret: %w", err)
	}
	secret := prefix + base64.RawURLEncoding.EncodeToString(b)
	return secret, hashSecret(secret), nil
}

func newID(prefix string) (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("access: generate id: %w", err)
	}
	return prefix + hex.EncodeToString(b), nil
}

func hashSecret(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
