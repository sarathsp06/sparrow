package accesslink

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sarathsp06/sparrow/internal/middleware"
	"github.com/sarathsp06/sparrow/pkg/crypto"
)

// memClaims is an in-memory Claimer.
type memClaims struct {
	mu   sync.Mutex
	seen map[string]bool
	err  error
}

func (m *memClaims) Claim(_ context.Context, nonce string, _ time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return false, m.err
	}
	if m.seen == nil {
		m.seen = map[string]bool{}
	}
	if m.seen[nonce] {
		return false, nil
	}
	m.seen[nonce] = true
	return true, nil
}

func keyring(t *testing.T, primary string, ids ...string) *crypto.Keyring {
	t.Helper()
	var keys []crypto.Key
	for i, id := range ids {
		keys = append(keys, crypto.Key{ID: id, Material: []byte(strings.Repeat(string(rune('a'+i)), 32))})
	}
	kr, err := crypto.NewKeyring(keys, primary)
	if err != nil {
		t.Fatal(err)
	}
	return kr
}

func newLinks(t *testing.T, apiKey string) *Links {
	t.Helper()
	return New(keyring(t, "main", "main"), apiKey, &memClaims{})
}

func TestMintRedeemSingleUse(t *testing.T) {
	l := newLinks(t, "secret")
	token, exp, err := l.Mint(0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, "sal_v1.main.") {
		t.Fatalf("token = %q", token)
	}
	if d := time.Until(exp); d < DefaultTTL-2*time.Second || d > DefaultTTL {
		t.Fatalf("default expiry in %v, want ~%v", d, DefaultTTL)
	}

	key, err := l.Redeem(context.Background(), token)
	if err != nil || key != "secret" {
		t.Fatalf("first redeem = %q, %v", key, err)
	}
	if _, err := l.Redeem(context.Background(), token); !errors.Is(err, ErrInvalid) {
		t.Fatalf("second redeem err = %v, want ErrInvalid", err)
	}
}

func TestMintClampsTTL(t *testing.T) {
	l := newLinks(t, "secret")
	_, exp, err := l.Mint(30 * 24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if time.Until(exp) > MaxTTL {
		t.Fatalf("expiry %v exceeds MaxTTL", time.Until(exp))
	}
}

func TestMintWithoutAPIKey(t *testing.T) {
	if _, _, err := newLinks(t, "").Mint(0); !errors.Is(err, ErrNoAPIKey) {
		t.Fatalf("err = %v, want ErrNoAPIKey", err)
	}
	var nilLinks *Links
	if _, _, err := nilLinks.Mint(0); err == nil {
		t.Fatal("nil Links minted a token")
	}
	if _, err := nilLinks.Redeem(context.Background(), "x"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil Links redeem err = %v", err)
	}
}

func TestRedeemRejectsExpired(t *testing.T) {
	l := newLinks(t, "secret")
	token, _, _ := l.Mint(time.Minute)
	l.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if _, err := l.Redeem(context.Background(), token); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestRedeemRejectsTampering(t *testing.T) {
	l := newLinks(t, "secret")
	token, _, _ := l.Mint(0)
	parts := strings.Split(token, ".")

	later := strings.Join([]string{parts[0], parts[1], "9999999999", parts[3], parts[4]}, ".")
	otherNonce := strings.Join([]string{parts[0], parts[1], parts[2], "AAAAAAAAAAAAAAAAAAAAAA", parts[4]}, ".")
	for name, bad := range map[string]string{
		"extended expiry": later,
		"swapped nonce":   otherNonce,
		"unknown key id":  strings.Replace(token, ".main.", ".other.", 1),
		"wrong prefix":    strings.Replace(token, "sal_v1", "sal_v2", 1),
		"truncated":       strings.Join(parts[:4], "."),
		"garbage":         "not-a-token",
	} {
		if _, err := l.Redeem(context.Background(), bad); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
}

func TestRotatingAPIKeyInvalidatesLinks(t *testing.T) {
	kr := keyring(t, "main", "main")
	token, _, _ := New(kr, "old-key", &memClaims{}).Mint(0)
	if _, err := New(kr, "new-key", &memClaims{}).Redeem(context.Background(), token); !errors.Is(err, ErrInvalid) {
		t.Fatalf("link survived API key rotation: err = %v", err)
	}
}

func TestLinksSurviveEncryptionKeyRotation(t *testing.T) {
	// Minted under primary "old"; after "new" becomes primary, "old" is still
	// in the ring, so outstanding links keep working.
	token, _, _ := New(keyring(t, "old", "old"), "secret", &memClaims{}).Mint(0)
	rotated := New(keyring(t, "new", "old", "new"), "secret", &memClaims{})
	if key, err := rotated.Redeem(context.Background(), token); err != nil || key != "secret" {
		t.Fatalf("redeem after keyring rotation = %q, %v", key, err)
	}
}

func TestPortalTokensAreNotAccessLinks(t *testing.T) {
	kr := keyring(t, "main", "main")
	portalToken, _, err := middleware.NewPortalTokensFromKeyring(kr).Mint("acme", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(kr, "secret", &memClaims{}).Redeem(context.Background(), portalToken); !errors.Is(err, ErrInvalid) {
		t.Fatalf("portal token redeemed as access link: err = %v", err)
	}
}

func TestRedeemSurfacesStoreErrors(t *testing.T) {
	l := New(keyring(t, "main", "main"), "secret", &memClaims{err: errors.New("db down")})
	token, _, _ := l.Mint(0)
	if _, err := l.Redeem(context.Background(), token); err == nil || errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want a store error", err)
	}
}

func TestRedeemHandler(t *testing.T) {
	l := newLinks(t, "secret")
	token, _, _ := l.Mint(0)
	h := l.RedeemHandler()

	call := func(method, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, RedeemPath, strings.NewReader(body)))
		return rec
	}

	rec := call(http.MethodPost, `{"token":"`+token+`"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"api_key":"secret"`) {
		t.Fatalf("redeem: %d %s", rec.Code, rec.Body)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}

	rec = call(http.MethodPost, `{"token":"`+token+`"}`)
	if rec.Code != http.StatusUnauthorized || strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("reuse: %d %s", rec.Code, rec.Body)
	}
	if rec := call(http.MethodPost, `not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad body: %d", rec.Code)
	}
	if rec := call(http.MethodGet, ``); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", rec.Code)
	}

	failing := New(keyring(t, "main", "main"), "secret", &memClaims{err: errors.New("db down")})
	tok, _, _ := failing.Mint(0)
	rec = httptest.NewRecorder()
	failing.RedeemHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, RedeemPath, strings.NewReader(`{"token":"`+tok+`"}`)))
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") {
		t.Fatalf("store failure: %d %s", rec.Code, rec.Body)
	}
}
