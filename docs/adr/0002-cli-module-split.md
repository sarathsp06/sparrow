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
- `satellites/sparrow/go.mod` (the CLI) — cobra, pflag, yaml, and the two
  `pkg` modules above.
- Root module (`github.com/sarathsp06/sparrow`) keeps the server, `recipes`,
  `sparrow-sinks`, and `sparrow-sources`, and `require`s the two light modules.

Development is wired with a committed `go.work` (`use` all four modules) plus
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
tagging. Release order:

1. Tag the leaves: `pkg/signature/vX.Y.Z`, then `pkg/template/vX.Y.Z`
   (template requires signature).
2. In `satellites/sparrow/go.mod` (and `pkg/template/go.mod`), remove the
   `replace` lines and set the `require`s to the real `pkg/*` versions from
   step 1.
3. Tag the CLI: `satellites/sparrow/vX.Y.Z`.
4. The server and other root binaries release under the existing `vX.Y.Z` tag.

The end-user command is unchanged:
`go install github.com/sarathsp06/sparrow/satellites/sparrow@latest`.

## When this is worth it

Only extract a module when a consumer is *distributed independently* and its
graph actually leaks (the CLI's `go install` path). Do **not** split for code
organisation alone — Go 1.17+ build pruning already keeps *binaries* lean, and
extra modules add a `go.work`/`replace`/ordered-tag release cost. `recipes`,
`sinks`, and `sources` stay in the root module for exactly this reason.

## Trigger to revisit

Collapse back to one module if the CLI ever needs the heavy server packages
directly (making isolation moot), or fold in another light module only when a
second independently-installed consumer needs the same client-safe code.
