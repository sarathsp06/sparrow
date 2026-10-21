package safetransport

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
)

// maxWeakScalar bounds the private scalars treated as compromised. eMudhra's
// 2025 key ceremony produced CA keys whose private scalar was an integer below
// 256 (Mozilla bug 2083449), recoverable by trying every value. The
// certificates were revoked, but Go checks neither CRLs nor Mozilla's OneCRL,
// and Sectigo cross-signed the same keys under roots Go still trusts. Matching
// the key itself covers the originals and the cross-signs without a hardcoded
// certificate list.
//
// This targets that flaw only; it is not a general weak-key detector. A
// scalar of 70000 is just as recoverable and passes. Widening the bound costs
// one scalar multiplication per value at first use, so it stays at what the
// incident needs.
const maxWeakScalar = 256

// ErrWeakKey is wrapped by the TLS handshake error when a server's chain
// contains a key with a trivially recoverable private key. Match it with
// errors.Is.
var ErrWeakKey = errors.New("tls: certificate chain contains a compromised key (private key below 256, see Mozilla bug 2083449)")

// weakECKeys holds the x-coordinates of k·G for 0 < k < 256 on P-256, P-384
// and P-521, built on first use. Keying on x alone also matches (n−k)·G, the
// negated point, whose private key is just as easy to recover. The
// coordinate lengths differ per curve, so the curves cannot collide.
var weakECKeys = sync.OnceValue(func() map[string]struct{} {
	curves := []struct {
		curve      ecdh.Curve
		scalarSize int
	}{{ecdh.P256(), 32}, {ecdh.P384(), 48}, {ecdh.P521(), 66}}
	keys := make(map[string]struct{}, len(curves)*(maxWeakScalar-1))
	for _, c := range curves {
		for k := 1; k < maxWeakScalar; k++ {
			scalar := make([]byte, c.scalarSize)
			scalar[c.scalarSize-1] = byte(k)
			priv, err := c.curve.NewPrivateKey(scalar)
			if err != nil {
				panic(fmt.Sprintf("weak key table: %v", err))
			}
			keys[string(xCoordinate(priv.PublicKey()))] = struct{}{}
		}
	}
	return keys
})

// xCoordinate returns X from the uncompressed encoding 0x04 || X || Y.
func xCoordinate(pub *ecdh.PublicKey) []byte {
	b := pub.Bytes()
	return b[1 : 1+(len(b)-1)/2]
}

// RejectWeakKeys is a tls.Config.VerifyConnection hook. It runs after normal
// chain verification and refuses any verified chain that contains a
// certificate with a compromised EC key. It only sees chains when
// verification ran, so it does nothing with InsecureSkipVerify.
func RejectWeakKeys(cs tls.ConnectionState) error {
	weak := weakECKeys()
	for _, chain := range cs.VerifiedChains {
		for _, cert := range chain {
			pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
			if !ok {
				continue
			}
			point, err := pub.ECDH()
			if err != nil {
				continue
			}
			if _, bad := weak[string(xCoordinate(point))]; bad {
				return fmt.Errorf("%w: %q", ErrWeakKey, cert.Subject.CommonName)
			}
		}
	}
	return nil
}
