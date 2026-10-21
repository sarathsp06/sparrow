---
title: The private key was 4
kicker: What the eMudhra key compromise means for a server that sends webhooks
author: sarathsp06
description: "In October 2026 eight intermediate CA certificates turned out to have private keys between 4 and 237. Browsers blocked them within hours. Go, and every server built on it, kept trusting them. Here is what went wrong, why revocation never reached us, and the two checks Sparrow now runs on every outbound TLS connection: one that refuses those keys, and one that makes Go check revocation at all."
pubDate: 2026-10-21
tags: [security, tls, pki, go]
---

On 17 October 2026 a researcher reported that eight intermediate certificates issued by eMudhra, a CA trusted by every major browser, had private keys that anyone could calculate ([Mozilla bug 2083449](https://bugzilla.mozilla.org/show_bug.cgi?id=2083449)). Not crack with a GPU farm. Calculate. One of them, `EM OV TLS CA - G3C`, had the private key **4**.

The full list, as later published:

| Intermediate | Private key |
|---|---|
| EM OV TLS CA - G3C | 4 |
| EM EV TLS CA - G3C | 56 |
| EM OV CS CA - G3B | 66 |
| EM EV TLS CA - G3B | 115 |
| EM OV TLS CA - G3B | 155 |
| EM DV TLS CA - G3B | 156 |
| EM DV TLS CA - G3C | 230 |
| EM OV CS CA - G3C | 237 |

Six of them issue TLS certificates. Anyone holding one of those keys can mint a certificate for any hostname, and it will verify.

Sparrow's whole job is opening TLS connections to other people's servers and handing them your data. So we looked at what this meant for us. The short answer: browsers were protected within three hours, and Sparrow would have trusted those certificates indefinitely.

## What a P-384 key is supposed to be

An ECDSA private key is a number *d*. The public key is the point *Q = d·G*, where *G* is a fixed point on the curve that everyone knows. Going from *d* to *Q* is fast. Going back is the discrete logarithm problem, and it is hard for exactly one reason: *d* is drawn at random from about 2^384 possibilities on P-384.

Nothing in the math hides *d*. The size of the haystack does.

eMudhra's keys were 47 zero bytes followed by one meaningful byte. Between 3 and 8 bits of secret where there should have been 384. To recover one, you compute *1·G*, *2·G*, *3·G* and compare each result with the public key in the certificate. For `G3C` you stop at four.

The signatures these keys made were perfectly valid. That is what makes it dangerous: nothing about the certificates looked wrong. The keys were real keys. They just weren't secret.

## How it happened

eMudhra's own account (comment 11 on the bug) describes a key ceremony on 30 April 2025, using Utimaco HSMs:

1. The real keys were generated correctly inside the HSM.
2. An "approved deviation" was added to test key wrapping and unwrapping.
3. A flaw in the ceremony script created a second, weak set of keys.
4. The two sets had similar labels, and the weak ones were used to make the certificate requests.
5. Those requests were signed by eMudhra's root, and also cross-signed by Sectigo.

The HSM did its job. The cryptography did its job. What failed was the procedure around them, in a few specific ways:

- **Test material in a production ceremony.** A key ceremony exists to produce keys whose entire history can be accounted for. A wrap/unwrap test brought extra code and extra keys into the one room where nothing extra belongs.
- **Keys chosen by name, not by fingerprint.** The weak keys were picked because their labels looked right. Comparing the public key fingerprint recorded at generation with the one in the certificate request would have caught this in seconds.
- **Nobody checked the key.** A public key equal to *4·G* is found by a loop that takes microseconds. Neither the ceremony nor any certificate linter in the pipeline ran one.
- **The cross-sign doubled the exposure.** Sectigo signed the same weak public keys under its own roots, so revoking eMudhra's copies, or even removing eMudhra's root, does not close every path to trust.

## Why the chain of trust accepted it

Certificate validation checks signatures. It does not, and cannot in general, check how a key was made. The root's signature on each intermediate was valid. A public key of *4·G* is a point on the curve that looks as random as any other. Root programs trust a CA because auditors watch its ceremonies; this ceremony was watched, the deviation was approved, and the weakness went through anyway.

## Why browsers were safe and we weren't

The response was fast. Mozilla pushed the certificates to OneCRL within about three hours. eMudhra and Sectigo revoked their certificates within a day. Mozilla and Chrome are removing the `emSign ECC Root CA - G3` root.

None of that reaches a Go program.

- Go's `crypto/x509` checks neither CRLs nor OneCRL. A revoked certificate verifies exactly like a valid one.
- Go trusts whatever root bundle the operating system provides. Sparrow's image is built on `gcr.io/distroless/static-debian12`; we pulled the latest digest, it was identical to the one we pin, and it still ships `emSign ECC Root CA - G3`. Rebuilding the image changes nothing until Debian updates `ca-certificates`.
- Even after the root goes, Sectigo's cross-signed copies chain to Sectigo roots that stay trusted.

This is not a Go quirk. Most server software, built on Go, OpenSSL or Java, is in the same position. Browsers are the exception, not the rule.

## What we considered

**Pulling in OneCRL.** It is a public JSON feed, and the eMudhra intermediates are already in it. But using it means a periodic fetch job, storage, an embedded fallback for instances that cannot reach Mozilla (most Sparrow deployments run behind a VPN), and a new setting. All to track a list that changes a few times a year. For Sparrow that is complexity, not protection.

**Hardcoding the sixteen certificates.** Simple, but it needs the exact certificates, and blocks only those exact certificates.

**Blocking the flaw itself.** Every compromised key has a private value below 256. That is a property we can check directly, today, with no network access.

**Checking revocation ourselves.** The real gap is that Go never asks whether a certificate was revoked. eMudhra put its intermediates on its root's CRL within a day, and Sectigo did the same for the cross-signs. A client that reads those CRLs would have stopped trusting them without anyone at Sparrow doing anything.

We did the third one first, because it could ship the same day, and then the fourth, because it is the one that fixes the cause.

## Fix one: refuse the keys

When a delivery client first needs it, Sparrow builds the 255 public points *k·G* for *k* from 1 to 255 on P-256, P-384 and P-521. After Go's normal chain verification, a `VerifyConnection` hook looks at every certificate in every verified chain and refuses the connection if any EC key is in that table:

```go
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
```

A few details that came out of review:

- **It matches the x-coordinate only.** For any key *k*, the key *n − k* (where *n* is the curve's order) has the same x-coordinate and the opposite y. It is just as trivial to recover. Matching on x catches both, with no extra entries.
- **It covers the cross-signs.** The check is on the key, not the certificate, so Sectigo's copies are caught as well as eMudhra's.
- **It runs on every server-side client that verifies TLS**: webhook deliveries, the fetcher that reads receiver documentation for AI drafting, and the OpenAI-compatible AI provider. Webhooks configured with `verify_ssl=false` skip verification entirely, as before.
- **A refused delivery fails as `tls_error`**, which is not retried, and the error names the certificate. That is a receiver-side problem, and it is recorded as one.

The tests run real TLS handshakes through a root, an intermediate and a leaf. A normal intermediate connects. Intermediates with private keys 1, 127 and 255 are refused.

## Can it reject a legitimate key?

No, and the reasoning is short.

A correctly generated P-384 key is uniform over about 2^384 values. The chance it lands below 256 is about 2^-376 (2^-248 on P-256). For comparison, guessing a random 128-bit key on the first try is 2^-128, and that is already treated as impossible.

And if a key ever did match, rejecting it would still be correct. Each private value gives exactly one public point, so a public key equal to *4·G* has private key 4, whatever its owner intended. Anyone can sign with it.

## What it does not do

This is a targeted check, not a weak-key detector. A private key of 70,000 is nearly as easy to recover and passes. Covering wider ranges would need a different technique, not a bigger table: building one with 65,536 entries per curve would cost tens of seconds of CPU on the first connection. The advisory says below 256, so that is where the bound sits.

And it doesn't make Go check revocation. The next incident will probably look different. That is what the second fix is for.

## Fix two: check revocation

Sparrow now checks every intermediate in a receiver's chain against the CRL its issuer publishes. The design came down to four decisions.

**Intermediates only, not leaves.** OCSP is on its way out: Let's Encrypt shut its responders down in 2025, and the industry rules made OCSP optional. Leaf CRLs can be large, sharded and change daily; browsers moved to compressed sets like CRLite because fetching them per connection does not scale. The CRLs that list revoked intermediates are small and change rarely, and an intermediate is where a CA key lives. That is the check that would have caught this incident.

**No schedule, no table.** Our first design had a periodic job and a Postgres table so every instance shared one cache. Both turned out to be unnecessary. CRLs are public and re-downloadable, so each instance keeps its own in memory, and a refresh is driven by use instead of a clock:

- The first delivery whose chain needs a CRL downloads it, and waits for it, up to five seconds.
- Once a cached CRL is older than `SPARROW_CRL_MAX_AGE` (six hours by default), the next delivery that uses it refreshes it in the background and carries on with the cached copy.
- A CRL past its own `nextUpdate` is downloaded again before it is used.
- `POST /v1/tls/crls:refresh` asks every instance to refresh now. It sends a Postgres `NOTIFY`, and each instance listens on the channel, so there is still nothing but Postgres.

An idle instance downloads nothing, and a busy one is never more than six hours behind, or seconds behind when someone calls the endpoint. Restarting an instance just empties its cache, and the next delivery fills it again.

**CRL addresses are attacker input.** The address of a CRL comes from a certificate the remote server sends. Someone holding a stolen CA key can mint a sub-CA whose CRL address points at an internal service or a cloud metadata endpoint, and a naive checker would fetch it from inside your network. Two things stop that. Downloads go through the same network policy as deliveries, so private and metadata addresses are refused when the connection is made. And the chain is checked from the root down, stopping at the first revoked intermediate:

```
attacker sub-CA  ->  EM OV TLS CA - G3C  ->  emSign root
   (address the        (address eMudhra        (trust anchor)
   attacker chose)      wrote; checked first)
```

The eMudhra intermediate's CRL address was written by eMudhra and signed by its root; an attacker cannot change it without the root key. That CRL lists the intermediate as revoked, the connection is refused there, and the attacker's address is never requested. Certificate pinning, which we were asked about, does not help here: it constrains which certificates a server may present, not which addresses we fetch, and CRLs are served over plain HTTP by design, since they are signed.

**Soft by default.** If a CRL cannot be downloaded or verified, `SPARROW_TLS_REVOCATION=soft` allows the connection and records the error in `GET /v1/tls/crls`; `hard` refuses it. An attacker who can intercept traffic can also block a CRL download, so `hard` is safer, but it turns every CA outage into failed deliveries, and most Sparrow installs run behind a VPN with no route to CRL servers at all. A certificate listed on a CRL is refused in both modes, even if the CRL has expired: a revocation is never undone.

The check lives in the same stdlib-only module, `safetransport.NewRevocationChecker`, so other Go programs that make outbound TLS calls can use it with its built-in in-memory cache.

## What we'd take from it

- **Key strength is decided at generation.** AES-256 or P-384 with a one-byte secret is a one-byte secret. Debian's OpenSSL bug in 2008 and ROCA in 2017 were the same failure.
- **Bind identity to the key.** Compare fingerprints at every hand-off, never labels.
- **Revocation only works for clients that check it.** If your software talks TLS from a server, assume the browser ecosystem's fixes won't reach you. Checking intermediate CRLs is cheap, and it is what turns a CA's "we revoked it" into protection for you.

Both checks ship in the next release; the [changelog](https://github.com/sarathsp06/sparrow/blob/main/CHANGELOG.md) has the details and the [security reference](/sparrow/reference/security/#certificate-revocation) describes the settings. If you run Sparrow, nothing needs configuring, but CRL checks need outbound access to the CAs' CRL servers; without it, `soft` mode lets deliveries through and `GET /v1/tls/crls` shows what failed. If a receiver's chain goes through a revoked intermediate, including these keys, its deliveries will start failing with `tls_error`, and the receiver needs a new certificate.
