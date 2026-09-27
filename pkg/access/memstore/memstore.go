// Package memstore is an in-memory access.Store for tests and single-process
// quick starts. Data is lost on restart.
package memstore

import (
	"context"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/sarathsp06/sparrow/pkg/access"
)

// Store is an in-memory access.Store. The zero value is ready to use.
type Store struct {
	mu      sync.Mutex
	tokens  map[string]access.Token  // by id
	tokHash map[string]string        // hex(hash) -> id
	invites map[string]access.Invite // by id
	invHash map[string]string        // hex(hash) -> id
}

// New returns an empty Store.
func New() *Store { return &Store{} }

func (s *Store) init() {
	if s.tokens == nil {
		s.tokens, s.tokHash = map[string]access.Token{}, map[string]string{}
		s.invites, s.invHash = map[string]access.Invite{}, map[string]string{}
	}
}

// CreateToken implements access.Store.
func (s *Store) CreateToken(_ context.Context, t access.Token, secretHash []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	s.putToken(t, secretHash)
	return nil
}

func (s *Store) putToken(t access.Token, secretHash []byte) {
	s.tokens[t.ID] = t
	s.tokHash[hex.EncodeToString(secretHash)] = t.ID
}

// TokenByHash implements access.Store.
func (s *Store) TokenByHash(_ context.Context, secretHash []byte) (access.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	id, ok := s.tokHash[hex.EncodeToString(secretHash)]
	if !ok {
		return access.Token{}, access.ErrNotFound
	}
	return s.tokens[id], nil
}

// ListTokens implements access.Store.
func (s *Store) ListTokens(_ context.Context, realm string) ([]access.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	var out []access.Token
	for _, t := range s.tokens {
		if t.Realm == realm {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return newer(out[i].CreatedAt, out[j].CreatedAt, out[i].ID, out[j].ID) })
	return out, nil
}

// RevokeToken implements access.Store.
func (s *Store) RevokeToken(_ context.Context, realm, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	t, ok := s.tokens[id]
	if !ok || t.Realm != realm {
		return access.ErrNotFound
	}
	if t.RevokedAt == nil {
		at := at.UTC()
		t.RevokedAt = &at
		s.tokens[id] = t
	}
	return nil
}

// TouchToken implements access.Store.
func (s *Store) TouchToken(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	if t, ok := s.tokens[id]; ok {
		at := at.UTC()
		t.LastUsedAt = &at
		s.tokens[id] = t
	}
	return nil
}

// CreateInvite implements access.Store.
func (s *Store) CreateInvite(_ context.Context, inv access.Invite, secretHash []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	s.invites[inv.ID] = inv
	s.invHash[hex.EncodeToString(secretHash)] = inv.ID
	return nil
}

// ListInvites implements access.Store.
func (s *Store) ListInvites(_ context.Context, realm string) ([]access.Invite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	var out []access.Invite
	for _, inv := range s.invites {
		if inv.Realm == realm {
			out = append(out, inv)
		}
	}
	sort.Slice(out, func(i, j int) bool { return newer(out[i].CreatedAt, out[j].CreatedAt, out[i].ID, out[j].ID) })
	return out, nil
}

// CancelInvite implements access.Store.
func (s *Store) CancelInvite(_ context.Context, realm, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	inv, ok := s.invites[id]
	if !ok || inv.Realm != realm {
		return access.ErrNotFound
	}
	if inv.Status(at) != access.StatusPending {
		return access.ErrInvalidInvite
	}
	at = at.UTC()
	inv.CancelledAt = &at
	s.invites[id] = inv
	return nil
}

// RedeemInvite implements access.Store.
func (s *Store) RedeemInvite(_ context.Context, inviteHash []byte, at time.Time, mint func(access.Invite) (access.Token, []byte, error)) (access.Invite, access.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	id, ok := s.invHash[hex.EncodeToString(inviteHash)]
	if !ok {
		return access.Invite{}, access.Token{}, access.ErrInvalidInvite
	}
	inv := s.invites[id]
	if inv.Status(at) != access.StatusPending {
		return access.Invite{}, access.Token{}, access.ErrInvalidInvite
	}
	t, hash, err := mint(inv)
	if err != nil {
		return access.Invite{}, access.Token{}, err
	}
	s.putToken(t, hash)
	at = at.UTC()
	inv.RedeemedAt, inv.TokenID = &at, &t.ID
	s.invites[id] = inv
	return inv, t, nil
}

func newer(a, b time.Time, idA, idB string) bool {
	if !a.Equal(b) {
		return a.After(b)
	}
	return idA > idB
}
