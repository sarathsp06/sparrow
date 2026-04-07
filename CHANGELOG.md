# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
## [0.11.3] - 2026-04-07

### Added

- **docs**: Theme-aware diagrams — separate light and dark SVGs that switch automatically with Starlight's theme toggle
- `ThemeDiagram.astro` component for zero-JS theme-based image switching via CSS `[data-theme]`
- Dark-mode mermaid config with slate-based palette (dark fills, light text, muted borders)

### Changed

- Diagram build script now generates both `-light.svg` and `-dark.svg` variants per `.mmd` source file
- Architecture docs page converted to `.mdx` to support Astro component imports

## [0.11.2] - 2026-04-07

### Fixed

- **ci**: fix data race in template timeout path (buffer returned to pool while goroutine still writing)
- **ci**: add `continue-on-error` for govulncheck (upstream panic with Go 1.25 + x/tools)
- **ci**: skip Puppeteer/Chromium diagram regeneration when pre-rendered SVGs already exist (fixes Ubuntu 24.04+ AppArmor sandbox failure)
- **ci**: add missing UI dist placeholder to test job (fixes `[setup failed]` for `cmd/server` and `internal/ui` tests)

### Documentation

- Fix Python HMAC verification example indentation in architecture docs
- Remove incorrect "OTel Collector sidecar" claim from docker-compose docs

## [0.11.1] - 2026-04-07

### Added

- Configurable gRPC and HTTP ports via `SPARROW_GRPC_PORT` and `SPARROW_HTTP_PORT` env vars
- `SPARROW_ALLOW_PRIVATE_NETWORKS` env var to allow localhost/private IPs as webhook URLs in dev/testing
- `ServiceError` type for structured gRPC error propagation with client-safe messages
- `formatAPIError()` frontend utility to strip gRPC code prefixes and avoid double-prefixed error messages
- `CopyableId` component with hover clipboard icon for click-to-copy IDs across all list views

### Changed

- Health badge shows em dash instead of "Unknown" for webhooks with no deliveries yet, with tooltip
- Health dashboard "Unknown" summary card renamed to "No Data"

### Fixed

- **security**: harden template execution (CPU timeout), add security headers middleware, restrict CORS in production
- Error messages from URL validation, encryption, event state, and batch operations now propagate to the client instead of being swallowed into generic "internal" errors

## [0.11.0] - 2026-04-07

### Added

- **health**: add unexpected_status error category to health metrics (full stack: migration, store, service, proto, gRPC, UI)

### Changed

- **docs**: replace client-side mermaid with pre-rendered SVGs (zero client JS, ~2MB lighter)
- Add Makefile `diagrams` target with Make pattern rule for incremental re-rendering
- Wire diagram re-rendering into `npm run dev` and `npm run build` in docs
- Clean up .gitignore, remove obsolete pre-commit hook for .gitkeep

### Fixed

- **delivery**: only require namespace for webhook-level retry (fixes /deliveries page retry button)
- **docs**: use default (light) mermaid theme
- **docs**: use rehype-mermaid-lite to fix mermaid diagram rendering

## [0.10.0] - 2026-04-06

### Added

- **ui**: add unexpected_status error category, deduplicate badge utilities
### Changed

- remove redundant root docker-compose.yml
- regenerate proto code with updated error_category docs
### Documentation

- add client-side mermaid rendering, remove volatile data model section
### Fixed

- add unexpected_status error category for status code mismatches## [0.9.3] - 2026-04-06

### Added

- add RePushEvent RPC for single event replay
### Fixed

- revert Railway to Dockerfile-based builds## [0.9.2] - 2026-04-06

### Fixed

- **ci**: restore clean git state before GoReleaser## [0.9.1] - 2026-04-06

### Changed

- add pre-commit hook to restore internal/ui/dist/.gitkeep## [0.9.0] - 2026-04-06

### Added

- **ui**: search filters, batch re-push, and batch retry for web UI
- lead with full flow diagram in docs and webhooks empty state
### Changed

- migrate release pipeline to GoReleaser## [0.8.1] - 2026-04-06

### Fixed

- use pre-built GHCR image for Railway deployment instead of Dockerfile build## [0.8.0] - 2026-04-06

### Added

- search filters, deterministic batch re-push and retry
- soft schema validation, template fallback, and implementation plan## [0.7.1] - 2026-04-06

### Fixed

- rewrite webhooks empty state with correct getting-started flow## [0.7.0] - 2026-04-05

### Added

- add getting-started empty state for webhooks, remove unused Inter font and otel-collector config## [0.6.0] - 2026-04-05

### Fixed

- replace stdlib mux with chi router to fix Connect-RPC routing bug## [0.5.4] - 2026-04-05

### Changed

- split API and UI into separate muxes for cleaner auth boundary## [0.5.3] - 2026-04-05

### Fixed

- allow UI routes through API key auth when embedded UI is served## [0.5.2] - 2026-04-05

### Changed

- migrate docs to proto2astro v0.4.1 JSON-based config pattern
### Fixed

- enable encryption in integration tests and add dev docker-compose
- remove Guides from sidebar and enforce correct section order
- restore missing sidebar sections in docs (Getting Started, Deployment, Reference)## [0.5.1] - 2026-04-05

### Documentation

- add Railway one-click deploy button to README, docs, and landing page## [0.5.0] - 2026-04-05

### Added

- envelope encryption for webhook secrets and sensitive headers
- add Railway deployment template and docs
### Documentation

- consolidate root markdown files into docs site
- add proto2astro attribution with repo link in README and comment guide
- fix inaccuracies in architecture, config, and technical docs
### Fixed

- remove broken Railway one-click deploy button## [0.4.0] - 2026-04-05

### Added

- migrate docs generation from custom gendocs to proto2astro## [0.3.2] - 2026-04-03

### Changed

- exclude SESSION_NOTES.md from version control
### Fixed

- multi-arch Docker build for Apple Silicon support## [0.3.1] - 2026-04-03

### Added

- unify Fira Code as default font across web app, docs, and landing page
### Documentation

- refactor README to follow open-source conventions## [0.3.0] - 2026-04-03

### Added

- standalone docker-compose, docs updates, and helm chart fix## [0.2.0] - 2026-04-03

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
- fix lint
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
- improve max attempt
- include request body in webhook delivery table and responses
- implement resend delivery
- report db metrics
- working with sqlx
- fix lint errors
- fix build errors
- temporary
- use custom sparrow
- added new favicon
- loading spin
- update README
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
- improve function signatures
- otel logs
- add UI
- add UI
- add more statistics managements
- add more functioins
- improve README
- more core functionalities
- more trace for grpc and connectrpc added
- basic working code
- remove unwanted sample code
- grpc service
### Fix

- Correct function calls in gRPC and Connect servers (#2)
### Fixed

- add table svelte component
- add meatures
- add meatures
- more telemetry
