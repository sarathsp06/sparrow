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

Releases are triggered by pushing a semantic version tag. CI builds frontend + backend artifacts and publishes via GoReleaser.

```bash
# create a release tag
git tag vX.Y.Z

# push main and tags
git push origin main --tags
```

If the tag already exists remotely, bump to the next version and push that tag.

## Notes

- Avoid committing generated files unless the change intentionally updates generated output.
