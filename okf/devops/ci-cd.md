---
type: DevOps Config
title: CI/CD and Release
description: GitHub Actions CI and the single-tag-push release (GoReleaser, split-module tags, Docker image)
tags: [ci-cd, release, goreleaser, github-actions]
timestamp: 2026-10-13T00:00:00Z
---

# CI/CD and Release

`AGENTS.md`'s Release section is the source of truth; this page summarizes it.

## Makefile Targets

| Target | Description |
|--------|-------------|
| `build` | Build server binary |
| `build-ui` | Build SvelteKit frontend |
| `build-with-ui` | Build server + embedded UI |
| `test` | Root module plus the nested workspace modules |
| `test-integration` | Integration tests (Docker required) |
| `lint` | golangci-lint |
| `generate` | Export the OpenAPI spec from Go (`cmd/openapi-export`), then `go generate`; no client is generated (users generate theirs from the spec) (gowrap OTel wrappers, pinned as a `tool` in `go.mod`) |
| `migrate` | Run DB migrations |
| `release-dry-run` | Test GoReleaser locally |
| `docker-dev` | Docker compose dev environment |

## CI (`.github/workflows/ci.yml`)

Runs on pushes and pull requests to `main`, with a read-only `GITHUB_TOKEN`.

- **Lint**: golangci-lint on the root module, `sparrow-sinks` and `sparrow-sources`; govulncheck on each module.
- **Test**: `go test -race` across the workspace modules against a Postgres service.
- **UI E2E** (after Test): frontend unit tests and `make test-ui`.
- **Integration Test** (after Test): `-tags integration` tests with testcontainers.

Third-party actions are pinned to commit SHAs; golangci-lint and govulncheck are pinned to versions.

## Release (`.github/workflows/release.yml`)

A release is one tag push:

```bash
git tag vX.Y.Z && git push origin vX.Y.Z
```

The workflow runs three jobs:

1. **GoReleaser**: builds the UI, then the server, CLI, `sparrow-sources` and `sparrow-sinks` binaries for linux/darwin/windows × amd64/arm64 (no windows/arm64), and publishes the GitHub release with `checksums.txt`. Archives are `sparrow[-cli|-sources|-sinks]-<version>-<os>-<arch>` (zip on Windows). Release notes come from Conventional Commits via the `changelog:` block.
2. **Release Split Modules**: `scripts/release-submodules.sh vX.Y.Z --push` tags the separate modules (`pkg/*`, `satellites/*`) under path-prefixed tags.
3. **Docker Image** (after GoReleaser): multi-arch image pushed to `ghcr.io/<repo>` tagged `X.Y.Z`, `X.Y` and `latest`.

There is no second manual step.

## Citations

- `AGENTS.md` (Release)
- `Makefile`
- `.goreleaser.yml`
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
