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

This keeps release notes grouped and readable.

## Release Process

A release is one tag push from an up-to-date `main`; `.github/workflows/release.yml` does the rest:

```bash
git tag v1.2.3
git push origin v1.2.3
```

The workflow runs the full CI suite first. Only when it passes does GoReleaser build the server, CLI, `sparrow-sinks` and `sparrow-sources` binaries, attach an SBOM per archive and a cosign-signed `checksums.txt`, and publish the GitHub release. After that, the split Go modules (`pkg/*`, `satellites/*`) get their path-prefixed tags (`pkg/signature/v1.2.3`, ...) via `scripts/release-submodules.sh`, and the Docker image is pushed to GHCR with provenance and an SBOM.

- Push only the tag. Don't push `main --tags`, and don't run `scripts/release-submodules.sh --push` yourself: the workflow creates those tags, and the Go module proxy caches a tag forever, so a bad one can't be taken back.
- **Release candidates**: tag `v1.2.3-rc.1`. GoReleaser marks it as a pre-release, Docker gets `1.2.3-rc.1` but not `latest`, and the split modules are tagged `-rc.1` too (Go never resolves `@latest` to a pre-release).
- **Never move or delete a pushed tag.** If a release is broken, fix forward with the next patch version.
- Check `goreleaser check` and `make release-dry-run` (a `--snapshot` build, nothing published) before tagging when the release config changed.
- Release notes come from Conventional Commit subjects, so write them for users. Mark breaking changes with `!` (`feat!:`) and say what to do in the body. See the [compatibility policy](https://sarathsp06.github.io/sparrow/reference/compatibility/) for what counts as breaking.

## Notes

- Avoid committing generated files unless the change intentionally updates generated output.
