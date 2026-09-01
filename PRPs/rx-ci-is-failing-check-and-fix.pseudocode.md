# Pseudocode Design — `rx-ci-is-failing-check-and-fix`

**Ticket (ad-hoc):** "ci is failing check and fix"
**Scope (approved by user):** timeout bump only — no test-code refactor.

## Goal

Make the **Integration Test** CI job green again. It is the only failing job on
`main` (Lint / Test / Build / Deploy Docs are green).

## Root cause (grounded in evidence)

- `.github/workflows/ci.yml:147`
  `go test -v -tags integration -timeout 120s -race ./internal/integration/...`
- The suite has ~14 `TestE2E_*` tests. Each calls `setupEnv(t)`
  (`internal/integration/testhelpers_test.go:89`) → `setupTestDB`
  (`:59`) which starts its **own** `postgres:16-alpine` testcontainer, runs all
  migrations, and boots the full River queue stack.
- Several tests exercise real retry / backoff / timeout paths (e.g.
  `TestE2E_TimeoutRetry`, `TestE2E_PauseWebhookStopsRetries`) with wall-clock waits.
- Under `-race`, cumulative runtime exceeds the 120s budget. CI log (run
  33432688725): first test `=== RUN` at 19:53:08, `panic: test timed out after
  2m0s` at 19:55:08 while `TestE2E_HappyPath` (14th) was still starting its
  container. Binary SIGABRT'd → job exit 1.
- Conclusion: the suite legitimately needs **> 120s**; the timeout is the
  limiter, not a hang.

## The change

1. FILE `.github/workflows/ci.yml`, step "Run integration tests" (line 147)
   - REPLACE `-timeout 120s` WITH `-timeout 600s`
   - keep everything else identical (`-v -tags integration -race
     ./internal/integration/...`)
   - WHY 600s: observed need ~130-150s of test runtime; 600s gives ~4x headroom
     for CI-runner variance without masking a genuine hang (the job still sits
     well under GitHub's implicit job ceiling). Human may tune this at the gate.

## What is explicitly NOT changed

- No change to `internal/integration/*_test.go` (container-per-test stays — the
  shared-container refactor was declined this round).
- No change to the govulncheck step (already `continue-on-error: true`,
  `ci.yml:40`) — its warnings are non-failing.
- No change for the Node 20 deprecation annotations — they are warnings, not
  failures.
- Deploy Docs workflow untouched (already green).

## Verification (see test plan)

- Local: `go test -tags integration -timeout 600s -race
  ./internal/integration/...` (needs Docker) → expect PASS; capture wall time to
  confirm headroom.
- Authoritative: the PR's own **Integration Test** job turns green.

## Open questions

- None. (Timeout value 600s is a proposed default; adjustable at the gate.)
