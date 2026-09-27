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

The dev server connects to the Sparrow Go backend at `http://localhost:8080` (override with `PUBLIC_API_URL` in `web/.env`). Make sure the backend is running (`make run` from the project root). That's a cross-origin setup: it works because the server allows any origin unless `ENVIRONMENT=production`. In production mode, also set `CORS_ALLOWED_ORIGINS=http://localhost:5173`.

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

The dashboard is a static SPA and can be served by any web server, on a different origin from the Sparrow API. Full guide: [Hosting the UI separately](https://sarathsp06.github.io/sparrow/deployment/separate-ui/).

In short:

1. `npm run build`, then upload `../internal/ui/dist/` to the static host. Configure it to fall back to `index.html` for unknown paths, and serve it at the origin root (not a sub-path).
2. Edit `config.js` on the static host (no rebuild needed):

   ```js
   window.__SPARROW_CONFIG__ = window.__SPARROW_CONFIG__ || {
     apiUrl: "https://sparrow-api.example.com",
   };
   ```

3. On the Sparrow server, allow the dashboard's origin:

   ```bash
   CORS_ALLOWED_ORIGINS=https://dashboard.example.com ./server
   ```

If the server has `SPARROW_API_KEY` set, the dashboard shows a **Sign in to Sparrow** prompt on the first `401`. You can paste the master key or an access token. A pasted master key is exchanged for a named browser token behind the scenes, so the master key is never stored. You can also send someone a one-time invite link (`sparrow invite alice --ui-url https://dashboard.example.com`) -- opening it redeems the invite and signs them in automatically. Setting `apiKey` in `config.js` skips the prompt, but then anyone who can load the dashboard can read the key.

## Configuration

Each setting is resolved in this order; the first match wins.

**API URL**

1. `apiUrl` in `window.__SPARROW_CONFIG__`, from `static/config.js` on the static host (or injected by the Go server). Read at runtime, so no rebuild is needed.
2. `PUBLIC_API_URL` at `vite build` / `vite dev` time. Baked into the bundle.
3. Built-in default: `http://localhost:8080` under `npm run dev`, same origin for builds.

**Credential**

1. A credential stored in this browser (`localStorage`, remembered until you sign out):
   - an access token from an invite link (`/#invite=…`);
   - a browser token created when someone pastes the master key into the sign-in prompt (the master key itself is never stored);
   - or a token pasted into the prompt as is.
2. `apiKey` in `window.__SPARROW_CONFIG__`: injected by the Go server when it serves the UI (unless `SPARROW_UI_INJECT_KEY=false`), or set in `config.js`.

`npm run build` respects `PUBLIC_API_URL` from the environment or `web/.env`. `make build-ui` and the Dockerfile set `PUBLIC_API_URL=/` explicitly, so a local `.env` can't leak into the embedded build.

## Tech Stack

- **SvelteKit 2** + **Svelte 5** (Runes)
- **Vite 7**
- **Tailwind CSS 4** (via Vite plugin)
- **openapi-fetch** (typed REST client, generated from `api/openapi.yaml`)
- **Static adapter** (embedded in Go binary via `go:embed`)
