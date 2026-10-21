package safetransport

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// crlServer serves CRLs by path; a path with no body answers 500. Requests
// wait hang first (for slow-server tests).
type crlServer struct {
	srv    *httptest.Server
	mu     sync.Mutex
	bodies map[string][]byte
	hits   map[string]int
	hang   time.Duration
}

func newCRLServer(t *testing.T) *crlServer {
	s := &crlServer{bodies: map[string][]byte{}, hits: map[string]int{}}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		hang := s.hang
		s.mu.Unlock()
		select {
		case <-time.After(hang):
		case <-r.Context().Done():
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.hits[r.URL.Path]++
		body, ok := s.bodies[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *crlServer) url(path string) string { return s.srv.URL + path }

func (s *crlServer) set(path string, der []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bodies[path] = der
}

func (s *crlServer) count(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits[path]
}

// makeCRL returns a CRL signed by issuer listing revoked.
func makeCRL(t *testing.T, issuer *x509.Certificate, key crypto.Signer, next time.Time, revoked ...*x509.Certificate) []byte {
	t.Helper()
	tmpl := &x509.RevocationList{
		Number:     big.NewInt(time.Now().UnixNano()),
		ThisUpdate: time.Now().Add(-2 * time.Hour),
		NextUpdate: next,
	}
	for _, c := range revoked {
		tmpl.RevokedCertificateEntries = append(tmpl.RevokedCertificateEntries,
			x509.RevocationListEntry{SerialNumber: c.SerialNumber, RevocationTime: time.Now().Add(-time.Hour)})
	}
	der, err := x509.CreateRevocationList(rand.Reader, tmpl, issuer, key)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

// testPKI is a root and the CRL server its intermediates point at.
type testPKI struct {
	rootKey *ecdsa.PrivateKey
	root    *x509.Certificate
	crls    *crlServer
}

func newTestPKI(t *testing.T) *testPKI {
	key := genKey(t, elliptic.P256())
	return &testPKI{rootKey: key, root: newCert(t, "test root", true, key.Public(), nil, key), crls: newCRLServer(t)}
}

// serve starts a TLS server for a leaf issued by the last of intermediates
// (each issued by the one before it, the first by the root) and returns its
// URL. intermediates holds signing keys in step with certs.
func (p *testPKI) serve(t *testing.T, certs []*x509.Certificate, keys []*ecdsa.PrivateKey) string {
	t.Helper()
	leafKey := genKey(t, elliptic.P256())
	leaf := newCert(t, "example.com", false, leafKey.Public(), certs[len(certs)-1], keys[len(keys)-1])
	chain := [][]byte{leaf.Raw}
	for i := len(certs) - 1; i >= 0; i-- {
		chain = append(chain, certs[i].Raw)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: chain, PrivateKey: leafKey}}}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv.URL
}

// intermediate issues a CA certificate under parent.
func (p *testPKI) intermediate(t *testing.T, cn string, parent *x509.Certificate, parentKey crypto.Signer, crlURLs ...string) (*x509.Certificate, *ecdsa.PrivateKey) {
	key := genKey(t, elliptic.P256())
	return newCert(t, cn, true, key.Public(), parent, parentKey, crlURLs...), key
}

// get makes one request through c, trusting the test root, on a fresh
// connection so every call runs the handshake hook.
func (p *testPKI) get(t *testing.T, c *RevocationChecker, url string) error {
	t.Helper()
	roots := x509.NewCertPool()
	roots.AddCert(p.root)
	tr := c.Transport()
	tr.TLSClientConfig.RootCAs = roots
	tr.TLSClientConfig.ServerName = "example.com"
	tr.DisableKeepAlives = true
	resp, err := (&http.Client{Transport: tr}).Get(url)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func TestRevocationCleanAndRevoked(t *testing.T) {
	p := newTestPKI(t)
	inter, interKey := p.intermediate(t, "inter", p.root, p.rootKey, p.crls.url("/root.crl"))
	url := p.serve(t, []*x509.Certificate{inter}, []*ecdsa.PrivateKey{interKey})

	p.crls.set("/root.crl", makeCRL(t, p.root, p.rootKey, time.Now().Add(24*time.Hour)))
	if err := p.get(t, NewRevocationChecker(RevocationOptions{Mode: RevocationHard}), url); err != nil {
		t.Fatalf("intermediate not on the CRL: %v", err)
	}

	p.crls.set("/root.crl", makeCRL(t, p.root, p.rootKey, time.Now().Add(24*time.Hour), inter))
	if err := p.get(t, NewRevocationChecker(RevocationOptions{}), url); !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked intermediate: got %v, want ErrRevoked", err)
	}
}

func TestRevocationUnavailable(t *testing.T) {
	p := newTestPKI(t)
	inter, interKey := p.intermediate(t, "inter", p.root, p.rootKey, p.crls.url("/missing.crl"))
	url := p.serve(t, []*x509.Certificate{inter}, []*ecdsa.PrivateKey{interKey})

	soft := NewRevocationChecker(RevocationOptions{})
	for range 2 {
		if err := p.get(t, soft, url); err != nil {
			t.Fatalf("soft mode with CRL unreachable: %v", err)
		}
	}
	if got := p.crls.count("/missing.crl"); got != 1 {
		t.Fatalf("CRL requested %d times within the retry delay, want 1", got)
	}
	if st := soft.Status(); len(st) != 1 || st[0].Error == "" {
		t.Fatalf("status should record the download error: %+v", st)
	}

	hard := NewRevocationChecker(RevocationOptions{Mode: RevocationHard})
	if err := p.get(t, hard, url); !errors.Is(err, ErrRevocationUnknown) {
		t.Fatalf("hard mode with CRL unreachable: got %v, want ErrRevocationUnknown", err)
	}
}

func TestRevocationBadSignature(t *testing.T) {
	p := newTestPKI(t)
	inter, interKey := p.intermediate(t, "inter", p.root, p.rootKey, p.crls.url("/root.crl"))
	url := p.serve(t, []*x509.Certificate{inter}, []*ecdsa.PrivateKey{interKey})

	// Same subject as the real root, different key: the CRL parses, its
	// issuer name matches, its signature does not verify.
	forgerKey := genKey(t, elliptic.P256())
	forger := newCert(t, "test root", true, forgerKey.Public(), nil, forgerKey)
	p.crls.set("/root.crl", makeCRL(t, forger, forgerKey, time.Now().Add(24*time.Hour)))

	if err := p.get(t, NewRevocationChecker(RevocationOptions{}), url); err != nil {
		t.Fatalf("soft mode with forged CRL: %v", err)
	}
	if err := p.get(t, NewRevocationChecker(RevocationOptions{Mode: RevocationHard}), url); !errors.Is(err, ErrRevocationUnknown) {
		t.Fatalf("hard mode with forged CRL: got %v, want ErrRevocationUnknown", err)
	}
}

func TestRevocationRefresh(t *testing.T) {
	p := newTestPKI(t)
	inter, interKey := p.intermediate(t, "inter", p.root, p.rootKey, p.crls.url("/root.crl"))
	url := p.serve(t, []*x509.Certificate{inter}, []*ecdsa.PrivateKey{interKey})
	c := NewRevocationChecker(RevocationOptions{})

	p.crls.set("/root.crl", makeCRL(t, p.root, p.rootKey, time.Now().Add(24*time.Hour)))
	if err := p.get(t, c, url); err != nil {
		t.Fatal(err)
	}

	p.crls.set("/root.crl", makeCRL(t, p.root, p.rootKey, time.Now().Add(24*time.Hour), inter))
	if err := p.get(t, c, url); err != nil {
		t.Fatalf("cached CRL younger than MaxAge should be used as is: %v", err)
	}
	if err := c.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := p.get(t, c, url); !errors.Is(err, ErrRevoked) {
		t.Fatalf("after Refresh: got %v, want ErrRevoked", err)
	}
}

func TestRevocationRefreshOnUse(t *testing.T) {
	p := newTestPKI(t)
	inter, interKey := p.intermediate(t, "inter", p.root, p.rootKey, p.crls.url("/root.crl"))
	url := p.serve(t, []*x509.Certificate{inter}, []*ecdsa.PrivateKey{interKey})
	c := NewRevocationChecker(RevocationOptions{MaxAge: time.Nanosecond})

	p.crls.set("/root.crl", makeCRL(t, p.root, p.rootKey, time.Now().Add(24*time.Hour)))
	if err := p.get(t, c, url); err != nil {
		t.Fatal(err)
	}

	p.crls.set("/root.crl", makeCRL(t, p.root, p.rootKey, time.Now().Add(24*time.Hour), inter))
	// The stale copy is used while a background download starts.
	if err := p.get(t, c, url); err != nil {
		t.Fatalf("stale CRL should be used while refreshing: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for c.Status()[0].Entries != 1 {
		if time.Now().After(deadline) {
			t.Fatal("background refresh never stored the new CRL")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := p.get(t, c, url); !errors.Is(err, ErrRevoked) {
		t.Fatalf("after background refresh: got %v, want ErrRevoked", err)
	}
}

func TestRevocationExpiredCRL(t *testing.T) {
	p := newTestPKI(t)
	inter, interKey := p.intermediate(t, "inter", p.root, p.rootKey, p.crls.url("/root.crl"))
	url := p.serve(t, []*x509.Certificate{inter}, []*ecdsa.PrivateKey{interKey})
	expired := time.Now().Add(-time.Hour)

	p.crls.set("/root.crl", makeCRL(t, p.root, p.rootKey, expired))
	if err := p.get(t, NewRevocationChecker(RevocationOptions{}), url); err != nil {
		t.Fatalf("soft mode with expired CRL: %v", err)
	}
	if err := p.get(t, NewRevocationChecker(RevocationOptions{Mode: RevocationHard}), url); !errors.Is(err, ErrRevocationUnknown) {
		t.Fatalf("hard mode with expired CRL: got %v, want ErrRevocationUnknown", err)
	}

	// Revocation is permanent: an expired CRL that lists it still counts.
	p.crls.set("/root.crl", makeCRL(t, p.root, p.rootKey, expired, inter))
	if err := p.get(t, NewRevocationChecker(RevocationOptions{}), url); !errors.Is(err, ErrRevoked) {
		t.Fatalf("expired CRL listing the intermediate: got %v, want ErrRevoked", err)
	}
}

func TestRevocationTopDown(t *testing.T) {
	p := newTestPKI(t)
	// root -> revoked -> attacker sub-CA (CRL address it chose) -> leaf.
	revoked, revokedKey := p.intermediate(t, "revoked", p.root, p.rootKey, p.crls.url("/root.crl"))
	sub, subKey := p.intermediate(t, "attacker sub-CA", revoked, revokedKey, p.crls.url("/attacker.crl"))
	url := p.serve(t, []*x509.Certificate{revoked, sub}, []*ecdsa.PrivateKey{revokedKey, subKey})
	p.crls.set("/root.crl", makeCRL(t, p.root, p.rootKey, time.Now().Add(24*time.Hour), revoked))

	if err := p.get(t, NewRevocationChecker(RevocationOptions{}), url); !errors.Is(err, ErrRevoked) {
		t.Fatalf("got %v, want ErrRevoked", err)
	}
	if got := p.crls.count("/attacker.crl"); got != 0 {
		t.Fatalf("CRL address below a revoked intermediate was fetched %d times", got)
	}
}

func TestRevocationNoCRLAddress(t *testing.T) {
	p := newTestPKI(t)
	inter, interKey := p.intermediate(t, "private CA", p.root, p.rootKey)
	url := p.serve(t, []*x509.Certificate{inter}, []*ecdsa.PrivateKey{interKey})
	if err := p.get(t, NewRevocationChecker(RevocationOptions{Mode: RevocationHard}), url); err != nil {
		t.Fatalf("intermediate without a CRL address: %v", err)
	}
}

func TestRevocationNilChecker(t *testing.T) {
	var c *RevocationChecker
	if c.TLSConfig().VerifyConnection == nil {
		t.Fatal("nil checker should still reject weak keys")
	}
}

func TestRevocationWaitBudgetIsPerHandshake(t *testing.T) {
	p := newTestPKI(t)
	a, aKey := p.intermediate(t, "a", p.root, p.rootKey, p.crls.url("/a.crl"))
	b, bKey := p.intermediate(t, "b", a, aKey, p.crls.url("/b.crl"))
	c, cKey := p.intermediate(t, "c", b, bKey, p.crls.url("/c.crl"))
	url := p.serve(t, []*x509.Certificate{a, b, c}, []*ecdsa.PrivateKey{aKey, bKey, cKey})
	p.crls.hang = 5 * time.Second

	const budget = 300 * time.Millisecond
	soft := NewRevocationChecker(RevocationOptions{FetchTimeout: budget})
	start := time.Now()
	if err := p.get(t, soft, url); err != nil {
		t.Fatalf("soft mode with a hanging CRL server: %v", err)
	}
	// Waiting per intermediate would take 3 x budget.
	if took := time.Since(start); took > 2*budget {
		t.Fatalf("handshake waited %s for CRLs, want about one %s budget", took, budget)
	}

	hard := NewRevocationChecker(RevocationOptions{Mode: RevocationHard, FetchTimeout: budget})
	if err := p.get(t, hard, url); !errors.Is(err, ErrRevocationUnknown) {
		t.Fatalf("hard mode with a hanging CRL server: got %v, want ErrRevocationUnknown", err)
	}
}
