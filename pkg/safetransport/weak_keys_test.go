package safetransport

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newCert issues a certificate for pub signed by parent/parentKey (self-signed
// when parent is nil), with crlURLs as its CRL distribution points.
func newCert(t *testing.T, cn string, isCA bool, pub crypto.PublicKey, parent *x509.Certificate, parentKey crypto.Signer, crlURLs ...string) *x509.Certificate {
	t.Helper()
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		CRLDistributionPoints: crlURLs,
	}
	if !isCA {
		tmpl.DNSNames = []string{"example.com"}
	}
	if parent == nil {
		parent = tmpl
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, pub, parentKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func genKey(t *testing.T, curve elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

// handshake serves a root -> intermediate -> leaf chain and reports the error
// of a request through New() configured to trust the root.
func handshake(t *testing.T, intermediateKey *ecdsa.PrivateKey) error {
	t.Helper()
	rootKey := genKey(t, elliptic.P384())
	root := newCert(t, "test root", true, rootKey.Public(), nil, rootKey)
	inter := newCert(t, "test intermediate", true, intermediateKey.Public(), root, rootKey)
	leafKey := genKey(t, elliptic.P384())
	leaf := newCert(t, "example.com", false, leafKey.Public(), inter, intermediateKey)

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{
		Certificate: [][]byte{leaf.Raw, inter.Raw},
		PrivateKey:  leafKey,
	}}}
	srv.StartTLS()
	t.Cleanup(srv.Close)

	roots := x509.NewCertPool()
	roots.AddCert(root)
	transport := New()
	transport.TLSClientConfig.RootCAs = roots
	transport.TLSClientConfig.ServerName = "example.com"
	c := &http.Client{Transport: transport}
	resp, err := c.Get(srv.URL)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

// scalarKey returns the key on curve whose private scalar is d.
func scalarKey(t *testing.T, curve elliptic.Curve, d *big.Int) *ecdsa.PrivateKey {
	t.Helper()
	scalar := d.FillBytes(make([]byte, (curve.Params().N.BitLen()+7)/8))
	key, err := ecdsa.ParseRawPrivateKey(curve, scalar)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestRejectWeakKeys(t *testing.T) {
	for _, curve := range []elliptic.Curve{elliptic.P256(), elliptic.P384()} {
		name := curve.Params().Name
		if err := handshake(t, genKey(t, curve)); err != nil {
			t.Fatalf("%s chain with normal keys: %v", name, err)
		}

		n := curve.Params().N
		for _, k := range []int64{1, 0x7f, 0xff} {
			for _, d := range []*big.Int{
				big.NewInt(k),
				new(big.Int).Sub(n, big.NewInt(k)), // negated point, same x
			} {
				if err := handshake(t, scalarKey(t, curve, d)); !errors.Is(err, ErrWeakKey) {
					t.Fatalf("%s intermediate with private key %v: got %v, want ErrWeakKey", name, d, err)
				}
			}
		}
	}

	if err := handshake(t, scalarKey(t, elliptic.P384(), big.NewInt(256))); err != nil {
		t.Fatalf("private key 256 is outside the table: %v", err)
	}
}

func TestWeakECKeysTable(t *testing.T) {
	if got, want := len(weakECKeys()), 3*(maxWeakScalar-1); got != want {
		t.Fatalf("weak key table has %d points, want %d", got, want)
	}
}
