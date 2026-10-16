# Contributing to Sparrow

Thanks for contributing to Sparrow.

## Development Setup

```bash
git clone https://github.com/sarathsp06/sparrow.git
cd sparrow
make build-with-ui
make test
make lint
```

## Requirements for Acceptable Contributions

A pull request is merged when it meets all of the following. CI checks the
first four on every pull request (`.github/workflows/ci.yml`).

1. **Coding standard.** Go code follows [Effective Go](https://go.dev/doc/effective_go)
   and the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments),
   plus the project conventions in [AGENTS.md](AGENTS.md#code-conventions)
   (error handling before the happy path, no SQL in handlers, tenant-scoped
   queries, `snake_case.go` files, verb-first `camelCase` OperationIDs).
   Frontend code under `web/` follows the existing Svelte 5 and TypeScript style.
2. **Formatted and lint-clean.** Run `make fmt` (`goimports` with local import
   grouping) and `make lint` ([golangci-lint](https://golangci-lint.run/) with the
   standard linters plus [gosec](https://github.com/securego/gosec), configured
   in `.golangci.yml`); both must pass with no new warnings. Silence a gosec
   finding only with `//nolint:gosec // reason` explaining why it is safe. `govulncheck` must report no reachable
   vulnerabilities.
3. **Tested.** New behavior comes with tests and bug fixes come with a test
   that fails without the fix. `make test` must pass; changes to storage or
   queue code should also pass `make test-integration`, and UI changes
   `make web-test`.
4. **Generated files are current.** After changing a REST handler, run
   `make generate` and commit the updated `api/openapi.{yaml,json}`
   (`internal/rest/openapi_drift_test.go` fails otherwise). Never hand-edit
   generated files such as `*_otel.go`.
5. **Documented.** User-visible changes update the docs in `docs/` and, for API
   fields, the handler descriptions (see [Documenting the API](#documenting-the-api)).
6. **Changelog entry.** User-visible changes add a line to `CHANGELOG.md` under
   `## [Unreleased]`, written for users; anything they must do when upgrading
   goes under `### Upgrading`.
7. **Conventional Commits.** Commit subjects follow the [Commit Style](#commit-style)
   below; breaking changes are marked with `!` and explained in the body.
8. **Fits the design.** Changes respect the design principles in
   [AGENTS.md](AGENTS.md#design-principles), for example Postgres-only (no new
   external services such as Redis without discussion). Open an issue first for
   large or architectural changes.
9. **Licensed.** By contributing you agree that your contribution is licensed
   under the project's [MIT License](LICENSE).

Security issues are reported privately as described in [SECURITY.md](SECURITY.md),
not through public issues or pull requests.

## Documenting the API

The REST API reference is generated from the handler structs in `internal/rest`.
See [docs/api-documentation.md](docs/api-documentation.md) before changing a
handler's fields or descriptions, and run `make generate` afterwards.

## Commit Style

Use Conventional Commits when possible:

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation-only changes
- `refactor:` for code restructuring without behavior changes
- `perf:` for performance improvements
- `chore:` for maintenance tasks

This keeps the history easy to scan and marks breaking changes.

## Release Process

A release is a `CHANGELOG.md` update and one tag push from an up-to-date `main`; `.github/workflows/release.yml` does the rest:

```bash
git tag v1.2.3
git push origin v1.2.3
```

The workflow runs the full CI suite first. Only when it passes does GoReleaser build the server, CLI, `sparrow-sinks` and `sparrow-sources` binaries, attach an SBOM per archive and a cosign-signed `checksums.txt`, and publish the GitHub release. After that, the split Go modules (`pkg/*`, `satellites/*`) get their path-prefixed tags (`pkg/signature/v1.2.3`, ...) via `scripts/release-submodules.sh`, and the Docker image is pushed to GHCR with provenance and an SBOM.

- Push only the tag. Don't push `main --tags`, and don't run `scripts/release-submodules.sh --push` yourself: the workflow creates those tags, and the Go module proxy caches a tag forever, so a bad one can't be taken back.
- **Release candidates**: tag `v1.2.3-rc.1`. GoReleaser marks it as a pre-release, Docker gets `1.2.3-rc.1` but not `latest`, and the split modules are tagged `-rc.1` too (Go never resolves `@latest` to a pre-release).
- **Never move or delete a pushed tag.** If a release is broken, fix forward with the next patch version.
- Check `goreleaser check` and `make release-dry-run` (a `--snapshot` build, nothing published) before tagging when the release config changed.
- Before tagging, rename `## [Unreleased]` in `CHANGELOG.md` to `## [v1.2.3] - YYYY-MM-DD`, add a new empty `Unreleased` section and the compare link at the bottom. That section becomes the GitHub release note; the workflow fails if it is missing. Mark breaking changes in commits with `!` (`feat!:`) too. See the [compatibility policy](https://sarathsp06.github.io/sparrow/reference/compatibility/) for what counts as breaking.

## Notes

- Avoid committing generated files unless the change intentionally updates generated output.
