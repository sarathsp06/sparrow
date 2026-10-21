// Package safetransport provides HTTP transports and TLS configs that refuse
// certificate chains Go's verifier would otherwise accept but that are known
// to be compromised. Go's crypto/x509 checks no revocation (no CRLs, no OCSP,
// no Mozilla OneCRL), so a revoked CA is trusted until its root leaves the
// system store, and a cross-signed copy may stay trusted after that.
//
// Today it rejects the eMudhra intermediate keys from Mozilla bug 2083449 (see
// RejectWeakKeys). Use New for a ready transport, TLSConfig when building
// your own, or RejectWeakKeys directly as a VerifyConnection hook. The
// package is stdlib-only, so any Go program can import it.
package safetransport

import (
	"crypto/tls"
	"net/http"
)

// TLSConfig returns a new tls.Config with RejectWeakKeys as its
// VerifyConnection hook. Normal certificate verification still runs first;
// leave InsecureSkipVerify off, since the hook only inspects verified chains.
func TLSConfig() *tls.Config {
	return &tls.Config{VerifyConnection: RejectWeakKeys}
}

// New returns a clone of http.DefaultTransport (proxy from the environment,
// HTTP/2, default timeouts) whose TLS config is TLSConfig.
func New() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = TLSConfig()
	return t
}
