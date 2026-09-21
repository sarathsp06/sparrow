---
type: DevOps Config
title: CI/CD and Release
description: GoReleaser-based release automation with conventional commits and cross-platform builds
tags: [ci-cd, release, goreleaser]
timestamp: 2026-08-29T00:00:00Z
---

# CI/CD and Release

## Makefile Targets

| Target | Description |
|--------|-------------|
| `build` | Build server binary |
| `build-ui` | Build SvelteKit frontend |
| `build-with-ui` | Build server + embedded UI |
| `test` | `go test -v ./...` |
| `test-integration` | Integration tests (Docker required) |
| `lint` | golangci-lint |
| `generate` | Export the OpenAPI spec from Go (`cmd/openapi-export`), regenerate the Python client, then `go generate` (gowrap OTel wrappers) |
| `migrate` | Run DB migrations |
| `release-dry-run` | Test GoReleaser locally |
| `docker-dev` | Docker compose dev environment |

## GoReleaser (.goreleaser.yml)

- **Builds**: Single `sparrow` binary from `./cmd/server`; CGO_ENABLED=0
- **Platforms**: linux/darwin/windows × amd64/arm64 (no windows/arm64)
- **Archives**: `sparrow-{{ .Version }}-{{ .Os }}-{{ .Arch }}` with LICENSE + README
- **Release Notes**: Generated from conventional commits (feat→Added, fix→Fixed, etc.) via `.goreleaser.yml`'s `changelog:` block — no separate CHANGELOG.md file to maintain

## Release Workflow

```bash
git tag v1.x.x
git push origin main --tags
```

CI runs GoReleaser which cross-compiles, generates release notes from commit history, and publishes GitHub release + artifacts.

## Citations

- `Makefile`
- `.goreleaser.yml`
