// Package storetest is a conformance suite for access.Store implementations.
//
//	func TestMyStore(t *testing.T) {
//		storetest.Run(t, func(t *testing.T) access.Store { return newEmptyStore(t) })
//	}
package storetest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sarathsp06/sparrow/pkg/access"
)

// Run exercises the full access.Store contract. newStore must return an empty
// store for every call.
func Run(t *testing.T, newStore func(t *testing.T) access.Store) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("token round trip", func(t *testing.T) {
		s := newStore(t)
		scope := "acme"
		exp := now.Add(time.Hour)
		want := access.Token{ID: "tok_1", Realm: "r1", Scope: &scope, Name: "alice", CreatedBy: "root", CreatedAt: now, ExpiresAt: &exp}
		must(t, s.CreateToken(ctx, want, []byte("hash-1")))

		got, err := s.TokenByHash(ctx, []byte("hash-1"))
		must(t, err)
		if got.ID != want.ID || got.Realm != "r1" || got.Name != "alice" || got.CreatedBy != "root" ||
			got.Scope == nil || *got.Scope != "acme" || got.ExpiresAt == nil || !got.ExpiresAt.Equal(exp) || !got.CreatedAt.Equal(now) {
			t.Fatalf("round trip = %+v", got)
		}
		if _, err := s.TokenByHash(ctx, []byte("nope")); !errors.Is(err, access.ErrNotFound) {
			t.Fatalf("unknown hash err = %v, want ErrNotFound", err)
		}
	})

	t.Run("full-access token has nil scope and expiry", func(t *testing.T) {
		s := newStore(t)
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_1", Realm: "r1", Name: "ci", CreatedBy: "root", CreatedAt: now}, []byte("h")))
		got, err := s.TokenByHash(ctx, []byte("h"))
		must(t, err)
		if got.Scope != nil || got.ExpiresAt != nil || got.RevokedAt != nil || got.LastUsedAt != nil {
			t.Fatalf("nullable fields = %+v", got)
		}
	})

	t.Run("list is per realm, newest first", func(t *testing.T) {
		s := newStore(t)
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_a", Realm: "r1", Name: "a", CreatedBy: "x", CreatedAt: now}, []byte("a")))
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_b", Realm: "r1", Name: "b", CreatedBy: "x", CreatedAt: now.Add(time.Second)}, []byte("b")))
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_c", Realm: "r2", Name: "c", CreatedBy: "x", CreatedAt: now}, []byte("c")))
		got, err := s.ListTokens(ctx, "r1")
		must(t, err)
		if len(got) != 2 || got[0].ID != "tok_b" || got[1].ID != "tok_a" {
			t.Fatalf("ListTokens = %v", ids(got))
		}
	})

	t.Run("revoke is idempotent and realm scoped", func(t *testing.T) {
		s := newStore(t)
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_1", Realm: "r1", Name: "a", CreatedBy: "x", CreatedAt: now}, []byte("h")))
		if err := s.RevokeToken(ctx, "r2", "tok_1", now); !errors.Is(err, access.ErrNotFound) {
			t.Fatalf("cross-realm revoke err = %v, want ErrNotFound", err)
		}
		must(t, s.RevokeToken(ctx, "r1", "tok_1", now))
		must(t, s.RevokeToken(ctx, "r1", "tok_1", now.Add(time.Hour)))
		got, _ := s.TokenByHash(ctx, []byte("h"))
		if got.RevokedAt == nil || !got.RevokedAt.Equal(now) {
			t.Fatalf("RevokedAt = %v, want first revocation %v", got.RevokedAt, now)
		}
		if err := s.RevokeToken(ctx, "r1", "tok_missing", now); !errors.Is(err, access.ErrNotFound) {
			t.Fatalf("missing revoke err = %v", err)
		}
	})

	t.Run("touch records last use", func(t *testing.T) {
		s := newStore(t)
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_1", Realm: "r1", Name: "a", CreatedBy: "x", CreatedAt: now}, []byte("h")))
		must(t, s.TouchToken(ctx, "tok_1", now.Add(time.Minute)))
		got, _ := s.TokenByHash(ctx, []byte("h"))
		if got.LastUsedAt == nil || !got.LastUsedAt.Equal(now.Add(time.Minute)) {
			t.Fatalf("LastUsedAt = %v", got.LastUsedAt)
		}
	})

	t.Run("purge deletes only tokens expired or revoked before the cutoff", func(t *testing.T) {
		s := newStore(t)
		old, recent := now.Add(-48*time.Hour), now.Add(-time.Hour)
		future := now.Add(time.Hour)
		cutoff := now.Add(-24 * time.Hour)
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_old_exp", Realm: "r1", Name: "a", CreatedBy: "x", CreatedAt: old.Add(-time.Hour), ExpiresAt: &old}, []byte("h1")))
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_recent_exp", Realm: "r1", Name: "b", CreatedBy: "x", CreatedAt: old, ExpiresAt: &recent}, []byte("h2")))
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_old_rev", Realm: "r2", Name: "c", CreatedBy: "x", CreatedAt: old.Add(-time.Hour), ExpiresAt: &future}, []byte("h3")))
		must(t, s.RevokeToken(ctx, "r2", "tok_old_rev", old))
		must(t, s.CreateToken(ctx, access.Token{ID: "tok_live", Realm: "r1", Name: "d", CreatedBy: "x", CreatedAt: old}, []byte("h4")))

		// A redeemed invite whose token is purged survives, without its token.
		must(t, s.CreateInvite(ctx, access.Invite{ID: "inv_1", Realm: "r1", Name: "e", CreatedBy: "x", CreatedAt: old, ExpiresAt: now}, []byte("ih")))
		_, _, err := s.RedeemInvite(ctx, []byte("ih"), old, func(inv access.Invite) (access.Token, []byte, error) {
			exp := old.Add(time.Minute)
			return access.Token{ID: "tok_inv", Realm: inv.Realm, Name: inv.Name, CreatedBy: inv.CreatedBy, CreatedAt: old, ExpiresAt: &exp}, []byte("h5"), nil
		})
		must(t, err)

		n, err := s.PurgeTokens(ctx, cutoff)
		must(t, err)
		if n != 3 {
			t.Fatalf("purged %d tokens, want 3 (old expired, old revoked, invite token)", n)
		}
		for _, h := range []string{"h1", "h3", "h5"} {
			if _, err := s.TokenByHash(ctx, []byte(h)); !errors.Is(err, access.ErrNotFound) {
				t.Fatalf("token %s still present: err = %v", h, err)
			}
		}
		for _, h := range []string{"h2", "h4"} {
			if _, err := s.TokenByHash(ctx, []byte(h)); err != nil {
				t.Fatalf("token %s purged too early: %v", h, err)
			}
		}
		invites, err := s.ListInvites(ctx, "r1")
		must(t, err)
		if len(invites) != 1 || invites[0].TokenID != nil || invites[0].RedeemedAt == nil {
			t.Fatalf("invite after purge = %+v", invites)
		}
	})

	idem := func(id, realm, scope, key string, exp time.Time) access.Token {
		tok := access.Token{ID: id, Realm: realm, Name: "portal", CreatedBy: "x", CreatedAt: now, ExpiresAt: &exp, ExternalID: key}
		if scope != "" {
			tok.Scope = &scope
		}
		return tok
	}

	t.Run("get-or-create returns the active token for an external id", func(t *testing.T) {
		s := newStore(t)
		exp := now.Add(time.Hour)
		got, sealed, created, err := s.GetOrCreateToken(ctx, idem("tok_1", "r1", "acme", "user-1", exp), []byte("h1"), []byte("sealed-1"), now)
		must(t, err)
		if !created || got.ID != "tok_1" || string(sealed) != "sealed-1" || got.ExternalID != "user-1" {
			t.Fatalf("first create = %+v %q %v", got, sealed, created)
		}
		got, sealed, created, err = s.GetOrCreateToken(ctx, idem("tok_2", "r1", "acme", "user-1", exp), []byte("h2"), []byte("sealed-2"), now)
		must(t, err)
		if created || got.ID != "tok_1" || string(sealed) != "sealed-1" {
			t.Fatalf("second create = %+v %q %v, want tok_1 reused", got, sealed, created)
		}
		if _, err := s.TokenByHash(ctx, []byte("h2")); !errors.Is(err, access.ErrNotFound) {
			t.Fatalf("reused call stored its token: err = %v", err)
		}

		// The key is per realm and scope, and a nil scope is its own scope.
		for i, c := range []struct{ realm, scope string }{{"r1", "globex"}, {"r2", "acme"}, {"r1", ""}} {
			id := fmt.Sprintf("tok_other_%d", i)
			_, _, created, err := s.GetOrCreateToken(ctx, idem(id, c.realm, c.scope, "user-1", exp), []byte(id), []byte("s"), now)
			must(t, err)
			if !created {
				t.Fatalf("realm %q scope %q reused another scope's token", c.realm, c.scope)
			}
		}
	})

	t.Run("get-or-create replaces an expired or revoked token", func(t *testing.T) {
		s := newStore(t)
		soon, later := now.Add(time.Minute), now.Add(time.Hour)
		_, _, _, err := s.GetOrCreateToken(ctx, idem("tok_old", "r1", "acme", "k", soon), []byte("h1"), []byte("s1"), now)
		must(t, err)

		got, sealed, created, err := s.GetOrCreateToken(ctx, idem("tok_new", "r1", "acme", "k", later), []byte("h2"), []byte("s2"), soon)
		must(t, err)
		if !created || got.ID != "tok_new" || string(sealed) != "s2" {
			t.Fatalf("after expiry = %+v %q %v, want tok_new", got, sealed, created)
		}
		old, err := s.TokenByHash(ctx, []byte("h1"))
		must(t, err)
		if old.ExternalID != "" {
			t.Fatalf("expired holder kept its key: %+v", old)
		}

		must(t, s.RevokeToken(ctx, "r1", "tok_new", now))
		revoked, _ := s.TokenByHash(ctx, []byte("h2"))
		if revoked.ExternalID != "" {
			t.Fatalf("revoked token kept its key: %+v", revoked)
		}
		got, _, created, err = s.GetOrCreateToken(ctx, idem("tok_3", "r1", "acme", "k", later), []byte("h3"), []byte("s3"), now)
		must(t, err)
		if !created || got.ID != "tok_3" {
			t.Fatalf("after revoke = %+v %v, want tok_3", got, created)
		}
	})

	t.Run("concurrent get-or-creates: exactly one wins", func(t *testing.T) {
		s := newStore(t)
		exp := now.Add(time.Hour)
		const n = 8
		var wg sync.WaitGroup
		ids := make([]string, n)
		createdCount := make([]bool, n)
		errs := make([]error, n)
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				id := fmt.Sprintf("tok_%d", i)
				got, _, created, err := s.GetOrCreateToken(ctx, idem(id, "r1", "acme", "race", exp), []byte(id), []byte("s"), now)
				ids[i], createdCount[i], errs[i] = got.ID, created, err
			}()
		}
		wg.Wait()
		wins := 0
		for i := range n {
			must(t, errs[i])
			if createdCount[i] {
				wins++
			}
			if ids[i] != ids[0] {
				t.Fatalf("callers got different tokens: %v", ids)
			}
		}
		if wins != 1 {
			t.Fatalf("%d callers created a token, want 1", wins)
		}

		// Same race once the holder has expired: exactly one replaces it.
		expired := ids[0]
		wins = 0
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				id := fmt.Sprintf("tok_late_%d", i)
				got, _, created, err := s.GetOrCreateToken(ctx, idem(id, "r1", "acme", "race", exp.Add(time.Hour)), []byte(id), []byte("s"), exp)
				ids[i], createdCount[i], errs[i] = got.ID, created, err
			}()
		}
		wg.Wait()
		for i := range n {
			must(t, errs[i])
			if createdCount[i] {
				wins++
			}
			if ids[i] != ids[0] || ids[i] == expired {
				t.Fatalf("callers after expiry got %v, want one shared replacement", ids)
			}
		}
		if wins != 1 {
			t.Fatalf("%d callers replaced the expired token, want 1", wins)
		}
	})

	t.Run("invite round trip and cancel", func(t *testing.T) {
		s := newStore(t)
		scope, ttl := "acme", 48*time.Hour
		inv := access.Invite{ID: "inv_1", Realm: "r1", Scope: &scope, Name: "bob", TokenTTL: &ttl, CreatedBy: "root", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
		must(t, s.CreateInvite(ctx, inv, []byte("ih")))
		got, err := s.ListInvites(ctx, "r1")
		must(t, err)
		if len(got) != 1 || got[0].TokenTTL == nil || *got[0].TokenTTL != ttl || *got[0].Scope != "acme" || got[0].Status(now) != access.StatusPending {
			t.Fatalf("ListInvites = %+v", got)
		}
		if others, _ := s.ListInvites(ctx, "r2"); len(others) != 0 {
			t.Fatalf("other realm sees invites: %+v", others)
		}

		if err := s.CancelInvite(ctx, "r2", "inv_1", now); !errors.Is(err, access.ErrNotFound) {
			t.Fatalf("cross-realm cancel err = %v", err)
		}
		must(t, s.CancelInvite(ctx, "r1", "inv_1", now))
		if err := s.CancelInvite(ctx, "r1", "inv_1", now); !errors.Is(err, access.ErrInvalidInvite) {
			t.Fatalf("second cancel err = %v, want ErrInvalidInvite", err)
		}
		if _, _, err := s.RedeemInvite(ctx, []byte("ih"), now, mintFn("tok_x", "tx")); !errors.Is(err, access.ErrInvalidInvite) {
			t.Fatalf("redeem cancelled err = %v", err)
		}
	})

	t.Run("redeem creates the token once", func(t *testing.T) {
		s := newStore(t)
		must(t, s.CreateInvite(ctx, access.Invite{ID: "inv_1", Realm: "r1", Name: "bob", CreatedBy: "alice", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}, []byte("ih")))

		inv, tok, err := s.RedeemInvite(ctx, []byte("ih"), now, mintFn("tok_bob", "th"))
		must(t, err)
		if inv.RedeemedAt == nil || inv.TokenID == nil || *inv.TokenID != "tok_bob" || tok.ID != "tok_bob" {
			t.Fatalf("redeem = %+v / %+v", inv, tok)
		}
		stored, err := s.TokenByHash(ctx, []byte("th"))
		must(t, err)
		if stored.Name != "bob" || stored.CreatedBy != "alice" {
			t.Fatalf("stored token = %+v", stored)
		}
		if _, _, err := s.RedeemInvite(ctx, []byte("ih"), now, mintFn("tok_2", "th2")); !errors.Is(err, access.ErrInvalidInvite) {
			t.Fatalf("second redeem err = %v", err)
		}
		if err := s.CancelInvite(ctx, "r1", "inv_1", now); !errors.Is(err, access.ErrInvalidInvite) {
			t.Fatalf("cancel redeemed err = %v", err)
		}
		list, _ := s.ListInvites(ctx, "r1")
		if len(list) != 1 || list[0].Status(now) != access.StatusRedeemed {
			t.Fatalf("status after redeem = %+v", list)
		}
	})

	t.Run("redeem rejects expired and unknown invites", func(t *testing.T) {
		s := newStore(t)
		must(t, s.CreateInvite(ctx, access.Invite{ID: "inv_1", Realm: "r1", Name: "bob", CreatedBy: "x", CreatedAt: now, ExpiresAt: now.Add(time.Minute)}, []byte("ih")))
		if _, _, err := s.RedeemInvite(ctx, []byte("ih"), now.Add(time.Minute), mintFn("tok_1", "t1")); !errors.Is(err, access.ErrInvalidInvite) {
			t.Fatalf("expired err = %v", err)
		}
		if _, _, err := s.RedeemInvite(ctx, []byte("unknown"), now, mintFn("tok_2", "t2")); !errors.Is(err, access.ErrInvalidInvite) {
			t.Fatalf("unknown err = %v", err)
		}
		if _, err := s.TokenByHash(ctx, []byte("t1")); !errors.Is(err, access.ErrNotFound) {
			t.Fatal("a token was created for an expired invite")
		}
	})

	t.Run("concurrent redeems: exactly one wins", func(t *testing.T) {
		s := newStore(t)
		must(t, s.CreateInvite(ctx, access.Invite{ID: "inv_1", Realm: "r1", Name: "bob", CreatedBy: "x", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}, []byte("ih")))
		const n = 8
		var wg sync.WaitGroup
		var mu sync.Mutex
		wins := 0
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _, err := s.RedeemInvite(ctx, []byte("ih"), now, mintFn(fmt.Sprintf("tok_%d", i), fmt.Sprintf("th%d", i)))
				if err == nil {
					mu.Lock()
					wins++
					mu.Unlock()
				} else if !errors.Is(err, access.ErrInvalidInvite) {
					t.Errorf("redeem %d: %v", i, err)
				}
			}()
		}
		wg.Wait()
		if wins != 1 {
			t.Fatalf("%d redeems succeeded, want 1", wins)
		}
		tokens, _ := s.ListTokens(ctx, "r1")
		if len(tokens) != 1 {
			t.Fatalf("%d tokens created, want 1", len(tokens))
		}
	})

	t.Run("failed mint leaves the invite pending", func(t *testing.T) {
		s := newStore(t)
		must(t, s.CreateInvite(ctx, access.Invite{ID: "inv_1", Realm: "r1", Name: "bob", CreatedBy: "x", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}, []byte("ih")))
		boom := errors.New("boom")
		if _, _, err := s.RedeemInvite(ctx, []byte("ih"), now, func(access.Invite) (access.Token, []byte, error) {
			return access.Token{}, nil, boom
		}); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want mint error", err)
		}
		if _, _, err := s.RedeemInvite(ctx, []byte("ih"), now, mintFn("tok_1", "t1")); err != nil {
			t.Fatalf("redeem after failed mint: %v", err)
		}
	})
}

func mintFn(id, hash string) func(access.Invite) (access.Token, []byte, error) {
	return func(inv access.Invite) (access.Token, []byte, error) {
		return access.Token{ID: id, Realm: inv.Realm, Scope: inv.Scope, Name: inv.Name, CreatedBy: inv.CreatedBy, CreatedAt: inv.CreatedAt}, []byte(hash), nil
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func ids(ts []access.Token) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.ID
	}
	return out
}
