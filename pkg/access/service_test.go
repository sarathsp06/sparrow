package access_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/memstore"
)

var ctx = context.Background()

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newClock() *clock                   { return &clock{t: time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)} }
func ptr(s string) *string               { return &s }
func reason(err error) access.Reason {
	var ae *access.AuthError
	if errors.As(err, &ae) {
		return ae.Reason
	}
	return ""
}

func newService(t *testing.T, c *clock, store access.Store) *access.Service {
	t.Helper()
	svc, err := access.New(access.Config{
		Store:        store,
		RootKeys:     []access.RootKey{{Secret: "root-secret", Realm: "r1", Name: "master key"}},
		TokenPrefix:  "app_tk_",
		InvitePrefix: "app_inv_",
		Now:          c.now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestNewValidates(t *testing.T) {
	if _, err := access.New(access.Config{}); err == nil {
		t.Fatal("New without store succeeded")
	}
	if _, err := access.New(access.Config{Store: memstore.New(), RootKeys: []access.RootKey{{Realm: "r"}}}); err == nil {
		t.Fatal("New with empty root key succeeded")
	}
}

func TestRootKey(t *testing.T) {
	svc := newService(t, newClock(), memstore.New())
	p, err := svc.Authenticate(ctx, "root-secret")
	if err != nil || !p.Root || !p.FullAccess() || p.Realm != "r1" || p.Name != "master key" || p.TokenID != "" {
		t.Fatalf("root = %+v, %v", p, err)
	}
	if _, err := svc.Authenticate(ctx, "root-secre"); reason(err) != access.ReasonInvalid {
		t.Fatalf("near-miss root err = %v", err)
	}
	if _, err := svc.Authenticate(ctx, ""); reason(err) != access.ReasonMissing {
		t.Fatalf("empty err = %v", err)
	}
}

func TestTokenLifecycle(t *testing.T) {
	c := newClock()
	svc := newService(t, c, memstore.New())

	tok, secret, err := svc.CreateToken(ctx, access.CreateTokenRequest{Realm: "r1", Name: " alice ", CreatedBy: "master key"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(secret, "app_tk_") || len(secret) < 40 || !strings.HasPrefix(tok.ID, "tok_") {
		t.Fatalf("secret %q id %q", secret, tok.ID)
	}
	if tok.Name != "alice" || tok.ExpiresAt != nil || tok.Status(c.now()) != access.StatusActive {
		t.Fatalf("token = %+v", tok)
	}

	p, err := svc.Authenticate(ctx, secret)
	if err != nil || p.Root || !p.FullAccess() || p.Name != "alice" || p.TokenID != tok.ID || p.Realm != "r1" {
		t.Fatalf("principal = %+v, %v", p, err)
	}

	if err := svc.RevokeToken(ctx, "r1", tok.ID); err != nil {
		t.Fatal(err)
	}
	// Revocation on this instance takes effect immediately, cache or not.
	if _, err := svc.Authenticate(ctx, secret); reason(err) != access.ReasonRevoked {
		t.Fatalf("after revoke err = %v, want revoked", err)
	}
	if err := svc.RevokeToken(ctx, "r2", tok.ID); !errors.Is(err, access.ErrNotFound) {
		t.Fatalf("cross-realm revoke err = %v", err)
	}
}

func TestScopedExpiringToken(t *testing.T) {
	c := newClock()
	svc := newService(t, c, memstore.New())
	_, secret, err := svc.CreateToken(ctx, access.CreateTokenRequest{Realm: "r1", Scope: ptr("acme"), Name: "acme ci", TTL: time.Hour, CreatedBy: "x"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := svc.Authenticate(ctx, secret)
	if err != nil || p.FullAccess() || *p.Scope != "acme" {
		t.Fatalf("principal = %+v, %v", p, err)
	}
	c.advance(time.Hour)
	if _, err := svc.Authenticate(ctx, secret); reason(err) != access.ReasonExpired {
		t.Fatalf("expired err = %v", err)
	}
}

func TestCreateValidation(t *testing.T) {
	svc := newService(t, newClock(), memstore.New())
	bad := []access.CreateTokenRequest{
		{Name: "no realm"},
		{Realm: "r1", Name: "  "},
		{Realm: "r1", Name: strings.Repeat("x", 201)},
		{Realm: "r1", Name: "x", Scope: ptr("")},
		{Realm: "r1", Name: "x", TTL: -time.Second},
	}
	for _, req := range bad {
		if _, _, err := svc.CreateToken(ctx, req); !errors.Is(err, access.ErrInvalidRequest) {
			t.Errorf("CreateToken(%+v) err = %v, want ErrInvalidRequest", req, err)
		}
	}
	if _, _, err := svc.CreateInvite(ctx, access.CreateInviteRequest{Realm: "r1", Name: "x"}); !errors.Is(err, access.ErrInvalidRequest) {
		t.Errorf("invite without TTL err = %v", err)
	}
}

func TestUnknownAndForeignCredentials(t *testing.T) {
	svc := newService(t, newClock(), memstore.New())
	for _, cred := range []string{"app_tk_doesnotexist", "sk_live_other_system", "app_inv_looks_like_invite"} {
		if _, err := svc.Authenticate(ctx, cred); reason(err) != access.ReasonInvalid {
			t.Errorf("Authenticate(%q) err = %v, want invalid", cred, err)
		}
	}
}

func TestInviteFlow(t *testing.T) {
	c := newClock()
	svc := newService(t, c, memstore.New())

	inv, secret, err := svc.CreateInvite(ctx, access.CreateInviteRequest{
		Realm: "r1", Scope: ptr("acme"), Name: "bob", TTL: 15 * time.Minute, TokenTTL: 24 * time.Hour, CreatedBy: "alice",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(secret, "app_inv_") || inv.Status(c.now()) != access.StatusPending {
		t.Fatalf("invite = %+v %q", inv, secret)
	}
	// An invite is not a credential.
	if _, err := svc.Authenticate(ctx, secret); reason(err) != access.ReasonInvalid {
		t.Fatalf("invite used as credential: %v", err)
	}

	tok, tokSecret, err := svc.RedeemInvite(ctx, secret)
	if err != nil {
		t.Fatal(err)
	}
	if tok.Name != "bob" || tok.CreatedBy != "alice" || *tok.Scope != "acme" || tok.ExpiresAt == nil || !tok.ExpiresAt.Equal(c.now().Add(24*time.Hour)) {
		t.Fatalf("redeemed token = %+v", tok)
	}
	p, err := svc.Authenticate(ctx, tokSecret)
	if err != nil || p.Name != "bob" || *p.Scope != "acme" {
		t.Fatalf("principal = %+v, %v", p, err)
	}
	if _, _, err := svc.RedeemInvite(ctx, secret); !errors.Is(err, access.ErrInvalidInvite) {
		t.Fatalf("second redeem err = %v", err)
	}
	if _, _, err := svc.RedeemInvite(ctx, "app_tk_notaninvite"); !errors.Is(err, access.ErrInvalidInvite) {
		t.Fatalf("token as invite err = %v", err)
	}

	invites, _ := svc.ListInvites(ctx, "r1")
	if len(invites) != 1 || invites[0].Status(c.now()) != access.StatusRedeemed || *invites[0].TokenID != tok.ID {
		t.Fatalf("invites = %+v", invites)
	}
}

func TestInviteExpiryAndCancel(t *testing.T) {
	c := newClock()
	svc := newService(t, c, memstore.New())
	_, expiring, _ := svc.CreateInvite(ctx, access.CreateInviteRequest{Realm: "r1", Name: "a", TTL: time.Minute, CreatedBy: "x"})
	cancelled, cancelledSecret, _ := svc.CreateInvite(ctx, access.CreateInviteRequest{Realm: "r1", Name: "b", TTL: time.Hour, CreatedBy: "x"})
	if err := svc.CancelInvite(ctx, "r1", cancelled.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.RedeemInvite(ctx, cancelledSecret); !errors.Is(err, access.ErrInvalidInvite) {
		t.Fatalf("cancelled redeem err = %v", err)
	}
	c.advance(time.Minute)
	if _, _, err := svc.RedeemInvite(ctx, expiring); !errors.Is(err, access.ErrInvalidInvite) {
		t.Fatalf("expired redeem err = %v", err)
	}
}

func TestInviteTokenWithoutTTLNeverExpires(t *testing.T) {
	c := newClock()
	svc := newService(t, c, memstore.New())
	_, secret, _ := svc.CreateInvite(ctx, access.CreateInviteRequest{Realm: "r1", Name: "a", TTL: time.Minute, CreatedBy: "x"})
	tok, _, err := svc.RedeemInvite(ctx, secret)
	if err != nil || tok.ExpiresAt != nil {
		t.Fatalf("token = %+v, %v", tok, err)
	}
}

// countingStore counts lookups and can simulate an outage.
type countingStore struct {
	access.Store
	lookups, touches atomic.Int32
	down             atomic.Bool
}

func (s *countingStore) TokenByHash(ctx context.Context, h []byte) (access.Token, error) {
	s.lookups.Add(1)
	if s.down.Load() {
		return access.Token{}, errors.New("connection refused")
	}
	return s.Store.TokenByHash(ctx, h)
}

func (s *countingStore) TouchToken(ctx context.Context, id string, at time.Time) error {
	s.touches.Add(1)
	return s.Store.TouchToken(ctx, id, at)
}

func TestCacheAndTouchThrottling(t *testing.T) {
	c := newClock()
	store := &countingStore{Store: memstore.New()}
	svc := newService(t, c, store)
	_, secret, _ := svc.CreateToken(ctx, access.CreateTokenRequest{Realm: "r1", Name: "a", CreatedBy: "x"})

	for range 5 {
		if _, err := svc.Authenticate(ctx, secret); err != nil {
			t.Fatal(err)
		}
	}
	if n := store.lookups.Load(); n != 1 {
		t.Fatalf("%d lookups within cache TTL, want 1", n)
	}
	if n := store.touches.Load(); n != 1 {
		t.Fatalf("%d touches, want 1", n)
	}

	c.advance(31 * time.Second)
	_, _ = svc.Authenticate(ctx, secret)
	if n := store.lookups.Load(); n != 2 {
		t.Fatalf("%d lookups after cache TTL, want 2", n)
	}
	if n := store.touches.Load(); n != 1 {
		t.Fatalf("%d touches before touch interval, want 1", n)
	}
	c.advance(time.Minute)
	_, _ = svc.Authenticate(ctx, secret)
	if n := store.touches.Load(); n != 2 {
		t.Fatalf("%d touches after interval, want 2", n)
	}
}

func TestRevocationFromAnotherInstanceWithinCacheTTL(t *testing.T) {
	c := newClock()
	store := memstore.New()
	a, b := newService(t, c, store), newService(t, c, store)
	tok, secret, _ := a.CreateToken(ctx, access.CreateTokenRequest{Realm: "r1", Name: "a", CreatedBy: "x"})
	if _, err := b.Authenticate(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if err := a.RevokeToken(ctx, "r1", tok.ID); err != nil {
		t.Fatal(err)
	}
	// b still trusts its cache until CacheTTL passes, then sees the revocation.
	if _, err := b.Authenticate(ctx, secret); err != nil {
		t.Fatalf("within cache TTL: %v", err)
	}
	c.advance(30 * time.Second)
	if _, err := b.Authenticate(ctx, secret); reason(err) != access.ReasonRevoked {
		t.Fatalf("after cache TTL err = %v, want revoked", err)
	}
}

func TestStoreOutageIsUnavailableNotUnauthenticated(t *testing.T) {
	c := newClock()
	store := &countingStore{Store: memstore.New()}
	svc := newService(t, c, store)
	_, secret, _ := svc.CreateToken(ctx, access.CreateTokenRequest{Realm: "r1", Name: "a", CreatedBy: "x"})
	store.down.Store(true)
	_, err := svc.Authenticate(ctx, secret)
	if !errors.Is(err, access.ErrUnavailable) || reason(err) != "" {
		t.Fatalf("outage err = %v, want ErrUnavailable", err)
	}
	// Root keys need no store.
	if _, err := svc.Authenticate(ctx, "root-secret"); err != nil {
		t.Fatalf("root during outage: %v", err)
	}
}

func TestConcurrentAuthenticate(t *testing.T) {
	svc, err := access.New(access.Config{Store: memstore.New(), TouchInterval: time.Nanosecond})
	if err != nil {
		t.Fatal(err)
	}
	_, secret, _ := svc.CreateToken(ctx, access.CreateTokenRequest{Realm: "r1", Name: "a", CreatedBy: "x"})
	done := make(chan error, 16)
	for range 16 {
		go func() {
			for range 50 {
				if _, err := svc.Authenticate(ctx, secret); err != nil {
					done <- err
					return
				}
			}
			done <- nil
		}()
	}
	for range 16 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

// xorSealer is a test SecretSealer; broken makes Open fail, as after the
// sealing key is removed.
type xorSealer struct{ broken bool }

func (xorSealer) Seal(p []byte) ([]byte, error) {
	out := make([]byte, len(p))
	for i, b := range p {
		out[i] = b ^ 0x5a
	}
	return out, nil
}

func (x *xorSealer) Open(s []byte) ([]byte, error) {
	if x.broken {
		return nil, errors.New("unknown key")
	}
	return xorSealer{}.Seal(s)
}

func newSealedService(t *testing.T, c *clock, sealer access.SecretSealer) *access.Service {
	t.Helper()
	svc, err := access.New(access.Config{Store: memstore.New(), TokenPrefix: "app_tk_", Now: c.now, Sealer: sealer})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestGetOrCreateTokenReturnsTheSameSecretUntilExpiry(t *testing.T) {
	c := newClock()
	svc := newSealedService(t, c, &xorSealer{})
	req := access.CreateTokenRequest{Realm: "r1", Scope: ptr("acme"), Name: "portal", TTL: time.Hour, CreatedBy: "root"}

	first, secret, created, err := svc.GetOrCreateToken(ctx, req, "user-42")
	if err != nil || !created || !strings.HasPrefix(secret, "app_tk_") {
		t.Fatalf("first = %+v %q %v %v", first, secret, created, err)
	}
	c.advance(30 * time.Minute)
	req.TTL = 5 * time.Hour // ignored while the holder is active
	again, secret2, created, err := svc.GetOrCreateToken(ctx, req, "user-42")
	if err != nil || created || again.ID != first.ID || secret2 != secret || !again.ExpiresAt.Equal(*first.ExpiresAt) {
		t.Fatalf("again = %+v %q %v %v, want the first token and secret", again, secret2, created, err)
	}
	if p, err := svc.Authenticate(ctx, secret2); err != nil || p.TokenID != first.ID {
		t.Fatalf("returned secret does not authenticate: %+v %v", p, err)
	}

	c.advance(time.Hour) // first token expired
	fresh, secret3, created, err := svc.GetOrCreateToken(ctx, req, "user-42")
	if err != nil || !created || fresh.ID == first.ID || secret3 == secret {
		t.Fatalf("after expiry = %+v %v %v, want a new token", fresh, created, err)
	}
	if got := fresh.ExpiresAt.Sub(c.now()); got != 5*time.Hour {
		t.Fatalf("new token lifetime = %s, want the requested 5h", got)
	}
}

func TestGetOrCreateTokenReplacesAnUnreadableSecret(t *testing.T) {
	c := newClock()
	sealer := &xorSealer{}
	svc := newSealedService(t, c, sealer)
	req := access.CreateTokenRequest{Realm: "r1", Scope: ptr("acme"), Name: "portal", TTL: time.Hour, CreatedBy: "root"}
	old, oldSecret, _, err := svc.GetOrCreateToken(ctx, req, "k")
	if err != nil {
		t.Fatal(err)
	}

	sealer.broken = true
	fresh, _, created, err := svc.GetOrCreateToken(ctx, req, "k")
	if err != nil || !created || fresh.ID == old.ID {
		t.Fatalf("unreadable holder: %+v %v %v, want a replacement", fresh, created, err)
	}
	if _, err := svc.Authenticate(ctx, oldSecret); reason(err) != access.ReasonRevoked {
		t.Fatalf("unreadable holder not revoked: %v", err)
	}
}

func TestGetOrCreateTokenValidation(t *testing.T) {
	c := newClock()
	req := access.CreateTokenRequest{Realm: "r1", Scope: ptr("acme"), Name: "portal", TTL: time.Hour, CreatedBy: "root"}
	if _, _, _, err := newSealedService(t, c, nil).GetOrCreateToken(ctx, req, "k"); !errors.Is(err, access.ErrNoSealer) {
		t.Fatalf("no sealer err = %v, want ErrNoSealer", err)
	}
	svc := newSealedService(t, c, &xorSealer{})
	for _, key := range []string{"", " k", strings.Repeat("k", 201)} {
		if _, _, _, err := svc.GetOrCreateToken(ctx, req, key); !errors.Is(err, access.ErrInvalidRequest) {
			t.Errorf("key %q err = %v, want ErrInvalidRequest", key, err)
		}
	}
}
