# Sparrow Web Dashboard

SvelteKit web dashboard for the Sparrow webhook delivery system. Builds to static files and can be embedded into the Go server binary or deployed independently.

## Prerequisites

- Node.js 18+
- npm

## Local Development

```bash
# Install dependencies
npm install

# Start dev server (default: http://localhost:5173)
npm run dev
```

The dev server connects to the Sparrow Go backend API. Make sure the backend is running (`make run` from the project root).

## Build

```bash
npm run build
```

Build output goes to `../internal/ui/dist/` (the Go embed directory). The static adapter produces a fully self-contained SPA with `index.html` as the fallback for client-side routing.

## Embedding in the Go Binary

1. Build the frontend: `npm run build` (or `make build-ui` from the project root)
2. Build the server: `go build ./cmd/server` (or `make build-with-ui` for both steps)
3. Run with `SPARROW_SERVE_UI=true` -- the UI is served at `http://localhost:8080/`

## Standalone Deployment

The dashboard is a static SPA and can be served independently from any web server. Point `PUBLIC_API_URL` at your running Sparrow backend.

```bash
cd web
npm install
PUBLIC_API_URL=https://sparrow.example.com npm run build
```

The build output in `../internal/ui/dist/` contains only static files (`index.html`, JS, CSS, assets). Serve them with any static file server. Configure your server to fall back to `index.html` for all routes (SPA routing).

When the dashboard runs on a different origin than the Sparrow API, set `CORS_ALLOWED_ORIGINS` on the Sparrow server:

```bash
CORS_ALLOWED_ORIGINS=https://dashboard.example.com ./server
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PUBLIC_API_URL` | `http://localhost:8080` | Sparrow backend REST API URL |

Set this in `web/.env` for development. The default `npm run build` script sets `PUBLIC_API_URL=/` for embedded mode (same-origin). For standalone deployment, override it at build time to point at your Sparrow API server.

## Tech Stack

- **SvelteKit 2** + **Svelte 5** (Runes)
- **Vite 7**
- **Tailwind CSS 4** (via Vite plugin)
- **openapi-fetch** (typed REST client, generated from `api/openapi.yaml`)
- **Static adapter** (embedded in Go binary via `go:embed`)
