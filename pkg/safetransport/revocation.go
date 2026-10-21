package safetransport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

// ErrRevoked is wrapped by the TLS handshake error when an intermediate in
// the server's chain is listed on its issuer's CRL.
var ErrRevoked = errors.New("tls: certificate chain contains a revoked intermediate")

// ErrRevocationUnknown is wrapped by the TLS handshake error in
// RevocationHard mode when an intermediate's CRL could not be obtained or
// verified.
var ErrRevocationUnknown = errors.New("tls: revocation status of an intermediate is unknown")

// RevocationMode says what a RevocationChecker does when it cannot tell
// whether an intermediate is revoked (CRL unreachable, unverifiable or
// expired).
type RevocationMode int

const (
	// RevocationSoft allows the connection and records the error in Status.
	// A certificate found on a CRL is refused either way.
	RevocationSoft RevocationMode = iota
	// RevocationHard refuses the connection with ErrRevocationUnknown.
	RevocationHard
)

// RevocationOptions configures a RevocationChecker. The zero value is
// usable: soft mode, 6h maximum age, 5s fetch timeout, 10 MiB CRL limit.
type RevocationOptions struct {
	// Client downloads CRLs. CRL addresses come from certificates a remote
	// server sends, so a server-side program should pass a client that
	// enforces its network (SSRF) policy. It must not use this checker
	// itself. Default: a client on New() with FetchTimeout.
	Client *http.Client
	// Mode is the policy when revocation status is unknown.
	Mode RevocationMode
	// MaxAge is how old a cached CRL may get before a handshake that uses
	// it starts a background refresh. The handshake still uses the cached
	// copy. Default 6h.
	MaxAge time.Duration
	// FetchTimeout bounds one CRL download, and is also the most one
	// handshake waits in total for downloads, however many intermediates
	// its chain has. A handshake waits only for CRLs it has no usable copy
	// of; a download still running when the budget is spent finishes in the
	// background for later handshakes, and this one treats that CRL as
	// unknown. Keep it below the transport's TLSHandshakeTimeout. Default 5s.
	FetchTimeout time.Duration
	// MaxCRLBytes caps a downloaded CRL. Default 10 MiB.
	MaxCRLBytes int64
	// Logger receives download failures. Default slog.Default().
	Logger *slog.Logger
}

// RevocationChecker checks intermediates in verified TLS chains against
// their issuers' CRLs. It keeps CRLs in memory and needs no scheduler:
//
//   - A CRL is downloaded the first time a chain needs it; that handshake
//     waits for it, at most FetchTimeout across the whole chain.
//   - A cached CRL older than MaxAge is refreshed in the background by the
//     next handshake that uses it.
//   - A CRL past its own NextUpdate is downloaded again before use.
//   - Refresh downloads every known CRL now.
//
// Only intermediates are checked, from the root down, and the check stops at
// the first revoked one, so a CRL address written by a revoked CA is never
// fetched. Leaf certificates are not checked. Intermediates without an
// http(s) CRL address (typical for private CAs) are skipped in both modes.
// It also runs RejectWeakKeys.
//
// The cache holds one entry per CRL address and is never pruned. That is
// bounded: an address is only added from a chain Go has already verified
// against the trusted roots, so a server cannot invent entries without a
// CA's signature, and real chains use a few dozen addresses.
type RevocationChecker struct {
	opts       RevocationOptions
	retryAfter time.Duration

	mu   sync.Mutex
	crls map[string]*crlEntry
}

// crlEntry is one CRL address. All fields are guarded by RevocationChecker.mu.
type crlEntry struct {
	url         string
	issuer      *x509.Certificate // verifies downloads
	list        *crlData          // nil until a download succeeded
	fetchedAt   time.Time
	lastAttempt time.Time
	lastErr     error
	inflight    chan struct{} // non-nil while downloading, closed when done
}

type crlData struct {
	list    *x509.RevocationList
	revoked map[string]struct{} // serial numbers
	// verified caches signature checks per issuer public key: one CRL
	// address can serve certificates whose issuer was reissued.
	verified map[string]bool
}

// CRLStatus describes one cached CRL.
type CRLStatus struct {
	URL        string    `json:"url"`
	Issuer     string    `json:"issuer"`
	FetchedAt  time.Time `json:"fetched_at,omitzero"`
	NextUpdate time.Time `json:"next_update,omitzero"`
	Entries    int       `json:"entries"`
	Error      string    `json:"error,omitempty"`
}

// NewRevocationChecker returns a checker with an empty cache.
func NewRevocationChecker(opts RevocationOptions) *RevocationChecker {
	if opts.MaxAge <= 0 {
		opts.MaxAge = 6 * time.Hour
	}
	if opts.FetchTimeout <= 0 {
		opts.FetchTimeout = 5 * time.Second
	}
	if opts.MaxCRLBytes <= 0 {
		opts.MaxCRLBytes = 10 << 20
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Client == nil {
		opts.Client = &http.Client{Transport: New(), Timeout: opts.FetchTimeout}
	}
	return &RevocationChecker{opts: opts, retryAfter: time.Minute, crls: map[string]*crlEntry{}}
}

// TLSConfig returns a tls.Config whose VerifyConnection runs the checker.
// A nil checker returns the package TLSConfig (weak keys only).
func (c *RevocationChecker) TLSConfig() *tls.Config {
	if c == nil {
		return TLSConfig()
	}
	return &tls.Config{VerifyConnection: c.VerifyConnection}
}

// Transport returns a clone of http.DefaultTransport using c.TLSConfig().
func (c *RevocationChecker) Transport() *http.Transport {
	t := New()
	t.TLSClientConfig = c.TLSConfig()
	return t
}

// VerifyConnection is a tls.Config.VerifyConnection hook: RejectWeakKeys,
// then the revocation check on every verified chain.
func (c *RevocationChecker) VerifyConnection(cs tls.ConnectionState) error {
	if err := RejectWeakKeys(cs); err != nil {
		return err
	}
	if c == nil {
		return nil
	}
	// One wait budget for the whole handshake, spent root-down: downloads
	// run one at a time so a CRL address below a revoked intermediate is
	// never fetched, and the total stays under the handshake timeout.
	deadline := time.Now().Add(c.opts.FetchTimeout)
	for _, chain := range cs.VerifiedChains {
		// chain[0] is the leaf, chain[len-1] the trust anchor.
		for i := len(chain) - 2; i >= 1; i-- {
			if err := c.checkCert(chain[i], chain[i+1], deadline); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *RevocationChecker) checkCert(cert, issuer *x509.Certificate, deadline time.Time) error {
	urls := crlURLs(cert)
	if len(urls) == 0 {
		return nil
	}
	var unknown error
	for _, u := range urls {
		revoked, err := c.lookup(u, cert, issuer, deadline)
		if revoked {
			return fmt.Errorf("%w: %q (serial %x)", ErrRevoked, cert.Subject.CommonName, cert.SerialNumber)
		}
		if err == nil {
			return nil
		}
		unknown = err
	}
	if c.opts.Mode == RevocationHard {
		return fmt.Errorf("%w: %q: %w", ErrRevocationUnknown, cert.Subject.CommonName, unknown)
	}
	return nil
}

// lookup reports whether cert is on the CRL at u. A non-nil error means the
// CRL gave no answer for cert (missing, unverifiable, expired, or still
// downloading at deadline); a serial listed on an expired CRL still counts
// as revoked.
func (c *RevocationChecker) lookup(u string, cert, issuer *x509.Certificate, deadline time.Time) (bool, error) {
	now := time.Now()
	c.mu.Lock()
	e, ok := c.crls[u]
	if !ok {
		e = &crlEntry{url: u, issuer: issuer}
		c.crls[u] = e
	}
	usable := e.list != nil && (e.list.list.NextUpdate.IsZero() || now.Before(e.list.list.NextUpdate))
	retryDue := e.lastErr == nil || now.Sub(e.lastAttempt) >= c.retryAfter
	var wait <-chan struct{}
	switch {
	case !usable && retryDue:
		wait = c.startFetchLocked(e)
	case usable && now.Sub(e.fetchedAt) >= c.opts.MaxAge && retryDue:
		c.startFetchLocked(e)
	}
	c.mu.Unlock()

	timedOut := false
	if wait != nil {
		timer := time.NewTimer(time.Until(deadline))
		select {
		case <-wait:
		case <-timer.C:
			timedOut = true
		}
		timer.Stop()
	}

	c.mu.Lock()
	data, lastErr := e.list, e.lastErr
	c.mu.Unlock()
	if data == nil {
		switch {
		case timedOut:
			lastErr = errors.New("CRL download did not finish within the handshake's wait budget")
		case lastErr == nil:
			lastErr = errors.New("CRL not downloaded")
		}
		return false, lastErr
	}
	if err := c.verifyFor(data, issuer); err != nil {
		return false, err
	}
	if string(data.list.RawIssuer) != string(cert.RawIssuer) {
		return false, errors.New("CRL issuer does not match the certificate's issuer")
	}
	if _, ok := data.revoked[cert.SerialNumber.String()]; ok {
		return true, nil
	}
	if next := data.list.NextUpdate; !next.IsZero() && !time.Now().Before(next) {
		return false, fmt.Errorf("CRL expired at %s", next.Format(time.RFC3339))
	}
	return false, nil
}

// verifyFor checks data's signature against issuer once per issuer key.
func (c *RevocationChecker) verifyFor(data *crlData, issuer *x509.Certificate) error {
	key := string(issuer.RawSubjectPublicKeyInfo)
	c.mu.Lock()
	ok, seen := data.verified[key]
	c.mu.Unlock()
	if !seen {
		ok = data.list.CheckSignatureFrom(issuer) == nil
		c.mu.Lock()
		data.verified[key] = ok
		c.mu.Unlock()
	}
	if !ok {
		return errors.New("CRL signature does not verify against the issuer")
	}
	return nil
}

// startFetchLocked starts a download of e unless one is running and returns
// the channel closed when it finishes. c.mu must be held.
func (c *RevocationChecker) startFetchLocked(e *crlEntry) <-chan struct{} {
	if e.inflight != nil {
		return e.inflight
	}
	done := make(chan struct{})
	e.inflight = done
	e.lastAttempt = time.Now()
	issuer := e.issuer
	go func() {
		defer close(done)
		data, err := c.download(e.url, issuer)
		c.mu.Lock()
		defer c.mu.Unlock()
		e.inflight = nil
		e.lastErr = err
		if err != nil {
			c.opts.Logger.Warn("CRL download failed", "url", e.url, "error", err)
			return
		}
		e.list = data
		e.fetchedAt = time.Now()
	}()
	return done
}

func (c *RevocationChecker) download(u string, issuer *x509.Certificate) (*crlData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.FetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.opts.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CRL download: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.opts.MaxCRLBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > c.opts.MaxCRLBytes {
		return nil, fmt.Errorf("CRL larger than %d bytes", c.opts.MaxCRLBytes)
	}
	if block, _ := pem.Decode(body); block != nil && block.Type == "X509 CRL" {
		body = block.Bytes
	}
	list, err := x509.ParseRevocationList(body)
	if err != nil {
		return nil, err
	}
	if err := list.CheckSignatureFrom(issuer); err != nil {
		return nil, fmt.Errorf("CRL signature: %w", err)
	}
	data := &crlData{
		list:     list,
		revoked:  make(map[string]struct{}, len(list.RevokedCertificateEntries)),
		verified: map[string]bool{string(issuer.RawSubjectPublicKeyInfo): true},
	}
	for _, r := range list.RevokedCertificateEntries {
		data.revoked[r.SerialNumber.String()] = struct{}{}
	}
	return data, nil
}

// Refresh downloads every known CRL now, waits for the downloads, and
// returns their errors joined. A CRL whose download fails keeps its cached
// copy.
func (c *RevocationChecker) Refresh(ctx context.Context) error {
	c.mu.Lock()
	waits := make([]<-chan struct{}, 0, len(c.crls))
	entries := make([]*crlEntry, 0, len(c.crls))
	for _, e := range c.crls {
		waits = append(waits, c.startFetchLocked(e))
		entries = append(entries, e)
	}
	c.mu.Unlock()

	for _, w := range waits {
		select {
		case <-w:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	var errs []error
	c.mu.Lock()
	for _, e := range entries {
		if e.lastErr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.url, e.lastErr))
		}
	}
	c.mu.Unlock()
	return errors.Join(errs...)
}

// Status lists the cached CRLs by URL.
func (c *RevocationChecker) Status() []CRLStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]CRLStatus, 0, len(c.crls))
	for _, e := range c.crls {
		s := CRLStatus{URL: e.url, Issuer: e.issuer.Subject.CommonName, FetchedAt: e.fetchedAt}
		if e.list != nil {
			s.NextUpdate = e.list.list.NextUpdate
			s.Entries = len(e.list.revoked)
		}
		if e.lastErr != nil {
			s.Error = e.lastErr.Error()
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].URL < out[j].URL })
	return out
}

// crlURLs returns cert's http(s) CRL distribution points.
func crlURLs(cert *x509.Certificate) []string {
	var out []string
	for _, raw := range cert.CRLDistributionPoints {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			continue
		}
		out = append(out, raw)
	}
	return out
}
