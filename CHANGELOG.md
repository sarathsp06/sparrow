# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.1] - 2026-04-03

### Changed

- Unified Fira Code as the default font across web app, docs site, and landing page
- Docs site (Starlight) now loads Google Fonts and uses Fira Code for all body text
- Landing page uses Fira Code as default body font, Inter for descriptive paragraphs
- Web app Tailwind theme overrides `--font-mono` to use Fira Code explicitly
- Refactored README to follow open-source conventions

## [0.3.0] - 2026-04-03

### Added

- standalone docker-compose, docs updates, and helm chart fix## [0.2.0] - 2026-04-03
- improved README

### Added

- **docs**: add proto-to-docs generator, hostname switcher, and build footer
- **web**: allow choosing and persisting namespace in UI (#11)
- move marketing landing page from web app to docs site
- add release workflow with git-cliff and changelog generation
- add optional API key authentication via SPARROW_API_KEY
- inject clean semver into web UI, docs site, and Docker builds
- add Helm chart, k8s Makefile targets, and update CI/CD for releases
- improve dependencoies
- add event subscription template dry-run API and UI (#10)
- use uuids
- add support for templates
### Changed

- **docs**: remove getting started section from landing page
- updat make file
- added back .gitkeep
- add docs
- fix tests
- fix test adding allow loopback
- use MIT License
- update ci-cd and cleanup repo tree
- improve docs
- update removing auth
- security fixes
- fix lint
- update web and README
- security improvements
- improved tenantisation and UI
- update tenats and license
- add clerk support
- improve jsonschema
- update README
- update README
- improve error categorization
- refactor and improve UX
- Refactor Webhook Service based on Protobuf Evaluation (#8)
- Optimize database interactions and fix performance bottlenecks (#7)
- fix lint
- Propagate protobuf changes for Timestamp and Struct types (#5)
- use TimeStamp
- fix lint
- improe README
- improve telementry generator
- improve README
- update package.json
- use latest githubcilint
- add github ci
- lint fix
- improve benchmarking
- improve benchmarking
- improve benchmarking
- update templates
- generate sample data for event schema
- remove handle registration event
- add simplified version of json schema meta schema for web
- improve readme
- improve signature
- cleanup migrations
- move webhook client to a package:
- fix max attempt retries
### Documentation

- add Kubernetes deployment page to docs site, slim down KUBERNETES.md
- move k8s details to KUBERNETES.md, slim down README
- add documentation site link to README
- add web UI quick start section and project badges
- add Protobuf evaluation and recommendations (#6)
### Fixed

- **ci**: fix lint issues and CI workflow for reliable builds
- **ci**: add .gitkeep to dist dir for go:embed
- **docs**: fix broken links and sync landing page with indigo theme
- **docs**: use ghcr.io image references in docs and config
- **docs**: point docs links to GitHub Pages, remove redundant overlay sections
- **docs**: remove stale references, add How It Works page, slim overlay, protocol toggle
- **docs**: rename installation.md to .mdx and remove Guides section
- **helm**: add PDB, PG security context, and initContainer resource limits
- **otel**: make OTLP export opt-in via OTEL_EXPORTER_OTLP_ENDPOINT
- **web**: remove static PUBLIC_API_KEY import that breaks SvelteKit build
- use changelog content as release tag message and curate CHANGELOG.md
- resolve integration test failures (SSRF config + webhook ID mismatch)
- add .gitkeep to dist dir for go:embed## [0.1.0] - 2025-11-14

### Added

- Add extensive test coverage to the webhook service (#4)
- Enhance SvelteKit frontend application (#3)
### Changed

- add api-key for completelness
- Update README.md
- improve max attempt
- include request body in webhook delivery table and responses
- implement resend delivery
- report db metrics
- working with sqlx
- Update favicon.svg
- Update favicon.svg
- Update favicon.svg
- fix lint errors
- fix build errors
- temporary
- Update README.md
- use custom sparrow
- added new favicon
- loading spin
- update README
- Update README.md
- update make
- improve makefile
- improved readme
- improved readme
- cleanup
- more ui improvements
- improve UI
- improve documentation
- fix code
- more test
- add JSON validation
- more UI
- improve web
- use MIT license
- some UI fun
- Correct function calls in gRPC and Connect servers (#2)
- improve function signatures
- otel logs
- Add event schema and validation
- Add event schema and validation
- UI (#1)
- add UI
- add UI
- add more statistics managements
- add more functioins
- improve README
- more core functionalities
- more trace for grpc and connectrpc added
- basic working code
- add connectrpx
- use buf
- remove unwanted sample code
- Working with gRPC commands
- grpc service
### Fixed

- add table svelte component
- add meatures
- add meatures
- more telemetry
