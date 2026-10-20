// Package receivercode generates a ready-to-run receiver project for a
// Sparrow webhook: an HTTP server that verifies Sparrow's signatures on the
// raw body, rejects stale timestamps, drops redeliveries, routes on the
// X-Sparrow-Subscription-ID and X-Sparrow-Event-Type headers and decodes each
// subscription's payload into an inferred model, with tests that replay
// signed vectors.
//
// The package is EXPERIMENTAL: it is not covered by Sparrow's 1.x
// compatibility promise, and both its API and the code it generates may
// change in any release. The Sparrow endpoint built on it
// (generateReceiverCode) is covered.
//
// Generate takes plain data (Input) and is deterministic: the same Input
// always gives byte-identical files. Verification code is never generated:
// each project carries a byte-identical copy of a tested helper
// (pkg/signature for Go, client/verify for Python and TypeScript). Secrets are
// read from SPARROW_WEBHOOK_SECRET at run time and never written into the
// output; tests use throwaway credentials derived from the webhook ID.
//
// Supported targets are listed by Targets: go-nethttp, python-fastapi and
// node-express (TypeScript run by Node's built-in type stripping).
package receivercode
