---
type: DevOps Config
title: CI/CD and Release
description: GoReleaser-based release automation with conventional commits, cross-platform builds, and Helm chart artifact
tags: [ci-cd, release, goreleaser]
timestamp: 2026-06-22T00:00:00Z
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
| `generate` | buf generate + go generate (protobuf, clients, gowrap) |
| `migrate` | Run DB migrations |
| `release-dry-run` | Test GoReleaser locally |
| `docker-dev` | Docker compose dev environment |

## GoReleaser (.goreleaser.yml)

- **Builds**: Single `sparrow` binary from `./cmd/server`; CGO_ENABLED=0
- **Platforms**: linux/darwin/windows × amd64/arm64 (no windows/arm64)
- **Archives**: `sparrow-{{ .Version }}-{{ .Os }}-{{ .Arch }}` with LICENSE + README
- **Release Notes**: Generated from conventional commits (feat→Added, fix→Fixed, etc.)
- **Changelog**: Hand-curated `CHANGELOG.md`
- **Artifacts**: Attaches Helm chart `.tgz`

## Release Workflow

```bash
git tag v1.x.x
git push origin main --tags
```

CI builds the UI, packages the Helm chart, runs GoReleaser which cross-compiles, creates release notes, and publishes GitHub release + artifacts.

## Citations

- `Makefile`
- `.goreleaser.yml`
