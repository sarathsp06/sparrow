# 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph

- Status: accepted
- Date: 2026-09-13

## Context

The `sparrow` CLI is published for end users via
`go install github.com/sarathsp06/sparrow/satellites/sparrow@latest`. It lived
in the repository's single root module, so its `go install` supply chain and
download graph inherited the **entire server** dependency tree: aws-sdk-go-v2,
pgx, River, Huma, testcontainers, OpenTelemetry, and more.

Measured with `go list -m all`, the CLI's module graph was **134 modules** —
almost none of which the CLI compiles. `go build` prunes the *binary*, but
the *install-time* module graph (what `go install` resolves, and what a
supply-chain / CVE scanner and dependabot see) is not pruned. That graph is the
thing that matters for a widely-installed client.

The CLI only needs its own command deps (cobra, pflag, yaml) plus two internal
libraries it imports directly: `pkg/signature` (`listen.go`) and `pkg/template`
(`template_cmd.go`). Those two packages are clean leaves — `pkg/signature` is
stdlib-only, `pkg/template` depends only on `golang-lru/v2`.

## Decision

Carve the shared client-safe packages and the CLI into their own Go modules,
leaving the heavy server tree in the root module:

- `pkg/signature/go.mod` — stdlib only.
- `pkg/template/go.mod` — `golang-lru/v2` + `pkg/signature`.
- `satellites/recipes/go.mod` — yaml only (`pkg/template` for tests); the
  recipe schema plus the `go:embed`ed built-in catalog, so the CLI ships the
  recipes inside the binary (2026-09 amendment: extracted from the root module
  when the CLI grew the embedded catalog).
- `satellites/sparrow/go.mod` (the CLI) — cobra, pflag, yaml, and the three
  light modules above.
- `satellites/sparrow-sinks/go.mod` and `satellites/sparrow-sources/go.mod`
  (2026-10 amendment, see below) — each satellite is also a Go library.
- Root module (`github.com/sarathsp06/sparrow`) keeps the server and
  `require`s the light modules it uses.

Development is wired with a committed `go.work` (`use` all five modules) plus
`replace` directives in each consuming `go.mod` pointing at the local relative
paths. `replace` (not `go.work` alone) is deliberate: it keeps `go mod tidy`,
`go build`, `go test`, and GoReleaser working per-module with no workspace-only
special-casing — `go mod tidy` does not honour `go.work` replaces.

## Consequences

- **Isolated CLI graph: 134 → 11 modules** (`go list -m all`). aws-sdk, pgx,
  River, Huma, testcontainers, and OTel are gone from the CLI's install graph,
  SBOM, and CVE surface. This is the load-bearing win.
- **Dev is unchanged in practice:** `go.work` + `replace` make every local and
  CI command resolve the siblings from source. Verified green: `go mod tidy`,
  `go build ./...`, the full test suite across all modules, and
  `goreleaser build` for all four binaries.
- **`make test` / CI list the nested modules explicitly** — root `./...` only
  covers the root module. See `MODULE_TEST_PATHS` in the Makefile and the test
  step in `.github/workflows/ci.yml`.
- **The server Docker image sets `GOWORK=off`** and copies the two `pkg`
  manifests before `go mod download`; the server resolves them via root
  `replace`, and never pulls the CLI module.
- **Release binaries are unaffected:** GoReleaser builds every binary from the
  checkout via `replace`; module tags are irrelevant to the produced archives.

## Releasing the split modules (tag scheme)

Nested modules are versioned by **path-prefixed tags**, and `go install
...@version` refuses a module whose `go.mod` contains `replace` directives.
So the working-tree `replace` must be stripped and real versions pinned before
tagging. `scripts/release-submodules.sh vX.Y.Z --push` automates this end to
end and is wired into `.github/workflows/release.yml` as the
`release-submodules` job, running on every `v*` tag push alongside GoReleaser:

1. Tag the leaf `pkg/signature/vX.Y.Z` directly (stdlib-only, no `replace` to
   strip).
2. In a disposable detached worktree (never a branch merged to `main`), strip
   `pkg/template/go.mod`'s `replace` line, pin its `pkg/signature` `require`
   to the version from step 1, `go mod tidy`, commit, and tag
   `pkg/template/vX.Y.Z`.
3. Same recipe for `satellites/recipes/go.mod` (pinning `pkg/template`), then
   `satellites/sparrow/go.mod`, pinning `pkg/signature`, `pkg/template`, and
   `satellites/recipes`; tag `satellites/recipes/vX.Y.Z` and
   `satellites/sparrow/vX.Y.Z`.
4. The server and other root binaries release under the existing `vX.Y.Z` tag
   as before.

`main`'s `go.work` + `replace` setup is untouched by any of this — the
replace-stripped `go.mod`s only ever exist in the disposable worktrees behind
each nested tag. The script is idempotent (skips any module already tagged at
the requested version) and defaults to a dry run; `--push` is required to
actually tag and push.

The end-user command is unchanged:
`go install github.com/sarathsp06/sparrow/satellites/sparrow@latest`.

## When this is worth it

Only extract a module when a consumer is *distributed independently* and its
graph actually leaks (the CLI's `go install` path). Do **not** split for code
organisation alone — Go 1.17+ build pruning already keeps *binaries* lean, and
extra modules add a `go.work`/`replace`/ordered-tag release cost. `recipes` was
extracted only when the CLI (an independently-installed consumer) needed to
embed it; `sinks` and `sources` stayed in the root module until they became
libraries (amendment below).

## Amendment (2026-10): sinks and sources are libraries

Go teams should be able to import a satellite and get what its command does
— mounted on their own HTTP server, extended with their own sinks or
providers — while the binary keeps working unchanged. Importing from the root
module would drag the server's graph (River, pgx, Huma, OTel) into theirs,
which is this ADR's extraction trigger, so `satellites/sparrow-sinks` and
`satellites/sparrow-sources` are now their own modules.

Shape, the same for every satellite with a binary:

- The binary stays at `satellites/<name>` (`package main`), so GoReleaser
  `main:` paths, `go build ./satellites/<name>` and the install script are
  unchanged. The library is two layers: a core package (`sinks`, `sources`)
  with the shared interfaces and definitions, and one package per sink or
  source (`sinks/email`, `sinks/otlp`, `sinks/s3`; `sources/stripe`,
  `sources/github`, `sources/cron`). Each implementation keeps its own
  dependencies: importing one sink or writing a custom one never compiles
  aws-sdk-go-v2 (97 packages, ~0.9 MB of binary) unless you import
  `sinks/s3`, and the same holds for any heavy client a future source (Kafka)
  brings. The binaries compose the packages themselves.
- The library API mirrors the command: config types are the YAML sections
  (with yaml tags), and it reads no files, env vars or flags. Those, plus the
  listen address and the `http.Server`, stay in `main`.
- It exposes `http.Handler`s, not a server: one per sink/source and a core
  `Handle(...)` that the built-ins and custom ones share. Background work (cron) is a validating
  constructor plus `Run(ctx)`.
- Release: `release-submodules.sh` tags `satellites/sparrow-sinks/vX.Y.Z`
  (pinning `pkg/signature`) and `satellites/sparrow-sources/vX.Y.Z` (a leaf).
  The root module `require`s `sparrow-sources` via `replace` for its
  integration tests, so the Dockerfile copies that manifest too.

## Trigger to revisit

Collapse back to one module if the CLI ever needs the heavy server packages
directly (making isolation moot), or fold in another light module only when a
second independently-installed consumer needs the same client-safe code.
